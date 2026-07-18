package workflow

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/lmtani/pumbaa/internal/application/ports"
	domain "github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/pkg/wdl"
)

// inputComparison decides, for one call, whether its inputs are what the
// reference recorded.
//
// The rule it implements has two halves, and both are needed:
//
//  1. **Explanation.** At least one leaf of the pending expression must account
//     for the value the reference recorded. If none does, the reference read
//     that input from somewhere this program no longer reads — the wiring
//     changed — and no amount of checking the leaves we *do* see can establish
//     that the input is unchanged.
//  2. **Stability.** Every leaf must then be unchanged. This covers the
//     predicate of a conditional as much as its branches: a predicate that
//     flips selects a different leaf, so an input can differ from the recorded
//     value even when both branches are individually untouched.
//
// Half 1 without half 2 misses a flipped branch; half 2 without half 1 misses a
// rewiring. Together they are what stands between the forecast and a false
// claim of reuse.
type inputComparison struct {
	files     ports.FileProvider
	reference *domain.Workflow
	// referenceParams is the parameter document the reference was submitted
	// with, which is how a parameter leaf is checked for stability.
	referenceParams map[string]any
	pendingParams   map[string]any
	specs           map[string]wdl.TaskSpec
}

// inputOutcome is what could be established about a single input.
type inputOutcome int

const (
	// outcomeUnchanged: verified equal to what the reference recorded.
	outcomeUnchanged inputOutcome = iota
	// outcomeChanged: verified different, so the call reruns on its own account.
	outcomeChanged
	// outcomeDeferred: rests on a producing call; the graph decides.
	outcomeDeferred
	// outcomeUnverifiable: nothing could be established, so reuse is off the
	// table even though nothing was found wrong.
	outcomeUnverifiable
)

// callInputs is the verdict over all of one call's inputs.
type callInputs struct {
	// Changed lists the inputs proven different, in the user's terms.
	Changed []string
	// Blocked, when non-empty, is why reuse cannot be established.
	Blocked string
	// Assumed lists inputs taken on trust because the program computes them.
	Assumed []string
}

// compare walks every input the reference fingerprinted. Inputs it did not
// fingerprint play no part in the cache key and are ignored.
func (c inputComparison) compare(ctx context.Context, node *wdl.CallNode, refCall domain.Call) callInputs {
	var out callInputs

	for _, inputName := range fingerprintInputNames(refCall.Fingerprint) {
		binding, bound := node.Bindings[inputName]
		if !bound {
			// Not written at the call site: either the program computes it, or
			// the submission supplies it under a call-scoped key.
			c.compareUnbound(ctx, node, refCall, inputName, &out)
			continue
		}

		switch outcome, detail := c.compareBinding(ctx, binding, refCall, inputName); outcome {
		case outcomeChanged:
			out.Changed = append(out.Changed, detail)
		case outcomeUnverifiable:
			if out.Blocked == "" {
				out.Blocked = detail
			}
		case outcomeUnchanged, outcomeDeferred:
		}
	}

	sort.Strings(out.Changed)
	return out
}

// compareBinding applies the two-part rule to one resolved input.
func (c inputComparison) compareBinding(
	ctx context.Context,
	binding wdl.ResolvedBinding,
	refCall domain.Call,
	inputName string,
) (inputOutcome, string) {
	if !binding.Complete {
		return outcomeUnverifiable, fmt.Sprintf("input %q: %s", inputName, binding.Incomplete)
	}

	recorded, ok := refCall.Inputs[inputName]
	if !ok {
		return outcomeUnverifiable, fmt.Sprintf("input %q was not recorded by the reference run", inputName)
	}
	recordedValue := valueString(recorded)

	// When the pending value can be computed outright, comparing it against the
	// recorded one settles the question on its own: equal values are equal
	// whatever expression produced them, so how the input was wired does not
	// matter. The two-part rule below is for the values we cannot compute.
	if value, ok := c.directValue(binding); ok {
		return c.compareValue(ctx, value, recordedValue, refCall, inputName)
	}

	// Half 1: something this program reads must account for the recorded value.
	if !c.explains(binding, recordedValue) {
		return outcomeUnverifiable, fmt.Sprintf(
			"input %q was wired differently in the reference run, so its value cannot be compared", inputName)
	}

	// Half 2: every leaf must be stable, including any that steers a choice.
	deferred := false
	for _, source := range binding.Sources {
		switch source.Kind {
		case wdl.SourceCall:
			deferred = true
		case wdl.SourceLiteral:
			// A literal is only known to match what the reference used when it
			// is what the reference recorded; the reference's own text is not
			// available for composites.
			if source.Literal != recordedValue {
				return outcomeUnverifiable, fmt.Sprintf(
					"input %q reads a literal this run cannot check against the reference", inputName)
			}
		case wdl.SourceInput:
			changed, detail := c.parameterChanged(ctx, source, refCall, inputName)
			if detail != "" && !changed {
				return outcomeUnverifiable, detail
			}
			if changed {
				return outcomeChanged, detail
			}
		}
	}

	if deferred {
		return outcomeDeferred, ""
	}
	return outcomeUnchanged, ""
}

// directValue computes the pending value of a binding when it does not depend
// on any producing call — a lone literal, or a lone parameter read from the
// submission. Anything richer has no single value until the run happens.
func (c inputComparison) directValue(binding wdl.ResolvedBinding) (string, bool) {
	if len(binding.Sources) != 1 {
		return "", false
	}
	source := binding.Sources[0]
	switch source.Kind {
	case wdl.SourceLiteral:
		return source.Literal, true
	case wdl.SourceInput:
		if v, ok := c.lookupParam(c.pendingParams, source); ok {
			return valueString(v), true
		}
	}
	return "", false
}

// compareValue settles an input whose pending value is known, falling back to
// content when the paths differ: the engine hashes what a file holds, not where
// it sits, so a file moved with its bytes intact is still the same input.
func (c inputComparison) compareValue(
	ctx context.Context,
	pending, recorded string,
	refCall domain.Call,
	inputName string,
) (inputOutcome, string) {
	// A file is decided by its content, never by its path — in both directions.
	// The same path may hold different bytes after an overwrite, and different
	// paths may hold identical bytes after a move; the engine sees only the
	// content, so a path comparison would be wrong either way round.
	if hash := c.recordedFileHash(refCall.Fingerprint, inputName); hash != "" {
		same, err := sameContent(ctx, c.files, pending, hash)
		if err != nil {
			return outcomeUnverifiable, fmt.Sprintf("input %q: %v", inputName, err)
		}
		if same {
			return outcomeUnchanged, ""
		}
		return outcomeChanged, fmt.Sprintf("input file %q changed", inputName)
	}

	if pending == recorded {
		return outcomeUnchanged, ""
	}
	return outcomeChanged, fmt.Sprintf("input %q changed", inputName)
}

// explains reports whether some leaf, evaluated with the reference's own
// values, yields the value the reference recorded for this input.
func (c inputComparison) explains(binding wdl.ResolvedBinding, recordedValue string) bool {
	for _, source := range binding.Sources {
		switch source.Kind {
		case wdl.SourceLiteral:
			if source.Literal == recordedValue {
				return true
			}
		case wdl.SourceInput:
			if v, ok := c.lookupParam(c.referenceParams, source); ok && valueString(v) == recordedValue {
				return true
			}
		case wdl.SourceCall:
			if c.producedBy(source.Name, recordedValue) {
				return true
			}
		}
	}
	// A single-leaf binding on a producing call whose outputs the reference did
	// not record cannot be checked either way; treat it as explained so the
	// graph still governs it, rather than blocking every such call.
	return len(binding.Sources) == 1 &&
		binding.Sources[0].Kind == wdl.SourceCall &&
		!c.hasRecordedOutputs(binding.Sources[0].Name)
}

// producedBy reports whether the given call recorded the value as one of its
// outputs — the evidence that an input really was fed by that call.
func (c inputComparison) producedBy(callPath, value string) bool {
	call, ok := findReferenceCall(c.reference, callPath)
	if !ok {
		return false
	}
	for _, out := range call.Outputs {
		if valueString(out) == value {
			return true
		}
	}
	return false
}

func (c inputComparison) hasRecordedOutputs(callPath string) bool {
	call, ok := findReferenceCall(c.reference, callPath)
	return ok && len(call.Outputs) > 0
}

// parameterChanged compares one parameter leaf between the two submissions.
// A file is compared by content; anything else by value.
func (c inputComparison) parameterChanged(
	ctx context.Context,
	source wdl.ValueSource,
	refCall domain.Call,
	inputName string,
) (changed bool, detail string) {
	pending, pendingOK := c.lookupParam(c.pendingParams, source)
	referenceValue, refOK := c.lookupParam(c.referenceParams, source)
	if !pendingOK || !refOK {
		// One side does not supply it: an optional that appeared or vanished
		// changes which branch a conditional takes, so this is a difference,
		// not an absence of information.
		if pendingOK != refOK {
			return true, fmt.Sprintf("input %q changed (%s was %s)", inputName, source.Name,
				presence(refOK))
		}
		return false, fmt.Sprintf("input %q reads %s, which neither run supplies", inputName, source.Name)
	}

	pendingValue, refValue := valueString(pending), valueString(referenceValue)
	if pendingValue == refValue {
		return false, ""
	}

	// Different paths may still hold identical bytes, which the engine treats
	// as the same input.
	if hash := c.recordedFileHash(refCall.Fingerprint, inputName); hash != "" {
		same, err := sameContent(ctx, c.files, pendingValue, hash)
		if err == nil && same {
			return false, ""
		}
		if err != nil {
			return false, fmt.Sprintf("input %q: %v", inputName, err)
		}
	}
	return true, fmt.Sprintf("input %q changed", inputName)
}

func presence(supplied bool) string {
	if supplied {
		return "supplied before and is not now"
	}
	return "not supplied before and is now"
}

// lookupParam finds a parameter in a submission document, honouring the call
// path an unbound subworkflow input must be qualified by.
func (c inputComparison) lookupParam(params map[string]any, source wdl.ValueSource) (any, bool) {
	if source.Scope != "" {
		return lookupCallInput(params, source.Scope, source.Name)
	}
	return lookupWorkflowInput(params, source.Name)
}

// compareUnbound handles an input the call site does not write: the program
// computes it, or the submission supplies it under a call-scoped key.
func (c inputComparison) compareUnbound(
	ctx context.Context,
	node *wdl.CallNode,
	refCall domain.Call,
	inputName string,
	out *callInputs,
) {
	pending, ok := c.pendingValue(node, inputName)
	if !ok {
		// Computed by the program. It is a deterministic function of the task
		// definition and the other inputs, both of which are compared, so it is
		// taken to follow them — except for a file, where not even the path is
		// known and there is nothing to stand on.
		if _, declared := declaredTypeOf(refCall.Fingerprint, inputName); strings.Contains(declared, "File") {
			if out.Blocked == "" {
				out.Blocked = fmt.Sprintf("input %q has no resolvable path", inputName)
			}
			return
		}
		out.Assumed = appendUnique(out.Assumed, inputName)
		return
	}

	recorded, recordedOK := refCall.Inputs[inputName]
	if !recordedOK {
		return
	}
	if valueString(recorded) == pending {
		return
	}
	if hash := c.recordedFileHash(refCall.Fingerprint, inputName); hash != "" {
		same, err := sameContent(ctx, c.files, pending, hash)
		if err != nil {
			if out.Blocked == "" {
				out.Blocked = fmt.Sprintf("input %q: %v", inputName, err)
			}
			return
		}
		if same {
			return
		}
	}
	out.Changed = append(out.Changed, fmt.Sprintf("input %q changed", inputName))
}

// pendingValue resolves an input the call site does not write, from a
// call-scoped override or the task's own default.
func (c inputComparison) pendingValue(node *wdl.CallNode, inputName string) (string, bool) {
	// Call-scoped overrides address calls of the top-level workflow only;
	// Cromwell rejects a submission that tries to address a nested one.
	if !strings.Contains(node.Name, ".") {
		if v, ok := lookupCallInput(c.pendingParams, node.Name, inputName); ok {
			return valueString(v), true
		}
	}
	if spec, ok := c.specs[node.Task]; ok {
		if def, ok := spec.InputDefaults[inputName]; ok {
			return def, true
		}
	}
	return "", false
}

// recordedFileHash returns the content digest the reference recorded for a file
// input, or "" when the input is not a file.
func (c inputComparison) recordedFileHash(fp domain.CallFingerprint, inputName string) string {
	key, declared := declaredTypeOf(fp, inputName)
	if !strings.Contains(declared, "File") {
		return ""
	}
	return fp[key]
}

// sameContent reports whether the file at path holds the bytes the reference
// hashed. The digest algorithm is read from the shape of the recorded hash, so
// a backend that hashes differently is detected rather than assumed.
func sameContent(ctx context.Context, files ports.FileProvider, path, referenceHash string) (bool, error) {
	kind := classifyFileHash(referenceHash)
	if kind == hashUnrecognised {
		return false, errors.New("reference hash is in an unrecognised format, cannot compare content")
	}

	digests, err := files.GetContentDigests(ctx, path)
	if err != nil {
		if errors.Is(err, ports.ErrFileNotFound) {
			return false, fmt.Errorf("file not found: %s", path)
		}
		return false, err
	}

	switch kind {
	case hashMD5:
		if digests.MD5 == "" {
			return false, errors.New("no MD5 available for this file")
		}
		return strings.EqualFold(digests.MD5, referenceHash), nil
	default:
		if digests.CRC32C == "" {
			return false, errors.New("no crc32c available for this file")
		}
		return digests.CRC32C == referenceHash, nil
	}
}

// fileHashKind is the digest algorithm a recorded file hash came from.
type fileHashKind int

const (
	hashUnrecognised fileHashKind = iota
	hashMD5
	hashCRC32C
)

// classifyFileHash infers the algorithm from the encoding the engine stored.
// A local backend records a 32-character MD5; GCS records a crc32c as the
// base64 of four bytes. Reading it from the hash itself means a deployment that
// hashes some third way is detected rather than assumed.
func classifyFileHash(h string) fileHashKind {
	if len(h) == md5HexLength && isHex(h) {
		return hashMD5
	}
	if raw, err := base64.StdEncoding.DecodeString(h); err == nil && len(raw) == crc32ByteLength {
		return hashCRC32C
	}
	return hashUnrecognised
}

func isHex(s string) bool {
	for _, c := range s {
		switch {
		case c >= '0' && c <= '9', c >= 'a' && c <= 'f', c >= 'A' && c <= 'F':
		default:
			return false
		}
	}
	return true
}
