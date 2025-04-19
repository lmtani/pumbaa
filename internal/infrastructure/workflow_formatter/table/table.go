package table

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/lmtani/pumbaa/internal/entities"
)

// Styles
var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#0366D6")).
			Align(lipgloss.Center).
			Padding(0, 1)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#2ECC40")).
			Bold(true)

	failureStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF4136")).
			Bold(true)

	runningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#0074D9")).
			Bold(true)

	cellStyle = lipgloss.NewStyle().
			Padding(0, 1)

	tableStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#AAAAAA")).
			Padding(0, 1).
			MarginTop(1).
			MarginBottom(1)

	baseTableStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#AAAAAA"))

	callHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#333333")).
			Background(lipgloss.Color("#E0E0E0")).
			Padding(0, 1).
			Width(80)
)

// WorkflowTableFormatter formats workflows as tables and writes to stdout
type WorkflowTableFormatter struct{}

// NewWorkflowTableFormatter creates a new table formatter
func NewWorkflowTableFormatter() *WorkflowTableFormatter {
	return &WorkflowTableFormatter{}
}

// Query formats and writes multiple workflows as a table
func (f *WorkflowTableFormatter) Query(workflows []entities.Workflow) error {
	if len(workflows) == 0 {
		fmt.Fprintln(os.Stdout, "No workflows found.")
		return nil
	}

	// Define headers with optimized widths
	headers := []string{"ID", "NAME", "STATUS", "START", "DURATION"}
	widths := []int{40, 30, 12, 12, 15}

	// Format header row
	headerRow := ""
	for i, header := range headers {
		headerRow += headerStyle.Width(widths[i]).Render(header)
	}

	rows := []string{headerRow}

	// Format data rows
	for _, workflow := range workflows {
		row := ""

		// ID column - truncate with ellipsis if needed
		id := workflow.ID
		if len(id) > widths[0]-3 {
			id = id[:widths[0]-3] + "..."
		}
		row += cellStyle.Width(widths[0]).Render(id)

		// Name column
		name := workflow.Name
		if len(name) > widths[1]-3 {
			name = name[:widths[1]-3] + "..."
		}
		row += cellStyle.Width(widths[1]).Render(name)

		// Status column with color
		status := workflow.Status
		statusCell := ""
		switch strings.ToLower(status) {
		case "succeeded":
			statusCell = successStyle.Render("Succeeded")
		case "failed":
			statusCell = failureStyle.Render("Failed")
		case "running":
			statusCell = runningStyle.Render("Running")
		default:
			statusCell = status
		}
		row += cellStyle.Width(widths[2]).Render(statusCell)

		// Start time column - use a more compact format
		start := workflow.Start.Format("2006-01-02")
		row += cellStyle.Width(widths[3]).Render(start)

		// Duration column
		duration := "-"
		if !workflow.End.IsZero() {
			d := workflow.End.Sub(workflow.Start)
			duration = formatDuration(d)
		} else if workflow.Start.Before(time.Now()) {
			// For running workflows, show duration since start
			d := time.Now().Sub(workflow.Start)
			duration = formatDuration(d) + "*" // asterisk indicates ongoing
		}
		row += cellStyle.Width(widths[4]).Render(duration)

		rows = append(rows, row)
	}

	// Join rows and apply table styling
	tableContent := strings.Join(rows, "\n")

	// Create a title for the table
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#0366D6")).
		Padding(0, 1).
		Align(lipgloss.Center).
		Width(widths[0] + widths[1] + widths[2] + widths[3] + widths[4] + 10).
		Render("WORKFLOWS")

	// Apply border to table content
	formattedTable := tableStyle.Render(tableContent)

	// Print title and table
	fmt.Fprintln(os.Stdout, title)
	fmt.Fprintln(os.Stdout, formattedTable)

	// Footer text
	fmt.Fprintln(os.Stdout, fmt.Sprintf("Total: %d workflows", len(workflows)))

	return nil
}

// Report formats and writes a single workflow as a detailed table
func (f *WorkflowTableFormatter) Report(workflow *entities.Workflow) error {
	if workflow.ID == "" {
		fmt.Fprintln(os.Stdout, "Workflow not found.")
		return nil
	}

	// Create a header for the workflow details
	header := headerStyle.Width(80).Render("WORKFLOW DETAILS")

	// Format basic workflow info
	statusColor := lipgloss.Color("#AAAAAA")
	switch strings.ToLower(workflow.Status) {
	case "succeeded":
		statusColor = lipgloss.Color("#2ECC40")
	case "failed":
		statusColor = lipgloss.Color("#FF4136")
	case "running":
		statusColor = lipgloss.Color("#0074D9")
	}

	// Create info table columns
	infoColumns := []table.Column{
		{Title: "FIELD", Width: 15},
		{Title: "VALUE", Width: 64},
	}

	// Create info table rows
	infoRows := []table.Row{
		{"ID", workflow.ID},
		{"Name", workflow.Name},
		{"Status", lipgloss.NewStyle().Foreground(statusColor).Render(workflow.Status)},
		{"Start Time", workflow.Start.Format("2006-01-02T15:04:05")},
	}

	if !workflow.End.IsZero() {
		infoRows = append(infoRows,
			table.Row{"End Time", workflow.End.Format("2006-01-02T15:04:05")},
			table.Row{"Duration", formatDuration(workflow.End.Sub(workflow.Start))},
		)
	}

	// Create and style the info table
	infoTable := table.New(
		table.WithColumns(infoColumns),
		table.WithRows(infoRows),
		table.WithFocused(false),
		table.WithHeight(len(infoRows)),
	)

	// Style the info table
	infoStyle := baseTableStyle.Copy()
	renderedInfoTable := infoStyle.Render(infoTable.View())

	// Create a section header style
	sectionHeaderStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#0366D6")).
		MarginTop(1)

	// Combine all sections
	output := []string{
		header,
		renderedInfoTable,
		sectionHeaderStyle.Render("CALLS"),
	}

	// Add call details if there are any
	if len(workflow.Calls) > 0 {
		for callName, steps := range workflow.Calls {
			callHeader := callHeaderStyle.Render(callName)
			output = append(output, callHeader)

			if len(steps) > 0 {
				// Create steps table columns
				stepColumns := []table.Column{
					{Title: "NAME", Width: 25},
					{Title: "SPOT", Width: 8},
					{Title: "STATUS", Width: 15},
					{Title: "START", Width: 20},
					{Title: "END", Width: 20},
				}

				// Create steps table rows
				stepRows := []table.Row{}
				for _, step := range steps {
					spot := "No"
					if step.Spot {
						spot = "Yes"
					}

					stepEnd := step.End
					if stepEnd == "" {
						stepEnd = "-"
					}

					stepRows = append(stepRows, table.Row{
						step.Name,
						spot,
						step.Status,
						step.Start,
						stepEnd,
					})
				}

				// Create and style the steps table
				stepsTable := table.New(
					table.WithColumns(stepColumns),
					table.WithRows(stepRows),
					table.WithFocused(false),
					table.WithHeight(len(stepRows)),
				)

				// Style the steps table
				stepsStyle := baseTableStyle.Copy()
				renderedStepsTable := stepsStyle.Render(stepsTable.View())
				output = append(output, renderedStepsTable)
			} else {
				output = append(output, "  No steps found for this call.")
			}
		}
	} else {
		output = append(output, "  No calls found for this workflow.")
	}

	// Print the final formatted output
	fmt.Fprintln(os.Stdout, strings.Join(output, "\n"))

	return nil
}

// Helper function to format duration in a human-readable format
func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}
