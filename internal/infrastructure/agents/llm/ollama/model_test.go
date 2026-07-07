package ollama

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

func userRequest(text string) *model.LLMRequest {
	return &model.LLMRequest{
		Contents: []*genai.Content{
			{Role: "user", Parts: []*genai.Part{genai.NewPartFromText(text)}},
		},
	}
}

func firstText(content *genai.Content) string {
	for _, part := range content.Parts {
		if part != nil && part.Text != "" {
			return part.Text
		}
	}
	return ""
}

func TestGenerateContentStreamingYieldsPartialsAndFinal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}
		if !req.Stream {
			t.Errorf("expected stream=true in request")
		}
		lines := []string{
			`{"message":{"role":"assistant","content":"Hel"},"done":false}`,
			`{"message":{"role":"assistant","content":"lo"},"done":false}`,
			`{"message":{"role":"assistant","content":""},"done":true,"prompt_eval_count":10,"eval_count":5}`,
		}
		for _, line := range lines {
			_, _ = w.Write([]byte(line + "\n"))
		}
	}))
	defer srv.Close()

	m := NewModel(srv.URL, "test-model")

	var partials []string
	var final *model.LLMResponse
	for r, err := range m.GenerateContent(context.Background(), userRequest("hi"), true) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if r.Partial {
			partials = append(partials, firstText(r.Content))
		} else {
			final = r
		}
	}

	if len(partials) != 2 || partials[0] != "Hel" || partials[1] != "lo" {
		t.Errorf("unexpected partials: %v", partials)
	}
	if final == nil {
		t.Fatalf("no final response yielded")
	}
	if got := firstText(final.Content); got != "Hello" {
		t.Errorf("final text = %q, want %q", got, "Hello")
	}
	if !final.TurnComplete {
		t.Errorf("final response without tool calls should be TurnComplete")
	}
	if final.UsageMetadata == nil || final.UsageMetadata.PromptTokenCount != 10 || final.UsageMetadata.CandidatesTokenCount != 5 {
		t.Errorf("usage metadata not aggregated: %+v", final.UsageMetadata)
	}
}

func TestGenerateContentStreamingCollectsToolCalls(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lines := []string{
			`{"message":{"role":"assistant","content":"","tool_calls":[{"type":"function","function":{"name":"pumbaa","arguments":{"action":"query"}}}]},"done":false}`,
			`{"message":{"role":"assistant","content":""},"done":true,"prompt_eval_count":3,"eval_count":2}`,
		}
		for _, line := range lines {
			_, _ = w.Write([]byte(line + "\n"))
		}
	}))
	defer srv.Close()

	m := NewModel(srv.URL, "test-model")

	var final *model.LLMResponse
	for r, err := range m.GenerateContent(context.Background(), userRequest("hi"), true) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !r.Partial {
			final = r
		}
	}

	if final == nil {
		t.Fatalf("no final response yielded")
	}
	var fnCall *genai.FunctionCall
	for _, part := range final.Content.Parts {
		if part != nil && part.FunctionCall != nil {
			fnCall = part.FunctionCall
		}
	}
	if fnCall == nil || fnCall.Name != "pumbaa" {
		t.Fatalf("tool call not propagated to final response: %+v", final.Content.Parts)
	}
	if final.TurnComplete {
		t.Errorf("final response with pending tool calls should not be TurnComplete")
	}
}
