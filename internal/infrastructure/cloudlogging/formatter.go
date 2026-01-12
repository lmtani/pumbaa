package cloudlogging

import (
	"encoding/json"

	"cloud.google.com/go/logging"

	"github.com/lmtani/pumbaa/internal/domain/ports"
)

// FormatLogEntry extracts a cleaned, human-readable message from a Cloud Logging entry.
// Message extraction priority:
//   1. String payload (text message)
//   2. JSON payload with "message" key
//   3. JSON payload (stringified)
//   4. Fallback: fmt.Sprintf("%v", payload), truncated
//
// The returned message is truncated to maxLen characters.
func FormatLogEntry(entry *logging.Entry, maxLen int) ports.BatchLogEntry {
	message := extractMessage(entry, maxLen)

	// Map severity
	severity := entry.Severity.String()
	if severity == "DEFAULT" {
		severity = "INFO"
	}

	return ports.BatchLogEntry{
		Timestamp: entry.Timestamp,
		Severity:  severity,
		Message:   message,
	}
}

// extractMessage extracts a cleaned message from a logging.Entry.
func extractMessage(entry *logging.Entry, maxLen int) string {
	// Priority 1: String payload (text message)
	if str, ok := entry.Payload.(string); ok && str != "" {
		return truncateString(str, maxLen)
	}

	// Priority 2: JSON payload (map) with "message" key
	if msgObj, ok := entry.Payload.(map[string]interface{}); ok {
		if msg, exists := msgObj["message"]; exists {
			if msgStr, ok := msg.(string); ok && msgStr != "" {
				return truncateString(msgStr, maxLen)
			}
		}
	}

	// Priority 3: JSON payload as fallback (stringify any other type)
	if entry.Payload != nil {
		jsonStr, err := jsonToString(entry.Payload)
		if err == nil && jsonStr != "" {
			return truncateString(jsonStr, maxLen)
		}
	}

	// Last resort: return empty string
	return ""
}

// jsonToString converts a JSON payload to a string.
func jsonToString(payload interface{}) (string, error) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// truncateString truncates a string to maxLen characters, appending "..." if truncated.
func truncateString(s string, maxLen int) string {
	if maxLen <= 0 {
		return s
	}
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
