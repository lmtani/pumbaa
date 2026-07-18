// cacheforecast.go answers "what will actually run if I submit this?" before
// the submission happens — a biopsy rather than an autopsy.
//
// It never recomputes Cromwell's hashes. Instead it compares the pending
// submission against a previous run along the axes Cromwell fingerprints, using
// two properties verified against a real server, and pinned by the fixtures
// under internal/infrastructure/cromwell/testdata/callcache/:
// the command template is hashed before input substitution, so comparing WDL
// text is equivalent to comparing that hash; and File inputs are hashed by
// content, so a file's MD5 can be compared against the hash the reference run
// recorded.
package workflow

import (
	"context"
	"encoding/json"
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
	reader   ports.WorkflowMetadataReader
	fetcher  ports.WorkflowMetadataFetcher
	querier  ports.WorkflowQuerier
	files    ports.FileProvider
	progress ports.ProgressReporter

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
	progress ports.ProgressReporter,
) *CacheForecastUseCase {
	return &CacheForecastUseCase{
		reader: reader, fetcher: fetcher, querier: querier, files: files, progress: progress,
	}
}

// step reports a stage, tolerating the absence of a reporter.
func (uc *CacheForecastUseCase) step(format string, args ...any) {
	if uc.progress != nil {
		uc.progress.Step(format, args...)
	}
}

func (uc *CacheForecastUseCase) doneReporting() {
	if uc.progress != nil {
		uc.progress.Done()
	}
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

	defer uc.doneReporting()

	uc.step("reading the workflow")
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

	uc.step("fetching the reference run")
	reference, err := uc.resolveReference(ctx, input.ReferenceID, graph.Workflow, pendingInputs)
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

	// Inputs are hashed one object at a time, and each hash is a round trip. The
	// paths are known up front, so they are fetched together before the
	// comparison walks them one by one.
	digests := newDigestCache(uc.files)
	prefetch := pathsWorthPrefetching(pendingInputs)
	if len(prefetch) > 0 {
		uc.step("checking %d input file(s)", len(prefetch))
		digests.warm(ctx, prefetch)
	}

	uc.step("comparing %d call(s)", len(graph.Nodes))
	comparison := inputComparison{
		files:           digests,
		reference:       reference,
		referenceParams: parseParameters(reference.SubmittedInputs),
		pendingParams:   pendingInputs,
		inputDefaults:   graph.InputDefaults,
		specs:           specs,
	}

	assessments := make(map[string]domain.CallAssessment, len(graph.Nodes))
	var assumed []string
	for _, name := range graph.Names() {
		assessments[name] = uc.assessCall(ctx, graph.Nodes[name], specs, refSpecs, comparison, &assumed)
	}

	if len(assumed) > 0 {
		sort.Strings(assumed)
		forecast.Warnings = append(forecast.Warnings, fmt.Sprintf(
			"not compared, and assumed unchanged: %s",
			strings.Join(assumed, ", ")))
	}

	forecast.Calls = domain.PredictCacheReuse(graph.Dependencies(), assessments)
	return forecast, nil
}

// assessCall determines what can be established about one call on its own
// account, before the dependency graph is consulted.
func (uc *CacheForecastUseCase) assessCall(
	ctx context.Context,
	node *wdl.CallNode,
	specs map[string]wdl.TaskSpec,
	refSpecs map[string]wdl.TaskSpec,
	comparison inputComparison,
	assumed *[]string,
) domain.CallAssessment {
	if node.Unresolved {
		if node.Subworkflow {
			return domain.CallAssessment{Unknown: "subworkflow source not available"}
		}
		// The task body is invisible, so a command or docker change would go
		// undetected. Predicting reuse here would be the dangerous direction.
		return domain.CallAssessment{Unknown: "task definition not available"}
	}

	refCall, ok := findReferenceCall(comparison.reference, node.Name)
	if !ok {
		// A call the reference never ran has nothing to be reused from.
		return domain.CallAssessment{Reasons: []string{"not present in the reference run"}}
	}
	if len(refCall.Fingerprint) == 0 {
		return domain.CallAssessment{Unknown: "reference run recorded no cache hashes"}
	}

	var assessment domain.CallAssessment
	if spec, ok := specs[node.Task]; ok {
		assessment.Reasons = append(assessment.Reasons, uc.compareDefinition(spec, refSpecs[node.Task], refCall, node, assumed)...)
	}

	if node.Fanout != nil {
		uc.assessInstances(ctx, node, comparison, &assessment, assumed)
	} else {
		inputs := comparison.compare(ctx, node, refCall)
		assessment.Reasons = append(assessment.Reasons, inputs.Changed...)
		assessment.ReuseBlocked = inputs.Blocked
		for _, name := range inputs.Assumed {
			*assumed = appendUnique(*assumed, name)
		}
	}
	sort.Strings(assessment.Reasons)
	return assessment
}

// assessInstances compares a fan-out call one instance at a time.
//
// The engine caches instances individually, so a verdict for the call as a whole
// would be either too optimistic or too pessimistic whenever they disagree. The
// instances are paired by position: the pending element at position k against
// what the reference recorded for position k.
func (uc *CacheForecastUseCase) assessInstances(
	ctx context.Context,
	node *wdl.CallNode,
	comparison inputComparison,
	assessment *domain.CallAssessment,
	assumed *[]string,
) {
	instances := findReferenceInstances(comparison.reference, node.Name)
	if len(instances) == 0 {
		assessment.Unknown = "the reference run recorded no instances of this call"
		return
	}

	// A collection that grew brings positions the reference never ran.
	elements, _ := comparison.enumerateCollection(node.Fanout.Collection)
	split := domain.InstanceSplit{Total: max(len(instances), len(elements))}

	changed := make(map[string]bool)
	for position, refInstance := range instances {
		inputs := comparison.forInstance(position, node.Fanout.Collection).compare(ctx, node, refInstance)
		for _, name := range inputs.Assumed {
			*assumed = appendUnique(*assumed, name)
		}
		switch {
		case inputs.Blocked != "":
			// One unverifiable instance makes the whole split unknowable: the
			// call cannot be called reused, and counting it as rerun would be a
			// claim we did not establish.
			assessment.ReuseBlocked = inputs.Blocked
			assessment.Instances = nil
			return
		case len(inputs.Changed) > 0:
			for _, reason := range inputs.Changed {
				changed[reason] = true
			}
		default:
			split.Reused++
		}
	}

	if len(elements) > len(instances) {
		changed[fmt.Sprintf("%d new instance(s) in the collection", len(elements)-len(instances))] = true
	}

	assessment.Instances = &split
	if split.Reused < split.Total && len(changed) == 0 {
		changed["some instances differ"] = true
	}
	// A split with instances on both sides is partial reuse, which the domain
	// reads from Instances; only a wholly changed fan-out is a plain rerun.
	if split.Reused == 0 {
		for reason := range changed {
			assessment.Reasons = append(assessment.Reasons, reason)
		}
	}
}

// compareDefinition checks the parts of a task's definition that feed the
// fingerprint and can be read from the WDL.
func (uc *CacheForecastUseCase) compareDefinition(
	spec wdl.TaskSpec,
	refSpec wdl.TaskSpec,
	refCall domain.Call,
	node *wdl.CallNode,
	assumed *[]string,
) []string {
	var reasons []string

	// The command is compared against the digest the reference recorded rather
	// than against its text: a run's metadata carries only the top-level
	// workflow, so a task inside an imported unit has no reference text — but
	// it always has a fingerprint.
	recorded := refCall.Fingerprint["command template"]
	computed, canonical := spec.CommandHash()
	switch {
	case !uc.commandHashTrusted || recorded == "":
	case !canonical:
		// The command interpolates an expression whose canonical form we cannot
		// reproduce, so a mismatch would be our rendering, not a change.
		*assumed = appendUnique(*assumed, node.Task+" (command)")
	case computed != recorded:
		reasons = append(reasons, "command template changed")
	}

	if refSpec.Name != "" {
		if r := compareDocker(spec, refSpec); r != "" {
			reasons = append(reasons, r)
		}
	}
	return reasons
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

// parseParameters decodes a submission's parameter document, tolerating an
// absent or unreadable one: the comparison degrades to "cannot verify" rather
// than failing the whole forecast.
func parseParameters(document string) map[string]any {
	if document == "" {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(document), &out); err != nil {
		return nil
	}
	return out
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

// findReferenceInstances returns every instance the reference recorded for a
// call, ordered by position. A call that did not fan out yields its single
// instance, so callers need no special case.
func findReferenceInstances(w *domain.Workflow, path string) []domain.Call {
	segments := strings.Split(path, ".")
	parent := w
	for _, segment := range segments[:len(segments)-1] {
		call, ok := lookupCallByName(parent, segment)
		if !ok {
			return nil
		}
		parent = call.SubWorkflowMetadata
		if parent == nil {
			return nil
		}
	}
	return instancesByName(parent, segments[len(segments)-1])
}

// instancesByName collects a call's instances within one workflow, keeping the
// latest attempt of each position.
func instancesByName(w *domain.Workflow, name string) []domain.Call {
	if w == nil {
		return nil
	}
	for key, calls := range w.Calls {
		if key != name && !strings.HasSuffix(key, "."+name) {
			continue
		}
		best := make(map[int]domain.Call, len(calls))
		for _, c := range calls {
			if prev, ok := best[c.ShardIndex]; !ok || c.Attempt > prev.Attempt {
				best[c.ShardIndex] = c
			}
		}
		shards := make([]int, 0, len(best))
		for shard := range best {
			shards = append(shards, shard)
		}
		sort.Ints(shards)
		out := make([]domain.Call, 0, len(shards))
		for _, shard := range shards {
			out = append(out, best[shard])
		}
		return out
	}
	return nil
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
