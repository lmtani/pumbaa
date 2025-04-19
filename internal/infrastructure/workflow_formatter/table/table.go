package table

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/lmtani/pumbaa/internal/entities"
)

// Styles
var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#0366D6")).
			Padding(0, 1)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#2ECC40"))

	failureStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF4136"))

	runningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#0074D9"))

	cellStyle = lipgloss.NewStyle().
			Padding(0, 1)

	tableStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#AAAAAA")).
			MarginTop(1).
			MarginBottom(1)
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

	// Define headers and their widths
	headers := []string{"ID", "NAME", "STATUS", "START", "END", "DURATION"}
	widths := []int{40, 25, 12, 25, 25, 15}

	// Format header row
	headerRow := ""
	for i, header := range headers {
		headerRow += headerStyle.Width(widths[i]).Render(header)
	}

	rows := []string{headerRow}

	// Format data rows
	for _, workflow := range workflows {
		row := ""

		// ID column
		row += cellStyle.Width(widths[0]).Render(workflow.ID)

		// Name column
		name := workflow.Name
		if len(name) > widths[1]-2 {
			name = name[:widths[1]-5] + "..."
		}
		row += cellStyle.Width(widths[1]).Render(name)

		// Status column with color
		status := workflow.Status
		statusCell := ""
		switch strings.ToLower(status) {
		case "succeeded":
			statusCell = successStyle.Render(status)
		case "failed":
			statusCell = failureStyle.Render(status)
		case "running":
			statusCell = runningStyle.Render(status)
		default:
			statusCell = status
		}
		row += cellStyle.Width(widths[2]).Render(statusCell)

		// Start time column
		start := workflow.Start.Format("2006-01-02T15:04:05")
		row += cellStyle.Width(widths[3]).Render(start)

		// End time column
		end := "-"
		if !workflow.End.IsZero() {
			end = workflow.End.Format("2006-01-02T15:04:05")
		}
		row += cellStyle.Width(widths[4]).Render(end)

		// Duration column
		duration := "-"
		if !workflow.End.IsZero() {
			d := workflow.End.Sub(workflow.Start)
			duration = formatDuration(d)
		}
		row += cellStyle.Width(widths[5]).Render(duration)

		rows = append(rows, row)
	}

	// Render the table
	table := strings.Join(rows, "\n")
	fmt.Fprintln(os.Stdout, tableStyle.Render(table))

	return nil
}

// Report formats and writes a single workflow as a detailed table
func (f *WorkflowTableFormatter) Report(workflow *entities.Workflow) error {
	if workflow.ID == "" {
		fmt.Fprintln(os.Stdout, "Workflow not found.")
		return nil
	}

	// Create a header for the workflow details
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#0366D6")).
		Padding(0, 1).
		Width(80).
		Render("WORKFLOW DETAILS")

	// Create a style for section headers
	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#0366D6")).
		MarginTop(1)

	// Format basic workflow info
	statusStyle := cellStyle
	switch strings.ToLower(workflow.Status) {
	case "succeeded":
		statusStyle = successStyle
	case "failed":
		statusStyle = failureStyle
	case "running":
		statusStyle = runningStyle
	}

	// Create a 2-column table for workflow info
	infoWidths := []int{15, 64}
	infoRows := []string{}

	// Row for ID
	infoRow := cellStyle.Bold(true).Width(infoWidths[0]).Render("ID")
	infoRow += cellStyle.Width(infoWidths[1]).Render(workflow.ID)
	infoRows = append(infoRows, infoRow)

	// Row for Name
	infoRow = cellStyle.Bold(true).Width(infoWidths[0]).Render("Name")
	infoRow += cellStyle.Width(infoWidths[1]).Render(workflow.Name)
	infoRows = append(infoRows, infoRow)

	// Row for Status
	infoRow = cellStyle.Bold(true).Width(infoWidths[0]).Render("Status")
	infoRow += cellStyle.Width(infoWidths[1]).Render(statusStyle.Render(workflow.Status))
	infoRows = append(infoRows, infoRow)

	// Row for Start Time
	infoRow = cellStyle.Bold(true).Width(infoWidths[0]).Render("Start Time")
	infoRow += cellStyle.Width(infoWidths[1]).Render(workflow.Start.Format("2006-01-02T15:04:05"))
	infoRows = append(infoRows, infoRow)

	// Rows for End Time and Duration if applicable
	if !workflow.End.IsZero() {
		infoRow = cellStyle.Bold(true).Width(infoWidths[0]).Render("End Time")
		infoRow += cellStyle.Width(infoWidths[1]).Render(workflow.End.Format("2006-01-02T15:04:05"))
		infoRows = append(infoRows, infoRow)

		duration := workflow.End.Sub(workflow.Start)
		infoRow = cellStyle.Bold(true).Width(infoWidths[0]).Render("Duration")
		infoRow += cellStyle.Width(infoWidths[1]).Render(formatDuration(duration))
		infoRows = append(infoRows, infoRow)
	}

	infoTable := tableStyle.Render(strings.Join(infoRows, "\n"))

	// Combine all sections
	output := []string{
		header,
		infoTable,
		sectionStyle.Render("CALLS"),
	}

	// Add call details if there are any
	if len(workflow.Calls) > 0 {
		for callName, steps := range workflow.Calls {
			callHeader := lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#333333")).
				Background(lipgloss.Color("#E0E0E0")).
				Padding(0, 1).
				Width(80).
				Render(callName)

			output = append(output, callHeader)

			if len(steps) > 0 {
				// Define headers and their widths for steps
				stepHeaders := []string{"NAME", "SPOT", "STATUS", "START", "END"}
				stepWidths := []int{25, 8, 15, 20, 20}

				// Format step header row
				stepHeaderRow := ""
				for i, header := range stepHeaders {
					stepHeaderRow += headerStyle.Width(stepWidths[i]).Render(header)
				}

				stepRows := []string{stepHeaderRow}

				// Format step rows
				for _, step := range steps {
					stepRow := ""

					// Name column
					stepName := step.Name
					if len(stepName) > stepWidths[0]-2 {
						stepName = stepName[:stepWidths[0]-5] + "..."
					}
					stepRow += cellStyle.Width(stepWidths[0]).Render(stepName)

					// Spot column
					spot := "No"
					if step.Spot {
						spot = "Yes"
					}
					stepRow += cellStyle.Width(stepWidths[1]).Render(spot)

					// Status column with color
					stepStatus := step.Status
					stepStatusCell := ""
					switch strings.ToLower(stepStatus) {
					case "succeeded":
						stepStatusCell = successStyle.Render(stepStatus)
					case "failed":
						stepStatusCell = failureStyle.Render(stepStatus)
					case "running":
						stepStatusCell = runningStyle.Render(stepStatus)
					default:
						stepStatusCell = stepStatus
					}
					stepRow += cellStyle.Width(stepWidths[2]).Render(stepStatusCell)

					// Start time column
					stepStart := step.Start
					if len(stepStart) > stepWidths[3]-2 {
						stepStart = stepStart[:stepWidths[3]-5] + "..."
					}
					stepRow += cellStyle.Width(stepWidths[3]).Render(stepStart)

					// End time column
					stepEnd := step.End
					if stepEnd == "" {
						stepEnd = "-"
					} else if len(stepEnd) > stepWidths[4]-2 {
						stepEnd = stepEnd[:stepWidths[4]-5] + "..."
					}
					stepRow += cellStyle.Width(stepWidths[4]).Render(stepEnd)

					stepRows = append(stepRows, stepRow)
				}

				// Render the steps table
				stepsTable := tableStyle.Render(strings.Join(stepRows, "\n"))
				output = append(output, stepsTable)
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
