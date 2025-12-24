package telemetry

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
)

// SentryService implements the Service interface using Sentry
type SentryService struct {
	initialized bool
	version     string
}

var (
	// DSN is the Sentry DSN for Pumbaa project.
	// Injected at build time via: -ldflags "-X github.com/lmtani/pumbaa/internal/infrastructure/telemetry.DSN=..."
	DSN = ""

	// knownCommands maps top-level commands to their subcommands for extraction
	knownCommands = map[string][]string{
		"workflow":  {"submit", "metadata", "abort", "query", "debug"},
		"wf":        {"submit", "metadata", "abort", "query", "debug"},
		"bundle":    {},
		"dashboard": {},
		"chat":      {},
		"config":    {},
	}
)

// NewSentryService creates a new Sentry telemetry service
func NewSentryService(clientID string, version string) (*SentryService, error) {
	// If DSN is not set (dev build without secret), return nil for graceful fallback to NoOp
	if DSN == "" {
		return nil, nil
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              DSN,
		Release:          version,
		Environment:      "production",
		AttachStacktrace: true,
		MaxErrorDepth:    10,
		TracesSampleRate: 1.0,
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			// Add anonymous client ID as tag
			event.Tags["client_id"] = clientID
			event.Tags["os"] = runtime.GOOS
			event.Tags["arch"] = runtime.GOARCH
			return event
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize sentry: %w", err)
	}

	return &SentryService{initialized: true, version: version}, nil
}

func (s *SentryService) Track(event Event) {
	if !s.initialized {
		return
	}

	// Use Sentry's scope to add context
	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetTag("command", event.Command)
		scope.SetTag("success", fmt.Sprintf("%t", event.Success))
		scope.SetExtra("duration_ms", event.Duration)
		scope.SetExtra("version", event.Version)
		scope.SetExtra("os", event.OS)
		scope.SetExtra("arch", event.Arch)

		if event.Success {
			// Track successful command execution as a message
			sentry.CaptureMessage(fmt.Sprintf("command:%s", event.Command))
		} else if event.Error != "" {
			// Capture errors with full context
			sentry.CaptureException(fmt.Errorf("command %s failed: %s", event.Command, event.Error))
		}
	})
}

// TrackCommand tracks a command execution with automatic event creation.
func (s *SentryService) TrackCommand(ctx CommandContext, err error) {
	if !s.initialized {
		return
	}

	cmdName := extractCommandName(ctx.AppName, ctx.Args)
	duration := time.Since(ctx.StartTime).Milliseconds()
	success := err == nil

	var errType string
	if err != nil {
		errType = categorizeError(err)
	}

	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetTag("command", cmdName)
		scope.SetTag("success", fmt.Sprintf("%t", success))
		scope.SetExtra("duration_ms", duration)
		scope.SetExtra("version", s.version)
		scope.SetExtra("os", runtime.GOOS)
		scope.SetExtra("arch", runtime.GOARCH)

		if success {
			sentry.CaptureMessage(fmt.Sprintf("command:%s", cmdName))
		} else {
			scope.SetTag("error_type", errType)
			scope.SetExtra("args", sanitizeArgs(ctx.Args))

			// Capture the original error with stack trace if available
			sentry.CaptureException(fmt.Errorf("command %s failed: %w", cmdName, err))
		}
	})
}

func (s *SentryService) Close() {
	if s.initialized {
		sentry.Flush(2 * time.Second)
	}
}

// CaptureError captures an error with operation context.
// Use for TUI errors or background operations that don't go through TrackCommand.
func (s *SentryService) CaptureError(operation string, err error) {
	if !s.initialized || err == nil {
		return
	}

	errType := categorizeError(err)

	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetTag("operation", operation)
		scope.SetTag("error_type", errType)
		scope.SetExtra("version", s.version)
		scope.SetExtra("os", runtime.GOOS)
		scope.SetExtra("arch", runtime.GOARCH)

		sentry.CaptureException(fmt.Errorf("%s: %w", operation, err))
	})
}

// extractCommandName extracts just the command/subcommand from args,
// ignoring flags and arguments.
func extractCommandName(appName string, args []string) string {
	if len(args) == 0 {
		return appName
	}

	// Check if first arg is a known command
	firstArg := args[0]
	subcommands, isKnown := knownCommands[firstArg]
	if !isKnown {
		// Not a known command, might be a flag like --help
		return appName
	}

	// Normalize aliases
	cmdName := firstArg
	if cmdName == "wf" {
		cmdName = "workflow"
	}

	// Check for subcommand
	if len(args) > 1 && len(subcommands) > 0 {
		secondArg := args[1]
		for _, sub := range subcommands {
			if secondArg == sub {
				return appName + " " + cmdName + " " + sub
			}
		}
	}

	return appName + " " + cmdName
}

// categorizeError attempts to categorize an error by its type
func categorizeError(err error) string {
	if err == nil {
		return "none"
	}

	errStr := strings.ToLower(err.Error())

	switch {
	case strings.Contains(errStr, "connection") || strings.Contains(errStr, "network") ||
		strings.Contains(errStr, "timeout") || strings.Contains(errStr, "dial"):
		return "network"
	case strings.Contains(errStr, "permission") || strings.Contains(errStr, "access denied"):
		return "permission"
	case strings.Contains(errStr, "not found") || strings.Contains(errStr, "no such file"):
		return "not_found"
	case strings.Contains(errStr, "invalid") || strings.Contains(errStr, "validation"):
		return "validation"
	case strings.Contains(errStr, "parse") || strings.Contains(errStr, "syntax"):
		return "parse"
	default:
		// Try to get the error type from wrapped errors
		var targetErr interface{ Unwrap() error }
		if errors.As(err, &targetErr) {
			return fmt.Sprintf("wrapped:%T", err)
		}
		return "internal"
	}
}

// sanitizeArgs removes sensitive information from command args
func sanitizeArgs(args []string) []string {
	sanitized := make([]string, 0, len(args))
	skipNext := false

	sensitiveFlags := []string{"--token", "--password", "--secret", "--key", "--api-key"}

	for _, arg := range args {
		if skipNext {
			sanitized = append(sanitized, "[REDACTED]")
			skipNext = false
			continue
		}

		isSensitive := false
		for _, flag := range sensitiveFlags {
			if strings.HasPrefix(arg, flag+"=") {
				// Flag with value: --token=xxx
				parts := strings.SplitN(arg, "=", 2)
				sanitized = append(sanitized, parts[0]+"=[REDACTED]")
				isSensitive = true
				break
			}
			if arg == flag {
				// Flag without value: --token xxx
				sanitized = append(sanitized, arg)
				skipNext = true
				isSensitive = true
				break
			}
		}

		if !isSensitive {
			sanitized = append(sanitized, arg)
		}
	}

	return sanitized
}
