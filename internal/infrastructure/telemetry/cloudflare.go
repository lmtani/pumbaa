package telemetry

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"
)

var (
	// TelemetryEndpoint is the URL of the Cloudflare Worker
	// Injected at build time via: -ldflags "-X github.com/lmtani/pumbaa/internal/infrastructure/telemetry.TelemetryEndpoint=..."
	TelemetryEndpoint = ""

	// TelemetryKey is the API Key for the Cloudflare Worker
	// Injected at build time via: -ldflags "-X github.com/lmtani/pumbaa/internal/infrastructure/telemetry.TelemetryKey=..."
	TelemetryKey = ""
)

// CloudflareService implements the Service interface using Cloudflare Workers
type CloudflareService struct {
	client    *http.Client
	version   string
	clientID  string
	userAgent string
	wg        sync.WaitGroup
}

// NewCloudflareService creates a new Cloudflare telemetry service
func NewCloudflareService(clientID string, version string) *CloudflareService {
	if TelemetryEndpoint == "" {
		return nil
	}

	return &CloudflareService{
		client: &http.Client{
			Timeout: 2 * time.Second, // Fail fast
		},
		version:   version,
		clientID:  clientID,
		userAgent: fmt.Sprintf("pumbaa-cli/%s (%s; %s)", version, runtime.GOOS, runtime.GOARCH),
	}
}

// Track captures an event asynchronously
func (s *CloudflareService) Track(event Event) {
	if s == nil {
		return
	}

	// Enrich event
	if event.Version == "" {
		event.Version = s.version
	}
	if event.OS == "" {
		event.OS = runtime.GOOS
	}
	if event.Arch == "" {
		event.Arch = runtime.GOARCH
	}

	// Payload for Worker
	payload := map[string]interface{}{
		"command":       event.Command,
		"duration_ms":   event.Duration,
		"success":       event.Success,
		"error_message": event.Error,
		"version":       event.Version,
		"os":            event.OS,
		"arch":          event.Arch,
		"client_id":     s.clientID,
		"timestamp":     time.Now().Unix(),
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.send(payload)
	}()
}

// TrackCommand tracks a command execution
func (s *CloudflareService) TrackCommand(ctx CommandContext, err error) {
	if s == nil {
		return
	}

	cmdName := extractCommandName(ctx.AppName, ctx.Args)
	duration := time.Since(ctx.StartTime).Milliseconds()
	success := err == nil

	var errType string
	var errMsg string
	if err != nil {
		errType = categorizeError(err)
		errMsg = err.Error()
	}

	payload := map[string]interface{}{
		"command":       cmdName,
		"duration_ms":   duration,
		"success":       success,
		"error_type":    errType,
		"error_message": errMsg,
		"version":       s.version,
		"os":            runtime.GOOS,
		"arch":          runtime.GOARCH,
		"client_id":     s.clientID,
		"timestamp":     time.Now().Unix(),
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.send(payload)
	}()
}

func (s *CloudflareService) CaptureError(operation string, err error) {
	if s == nil || err == nil {
		return
	}

	payload := map[string]interface{}{
		"command":       "error:" + operation,
		"success":       false,
		"error_type":    categorizeError(err),
		"error_message": err.Error(),
		"version":       s.version,
		"os":            runtime.GOOS,
		"arch":          runtime.GOARCH,
		"client_id":     s.clientID,
		"timestamp":     time.Now().Unix(),
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.send(payload)
	}()
}

func (s *CloudflareService) AddBreadcrumb(category, message string) {
	// Cloudflare implementation relies on stateless events,
	// breadcrumbs are not typically sent unless we batch them locally.
	// For now, this is a no-op to satisfy interface.
}

func (s *CloudflareService) Close() {
	// Wait for all pending telemetry events to be sent
	// with a hard timeout to avoid hanging the CLI indefinitely
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All events sent
	case <-time.After(2 * time.Second):
		// Telemetry flush timed out, fail silently
	}
}

func (s *CloudflareService) send(payload map[string]interface{}) {
	// Recover from panics to never crash the app
	defer func() {
		if r := recover(); r != nil {
			// Fail silently
		}
	}()

	body, err := json.Marshal(payload)
	if err != nil {
		return
	}

	req, err := http.NewRequest("POST", TelemetryEndpoint, bytes.NewBuffer(body))
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", s.userAgent)
	if TelemetryKey != "" {
		req.Header.Set("Authorization", "Bearer "+TelemetryKey)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
}

// Utility functions (migrated from sentry.go)

func extractCommandName(appName string, args []string) string {
	if len(args) == 0 {
		return appName
	}

	// Default known commands matching Sentry implementation
	knownCommands := map[string][]string{
		"workflow":  {"submit", "metadata", "abort", "query", "debug"},
		"wf":        {"submit", "metadata", "abort", "query", "debug"},
		"bundle":    {},
		"dashboard": {},
		"chat":      {},
		"config":    {},
	}

	firstArg := args[0]
	subcommands, isKnown := knownCommands[firstArg]
	if !isKnown {
		return appName
	}

	cmdName := firstArg
	if cmdName == "wf" {
		cmdName = "workflow"
	}

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
		var targetErr interface{ Unwrap() error }
		if errors.As(err, &targetErr) {
			return fmt.Sprintf("wrapped:%T", err)
		}
		return "internal"
	}
}
