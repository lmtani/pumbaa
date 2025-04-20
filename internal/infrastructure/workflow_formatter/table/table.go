package table

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/lmtani/pumbaa/internal/entities"
)

// Define colors for better readability
const (
	headerColor  = lipgloss.Color("#0366D6")
	successColor = lipgloss.Color("#2ECC40")
	failureColor = lipgloss.Color("#FF4136")
	runningColor = lipgloss.Color("#0074D9")
	borderColor  = lipgloss.Color("#AAAAAA")
	purpleColor  = lipgloss.Color("#8A2BE2") // Added purple color for IDs and key values
)

// Style functions for different elements
var (
	successStyle = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true)

	failureStyle = lipgloss.NewStyle().
			Foreground(failureColor).
			Bold(true)

	runningStyle = lipgloss.NewStyle().
			Foreground(runningColor).
			Bold(true)

	idStyle = lipgloss.NewStyle().
		Foreground(purpleColor).
		Bold(true)

	valueStyle = lipgloss.NewStyle().
			Foreground(purpleColor)
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

	// Convert workflows to table rows
	rows := [][]string{}
	for _, workflow := range workflows {
		// Format duration
		duration := "-"
		if !workflow.End.IsZero() {
			d := workflow.End.Sub(workflow.Start)
			duration = formatDuration(d)
		} else if workflow.Start.Before(time.Now()) {
			d := time.Since(workflow.Start)
			duration = formatDuration(d) + "*" // asterisk indicates ongoing
		}

		// Format status
		status := workflow.Status

		// Add table row
		rows = append(rows, []string{
			workflow.ID,
			workflow.Name,
			status,
			workflow.Start.Format("2006-01-02"),
			duration,
		})
	}

	// Create and style the table
	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(borderColor)).
		Headers("ID", "NAME", "STATUS", "START", "DURATION").
		Rows(rows...)

	// Add a title
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(headerColor).
		Padding(0, 1).
		Align(lipgloss.Center).
		Width(100). // Adjust width as needed
		Render("WORKFLOWS")

	// Print to stdout
	fmt.Fprintln(os.Stdout, title)
	fmt.Fprintln(os.Stdout, t.Render())
	fmt.Fprintln(os.Stdout, fmt.Sprintf("Total: %d workflows", len(workflows)))

	return nil
}

// Report formats and writes a single workflow as a detailed table with a summary of steps
func (f *WorkflowTableFormatter) Report(workflow *entities.Workflow) error {
	if workflow.ID == "" {
		fmt.Fprintln(os.Stdout, "Workflow not found.")
		return nil
	}

	// Header for workflow details
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(headerColor).
		Padding(0, 1).
		Align(lipgloss.Center).
		Width(80).
		Render("WORKFLOW DETAILS")
	fmt.Fprintln(os.Stdout, header)

	// Create workflow details rows
	detailsRows := [][]string{
		{"ID", workflow.ID},
		{"Name", workflow.Name},
		{"Status", workflow.Status},
		{"Start Time", workflow.Start.Format("2006-01-02 15:04:05")},
	}

	if !workflow.End.IsZero() {
		detailsRows = append(detailsRows,
			[]string{"End Time", workflow.End.Format("2006-01-02 15:04:05")},
			[]string{"Duration", formatDuration(workflow.End.Sub(workflow.Start))},
		)
	}

	// Create workflow details table
	detailsTable := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(borderColor)).
		Headers("FIELD", "VALUE").
		Rows(detailsRows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			// Apply styles to specific cells
			if col == 0 {
				return lipgloss.NewStyle().
					Padding(0, 1).
					Bold(true)
			}
			if col == 1 && row >= 0 && row < len(detailsRows) {
				// Style the values appropriately
				if detailsRows[row][0] == "ID" {
					return idStyle.Padding(0, 1)
				} else if detailsRows[row][0] == "Status" {
					switch strings.ToLower(detailsRows[row][1]) {
					case "succeeded":
						return successStyle.Padding(0, 1)
					case "failed":
						return failureStyle.Padding(0, 1)
					case "running":
						return runningStyle.Padding(0, 1)
					}
				}
			}
			return lipgloss.NewStyle().Padding(0, 1)
		})

	fmt.Fprintln(os.Stdout, detailsTable.Render())

	// Section header for calls summary
	callsHeader := lipgloss.NewStyle().
		Bold(true).
		Foreground(headerColor).
		MarginTop(1).
		Render("CALLS SUMMARY")
	fmt.Fprintln(os.Stdout, callsHeader)

	if len(workflow.Calls) > 0 {
		// Prepare summary rows
		summaryRows := [][]string{}

		for callName, steps := range workflow.Calls {
			// Count statuses
			succeeded := 0
			failed := 0
			running := 0

			// Count cache hits
			cacheHits := 0

			for _, step := range steps {
				if step.Spot {
					cacheHits++
				}

				switch strings.ToLower(step.Status) {
				case "succeeded":
					succeeded++
				case "failed":
					failed++
				case "running":
					running++
				}
			}

			// Determine overall status
			status := "Succeeded"
			if failed > 0 {
				status = "Failed"
			} else if running > 0 {
				status = "Running"
			}

			// Extract call name without workflow name prefix
			displayName := callName
			if dotIndex := strings.Index(callName, "."); dotIndex >= 0 {
				displayName = callName[dotIndex+1:]
			}

			summaryRows = append(summaryRows, []string{
				displayName,
				fmt.Sprintf("%d", len(steps)),
				fmt.Sprintf("%d", cacheHits),
				status,
			})
		}

		if len(summaryRows) > 0 {
			// Create calls summary table
			callsTable := table.New().
				Border(lipgloss.NormalBorder()).
				BorderStyle(lipgloss.NewStyle().Foreground(borderColor)).
				Headers("CALL", "TASKS", "CACHE HITS", "STATUS").
				Rows(summaryRows...).
				StyleFunc(func(row, col int) lipgloss.Style {
					return lipgloss.NewStyle().Padding(0, 1) // Add horizontal padding
				})

			fmt.Fprintln(os.Stdout, callsTable.Render())
		} else {
			fmt.Fprintln(os.Stdout, "  No steps found for this workflow.")
		}
	} else {
		fmt.Fprintln(os.Stdout, "  No calls found for this workflow.")
	}

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
