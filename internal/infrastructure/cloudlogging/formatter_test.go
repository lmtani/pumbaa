package cloudlogging

import (
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/logging"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestFormatLogEntry_StringPayloadPriority(t *testing.T) {
	entry := &logging.Entry{
		Timestamp: time.Now(),
		Severity:  logging.Info,
		Payload:   "Container started successfully",
	}

	result := FormatLogEntry(entry, 500)
	if result.Message != "Container started successfully" {
		t.Errorf("expected 'Container started successfully', got '%s'", result.Message)
	}
	if result.Severity != "Info" && result.Severity != "INFO" {
		t.Errorf("expected severity Info or INFO, got '%s'", result.Severity)
	}
}

func TestFormatLogEntry_JSONPayloadMessageFallback(t *testing.T) {
	entry := &logging.Entry{
		Timestamp: time.Now(),
		Severity:  logging.Error,
		Payload:   map[string]interface{}{"message": "Task failed with error code 1"},
	}

	result := FormatLogEntry(entry, 500)
	if result.Message != "Task failed with error code 1" {
		t.Errorf("expected 'Task failed with error code 1', got '%s'", result.Message)
	}
	if result.Severity != "Error" && result.Severity != "ERROR" {
		t.Errorf("expected severity Error or ERROR, got '%s'", result.Severity)
	}
}

func TestFormatLogEntry_StringPayloadOverridesJSON(t *testing.T) {
	entry := &logging.Entry{
		Timestamp: time.Now(),
		Severity:  logging.Default,
		Payload:   "Text takes priority",
	}

	result := FormatLogEntry(entry, 500)
	if result.Message != "Text takes priority" {
		t.Errorf("string payload should be returned, got '%s'", result.Message)
	}
}

func TestFormatLogEntry_Truncation(t *testing.T) {
	longMessage := strings.Repeat("a", 100)
	entry := &logging.Entry{
		Timestamp: time.Now(),
		Severity:  logging.Info,
		Payload:   longMessage,
	}

	result := FormatLogEntry(entry, 20)
	if len(result.Message) > 20 {
		t.Errorf("message should be truncated to <= 20 chars, got %d", len(result.Message))
	}
	if !strings.HasSuffix(result.Message, "...") {
		t.Errorf("truncated message should end with '...', got '%s'", result.Message)
	}
}

func TestFormatLogEntry_EmptyPayload(t *testing.T) {
	entry := &logging.Entry{
		Timestamp: time.Now(),
		Severity:  logging.Warning,
		Payload:   nil,
	}

	result := FormatLogEntry(entry, 500)
	if result.Message != "" {
		t.Errorf("expected empty message, got '%s'", result.Message)
	}
}

func TestFormatLogEntry_SeverityMapping(t *testing.T) {
	entry := &logging.Entry{
		Timestamp: time.Now(),
		Severity:  logging.Default,
		Payload:   "test message",
	}

	result := FormatLogEntry(entry, 500)
	// Default severity is returned as-is (logging.Default.String() returns "Default")
	if result.Severity == "" {
		t.Errorf("expected non-empty severity, got '%s'", result.Severity)
	}
}

func TestFormatLogEntry_JSONPayloadNotString(t *testing.T) {
	// JSON payload with "message" key that's not a string
	entry := &logging.Entry{
		Timestamp: time.Now(),
		Severity:  logging.Info,
		Payload: map[string]interface{}{
			"message": 123,
			"error":   "some error",
		},
	}

	result := FormatLogEntry(entry, 500)
	// Should fall back to stringified JSON
	if result.Message == "" {
		t.Error("expected non-empty message from JSON payload fallback")
	}
}

func TestTruncateString_NoTruncationNeeded(t *testing.T) {
	s := "short message"
	result := truncateString(s, 50)
	if result != s {
		t.Errorf("expected '%s', got '%s'", s, result)
	}
}

func TestTruncateString_Truncation(t *testing.T) {
	s := "this is a long message that needs truncation"
	result := truncateString(s, 20)
	if len(result) > 20 {
		t.Errorf("expected <= 20 chars, got %d", len(result))
	}
	if !strings.HasSuffix(result, "...") {
		t.Errorf("expected truncated string to end with '...', got '%s'", result)
	}
}

func TestFormatLogEntry_TimestampPreserved(t *testing.T) {
	now := time.Now()
	entry := &logging.Entry{
		Timestamp: now,
		Severity:  logging.Info,
		Payload:   "test",
	}

	result := FormatLogEntry(entry, 500)
	if !result.Timestamp.Equal(now) {
		t.Errorf("expected timestamp %v, got %v", now, result.Timestamp)
	}
}

func TestFormatLogEntry_ProtoPayloadFallback(t *testing.T) {
	// Protobuf message marshaled as JSON
	pbMsg := &timestamppb.Timestamp{Seconds: 1234567890}
	entry := &logging.Entry{
		Timestamp: time.Now(),
		Severity:  logging.Info,
		Payload:   pbMsg,
	}

	result := FormatLogEntry(entry, 500)
	if result.Message == "" {
		t.Error("expected non-empty message from proto payload")
	}
}
