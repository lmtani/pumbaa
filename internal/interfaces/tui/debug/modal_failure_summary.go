package debug

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// failureGroup aggregates failed tasks sharing the same error signature.
type failureGroup struct {
	signature string   // normalized message used for grouping
	sample    string   // one raw message, shown in the modal
	tasks     []string // node names, in tree order
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

// normalizeFailureSignature strips run-specific details (paths, IDs, shard
// numbers) from a failure message so equivalent errors group together.
func normalizeFailureSignature(msg string) string {
	s := sigUUIDRe.ReplaceAllString(msg, "<id>")
	s = sigGCSRe.ReplaceAllString(s, "<path>")
	s = sigPathRe.ReplaceAllString(s, "<path>")
	s = sigShardRe.ReplaceAllString(s, "$1 <n>")
	s = sigLongNumRe.ReplaceAllString(s, "<n>")
	s = sigSpaceRe.ReplaceAllString(strings.TrimSpace(s), " ")
	return s
}

// rootCauseMessages returns the deepest CausedBy messages of a failure —
// the actual root causes rather than the "Workflow failed" wrappers.
func rootCauseMessages(f Failure) []string {
	if len(f.CausedBy) == 0 {
		if strings.TrimSpace(f.Message) == "" {
			return nil
		}
		return []string{f.Message}
	}
	var msgs []string
	for _, cause := range f.CausedBy {
		msgs = append(msgs, rootCauseMessages(cause)...)
	}
	if len(msgs) == 0 && strings.TrimSpace(f.Message) != "" {
		msgs = []string{f.Message}
	}
	return msgs
}

// failureGrouper accumulates failure messages into deduplicated groups.
type failureGrouper struct {
	index  map[string]int
	groups []failureGroup
}

func newFailureGrouper() *failureGrouper {
	return &failureGrouper{index: make(map[string]int)}
}

func (g *failureGrouper) add(taskName, msg string) {
	sig := normalizeFailureSignature(msg)
	idx, ok := g.index[sig]
	if !ok {
		g.groups = append(g.groups, failureGroup{signature: sig, sample: msg})
		idx = len(g.groups) - 1
		g.index[sig] = idx
	}
	g.groups[idx].tasks = append(g.groups[idx].tasks, taskName)
}

// addFailures adds the root causes of a failure list, counting each
// signature at most once per task.
func (g *failureGrouper) addFailures(taskName string, failures []Failure) {
	seen := make(map[string]bool)
	for _, f := range failures {
		for _, msg := range rootCauseMessages(f) {
			sig := normalizeFailureSignature(msg)
			if seen[sig] {
				continue
			}
			seen[sig] = true
			g.add(taskName, msg)
		}
	}
}

// sorted returns the groups ordered by task count, largest first.
func (g *failureGrouper) sorted() []failureGroup {
	groups := g.groups
	sort.SliceStable(groups, func(i, j int) bool {
		return len(groups[i].tasks) > len(groups[j].tasks)
	})
	return groups
}

// collectFailureGroups groups every failed leaf in the tree by error
// signature. When no failed leaves carry messages, it falls back to the
// workflow-level failures.
func collectFailureGroups(root *TreeNode, workflowFailures []Failure) []failureGroup {
	grouper := newFailureGrouper()

	for _, node := range flattenTree(root) {
		if len(node.Children) > 0 || !isFailedStatus(node.Status) || node.Type == NodeTypeWorkflow {
			continue
		}
		if node.CallData != nil && len(node.CallData.Failures) > 0 {
			grouper.addFailures(node.Name, node.CallData.Failures)
		} else {
			grouper.add(node.Name, "(no failure message in metadata)")
		}
	}

	if len(grouper.groups) == 0 {
		grouper.addFailures("(workflow)", workflowFailures)
	}

	return grouper.sorted()
}

// openFailureSummary opens the aggregated failure summary modal.
func (m Model) openFailureSummary() (tea.Model, tea.Cmd) {
	groups := collectFailureGroups(m.tree, m.metadata.Failures)
	if len(groups) == 0 {
		m.setStatusMessage("No failures found")
		return m, getClearStatusCmd()
	}

	styled, raw := m.formatFailureSummary(groups)
	m.failureSummaryRaw = raw
	m.activeModal = ModalFailureSummary
	m.failureSummaryViewport = viewport.New(m.width-10, m.height-8)
	m.failureSummaryViewport.SetContent(styled)
	return m, nil
}

// formatFailureSummary renders the groups for the modal viewport and as raw
// text for the clipboard.
func (m Model) formatFailureSummary(groups []failureGroup) (styled, raw string) {
	const maxTasksShown = 5
	width := m.width - 14

	var sb, rawSB strings.Builder
	for i, group := range groups {
		count := len(group.tasks)

		header := fmt.Sprintf("%d× %s", count, group.sample)
		sb.WriteString(errorStyle.Render(fmt.Sprintf("✗ %d×", count)) + " " +
			errorMsgStyle.Render(wrapText(group.sample, width-7)) + "\n")
		rawSB.WriteString(header + "\n")

		shown := group.tasks
		if len(shown) > maxTasksShown {
			shown = shown[:maxTasksShown]
		}
		taskLine := strings.Join(shown, ", ")
		if extra := count - len(shown); extra > 0 {
			taskLine += fmt.Sprintf(" (+%d more)", extra)
		}
		sb.WriteString(mutedStyle.Render(wrapText("  "+taskLine, width)) + "\n")
		rawSB.WriteString("  " + strings.Join(group.tasks, ", ") + "\n")

		if i < len(groups)-1 {
			sb.WriteString("\n")
			rawSB.WriteString("\n")
		}
	}

	return sb.String(), rawSB.String()
}

// renderFailureSummaryModal renders the failure summary modal.
func (m Model) renderFailureSummaryModal() string {
	title := errorStyle.Render("⚠  Failure Summary") + " " +
		mutedStyle.Render("(grouped by error signature)")
	content := renderModalViewportContent(m.failureSummaryViewport.View(), m.failureSummaryViewport.Width, false, "")
	return m.renderStandardModal(title, content, m.modalFooterWithHints("↑↓ scroll", "y copy", "esc close"))
}

// handleFailureSummaryModalKeys handles keyboard input in the failure
// summary modal.
func (m Model) handleFailureSummaryModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	actions := viewportModalActions{
		onClose: func(m *Model) {
			m.activeModal = ModalNone
			m.failureSummaryRaw = ""
		},
		onCopy: func(m *Model) tea.Cmd {
			if m.failureSummaryRaw != "" {
				return copyToClipboard(m.failureSummaryRaw, "failure summary")
			}
			return nil
		},
	}
	cmd, handled := m.handleViewportModalKeys(msg, &m.failureSummaryViewport, actions)
	if handled {
		return m, cmd
	}
	return m, nil
}
