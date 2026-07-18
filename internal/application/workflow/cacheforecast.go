// cacheforecast.go answers "what will actually run if I submit this?" before
// the submission happens — a biopsy rather than an autopsy.
//
// It never recomputes Cromwell's hashes. Instead it compares the pending
// submission against a previous run along the axes Cromwell fingerprints, using
// two properties verified against a real server (docs/design/cache-explainer.md):
// the command template is hashed before input substitution, so comparing WDL
// text is equivalent to comparing that hash; and File inputs are hashed by
// content, so a file's MD5 can be compared against the hash the reference run
// recorded.
package workflow

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/lmtani/pumbaa/internal/application"
	"github.com/lmtani/pumbaa/internal/application/ports"
	domain "github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/pkg/wdl"
)

// md5HexLength is the length of the MD5 hashes Cromwell records for files under
// content-based hashing. A reference hash of any other length came from a
// different algorithm (a GCS crc32c, say) and must not be compared to an MD5.
const (
	md5HexLength    = 32
	crc32ByteLength = 4
)

// CacheForecastUseCase predicts which calls of a pending submission will be
// served from the call cache.
type CacheForecastUseCase struct {
	reader  ports.WorkflowMetadataReader
	fetcher ports.WorkflowMetadataFetcher
	querier ports.WorkflowQuerier
	files   ports.FileProvider

	// commandHashTrusted records whether this run's reference confirmed that we
	// reproduce Cromwell's command-template hash. It is set per Execute call,
	// not configured.
	commandHashTrusted bool
}

// NewCacheForecastUseCase builds the use case.
//
// fetcher supplies metadata with subworkflows expanded, which is what makes
// calls inside a subworkflow comparable; without it the forecast falls back to
// flat metadata and reports those calls as undetermined. querier may be nil, in
// which case a reference run must be named explicitly.
func NewCacheForecastUseCase(
	reader ports.WorkflowMetadataReader,
	fetcher ports.WorkflowMetadataFetcher,
	querier ports.WorkflowQuerier,
	files ports.FileProvider,
) *CacheForecastUseCase {
	return &CacheForecastUseCase{reader: reader, fetcher: fetcher, querier: querier, files: files}
}

// CacheForecastInput describes the submission to forecast.
type CacheForecastInput struct {
	WorkflowFile string
	InputsFile   string
	// DependenciesFile is an imports zip. Without it, imports are resolved
	// from WDL files sitting next to the workflow, and anything still missing
	// is reported as undetermined rather than assumed unchanged.
	DependenciesFile string
	// ReferenceID names the run to compare against. When empty, the most
	// recent successful run of the same workflow is used.
	ReferenceID string
}

// Execute produces the forecast.
func (uc *CacheForecastUseCase) Execute(ctx context.Context, input CacheForecastInput) (*domain.CacheForecast, error) {
	if input.WorkflowFile == "" {
		return nil, application.NewInputValidationError("workflowFile", "is required")
	}

	source, err := uc.files.ReadBytes(ctx, input.WorkflowFile)
	if err != nil {
		return nil, application.NewUseCaseError("cache forecast", "failed to read workflow file", err)
	}

	deps, depErr := uc.resolveImportSources(ctx, input)

	graph, err := wdl.BuildCallGraphWithSources(source, deps)
	if err != nil {
		return nil, application.NewUseCaseError("cache forecast", "failed to parse workflow", err)
	}
	specs := map[string]wdl.TaskSpec{}
	if s, err := wdl.TaskSpecsWithSources(source, deps); err == nil {
		specs = s
	}

	pendingInputs, err := uc.readInputs(ctx, input.InputsFile)
	if err != nil {
		return nil, err
	}

	reference, err := uc.resolveReference(ctx, input.ReferenceID, graph.Workflow)
	if err != nil {
		return nil, err
	}

	forecast := &domain.CacheForecast{Reference: reference.ID}
	if depErr != "" {
		forecast.Warnings = append(forecast.Warnings, depErr)
	}
	if names := unresolvedCalls(graph); len(names) > 0 {
		forecast.Warnings = append(forecast.Warnings, fmt.Sprintf(
			"could not read the definition of %s; pass --dependencies with the imports zip to cover them",
			strings.Join(names, ", ")))
	}

	forecast.Backend = referenceBackend(reference)
	if !forecast.Backend.Supported() {
		forecast.Warnings = append(forecast.Warnings, fmt.Sprintf(
			"reference run used an unsupported backend (%s); cache prediction covers local and GCP only",
			backendLabel(reference)))
		forecast.Calls = allUnknown(graph, "backend not supported")
		return forecast, nil
	}

	refSpecs := uc.referenceTaskSpecs(reference)
	uc.commandHashTrusted = calibrateCommandHash(graph, specs, reference)
	if !uc.commandHashTrusted {
		forecast.Warnings = append(forecast.Warnings,
			"could not reproduce this server's command-template hash, so command changes are not reported; "+
				"input and docker changes still are")
	}

	changed := make(map[string][]string)
	unknown := make(map[string]string)
	var assumed []string
	for _, name := range graph.Names() {
		node := graph.Nodes[name]
		reasons, why := uc.assessCall(ctx, node, specs, refSpecs, reference, pendingInputs, &assumed)
		if why != "" {
			unknown[name] = why
			continue
		}
		if len(reasons) > 0 {
			changed[name] = reasons
		}
	}

	if len(assumed) > 0 {
		sort.Strings(assumed)
		forecast.Warnings = append(forecast.Warnings, fmt.Sprintf(
			"not compared, and assumed unchanged: %s",
			strings.Join(assumed, ", ")))
	}

	forecast.Calls = domain.PredictCacheReuse(graph.Dependencies(), changed, unknown)
	return forecast, nil
}

// assessCall decides whether one call's own fingerprint changed. It returns the
// reasons it will rerun, or a non-empty why when the call cannot be judged.
func (uc *CacheForecastUseCase) assessCall(
	ctx context.Context,
	node *wdl.CallNode,
	specs, refSpecs map[string]wdl.TaskSpec,
	reference *domain.Workflow,
	pendingInputs map[string]any,
	assumed *[]string,
) (reasons []string, why string) {
	if node.Unresolved {
		if node.Subworkflow {
			return nil, "subworkflow source not available"
		}
		// The task body is invisible, so a command or docker change would go
		// undetected. Predicting reuse here would be the dangerous direction.
		return nil, "task definition not available"
	}

	refCall, ok := findReferenceCall(reference, node.Name)
	if !ok {
		// A call the reference never ran has nothing to be reused from.
		return []string{"not present in the reference run"}, ""
	}
	if len(refCall.Fingerprint) == 0 {
		return nil, "reference run recorded no cache hashes"
	}

	spec, hasSpec := specs[node.Task]
	if hasSpec {
		// The command template is compared against the hash the reference
		// recorded, not against its WDL text: a run's metadata carries only the
		// top-level workflow, so a task inside an imported subworkflow has no
		// reference text to compare with — but it always has a fingerprint.
		recordedCommand := refCall.Fingerprint["command template"]
		computed, canonical := spec.CommandHash()
		switch {
		case !uc.commandHashTrusted || recordedCommand == "":
		case !canonical:
			// The command interpolates an expression whose canonical form we
			// cannot reproduce, so a mismatch would be our rendering, not a
			// change. Skipping is the only honest option.
			*assumed = appendUnique(*assumed, node.Task+" (command)")
		case computed != recordedCommand:
			reasons = append(reasons, "command template changed")
		}
		if refSpec, ok := refSpecs[node.Task]; ok {
			if r := compareDocker(spec, refSpec); r != "" {
				reasons = append(reasons, r)
			}
		}
	}

	inputReasons, inputWhy := uc.compareInputs(ctx, node, refCall, pendingInputs, specs, assumed)
	if inputWhy != "" {
		return nil, inputWhy
	}
	reasons = append(reasons, inputReasons...)
	sort.Strings(reasons)
	return reasons, ""
}

// compareInputs checks every input of a call against the value the reference
// run used.
//
// The list of inputs comes from the reference call rather than from the WDL's
// explicit call bindings, because Cromwell also accepts call-scoped overrides
// in the inputs JSON ("Workflow.Call.docker"). Those never appear as bindings,
// and missing them would silently predict reuse for a submission that changed
// a task's image.
func (uc *CacheForecastUseCase) compareInputs(
	ctx context.Context,
	node *wdl.CallNode,
	refCall domain.Call,
	pendingInputs map[string]any,
	specs map[string]wdl.TaskSpec,
	assumed *[]string,
) (reasons []string, why string) {
	// Only inputs the reference actually fingerprinted can affect reuse. A task
	// may receive more than that (Cromwell renames some private declarations),
	// and comparing those would invent differences the cache never sees.
	names := fingerprintInputNames(refCall.Fingerprint)

	for _, inputName := range names {
		binding, bound := node.Bindings[inputName]
		if bound && binding.Kind == wdl.BindingCall {
			// Handled by cascade propagation, not here.
			continue
		}

		pending, ok := uc.pendingValue(binding, node, inputName, pendingInputs, specs)
		if !ok {
			// The value is computed by the WDL rather than supplied — a disk
			// size derived from an input's size, say. It is a deterministic
			// function of the task definition and the other inputs, both of
			// which are compared, so it is assumed to follow them rather than
			// being treated as unknowable. Assumptions are reported, because
			// editing such an expression without touching the command would
			// slip past.
			_, declaredType := declaredTypeOf(refCall.Fingerprint, inputName)
			if strings.Contains(declaredType, "File") {
				// A file we cannot even name is genuine uncertainty about data.
				return nil, fmt.Sprintf("input %q has no resolvable path", inputName)
			}
			*assumed = appendUnique(*assumed, inputName)
			continue
		}

		hashKey, isFile := fileHashKey(refCall.Fingerprint, inputName)
		if isFile {
			same, err := uc.sameFile(ctx, pending, refCall.Fingerprint[hashKey])
			if err != nil {
				return nil, fmt.Sprintf("input %q: %v", inputName, err)
			}
			if !same {
				reasons = append(reasons, fmt.Sprintf("input file %q changed", inputName))
			}
			continue
		}

		refValue, ok := refCall.Inputs[inputName]
		if !ok {
			continue
		}
		if valueString(refValue) != pending {
			reasons = append(reasons, fmt.Sprintf("input %q changed", inputName))
		}
	}
	return reasons, ""
}

// pendingValue resolves what the submission will pass for one call input.
func (uc *CacheForecastUseCase) pendingValue(
	binding wdl.CallBinding,
	node *wdl.CallNode,
	inputName string,
	pendingInputs map[string]any,
	specs map[string]wdl.TaskSpec,
) (string, bool) {
	if binding.Kind == wdl.BindingLiteral {
		return binding.Literal, true
	}
	// An input a subworkflow declares but its caller does not pass must be
	// supplied qualified by the call path, so look there and nowhere else —
	// a top-level input of the same name is a different value.
	if binding.Kind == wdl.BindingWorkflowInput && binding.Scope != "" {
		if v, ok := lookupCallInput(pendingInputs, binding.Scope, binding.Source); ok {
			return valueString(v), true
		}
		return "", false
	}
	// A workflow-level input: the inputs JSON wins, then the WDL default.
	if binding.Kind == wdl.BindingWorkflowInput {
		if v, ok := lookupWorkflowInput(pendingInputs, binding.Source); ok {
			return valueString(v), true
		}
	}
	// Cromwell accepts call-scoped overrides ("WF.Call.input") only for calls
	// of the top-level workflow. A task inside a subworkflow cannot be
	// addressed that way — Cromwell rejects the submission outright — so the
	// override is not consulted for nested calls, whose values come from the
	// subworkflow's own WDL.
	if !strings.Contains(node.Name, ".") {
		if v, ok := lookupCallInput(pendingInputs, node.Name, inputName); ok {
			return valueString(v), true
		}
	}
	if spec, ok := specs[node.Task]; ok {
		if def, ok := spec.InputDefaults[inputName]; ok {
			return def, true
		}
	}
	return "", false
}

// sameFile reports whether the candidate path holds the same bytes the
// reference run hashed.
//
// Which digest to compare is decided by the shape of the recorded hash rather
// than by the backend: a local Cromwell records a 32-character MD5, while GCS
// records a crc32c as base64 of four bytes. Reading it from the hash itself
// means a deployment that hashes differently is detected rather than assumed.
func (uc *CacheForecastUseCase) sameFile(ctx context.Context, path, referenceHash string) (bool, error) {
	kind := classifyFileHash(referenceHash)
	if kind == hashUnrecognised {
		return false, errors.New("reference hash is in an unrecognised format, cannot compare content")
	}

	digests, err := uc.files.GetContentDigests(ctx, path)
	if err != nil {
		if errors.Is(err, ports.ErrFileNotFound) {
			// A missing input is a submission problem, not a cache question;
			// preflight reports it. Here it simply cannot be compared.
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

// classifyFileHash infers the algorithm from the encoding Cromwell stored.
func classifyFileHash(h string) fileHashKind {
	if len(h) == md5HexLength && isHex(h) {
		return hashMD5
	}
	// GCS crc32c: four bytes, base64 encoded — "tBGf4Q==".
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

// referenceTaskSpecs parses the WDL the reference run was submitted with. It
// covers only the top-level workflow — metadata never carries imported files —
// so it is a bonus source for the docker comparison, not a requirement.
func (uc *CacheForecastUseCase) referenceTaskSpecs(reference *domain.Workflow) map[string]wdl.TaskSpec {
	if reference.SubmittedWorkflow == "" {
		return nil
	}
	specs, err := wdl.TaskSpecs([]byte(reference.SubmittedWorkflow))
	if err != nil {
		return nil
	}
	return specs
}

// calibrateCommandHash checks that we reproduce this server's command-template
// hash before any mismatch is reported as a change.
//
// The normalisation Cromwell applies is not a documented contract, so a version
// that changed it would otherwise make every task look modified. Finding a
// single call whose computed hash matches its recorded one proves the formula
// holds here; finding none means a mismatch says more about us than about the
// submission, and the axis is dropped rather than reported wrongly.
func calibrateCommandHash(graph *wdl.CallGraph, specs map[string]wdl.TaskSpec, reference *domain.Workflow) bool {
	compared := 0
	for _, name := range graph.Names() {
		node := graph.Nodes[name]
		spec, ok := specs[node.Task]
		if !ok {
			continue
		}
		refCall, ok := findReferenceCall(reference, node.Name)
		if !ok {
			continue
		}
		recorded := refCall.Fingerprint["command template"]
		if recorded == "" {
			continue
		}
		computed, canonical := spec.CommandHash()
		if !canonical {
			continue
		}
		compared++
		if computed == recorded {
			return true
		}
	}
	// Nothing to calibrate against: withhold the axis rather than assume the
	// formula holds. Such a reference has no fingerprints either, so those
	// calls are already reported as undetermined.
	_ = compared
	return false
}

// fetchReference reads a run's metadata with subworkflows expanded, so calls
// inside a subworkflow carry their fingerprints. It falls back to flat metadata
// when no expanding fetcher is wired or the expanded read fails; those calls
// then come out as undetermined rather than wrong.
func (uc *CacheForecastUseCase) fetchReference(ctx context.Context, id string) (*domain.Workflow, error) {
	if uc.fetcher != nil {
		raw, err := uc.fetcher.GetRawMetadataWithOptions(ctx, id, true)
		if err == nil {
			if w, err := uc.fetcher.ParseMetadata(raw); err == nil {
				return w, nil
			}
		}
	}
	return uc.reader.GetMetadata(ctx, id)
}

// resolveReference finds the run to compare against.
func (uc *CacheForecastUseCase) resolveReference(ctx context.Context, id, workflowName string) (*domain.Workflow, error) {
	if id != "" {
		w, err := uc.fetchReference(ctx, id)
		if err != nil {
			return nil, application.NewUseCaseError("cache forecast", "failed to fetch reference run", err)
		}
		return w, nil
	}
	if uc.querier == nil {
		return nil, application.NewInputValidationError("reference", "is required when no querier is configured")
	}

	result, err := uc.querier.Query(ctx, domain.QueryFilter{
		Name:     workflowName,
		Status:   []domain.Status{domain.StatusSucceeded},
		PageSize: 1,
	})
	if err != nil {
		return nil, application.NewUseCaseError("cache forecast", "failed to look for a reference run", err)
	}
	if result == nil || len(result.Workflows) == 0 {
		return nil, application.NewUseCaseError("cache forecast",
			fmt.Sprintf("no successful previous run of %q to compare against", workflowName), nil)
	}
	w, err := uc.fetchReference(ctx, result.Workflows[0].ID)
	if err != nil {
		return nil, application.NewUseCaseError("cache forecast", "failed to fetch reference run", err)
	}
	return w, nil
}

func (uc *CacheForecastUseCase) readInputs(ctx context.Context, path string) (map[string]any, error) {
	if path == "" {
		return map[string]any{}, nil
	}
	data, err := uc.files.ReadBytes(ctx, path)
	if err != nil {
		return nil, application.NewUseCaseError("cache forecast", "failed to read inputs file", err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, application.NewUseCaseError("cache forecast", "inputs file is not valid JSON", err)
	}
	return out, nil
}

// valueString renders an inputs-JSON scalar the way a comparison needs it.
// Numbers arrive from encoding/json as float64, so an integer must not render
// as "1e+06" and compare unequal to the reference's "1000000".
func valueString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", t)
	}
}

// fingerprintInputNames lists the input names the reference fingerprinted,
// sorted, since only those participate in the cache key.
func fingerprintInputNames(fp domain.CallFingerprint) []string {
	var out []string
	for key := range fp {
		if _, name := domain.ParseInputHashKey(key); name != "" {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// declaredTypeOf returns the fingerprint key and declared WDL type of an input.
func declaredTypeOf(fp domain.CallFingerprint, inputName string) (key, declaredType string) {
	for k := range fp {
		typ, name := domain.ParseInputHashKey(k)
		if name == inputName {
			return k, typ
		}
	}
	return "", ""
}

func appendUnique(list []string, v string) []string {
	for _, existing := range list {
		if existing == v {
			return list
		}
	}
	return append(list, v)
}

// compareDocker reports a docker change, or "" when the images match or cannot
// both be resolved from the WDL.
func compareDocker(pending, reference wdl.TaskSpec) string {
	p, pok := pending.DockerValue()
	r, rok := reference.DockerValue()
	if !pok || !rok || p == r {
		return ""
	}
	return fmt.Sprintf("docker image changed (%s → %s)", r, p)
}

// findReferenceCall locates a call in the reference run by its path in the
// flattened graph ("AlignSample.Align").
//
// Metadata keys are qualified by the enclosing workflow ("Cohort.AlignSample"),
// and a subworkflow's calls live under that call's own metadata rather than at
// the top level, so each path segment is resolved one level down.
func findReferenceCall(w *domain.Workflow, path string) (domain.Call, bool) {
	return findReferenceCallPath(w, strings.Split(path, "."))
}

func findReferenceCallPath(w *domain.Workflow, segments []string) (domain.Call, bool) {
	if w == nil || len(segments) == 0 {
		return domain.Call{}, false
	}
	call, ok := lookupCallByName(w, segments[0])
	if !ok {
		return domain.Call{}, false
	}
	if len(segments) == 1 {
		return call, true
	}
	return findReferenceCallPath(call.SubWorkflowMetadata, segments[1:])
}

// lookupCallByName finds a call by its unqualified name within one workflow,
// choosing the latest attempt because that is the one whose fingerprint decided
// reuse.
func lookupCallByName(w *domain.Workflow, name string) (domain.Call, bool) {
	for key, calls := range w.Calls {
		if len(calls) == 0 {
			continue
		}
		if key != name && !strings.HasSuffix(key, "."+name) {
			continue
		}
		best := calls[0]
		for _, c := range calls[1:] {
			if c.Attempt > best.Attempt {
				best = c
			}
		}
		return best, true
	}
	return domain.Call{}, false
}

// unresolvedCalls lists calls whose definition could not be read, so the user
// is told exactly what to bundle rather than just seeing "undetermined".
func unresolvedCalls(graph *wdl.CallGraph) []string {
	var out []string
	for _, name := range graph.Names() {
		if graph.Nodes[name].Unresolved {
			out = append(out, name)
		}
	}
	return out
}

// resolveImportSources gathers the WDL sources needed to see inside imports:
// the dependencies zip when given, otherwise the WDL files sitting beside the
// workflow, which is how a workflow run from a checkout resolves. It returns a
// warning string rather than an error — a forecast without imports is degraded,
// not impossible.
func (uc *CacheForecastUseCase) resolveImportSources(ctx context.Context, input CacheForecastInput) (wdl.SourceSet, string) {
	if input.DependenciesFile != "" {
		data, err := uc.files.ReadBytes(ctx, input.DependenciesFile)
		if err != nil {
			return nil, fmt.Sprintf("could not read the dependencies zip: %v", err)
		}
		sources, err := wdl.SourcesFromZip(data)
		if err != nil {
			return nil, fmt.Sprintf("could not read the dependencies zip: %v", err)
		}
		return sources, ""
	}

	// Only local paths have a directory to scan; a remote WDL must be bundled.
	if strings.Contains(input.WorkflowFile, "://") {
		return nil, ""
	}
	sources, err := wdl.SourcesFromDir(filepath.Dir(input.WorkflowFile))
	if err != nil {
		return nil, ""
	}
	return sources, ""
}

// fileHashKey finds the fingerprint entry for a File input, returning the key
// and whether the input is a File at all.
func fileHashKey(fp domain.CallFingerprint, inputName string) (string, bool) {
	for key := range fp {
		declaredType, name := domain.ParseInputHashKey(key)
		if name != inputName {
			continue
		}
		return key, strings.Contains(declaredType, "File")
	}
	return "", false
}

// referenceBackend reports the backend the reference run used, treating a
// disagreement between calls as unsupported rather than picking one.
func referenceBackend(w *domain.Workflow) domain.BackendKind {
	var found domain.BackendKind
	seen := false
	for _, calls := range w.Calls {
		for _, c := range calls {
			if c.Backend == "" {
				continue
			}
			kind := domain.ClassifyBackend(c.Backend)
			if !seen {
				found, seen = kind, true
				continue
			}
			if kind != found {
				return domain.BackendUnsupported
			}
		}
	}
	if !seen {
		return domain.BackendUnsupported
	}
	return found
}

func backendLabel(w *domain.Workflow) string {
	for _, calls := range w.Calls {
		for _, c := range calls {
			if c.Backend != "" {
				return c.Backend
			}
		}
	}
	return "unknown"
}

func allUnknown(graph *wdl.CallGraph, why string) []domain.CallPrediction {
	names := graph.Names()
	out := make([]domain.CallPrediction, 0, len(names))
	for _, n := range names {
		out = append(out, domain.CallPrediction{Call: n, Fate: domain.FateUnknown, Reasons: []string{why}})
	}
	return out
}

// lookupWorkflowInput finds a workflow-level input by its unqualified name,
// accepting both "name" and "Workflow.name" spellings.
func lookupWorkflowInput(inputs map[string]any, name string) (any, bool) {
	if v, ok := inputs[name]; ok {
		return v, true
	}
	for key, v := range inputs {
		if strings.HasSuffix(key, "."+name) && strings.Count(key, ".") == 1 {
			return v, true
		}
	}
	return nil, false
}

// lookupCallInput finds a call-scoped override, "Workflow.Call.input".
func lookupCallInput(inputs map[string]any, call, input string) (any, bool) {
	suffix := "." + call + "." + input
	for key, v := range inputs {
		if strings.HasSuffix(key, suffix) {
			return v, true
		}
	}
	return nil, false
}
