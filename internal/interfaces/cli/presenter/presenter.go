// Package presenter provides terminal output formatting utilities.
package presenter

import (
	"fmt"
	"io"
	"time"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
)

// Presenter handles terminal output formatting.
type Presenter struct {
	out io.Writer
}

// New creates a new Presenter.
func New(out io.Writer) *Presenter {
	return &Presenter{out: out}
}

// Success prints a success message.
func (p *Presenter) Success(format string, args ...interface{}) {
	green := color.New(color.FgGreen, color.Bold)
	green.Fprintf(p.out, "✓ "+format+"\n", args...)
}

// Error prints an error message.
func (p *Presenter) Error(format string, args ...interface{}) {
	red := color.New(color.FgRed, color.Bold)
	red.Fprintf(p.out, "✗ "+format+"\n", args...)
}

// Info prints an info message.
func (p *Presenter) Info(format string, args ...interface{}) {
	cyan := color.New(color.FgCyan)
	cyan.Fprintf(p.out, "ℹ "+format+"\n", args...)
}

// Warning prints a warning message.
func (p *Presenter) Warning(format string, args ...interface{}) {
	yellow := color.New(color.FgYellow)
	yellow.Fprintf(p.out, "⚠ "+format+"\n", args...)
}

// Title prints a title/header.
func (p *Presenter) Title(title string) {
	bold := color.New(color.Bold)
	bold.Fprintf(p.out, "\n%s\n", title)
	fmt.Fprintln(p.out, "─────────────────────────────────────────")
}

// KeyValue prints a key-value pair.
func (p *Presenter) KeyValue(key string, value interface{}) {
	gray := color.New(color.FgHiBlack)
	gray.Fprintf(p.out, "  %s: ", key)
	fmt.Fprintf(p.out, "%v\n", value)
}

// NewTable creates a new table writer.
func (p *Presenter) NewTable(headers []string) *tablewriter.Table {
	table := tablewriter.NewWriter(p.out)
	table.SetHeader(headers)
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding("  ")
	table.SetNoWhiteSpace(true)
	return table
}

// StatusColor returns a colored status string.
func (p *Presenter) StatusColor(status string) string {
	switch status {
	case "Succeeded":
		return color.GreenString(status)
	case "Running":
		return color.CyanString(status)
	case "Failed":
		return color.RedString(status)
	case "Aborted", "Aborting":
		return color.YellowString(status)
	case "Submitted", "On Hold":
		return color.BlueString(status)
	default:
		return status
	}
}

// FormatDuration formats a duration for display.
func (p *Presenter) FormatDuration(d time.Duration) string {
	if d == 0 {
		return "-"
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

// FormatTime formats a time for display.
func (p *Presenter) FormatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("2006-01-02 15:04:05")
}

// Newline prints a newline.
func (p *Presenter) Newline() {
	fmt.Fprintln(p.out)
}

// Print prints a formatted string.
func (p *Presenter) Print(format string, args ...interface{}) {
	fmt.Fprintf(p.out, format, args...)
}

// Println prints a line.
func (p *Presenter) Println(args ...interface{}) {
	fmt.Fprintln(p.out, args...)
}
