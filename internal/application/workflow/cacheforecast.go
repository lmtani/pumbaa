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
	"encoding/json"
	"errors"
	"fmt"
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
const md5HexLength = 32

// CacheForecastUseCase predicts which calls of a pending submission will be
// served from the call cache.
type CacheForecastUseCase struct {
	reader  ports.WorkflowMetadataReader
	querier ports.WorkflowQuerier
	files   ports.FileProvider
}

// NewCacheForecastUseCase builds the use case. querier may be nil, in which
// case a reference run must be named explicitly.
func NewCacheForecastUseCase(reader ports.WorkflowMetadataReader, querier ports.WorkflowQuerier, files ports.FileProvider) *CacheForecastUseCase {
	return &CacheForecastUseCase{reader: reader, querier: querier, files: files}
}

// CacheForecastInput describes the submission to forecast.
type CacheForecastInput struct {
	WorkflowFile string
	InputsFile   string
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

	graph, err := wdl.BuildCallGraph(source)
	if err != nil {
		return nil, application.NewUseCaseError("cache forecast", "failed to parse workflow", err)
	}
	specs := map[string]wdl.TaskSpec{}
	if s, err := wdl.TaskSpecs(source); err == nil {
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

	forecast.Backend = referenceBackend(reference)
	if !forecast.Backend.Supported() {
		forecast.Warnings = append(forecast.Warnings, fmt.Sprintf(
			"reference run used an unsupported backend (%s); cache prediction covers local and GCP only",
			backendLabel(reference)))
		forecast.Calls = allUnknown(graph, "backend not supported")
		return forecast, nil
	}

	refSpecs := uc.referenceTaskSpecs(reference, forecast)

	changed := make(map[string][]string)
	unknown := make(map[string]string)
	for _, name := range graph.Names() {
		node := graph.Nodes[name]
		reasons, why := uc.assessCall(ctx, node, specs, refSpecs, reference, pendingInputs, forecast)
		if why != "" {
			unknown[name] = why
			continue
		}
		if len(reasons) > 0 {
			changed[name] = reasons
		}
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
	forecast *domain.CacheForecast,
) (reasons []string, why string) {
	if node.Subworkflow {
		return nil, "subworkflow internals not analysed"
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
	refSpec, hasRefSpec := refSpecs[node.Task]
	if hasSpec && hasRefSpec {
		if spec.Command != refSpec.Command {
			reasons = append(reasons, "command template changed")
		}
		if r := compareDocker(spec, refSpec); r != "" {
			reasons = append(reasons, r)
		}
	} else if !hasRefSpec {
		forecast.Warnings = append(forecast.Warnings,
			fmt.Sprintf("%s: reference WDL unavailable, compared inputs only", node.Name))
	}

	inputReasons, inputWhy := uc.compareInputs(ctx, node, refCall, pendingInputs, specs)
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
) (reasons []string, why string) {
	names := make([]string, 0, len(refCall.Inputs))
	for n := range refCall.Inputs {
		names = append(names, n)
	}
	for n := range node.Bindings {
		if _, ok := refCall.Inputs[n]; !ok {
			names = append(names, n)
		}
	}
	sort.Strings(names)

	for _, inputName := range names {
		binding, bound := node.Bindings[inputName]
		switch {
		case bound && binding.Kind == wdl.BindingCall:
			// Handled by cascade propagation, not here.
			continue
		case bound && binding.Kind == wdl.BindingUnknown:
			return nil, fmt.Sprintf("input %q is not statically resolvable", inputName)
		}

		pending, ok := uc.pendingValue(binding, node, inputName, pendingInputs, specs)
		if !ok {
			// An input the reference recorded but this submission cannot
			// resolve statically (a private declaration, a scatter variable).
			return nil, fmt.Sprintf("input %q has no resolvable value", inputName)
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
	// A workflow-level input: the inputs JSON wins, then the WDL default.
	if v, ok := lookupWorkflowInput(pendingInputs, binding.Source); ok {
		return valueString(v), true
	}
	// Cromwell also accepts call-scoped overrides ("WF.Call.input").
	if v, ok := lookupCallInput(pendingInputs, node.Name, inputName); ok {
		return valueString(v), true
	}
	if spec, ok := specs[node.Task]; ok {
		if def, ok := spec.InputDefaults[inputName]; ok {
			return def, true
		}
	}
	return "", false
}

// sameFile reports whether the candidate path holds the same bytes the
// reference run hashed. Comparison only happens when the reference hash is an
// MD5; anything else means the backend hashed differently and the answer is
// genuinely unknown.
func (uc *CacheForecastUseCase) sameFile(ctx context.Context, path, referenceHash string) (bool, error) {
	if len(referenceHash) != md5HexLength {
		return false, errors.New("reference hash is not an MD5, cannot compare content")
	}
	hash, err := uc.files.GetContentHash(ctx, path)
	if err != nil {
		if errors.Is(err, ports.ErrFileNotFound) {
			// A missing input is a submission problem, not a cache question;
			// preflight reports it. Here it simply cannot be compared.
			return false, fmt.Errorf("file not found: %s", path)
		}
		return false, err
	}
	return strings.EqualFold(hash, referenceHash), nil
}

// referenceTaskSpecs parses the WDL the reference run was submitted with, which
// is what makes "the command changed" answerable.
func (uc *CacheForecastUseCase) referenceTaskSpecs(reference *domain.Workflow, forecast *domain.CacheForecast) map[string]wdl.TaskSpec {
	if reference.SubmittedWorkflow == "" {
		forecast.Warnings = append(forecast.Warnings,
			"reference run did not record its WDL source; command and docker changes cannot be detected")
		return nil
	}
	specs, err := wdl.TaskSpecs([]byte(reference.SubmittedWorkflow))
	if err != nil {
		forecast.Warnings = append(forecast.Warnings,
			"reference WDL could not be parsed; command and docker changes cannot be detected")
		return nil
	}
	return specs
}

// resolveReference finds the run to compare against.
func (uc *CacheForecastUseCase) resolveReference(ctx context.Context, id, workflowName string) (*domain.Workflow, error) {
	if id != "" {
		w, err := uc.reader.GetMetadata(ctx, id)
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
	w, err := uc.reader.GetMetadata(ctx, result.Workflows[0].ID)
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

// findReferenceCall locates a call in the reference run by its unqualified
// name, since metadata keys are "Workflow.Call" while the graph uses "Call".
func findReferenceCall(w *domain.Workflow, callName string) (domain.Call, bool) {
	for key, calls := range w.Calls {
		if len(calls) == 0 {
			continue
		}
		if key == callName || strings.HasSuffix(key, "."+callName) {
			// The latest attempt carries the fingerprint that decided reuse.
			best := calls[0]
			for _, c := range calls[1:] {
				if c.Attempt > best.Attempt {
					best = c
				}
			}
			return best, true
		}
	}
	return domain.Call{}, false
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
