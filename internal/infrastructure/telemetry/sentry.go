package telemetry

import (
	"fmt"
	"runtime"
	"time"

	"github.com/getsentry/sentry-go"
)

// SentryService implements the Service interface using Sentry
type SentryService struct {
	initialized bool
}

var (
	// DSN is the Sentry DSN for Pumbaa project.
	// Injected at build time via: -ldflags "-X github.com/lmtani/pumbaa/internal/infrastructure/telemetry.DSN=..."
	DSN = ""
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

	return &SentryService{initialized: true}, nil
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

func (s *SentryService) Close() {
	if s.initialized {
		sentry.Flush(2 * time.Second)
	}
}
