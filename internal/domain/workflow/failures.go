// failures.go aggregates workflow failures into deduplicated groups: the
// same root cause across hundreds of shards collapses into one entry. Used
// by the debug TUI's failure summary and by the chat agent's failures action.
package workflow

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// FailureTask identifies one failed task instance within a failure group.
type FailureTask struct {
	Name       string // Short task name (tree label or task name)
	ShardIndex int    // -1 for non-scattered tasks
	Stderr     string // Path to the attempt's stderr, when known
}

// FailureGroup aggregates failed tasks sharing the same error signature.
type FailureGroup struct {
	Signature string        // Normalized message used for grouping
	Sample    string        // One raw message, representative of the group
	Tasks     []FailureTask // Affected task instances, in traversal order
}

// FailureSummary is a Value Object with the workflow's failures grouped by
// root-cause signature, largest group first.
type FailureSummary struct {
	Groups      []FailureGroup
	FailedTasks int // Distinct failed task instances across all groups
}

// Patterns of run-specific noise stripped from messages before grouping, so
// the same error across hundreds of shards collapses into one group.
var (
	sigUUIDRe    = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
	sigGCSRe     = regexp.MustCompile(`gs://\S+`)
	sigPathRe    = regexp.MustCompile(`(/[\w.\-]+){3,}`)
	sigShardRe   = regexp.MustCompile(`(?i)\b(shard|index|attempt)[ :=]+\d+`)
	sigLongNumRe = regexp.MustCompile(`\b\d{4,}\b`)
	sigSpaceRe   = regexp.MustCompile(`\s+`)
)

// NormalizeFailureSignature strips run-specific details (paths, IDs, shard
// numbers) from a failure message so equivalent errors group together.
func NormalizeFailureSignature(msg string) string {
	s := sigUUIDRe.ReplaceAllString(msg, "<id>")
	s = sigGCSRe.ReplaceAllString(s, "<path>")
	s = sigPathRe.ReplaceAllString(s, "<path>")
	s = sigShardRe.ReplaceAllString(s, "$1 <n>")
	s = sigLongNumRe.ReplaceAllString(s, "<n>")
	s = sigSpaceRe.ReplaceAllString(strings.TrimSpace(s), " ")
	return s
}

// RootCauseMessages returns the deepest CausedBy messages of a failure —
// the actual root causes rather than the "Workflow failed" wrappers.
func RootCauseMessages(f Failure) []string {
	if len(f.CausedBy) == 0 {
		if strings.TrimSpace(f.Message) == "" {
			return nil
		}
		return []string{f.Message}
	}
	var msgs []string
	for _, cause := range f.CausedBy {
		msgs = append(msgs, RootCauseMessages(cause)...)
	}
	if len(msgs) == 0 && strings.TrimSpace(f.Message) != "" {
		msgs = []string{f.Message}
	}
	return msgs
}

// FailureGrouper accumulates failure messages into deduplicated groups.
// Callers feed it failed task instances (from the workflow itself or from a
// UI tree) and read the result with Sorted.
type FailureGrouper struct {
	index  map[string]int
	groups []FailureGroup
}

// NewFailureGrouper creates an empty grouper.
func NewFailureGrouper() *FailureGrouper {
	return &FailureGrouper{index: make(map[string]int)}
}

// Add records one failure message for a task instance.
func (g *FailureGrouper) Add(task FailureTask, msg string) {
	sig := NormalizeFailureSignature(msg)
	idx, ok := g.index[sig]
	if !ok {
		g.groups = append(g.groups, FailureGroup{Signature: sig, Sample: msg})
		idx = len(g.groups) - 1
		g.index[sig] = idx
	}
	g.groups[idx].Tasks = append(g.groups[idx].Tasks, task)
}

// AddFailures adds the root causes of a failure list, counting each
// signature at most once per task.
func (g *FailureGrouper) AddFailures(task FailureTask, failures []Failure) {
	seen := make(map[string]bool)
	for _, f := range failures {
		for _, msg := range RootCauseMessages(f) {
			sig := NormalizeFailureSignature(msg)
			if seen[sig] {
				continue
			}
			seen[sig] = true
			g.Add(task, msg)
		}
	}
}

// Sorted returns the groups ordered by task count, largest first.
func (g *FailureGrouper) Sorted() []FailureGroup {
	groups := g.groups
	sort.SliceStable(groups, func(i, j int) bool {
		return len(groups[i].Tasks) > len(groups[j].Tasks)
	})
	return groups
}

// CalculateFailureSummary groups every failed task instance in the workflow
// by root-cause signature, recursing into loaded subworkflow metadata. Only
// the final attempt of each task/shard counts: preempted-then-retried
// attempts are not failures of the task. When no failed task carries a
// message, the workflow-level failures are used as fallback.
func (w *Workflow) CalculateFailureSummary() *FailureSummary {
	grouper := NewFailureGrouper()
	failed := 0

	var walk func(calls map[string][]Call, scope string)
	walk = func(calls map[string][]Call, scope string) {
		for callName, callList := range calls {
			short := preemptionShortTaskName(callName)

			shardGroups := make(map[int][]Call)
			for _, call := range callList {
				if call.SubWorkflowMetadata != nil {
					walk(call.SubWorkflowMetadata.Calls, subworkflowScope(scope, callName, call.ShardIndex))
					continue
				}
				if call.SubWorkflowID != "" {
					// Subworkflow not loaded: its tasks are absent here.
					continue
				}
				shardGroups[call.ShardIndex] = append(shardGroups[call.ShardIndex], call)
			}

			for shardIndex, attempts := range shardGroups {
				sort.Slice(attempts, func(i, j int) bool {
					return attempts[i].Attempt < attempts[j].Attempt
				})
				last := attempts[len(attempts)-1]
				if last.Status != StatusFailed {
					continue
				}
				failed++
				task := FailureTask{
					Name:       failureTaskLabel(short, shardIndex),
					ShardIndex: shardIndex,
					Stderr:     last.Stderr,
				}
				if len(last.Failures) > 0 {
					grouper.AddFailures(task, last.Failures)
				} else {
					grouper.Add(task, "(no failure message in metadata)")
				}
			}
		}
	}
	walk(w.Calls, "")

	if len(grouper.groups) == 0 && len(w.Failures) > 0 {
		grouper.AddFailures(FailureTask{Name: "(workflow)", ShardIndex: -1}, w.Failures)
	}

	return &FailureSummary{
		Groups:      grouper.Sorted(),
		FailedTasks: failed,
	}
}

// failureTaskLabel names a failed task instance, with the shard suffix for
// scattered tasks.
func failureTaskLabel(short string, shardIndex int) string {
	if shardIndex >= 0 {
		return short + "[" + strconv.Itoa(shardIndex) + "]"
	}
	return short
}
