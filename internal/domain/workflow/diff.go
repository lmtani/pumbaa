package workflow

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ChangeKind classifies how an item differs between two workflow runs.
type ChangeKind int

const (
	ChangeUnchanged ChangeKind = iota
	ChangeAdded                // present only in run B
	ChangeRemoved              // present only in run A
	ChangeModified             // present in both runs with a different value
)

func (k ChangeKind) String() string {
	switch k {
	case ChangeAdded:
		return "added"
	case ChangeRemoved:
		return "removed"
	case ChangeModified:
		return "modified"
	default:
		return "unchanged"
	}
}

// MarshalJSON emits the kind as its lowercase label rather than an integer.
func (k ChangeKind) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.String())
}

// KeyDiff is a single inputs/options key whose value differs between runs.
type KeyDiff struct {
	Key    string     `json:"key"`
	Kind   ChangeKind `json:"kind"`
	ValueA string     `json:"valueA,omitempty"`
	ValueB string     `json:"valueB,omitempty"`
}

// Duration-change thresholds: a task is flagged as slower/faster only when the
// change is both proportionally large and not negligible in absolute terms, so
// short tasks do not show up as noise on every run.
const (
	durationChangeRatio   = 1.5
	durationChangeMinDiff = 30 * time.Second
)

// TaskDiff summarizes how a single call (grouped by name) differs between runs.
type TaskDiff struct {
	Name      string        `json:"name"`
	Kind      ChangeKind    `json:"kind"`
	StatusA   string        `json:"statusA,omitempty"`
	StatusB   string        `json:"statusB,omitempty"`
	DurationA time.Duration `json:"durationA"`
	DurationB time.Duration `json:"durationB"`
	DockerA   string        `json:"dockerA,omitempty"`
	DockerB   string        `json:"dockerB,omitempty"`
	ShardsA   int           `json:"shardsA"`
	ShardsB   int           `json:"shardsB"`
	AttemptsA int           `json:"attemptsA"`
	AttemptsB int           `json:"attemptsB"`
	RunningA  bool          `json:"runningA,omitempty"`
	RunningB  bool          `json:"runningB,omitempty"`
}

// StatusChanged reports whether the aggregate task status differs.
func (t TaskDiff) StatusChanged() bool { return t.StatusA != t.StatusB }

// DockerChanged reports whether the task's Docker image differs.
func (t TaskDiff) DockerChanged() bool { return t.DockerA != t.DockerB }

// ShardsChanged reports whether the number of shards differs.
func (t TaskDiff) ShardsChanged() bool { return t.ShardsA != t.ShardsB }

// DurationChangedSignificantly reports whether the wall-clock duration changed
// enough (both proportionally and absolutely) to be worth surfacing.
func (t TaskDiff) DurationChangedSignificantly() bool {
	a, b := t.DurationA, t.DurationB
	if a <= 0 || b <= 0 {
		return false
	}
	hi, lo := a, b
	if b > a {
		hi, lo = b, a
	}
	if hi-lo < durationChangeMinDiff {
		return false
	}
	return float64(hi)/float64(lo) >= durationChangeRatio
}

// DurationRatio is B/A, useful to phrase "N× slower/faster". Zero when A is 0.
func (t TaskDiff) DurationRatio() float64 {
	if t.DurationA <= 0 {
		return 0
	}
	return float64(t.DurationB) / float64(t.DurationA)
}

// RunDiff is the full comparison of two workflow runs. The Inputs, Options and
// Tasks slices contain only the items that actually differ.
type RunDiff struct {
	IDA       string        `json:"idA"`
	IDB       string        `json:"idB"`
	NameA     string        `json:"nameA"`
	NameB     string        `json:"nameB"`
	StatusA   Status        `json:"statusA"`
	StatusB   Status        `json:"statusB"`
	DurationA time.Duration `json:"durationA"`
	DurationB time.Duration `json:"durationB"`

	// NameMismatch is true when both runs are named but the names differ.
	NameMismatch bool `json:"nameMismatch"`

	Inputs  []KeyDiff `json:"inputs"`
	Options []KeyDiff `json:"options"`

	SourceChanged bool `json:"sourceChanged"`
	SourceLinesA  int  `json:"sourceLinesA"`
	SourceLinesB  int  `json:"sourceLinesB"`

	Tasks      []TaskDiff `json:"tasks"`
	TotalTasks int        `json:"totalTasks"` // union of task names across both runs
}

// HasDifferences reports whether the two runs differ in any compared dimension.
func (d *RunDiff) HasDifferences() bool {
	return len(d.Inputs) > 0 || len(d.Options) > 0 || d.SourceChanged || len(d.Tasks) > 0
}

// CompareWorkflows compares two workflow runs and reports their differences.
// It is a pure function: it reads only the two aggregates and does not perform
// any I/O. Only first-level calls are compared; subworkflow internals are not
// expanded.
func CompareWorkflows(a, b *Workflow) *RunDiff {
	d := &RunDiff{
		IDA:          a.ID,
		IDB:          b.ID,
		NameA:        a.Name,
		NameB:        b.Name,
		StatusA:      a.Status,
		StatusB:      b.Status,
		DurationA:    a.Duration(),
		DurationB:    b.Duration(),
		NameMismatch: a.Name != "" && b.Name != "" && a.Name != b.Name,
	}

	d.Inputs = diffJSON(a.SubmittedInputs, b.SubmittedInputs)
	d.Options = diffJSON(a.SubmittedOptions, b.SubmittedOptions)

	sa := strings.TrimSpace(a.SubmittedWorkflow)
	sb := strings.TrimSpace(b.SubmittedWorkflow)
	d.SourceChanged = sa != sb
	d.SourceLinesA = countLines(sa)
	d.SourceLinesB = countLines(sb)

	d.Tasks, d.TotalTasks = diffTasks(a.Calls, b.Calls)

	return d
}

// --- inputs / options ---

func diffJSON(a, b string) []KeyDiff {
	fa := flattenJSON(a)
	fb := flattenJSON(b)

	var diffs []KeyDiff
	for _, key := range unionKeys(fa, fb) {
		va, okA := fa[key]
		vb, okB := fb[key]
		switch {
		case okA && okB:
			if va != vb {
				diffs = append(diffs, KeyDiff{Key: key, Kind: ChangeModified, ValueA: va, ValueB: vb})
			}
		case okA:
			diffs = append(diffs, KeyDiff{Key: key, Kind: ChangeRemoved, ValueA: va})
		default:
			diffs = append(diffs, KeyDiff{Key: key, Kind: ChangeAdded, ValueB: vb})
		}
	}
	return diffs
}

// flattenJSON parses a JSON object string into a flat map keyed by dot/bracket
// paths (e.g. "wf.config.threads", "wf.fastqs[0]"). Empty input yields an empty
// map; unparseable input is represented as a single "(unparseable)" entry so a
// malformed payload still shows up as a difference instead of being dropped.
func flattenJSON(raw string) map[string]string {
	out := make(map[string]string)
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return out
	}
	var v any
	if err := json.Unmarshal([]byte(trimmed), &v); err != nil {
		out["(unparseable)"] = trimmed
		return out
	}
	flattenValue("", v, out)
	return out
}

func flattenValue(prefix string, v any, out map[string]string) {
	switch val := v.(type) {
	case map[string]any:
		if len(val) == 0 {
			out[orRoot(prefix)] = "{}"
			return
		}
		for k, child := range val {
			key := k
			if prefix != "" {
				key = prefix + "." + k
			}
			flattenValue(key, child, out)
		}
	case []any:
		if len(val) == 0 {
			out[orRoot(prefix)] = "[]"
			return
		}
		for i, child := range val {
			flattenValue(fmt.Sprintf("%s[%d]", prefix, i), child, out)
		}
	default:
		out[orRoot(prefix)] = scalarString(val)
	}
}

func orRoot(prefix string) string {
	if prefix == "" {
		return "(root)"
	}
	return prefix
}

func scalarString(v any) string {
	switch val := v.(type) {
	case nil:
		return "null"
	case bool:
		return strconv.FormatBool(val)
	case float64:
		if val == math.Trunc(val) && !math.IsInf(val, 0) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'g', -1, 64)
	case string:
		return val
	default:
		b, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		return string(b)
	}
}

func unionKeys(a, b map[string]string) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	for k := range a {
		seen[k] = struct{}{}
	}
	for k := range b {
		seen[k] = struct{}{}
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// --- tasks ---

func diffTasks(a, b map[string][]Call) ([]TaskDiff, int) {
	names := unionTaskNames(a, b)

	var diffs []TaskDiff
	for _, name := range names {
		callsA, okA := a[name]
		callsB, okB := b[name]

		td := TaskDiff{Name: name}
		if okA {
			fillTaskSide(&td, callsA, true)
		}
		if okB {
			fillTaskSide(&td, callsB, false)
		}

		switch {
		case okA && okB:
			td.Kind = classifyTask(td)
		case okB:
			td.Kind = ChangeAdded
		default:
			td.Kind = ChangeRemoved
		}

		if td.Kind != ChangeUnchanged {
			diffs = append(diffs, td)
		}
	}
	return diffs, len(names)
}

func classifyTask(t TaskDiff) ChangeKind {
	if t.StatusChanged() || t.DockerChanged() || t.ShardsChanged() ||
		t.AttemptsA != t.AttemptsB || t.DurationChangedSignificantly() {
		return ChangeModified
	}
	return ChangeUnchanged
}

func fillTaskSide(td *TaskDiff, calls []Call, isA bool) {
	status, running := aggregateCallStatus(calls)
	duration := wallClockDuration(calls)
	docker := firstDockerImage(calls)
	shards := distinctShardCount(calls)
	attempts := maxAttempt(calls)

	if isA {
		td.StatusA, td.RunningA = status, running
		td.DurationA, td.DockerA = duration, docker
		td.ShardsA, td.AttemptsA = shards, attempts
	} else {
		td.StatusB, td.RunningB = status, running
		td.DurationB, td.DockerB = duration, docker
		td.ShardsB, td.AttemptsB = shards, attempts
	}
}

// aggregateCallStatus returns the overall status of a group of calls,
// prioritizing Failed, then Running/Submitted, otherwise the latest attempt.
func aggregateCallStatus(calls []Call) (status string, running bool) {
	hasFailed, hasRunning := false, false
	for _, c := range calls {
		switch c.Status {
		case StatusFailed:
			hasFailed = true
		case StatusRunning, StatusSubmitted:
			hasRunning = true
		}
	}
	if hasFailed {
		return string(StatusFailed), false
	}
	if hasRunning {
		return string(StatusRunning), true
	}
	latest := calls[0]
	for _, c := range calls[1:] {
		if c.Attempt > latest.Attempt {
			latest = c
		}
	}
	return string(latest.Status), false
}

// wallClockDuration is the span from the earliest start to the latest end
// across all calls; zero when timestamps are missing.
func wallClockDuration(calls []Call) time.Duration {
	var start, end time.Time
	for _, c := range calls {
		if !c.Start.IsZero() && (start.IsZero() || c.Start.Before(start)) {
			start = c.Start
		}
		if !c.End.IsZero() && c.End.After(end) {
			end = c.End
		}
	}
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return 0
	}
	return end.Sub(start)
}

func firstDockerImage(calls []Call) string {
	for _, c := range calls {
		if c.DockerImage != "" {
			return c.DockerImage
		}
	}
	return ""
}

func distinctShardCount(calls []Call) int {
	shards := make(map[int]struct{})
	for _, c := range calls {
		shards[c.ShardIndex] = struct{}{}
	}
	return len(shards)
}

func maxAttempt(calls []Call) int {
	maxAtt := 0
	for _, c := range calls {
		if c.Attempt > maxAtt {
			maxAtt = c.Attempt
		}
	}
	return maxAtt
}

func unionTaskNames(a, b map[string][]Call) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	for k := range a {
		seen[k] = struct{}{}
	}
	for k := range b {
		seen[k] = struct{}{}
	}
	names := make([]string, 0, len(seen))
	for k := range seen {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

func countLines(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}
