// Package dashboard provides the dashboard screen for the TUI.
package dashboard

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// WorkflowFetcher interface for fetching workflows
type WorkflowFetcher interface {
	Query(ctx context.Context, filter workflow.QueryFilter) (*workflow.QueryResult, error)
	Abort(ctx context.Context, workflowID string) error
}

// Model represents the dashboard screen state.
type Model struct {
	width       int
	height      int
	workflows   []workflow.Workflow
	totalCount  int
	cursor      int
	scrollY     int
	keys        KeyMap
	globalKeys  common.GlobalKeys
	fetcher     WorkflowFetcher
	loading     bool
	spinner     spinner.Model
	error       string
	statusMsg   string
	lastRefresh time.Time

	// Filtering
	filterInput   textinput.Model
	showFilter    bool
	activeFilters FilterState

	// Confirmation modal
	showConfirm   bool
	confirmAction string
	confirmID     string

	// Navigation state (for external handlers to check)
	NavigateToDebugID string
	ShouldQuit        bool
}

// FilterState holds the current filter configuration
type FilterState struct {
	Status []workflow.Status
	Name   string
}

// KeyMap defines the key bindings specific to the dashboard.
type KeyMap struct {
	common.NavigationKeys
	Refresh      key.Binding
	Open         key.Binding
	Abort        key.Binding
	Filter       key.Binding
	ClearFilter  key.Binding
	StatusFilter key.Binding
}

// DefaultKeyMap returns the default key bindings for the dashboard.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		NavigationKeys: common.DefaultNavigationKeys(),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Open: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "open/debug"),
		),
		Abort: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "abort workflow"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter by name"),
		),
		ClearFilter: key.NewBinding(
			key.WithKeys("ctrl+x"),
			key.WithHelp("ctrl+x", "clear filters"),
		),
		StatusFilter: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "toggle status filter"),
		),
	}
}

// NewModel creates a new dashboard model.
func NewModel() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(common.PrimaryColor)

	ti := textinput.New()
	ti.Placeholder = "Filter by workflow name..."
	ti.CharLimit = 100
	ti.Width = 40

	return Model{
		keys:        DefaultKeyMap(),
		globalKeys:  common.DefaultGlobalKeys(),
		spinner:     s,
		filterInput: ti,
		activeFilters: FilterState{
			Status: []workflow.Status{}, // Empty means all
		},
	}
}

// NewModelWithFetcher creates a new dashboard model with a workflow fetcher.
func NewModelWithFetcher(fetcher WorkflowFetcher) Model {
	m := NewModel()
	m.fetcher = fetcher
	m.loading = true
	return m
}

// Messages
type workflowsLoadedMsg struct {
	workflows  []workflow.Workflow
	totalCount int
}

type workflowsErrorMsg struct {
	err error
}

type abortResultMsg struct {
	success bool
	id      string
	err     error
}

// NavigateToDebugMsg is sent when user wants to open debug view
type NavigateToDebugMsg struct {
	WorkflowID string
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	if m.fetcher != nil {
		return tea.Batch(m.spinner.Tick, m.fetchWorkflows())
	}
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.filterInput.Width = minInt(40, m.width-20)

	case spinner.TickMsg:
		if m.loading {
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case workflowsLoadedMsg:
		m.loading = false
		m.workflows = msg.workflows
		m.totalCount = msg.totalCount
		m.lastRefresh = time.Now()
		m.error = ""
		// Reset cursor if out of bounds
		if m.cursor >= len(m.workflows) {
			m.cursor = maxInt(0, len(m.workflows)-1)
		}

	case workflowsErrorMsg:
		m.loading = false
		m.error = msg.err.Error()

	case abortResultMsg:
		m.showConfirm = false
		if msg.success {
			m.statusMsg = fmt.Sprintf("✓ Workflow %s abort requested", truncateID(msg.id))
			// Refresh the list
			m.loading = true
			cmds = append(cmds, m.spinner.Tick, m.fetchWorkflows())
		} else {
			m.statusMsg = fmt.Sprintf("✗ Failed to abort: %v", msg.err)
		}

	case tea.KeyMsg:
		// Handle confirmation modal first
		if m.showConfirm {
			return m.handleConfirmKeys(msg)
		}

		// Handle filter input
		if m.showFilter {
			return m.handleFilterKeys(msg)
		}

		return m.handleMainKeys(msg)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) handleMainKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch {
	case key.Matches(msg, m.globalKeys.Quit):
		m.ShouldQuit = true
		return m, tea.Quit

	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
			m.ensureVisible()
		}

	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(m.workflows)-1 {
			m.cursor++
			m.ensureVisible()
		}

	case key.Matches(msg, m.keys.Home):
		m.cursor = 0
		m.scrollY = 0

	case key.Matches(msg, m.keys.End):
		m.cursor = maxInt(0, len(m.workflows)-1)
		m.ensureVisible()

	case key.Matches(msg, m.keys.PageUp):
		m.cursor = maxInt(0, m.cursor-10)
		m.ensureVisible()

	case key.Matches(msg, m.keys.PageDown):
		m.cursor = minInt(len(m.workflows)-1, m.cursor+10)
		m.ensureVisible()

	case key.Matches(msg, m.keys.Refresh):
		if m.fetcher != nil && !m.loading {
			m.loading = true
			m.statusMsg = ""
			cmds = append(cmds, m.spinner.Tick, m.fetchWorkflows())
		}

	case key.Matches(msg, m.keys.Open):
		if len(m.workflows) > 0 && m.cursor < len(m.workflows) {
			wf := m.workflows[m.cursor]
			m.NavigateToDebugID = wf.ID
			return m, tea.Quit
		}

	case key.Matches(msg, m.keys.Abort):
		if len(m.workflows) > 0 && m.cursor < len(m.workflows) {
			wf := m.workflows[m.cursor]
			// Only allow aborting running/submitted workflows
			if wf.Status == workflow.StatusRunning || wf.Status == workflow.StatusSubmitted {
				m.showConfirm = true
				m.confirmAction = "abort"
				m.confirmID = wf.ID
			} else {
				m.statusMsg = "Can only abort Running or Submitted workflows"
			}
		}

	case key.Matches(msg, m.keys.Filter):
		m.showFilter = true
		m.filterInput.Focus()
		return m, textinput.Blink

	case key.Matches(msg, m.keys.ClearFilter):
		m.activeFilters = FilterState{Status: []workflow.Status{}}
		m.filterInput.SetValue("")
		if m.fetcher != nil {
			m.loading = true
			cmds = append(cmds, m.spinner.Tick, m.fetchWorkflows())
		}

	case key.Matches(msg, m.keys.StatusFilter):
		m.cycleStatusFilter()
		if m.fetcher != nil {
			m.loading = true
			cmds = append(cmds, m.spinner.Tick, m.fetchWorkflows())
		}
	}

	return m, tea.Batch(cmds...)
}

func (m Model) handleFilterKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.Type {
	case tea.KeyEsc:
		m.showFilter = false
		m.filterInput.Blur()
		return m, nil

	case tea.KeyEnter:
		m.showFilter = false
		m.filterInput.Blur()
		m.activeFilters.Name = m.filterInput.Value()
		if m.fetcher != nil {
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.fetchWorkflows())
		}
		return m, nil
	}

	m.filterInput, cmd = m.filterInput.Update(msg)
	return m, cmd
}

func (m Model) handleConfirmKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if m.confirmAction == "abort" && m.fetcher != nil {
			return m, m.abortWorkflow(m.confirmID)
		}
		m.showConfirm = false

	case "n", "N", "esc":
		m.showConfirm = false
	}

	return m, nil
}

func (m *Model) cycleStatusFilter() {
	// Cycle through: All -> Running -> Failed -> Succeeded -> All
	if len(m.activeFilters.Status) == 0 {
		m.activeFilters.Status = []workflow.Status{workflow.StatusRunning, workflow.StatusSubmitted}
	} else if containsStatus(m.activeFilters.Status, workflow.StatusRunning) {
		m.activeFilters.Status = []workflow.Status{workflow.StatusFailed}
	} else if containsStatus(m.activeFilters.Status, workflow.StatusFailed) {
		m.activeFilters.Status = []workflow.Status{workflow.StatusSucceeded}
	} else {
		m.activeFilters.Status = []workflow.Status{}
	}
}

func (m *Model) ensureVisible() {
	visibleRows := m.getVisibleRows()
	if m.cursor < m.scrollY {
		m.scrollY = m.cursor
	} else if m.cursor >= m.scrollY+visibleRows {
		m.scrollY = m.cursor - visibleRows + 1
	}
}

func (m Model) getVisibleRows() int {
	// Account for header, table header, footer, etc.
	return maxInt(1, m.height-12)
}

func (m Model) fetchWorkflows() tea.Cmd {
	return func() tea.Msg {
		if m.fetcher == nil {
			return workflowsErrorMsg{err: fmt.Errorf("no fetcher configured")}
		}

		filter := workflow.QueryFilter{
			Status:   m.activeFilters.Status,
			Name:     m.activeFilters.Name,
			PageSize: 100,
		}

		result, err := m.fetcher.Query(context.Background(), filter)
		if err != nil {
			return workflowsErrorMsg{err: err}
		}

		// Sort by submission time (newest first)
		sort.Slice(result.Workflows, func(i, j int) bool {
			return result.Workflows[i].SubmittedAt.After(result.Workflows[j].SubmittedAt)
		})

		return workflowsLoadedMsg{
			workflows:  result.Workflows,
			totalCount: result.TotalCount,
		}
	}
}

func (m Model) abortWorkflow(id string) tea.Cmd {
	return func() tea.Msg {
		if m.fetcher == nil {
			return abortResultMsg{success: false, id: id, err: fmt.Errorf("no fetcher configured")}
		}

		err := m.fetcher.Abort(context.Background(), id)
		if err != nil {
			return abortResultMsg{success: false, id: id, err: err}
		}

		return abortResultMsg{success: true, id: id}
	}
}

// View implements tea.Model.
func (m Model) View() string {
	if m.showConfirm {
		return m.renderConfirmModal()
	}

	header := m.renderHeader()
	content := m.renderContent()
	footer := m.renderFooter()

	return lipgloss.JoinVertical(lipgloss.Left, header, content, footer)
}

func (m Model) renderHeader() string {
	// Title
	title := common.HeaderTitleStyle.Render("📊 Cromwell Dashboard")

	// Status badges
	var badges []string

	// Connection status
	if m.loading {
		badges = append(badges, m.spinner.View()+" Loading...")
	} else if m.error != "" {
		badges = append(badges, common.ErrorStyle.Render("⚠ Error"))
	} else {
		badges = append(badges, common.SuccessStyle.Render("● Connected"))
	}

	// Workflow count
	countBadge := common.BadgeStyle.
		Foreground(lipgloss.Color("#000000")).
		Background(lipgloss.Color("#87CEEB")).
		Render(fmt.Sprintf("%d workflows", m.totalCount))
	badges = append(badges, countBadge)

	// Active filter indicator
	if len(m.activeFilters.Status) > 0 || m.activeFilters.Name != "" {
		filterBadge := common.BadgeStyle.
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#FFD700")).
			Render("🔍 Filtered")
		badges = append(badges, filterBadge)
	}

	// Last refresh
	if !m.lastRefresh.IsZero() {
		refreshBadge := common.MutedStyle.Render(
			fmt.Sprintf("Updated %s", m.lastRefresh.Format("15:04:05")),
		)
		badges = append(badges, refreshBadge)
	}

	headerContent := lipgloss.JoinHorizontal(lipgloss.Center, title, "  ", strings.Join(badges, " "))

	return common.HeaderStyle.
		Width(m.width - 2).
		Render(headerContent)
}

func (m Model) renderContent() string {
	if m.showFilter {
		return m.renderFilterInput()
	}

	if m.error != "" {
		errorBox := common.ErrorStyle.Render(fmt.Sprintf("Error: %s\n\nPress 'r' to retry", m.error))
		return common.PanelStyle.
			Width(m.width - 2).
			Height(m.height - 8).
			Render(lipgloss.Place(m.width-4, m.height-10, lipgloss.Center, lipgloss.Center, errorBox))
	}

	if len(m.workflows) == 0 && !m.loading {
		emptyMsg := common.MutedStyle.Render("No workflows found\n\nPress 'r' to refresh or '/' to filter")
		return common.PanelStyle.
			Width(m.width - 2).
			Height(m.height - 8).
			Render(lipgloss.Place(m.width-4, m.height-10, lipgloss.Center, lipgloss.Center, emptyMsg))
	}

	return m.renderTable()
}

func (m Model) renderFilterInput() string {
	filterBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(common.PrimaryColor).
		Padding(1, 2).
		Width(50).
		Render(
			lipgloss.JoinVertical(lipgloss.Left,
				common.TitleStyle.Render("Filter Workflows"),
				"",
				m.filterInput.View(),
				"",
				common.MutedStyle.Render("Enter to apply • Esc to cancel"),
			),
		)

	return common.PanelStyle.
		Width(m.width - 2).
		Height(m.height - 8).
		Render(lipgloss.Place(m.width-4, m.height-10, lipgloss.Center, lipgloss.Center, filterBox))
}

func (m Model) renderTable() string {
	var b strings.Builder

	// Table header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(common.TextColor).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(common.BorderColor)

	colWidths := m.getColumnWidths()
	header := fmt.Sprintf("%-*s  %-*s  %-*s  %-*s  %-*s",
		colWidths[0], "STATUS",
		colWidths[1], "ID",
		colWidths[2], "NAME",
		colWidths[3], "SUBMITTED",
		colWidths[4], "DURATION",
	)
	b.WriteString(headerStyle.Render(header) + "\n")

	// Table rows
	visibleRows := m.getVisibleRows()
	startIdx := m.scrollY
	endIdx := minInt(startIdx+visibleRows, len(m.workflows))

	for i := startIdx; i < endIdx; i++ {
		wf := m.workflows[i]
		row := m.renderWorkflowRow(wf, colWidths, i == m.cursor)
		b.WriteString(row + "\n")
	}

	// Scrollbar indicator
	if len(m.workflows) > visibleRows {
		scrollInfo := common.MutedStyle.Render(
			fmt.Sprintf("\n  Showing %d-%d of %d (↑↓ to scroll)", startIdx+1, endIdx, len(m.workflows)),
		)
		b.WriteString(scrollInfo)
	}

	return common.PanelStyle.
		Width(m.width - 2).
		Height(m.height - 8).
		Render(b.String())
}

func (m Model) renderWorkflowRow(wf workflow.Workflow, colWidths []int, selected bool) string {
	// Status with color
	statusIcon := common.StatusIcon(string(wf.Status))
	statusStyle := common.StatusStyle(string(wf.Status))
	status := statusStyle.Render(fmt.Sprintf("%-*s", colWidths[0]-2, statusIcon+" "+string(wf.Status)))

	// ID (truncated)
	id := truncateID(wf.ID)
	if len(id) > colWidths[1] {
		id = id[:colWidths[1]-3] + "..."
	}

	// Name
	name := wf.Name
	if len(name) > colWidths[2] {
		name = name[:colWidths[2]-3] + "..."
	}

	// Submitted time
	submitted := wf.SubmittedAt.Format("2006-01-02 15:04")

	// Duration
	duration := "-"
	if !wf.Start.IsZero() {
		endTime := wf.End
		if endTime.IsZero() {
			endTime = time.Now()
		}
		dur := endTime.Sub(wf.Start)
		duration = formatDuration(dur)
	}

	row := fmt.Sprintf("%s  %-*s  %-*s  %-*s  %-*s",
		status,
		colWidths[1], id,
		colWidths[2], name,
		colWidths[3], submitted,
		colWidths[4], duration,
	)

	if selected {
		return lipgloss.NewStyle().
			Background(common.HighlightColor).
			Foreground(common.TextColor).
			Width(m.width - 6).
			Render(row)
	}

	return row
}

func (m Model) getColumnWidths() []int {
	// STATUS, ID, NAME, SUBMITTED, DURATION
	available := m.width - 20 // margins and separators
	return []int{
		12,                       // STATUS
		14,                       // ID (truncated)
		maxInt(20, available-60), // NAME (flexible)
		18,                       // SUBMITTED
		12,                       // DURATION
	}
}

func (m Model) renderFooter() string {
	var parts []string

	// Status message
	if m.statusMsg != "" {
		parts = append(parts, m.statusMsg)
		parts = append(parts, " • ")
	}

	// Filter status
	if len(m.activeFilters.Status) > 0 {
		statusNames := make([]string, len(m.activeFilters.Status))
		for i, s := range m.activeFilters.Status {
			statusNames[i] = string(s)
		}
		parts = append(parts, fmt.Sprintf("Status: %s", strings.Join(statusNames, "/")))
		parts = append(parts, " • ")
	}

	// Help
	help := fmt.Sprintf(
		"%s %s  %s %s  %s %s  %s %s  %s %s  %s %s",
		common.KeyStyle.Render("↑↓"),
		common.DescStyle.Render("navigate"),
		common.KeyStyle.Render("enter"),
		common.DescStyle.Render("debug"),
		common.KeyStyle.Render("a"),
		common.DescStyle.Render("abort"),
		common.KeyStyle.Render("s"),
		common.DescStyle.Render("filter status"),
		common.KeyStyle.Render("r"),
		common.DescStyle.Render("refresh"),
		common.KeyStyle.Render("q"),
		common.DescStyle.Render("quit"),
	)
	parts = append(parts, help)

	return common.HelpBarStyle.
		Width(m.width - 2).
		Render(strings.Join(parts, ""))
}

func (m Model) renderConfirmModal() string {
	modalContent := lipgloss.JoinVertical(lipgloss.Center,
		common.TitleStyle.Render("⚠️  Confirm Abort"),
		"",
		"Are you sure you want to abort workflow",
		common.MutedStyle.Render(truncateID(m.confirmID)),
		"",
		lipgloss.JoinHorizontal(lipgloss.Center,
			common.KeyStyle.Render("y")+" Yes  ",
			common.KeyStyle.Render("n")+" No",
		),
	)

	modal := common.ModalStyle.
		Width(50).
		Render(modalContent)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		modal,
		lipgloss.WithWhitespaceChars(" "),
	)
}

// Helper functions
func truncateID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%dm", h, m)
}

func containsStatus(statuses []workflow.Status, status workflow.Status) bool {
	for _, s := range statuses {
		if s == status {
			return true
		}
	}
	return false
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
