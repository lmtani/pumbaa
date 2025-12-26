package cromwell

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

func TestNewClient(t *testing.T) {
	cfg := Config{
		Host:    "http://localhost:8000",
		Timeout: 10 * time.Second,
	}

	client := NewClient(cfg)

	if client.BaseURL != cfg.Host {
		t.Errorf("expected BaseURL=%s, got %s", cfg.Host, client.BaseURL)
	}
	if client.httpClient.Timeout != cfg.Timeout {
		t.Errorf("expected Timeout=%v, got %v", cfg.Timeout, client.httpClient.Timeout)
	}
}

func TestNewClient_DefaultTimeout(t *testing.T) {
	cfg := Config{
		Host: "http://localhost:8000",
		// No timeout specified
	}

	client := NewClient(cfg)

	if client.httpClient.Timeout != 30*time.Second {
		t.Errorf("expected default Timeout=30s, got %v", client.httpClient.Timeout)
	}
}

func TestClient_GetStatus_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/workflows/v1/test-id/status" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"id":     "test-id",
			"status": "Running",
		})
	}))
	defer server.Close()

	client := NewClient(Config{Host: server.URL})
	status, err := client.GetStatus(context.Background(), "test-id")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != workflow.StatusRunning {
		t.Errorf("expected StatusRunning, got %s", status)
	}
}

func TestClient_GetStatus_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(Config{Host: server.URL})
	_, err := client.GetStatus(context.Background(), "non-existent")

	if !errors.Is(err, workflow.ErrWorkflowNotFound) {
		t.Errorf("expected ErrWorkflowNotFound, got %v", err)
	}
}

func TestClient_GetStatus_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	client := NewClient(Config{Host: server.URL})
	_, err := client.GetStatus(context.Background(), "test-id")

	var apiErr workflow.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("expected APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", apiErr.StatusCode)
	}
}

func TestClient_Abort_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/workflows/v1/test-id/abort" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "Aborting"})
	}))
	defer server.Close()

	client := NewClient(Config{Host: server.URL})
	err := client.Abort(context.Background(), "test-id")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_Abort_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(Config{Host: server.URL})
	err := client.Abort(context.Background(), "non-existent")

	if !errors.Is(err, workflow.ErrWorkflowNotFound) {
		t.Errorf("expected ErrWorkflowNotFound, got %v", err)
	}
}

func TestClient_GetHealthStatus_AllHealthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/engine/v1/status" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"Engine Database": map[string]bool{"ok": true},
			"PAPI":            map[string]bool{"ok": true},
		})
	}))
	defer server.Close()

	client := NewClient(Config{Host: server.URL})
	health, err := client.GetHealthStatus(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !health.OK {
		t.Error("expected health.OK=true")
	}
	if health.Degraded {
		t.Error("expected health.Degraded=false")
	}
}

func TestClient_GetHealthStatus_Degraded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"Engine Database": map[string]bool{"ok": true},
			"PAPI":            map[string]bool{"ok": false},
		})
	}))
	defer server.Close()

	client := NewClient(Config{Host: server.URL})
	health, err := client.GetHealthStatus(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if health.OK {
		t.Error("expected health.OK=false")
	}
	if !health.Degraded {
		t.Error("expected health.Degraded=true")
	}
	if len(health.UnhealthySystems) != 1 || health.UnhealthySystems[0] != "PAPI" {
		t.Errorf("expected PAPI in unhealthy systems, got %v", health.UnhealthySystems)
	}
}

func TestClient_Query_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/workflows/v1/query" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		// Verify query params
		q := r.URL.Query()
		if q.Get("name") != "TestWorkflow" {
			t.Errorf("expected name=TestWorkflow, got %s", q.Get("name"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []map[string]interface{}{
				{
					"id":     "wf-1",
					"name":   "TestWorkflow",
					"status": "Succeeded",
				},
				{
					"id":     "wf-2",
					"name":   "TestWorkflow",
					"status": "Running",
				},
			},
			"totalResultsCount": 2,
		})
	}))
	defer server.Close()

	client := NewClient(Config{Host: server.URL})
	result, err := client.Query(context.Background(), workflow.QueryFilter{
		Name: "TestWorkflow",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalCount != 2 {
		t.Errorf("expected TotalCount=2, got %d", result.TotalCount)
	}
	if len(result.Workflows) != 2 {
		t.Errorf("expected 2 workflows, got %d", len(result.Workflows))
	}
}

func TestClient_GetLabels_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/workflows/v1/test-id/labels" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "test-id",
			"labels": map[string]string{
				"project": "test-project",
				"env":     "dev",
			},
		})
	}))
	defer server.Close()

	client := NewClient(Config{Host: server.URL})
	labels, err := client.GetLabels(context.Background(), "test-id")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if labels["project"] != "test-project" {
		t.Errorf("expected project=test-project, got %s", labels["project"])
	}
	if labels["env"] != "dev" {
		t.Errorf("expected env=dev, got %s", labels["env"])
	}
}

func TestClient_UpdateLabels_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type=application/json")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "test-id",
			"labels": map[string]string{
				"new-label": "new-value",
			},
		})
	}))
	defer server.Close()

	client := NewClient(Config{Host: server.URL})
	err := client.UpdateLabels(context.Background(), "test-id", map[string]string{
		"new-label": "new-value",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_ConnectionError(t *testing.T) {
	// Use invalid host to trigger connection error
	client := NewClient(Config{
		Host:    "http://localhost:99999",
		Timeout: 100 * time.Millisecond,
	})

	_, err := client.GetStatus(context.Background(), "test-id")

	if !errors.Is(err, workflow.ErrConnectionFailed) {
		t.Errorf("expected ErrConnectionFailed, got %v", err)
	}
}
