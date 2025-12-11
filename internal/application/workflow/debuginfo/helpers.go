package debuginfo

import (
	"fmt"
	"strings"
	"time"
)

// getString extracts a string value from a map.
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// getInt extracts an int value from a map.
func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}

// getBool extracts a bool value from a map.
func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

// parseTime parses a time string in RFC3339 format.
func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		// Try alternate format
		t, _ = time.Parse("2006-01-02T15:04:05.000Z", s)
	}
	return t
}

// FormatBytes formats bytes into a human-readable string.
func FormatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// AggregateStatus returns the aggregate status for a list of calls.
// It considers retries and preemptions - if any attempt succeeded, the status is Done.
func AggregateStatus(calls []CallDetails) string {
	// Check for failures first (excluding preemptions that were retried successfully)
	hasDone := false
	hasRunning := false
	hasFailed := false

	for _, c := range calls {
		switch c.ExecutionStatus {
		case "Done":
			hasDone = true
		case "Running":
			hasRunning = true
		case "Failed":
			hasFailed = true
		}
	}

	// If any attempt succeeded, the call is Done
	if hasDone {
		return "Done"
	}

	// If still running, show Running
	if hasRunning {
		return "Running"
	}

	// If failed (and no success), show Failed
	if hasFailed {
		return "Failed"
	}

	// Check if all are preempted (no retry yet)
	allPreempted := true
	for _, c := range calls {
		if c.ExecutionStatus != "Preempted" && c.ExecutionStatus != "RetryableFailure" {
			allPreempted = false
			break
		}
	}
	if allPreempted && len(calls) > 0 {
		return "Preempted"
	}

	return "Unknown"
}

// EarliestStart returns the earliest start time from a list of calls.
func EarliestStart(calls []CallDetails) time.Time {
	var earliest time.Time
	for _, c := range calls {
		if earliest.IsZero() || c.Start.Before(earliest) {
			earliest = c.Start
		}
	}
	return earliest
}

// LatestEnd returns the latest end time from a list of calls.
func LatestEnd(calls []CallDetails) time.Time {
	var latest time.Time
	for _, c := range calls {
		if c.End.After(latest) {
			latest = c.End
		}
	}
	return latest
}

// ParseCPU parses CPU string (e.g., "4", "4.0") to float64.
func ParseCPU(s string) float64 {
	if s == "" {
		return 0
	}
	var cpu float64
	for i, c := range s {
		if c == '.' {
			// Parse decimal part
			var decimal float64
			var divisor float64 = 10
			for _, d := range s[i+1:] {
				if d < '0' || d > '9' {
					break
				}
				decimal += float64(d-'0') / divisor
				divisor *= 10
			}
			return cpu + decimal
		}
		if c < '0' || c > '9' {
			break
		}
		cpu = cpu*10 + float64(c-'0')
	}
	return cpu
}

// ParseMemoryGB parses memory string (e.g., "8 GB", "8GB", "8192 MB") to GB.
func ParseMemoryGB(s string) float64 {
	if s == "" {
		return 0
	}

	// Extract numeric part
	var num float64
	var i int
	for i = 0; i < len(s); i++ {
		c := s[i]
		if c == '.' {
			// Parse decimal part
			var decimal float64
			var divisor float64 = 10
			for j := i + 1; j < len(s); j++ {
				d := s[j]
				if d < '0' || d > '9' {
					i = j
					break
				}
				decimal += float64(d-'0') / divisor
				divisor *= 10
				i = j + 1
			}
			num += decimal
			break
		}
		if c < '0' || c > '9' {
			break
		}
		num = num*10 + float64(c-'0')
	}

	// Check for unit
	rest := strings.ToUpper(strings.TrimSpace(s[i:]))
	if strings.HasPrefix(rest, "MB") {
		return num / 1024
	}
	if strings.HasPrefix(rest, "TB") {
		return num * 1024
	}
	// Default to GB
	return num
}

// FormatDuration formats a duration in a human-readable format.
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%ds", mins, secs)
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%dm", hours, mins)
}
