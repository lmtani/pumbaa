package chat

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"google.golang.org/adk/tool"
	"google.golang.org/genai"
)

// fakeTool implements tool.Tool and the chat's toolWithDefinition interface,
// mirroring how the real pumbaa function tool is executed by the agent loop.
type fakeTool struct {
	name    string
	result  map[string]any
	err     error
	gotArgs any
}

func (f *fakeTool) Name() string        { return f.name }
func (f *fakeTool) Description() string { return "fake" }
func (f *fakeTool) IsLongRunning() bool { return false }

func (f *fakeTool) Declaration() *genai.FunctionDeclaration {
	return &genai.FunctionDeclaration{Name: f.name}
}

func (f *fakeTool) Run(_ tool.Context, args any) (map[string]any, error) {
	f.gotArgs = args
	return f.result, f.err
}

func TestToolFailure(t *testing.T) {
	if got := toolFailure(nil, errors.New("boom")); got != "boom" {
		t.Errorf("transport error should win, got %q", got)
	}
	if got := toolFailure(map[string]any{"success": false, "error": "bad input"}, nil); got != "bad input" {
		t.Errorf("handler error should be extracted, got %q", got)
	}
	if got := toolFailure(map[string]any{"success": false}, nil); got != "failed" {
		t.Errorf("failure without message should fall back, got %q", got)
	}
	if got := toolFailure(map[string]any{"success": true}, nil); got != "" {
		t.Errorf("success should be empty, got %q", got)
	}
	if got := toolFailure(nil, nil); got != "" {
		t.Errorf("no result and no error should be empty, got %q", got)
	}
}

func TestFormatToolRecord(t *testing.T) {
	got := formatToolRecord("pumbaa", "query", map[string]any{"status": "Failed", "name": "wf"}, 800*time.Millisecond, "")
	// Params are sorted by key for a stable transcript.
	if want := "pumbaa query (name=wf, status=Failed) ✓ 0.8s"; got != want {
		t.Errorf("formatToolRecord = %q, want %q", got, want)
	}

	got = formatToolRecord("pumbaa", "logs", nil, 2*time.Second, "workflow not found")
	if !strings.Contains(got, "✗") || !strings.Contains(got, "workflow not found") {
		t.Errorf("failure record should carry marker and reason, got %q", got)
	}
}

func TestGetToolCalls(t *testing.T) {
	content := &genai.Content{
		Parts: []*genai.Part{
			genai.NewPartFromText("thinking..."),
			{FunctionCall: &genai.FunctionCall{Name: "pumbaa"}},
			{FunctionCall: &genai.FunctionCall{Name: "other"}},
		},
	}
	calls := getToolCalls(content)
	if len(calls) != 2 || calls[0].Name != "pumbaa" || calls[1].Name != "other" {
		t.Errorf("getToolCalls = %+v, want the two function calls in order", calls)
	}

	if calls := getToolCalls(&genai.Content{}); calls != nil {
		t.Errorf("content without calls should yield nil, got %+v", calls)
	}
}

func TestConvertToolsToGenAI(t *testing.T) {
	ft := &fakeTool{name: "pumbaa"}
	tools := convertToolsToGenAI([]tool.Tool{ft})
	if len(tools) != 1 || len(tools[0].FunctionDeclarations) != 1 || tools[0].FunctionDeclarations[0].Name != "pumbaa" {
		t.Errorf("convertToolsToGenAI = %+v, want one declaration named pumbaa", tools)
	}
}

func TestExecuteTool(t *testing.T) {
	ft := &fakeTool{name: "pumbaa", result: map[string]any{"success": true}}
	m := Model{tools: []tool.Tool{ft}}

	args := map[string]any{"action": "query"}
	result, err := m.executeTool(context.Background(), &genai.FunctionCall{Name: "pumbaa", Args: args})
	if err != nil {
		t.Fatalf("executeTool returned error: %v", err)
	}
	if success, _ := result["success"].(bool); !success {
		t.Errorf("expected the fake tool's result, got %+v", result)
	}
	if got, ok := ft.gotArgs.(map[string]any); !ok || got["action"] != "query" {
		t.Errorf("tool should receive the call args, got %+v", ft.gotArgs)
	}

	if _, err := m.executeTool(context.Background(), &genai.FunctionCall{Name: "unknown"}); err == nil {
		t.Errorf("unknown tool should return an error")
	}
}
