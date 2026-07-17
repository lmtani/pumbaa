package chat

import (
	"context"
	"errors"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"google.golang.org/adk/session"
	"google.golang.org/genai"

	"github.com/lmtani/pumbaa/internal/application/ports"
)

// newTestModel builds a chat model with no LLM and an in-memory ADK session
// service, sized so the viewport is initialized like in a real terminal.
func newTestModel(t *testing.T, svc session.Service) *Model {
	t.Helper()
	m := NewModel(nil, nil, "", svc, nil)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	mp, ok := updated.(*Model)
	if !ok {
		t.Fatalf("Update returned %T, want *Model", updated)
	}
	return mp
}

func lastMsg(t *testing.T, m *Model) ChatMessage {
	t.Helper()
	if m.msgs == nil || len(*m.msgs) == 0 {
		t.Fatal("no messages in transcript")
	}
	return (*m.msgs)[len(*m.msgs)-1]
}

func TestSubmitInputStartsGenerationAndLazySession(t *testing.T) {
	m := newTestModel(t, session.InMemoryService())

	m.textarea.SetValue("what failed?")
	_, cmd := m.submitInput()

	if !m.loading {
		t.Errorf("submit should set loading")
	}
	if cmd == nil {
		t.Errorf("submit should return the generation command batch")
	}
	if got := lastMsg(t, m); got.Role != "user" || got.Content != "what failed?" {
		t.Errorf("user message not recorded, got %+v", got)
	}
	if m.textarea.Value() != "" {
		t.Errorf("textarea should be cleared after submit")
	}
	if !m.sessionCreating {
		t.Errorf("first submit should trigger lazy session creation")
	}
}

func TestSubmitInputIgnoredWhenEmptyOrLoading(t *testing.T) {
	m := newTestModel(t, nil)

	m.textarea.SetValue("   ")
	if _, cmd := m.submitInput(); cmd != nil || m.loading {
		t.Errorf("blank input must not start a generation")
	}

	m.loading = true
	m.textarea.SetValue("hello")
	if _, cmd := m.submitInput(); cmd != nil {
		t.Errorf("submit while loading must be a no-op")
	}
	if len(*m.msgs) != 0 {
		t.Errorf("no message should be recorded, got %+v", *m.msgs)
	}
}

func TestStreamChunkUpdatesOnlyCurrentConversation(t *testing.T) {
	m := newTestModel(t, nil)
	m.loading = true

	m.Update(streamChunkMsg{owner: m.msgs, text: "partial answer"})
	if m.streamingText != "partial answer" {
		t.Errorf("streamingText = %q, want the chunk", m.streamingText)
	}

	other := []ChatMessage{}
	m.Update(streamChunkMsg{owner: &other, text: "stale"})
	if m.streamingText != "partial answer" {
		t.Errorf("a chunk from another conversation must be dropped")
	}

	m.loading = false
	m.Update(streamChunkMsg{owner: m.msgs, text: "late"})
	if m.streamingText != "partial answer" {
		t.Errorf("chunks after the response landed must be dropped")
	}
}

func TestToolRecordAppendsToTranscript(t *testing.T) {
	m := newTestModel(t, nil)

	m.Update(toolRecordMsg{owner: m.msgs, line: "pumbaa query ✓ 0.3s"})
	if got := lastMsg(t, m); got.Role != "tool" || got.Content != "pumbaa query ✓ 0.3s" {
		t.Errorf("tool record not appended, got %+v", got)
	}
}

func TestResponseMsgSuccess(t *testing.T) {
	m := newTestModel(t, nil)
	m.loading = true
	m.streamingText = "st"
	m.focusMode = FocusMessages

	m.Update(ResponseMsg{Content: "final answer", owner: m.msgs, InputTokens: 10, OutputTokens: 5})

	if m.loading {
		t.Errorf("response should clear loading")
	}
	if m.streamingText != "" {
		t.Errorf("streaming buffer should be cleared")
	}
	if got := lastMsg(t, m); got.Role != "agent" || got.Content != "final answer" {
		t.Errorf("agent message not recorded, got %+v", got)
	}
	if m.inputTokens != 10 || m.outputTokens != 5 {
		t.Errorf("token usage not accumulated: %d/%d", m.inputTokens, m.outputTokens)
	}
	if m.focusMode != FocusInput {
		t.Errorf("focus should return to the input after a response")
	}
}

func TestResponseMsgCancelKeepsPartialText(t *testing.T) {
	m := newTestModel(t, nil)
	m.loading = true
	m.streamingText = "partial so far"

	m.Update(ResponseMsg{Err: context.Canceled, owner: m.msgs})

	msgs := *m.msgs
	if len(msgs) != 2 {
		t.Fatalf("expected partial + notice, got %+v", msgs)
	}
	if msgs[0].Role != "agent" || msgs[0].Content != "partial so far" {
		t.Errorf("cancelled generation should keep the streamed text, got %+v", msgs[0])
	}
	if msgs[1].Role != "notice" {
		t.Errorf("cancellation should leave a notice, got %+v", msgs[1])
	}
}

func TestResponseMsgError(t *testing.T) {
	m := newTestModel(t, nil)
	m.loading = true

	m.Update(ResponseMsg{Err: errors.New("LLM unavailable"), owner: m.msgs})

	if got := lastMsg(t, m); got.Role != "error" || got.Content != "LLM unavailable" {
		t.Errorf("error message not recorded, got %+v", got)
	}
}

func TestResponseMsgFromPreviousConversationIsDropped(t *testing.T) {
	m := newTestModel(t, nil)
	m.loading = true
	other := []ChatMessage{}

	m.Update(ResponseMsg{Content: "stale", owner: &other})

	if !m.loading {
		t.Errorf("a stale response must not end the current generation")
	}
	if len(*m.msgs) != 0 {
		t.Errorf("a stale response must not touch the transcript, got %+v", *m.msgs)
	}
}

func TestSessionCreatedAttachesSession(t *testing.T) {
	svc := session.InMemoryService()
	m := newTestModel(t, svc)
	m.sessionCreating = true

	resp, err := svc.Create(context.Background(), &session.CreateRequest{
		AppName: ports.DefaultChatAppName, UserID: ports.DefaultChatUserID,
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	m.Update(sessionCreatedMsg{owner: m.msgs, session: resp.Session})

	if m.sessionCreating {
		t.Errorf("creation flag should be cleared")
	}
	if m.session == nil || m.session.ID() != resp.Session.ID() {
		t.Errorf("session not attached")
	}
}

func TestSessionCreatedFailureDegradesToNotice(t *testing.T) {
	m := newTestModel(t, nil)
	m.sessionCreating = true

	m.Update(sessionCreatedMsg{owner: m.msgs, err: errors.New("disk full")})

	if m.session != nil {
		t.Errorf("no session should be attached on failure")
	}
	if got := lastMsg(t, m); got.Role != "notice" {
		t.Errorf("failure should degrade to a notice, got %+v", got)
	}
}

func TestResumableFoundOffersResume(t *testing.T) {
	m := newTestModel(t, nil)

	m.Update(resumableFoundMsg{owner: m.msgs, info: ports.ChatSessionInfo{
		ID: "sess-1", Summary: "debugging align task", EventCount: 4, UpdatedAt: time.Now().Add(-2 * time.Hour),
	}})

	if m.resumableID != "sess-1" {
		t.Errorf("resumableID = %q, want sess-1", m.resumableID)
	}
	if got := lastMsg(t, m); got.Role != "notice" {
		t.Errorf("resume offer should be a notice, got %+v", got)
	}
}

func TestSessionSwitchedRebuildsTranscript(t *testing.T) {
	m := newTestModel(t, session.InMemoryService())

	history := []*genai.Content{
		{Role: "user", Parts: []*genai.Part{genai.NewPartFromText("hi")}},
		{Role: "model", Parts: []*genai.Part{genai.NewPartFromText("hello!")}},
	}
	m.Update(sessionSwitchedMsg{history: history})

	msgs := *m.msgs
	if len(msgs) != 2 {
		t.Fatalf("expected 2 rebuilt messages, got %+v", msgs)
	}
	if msgs[0].Role != "user" || msgs[1].Role != "agent" {
		t.Errorf("roles not mapped (model→agent), got %+v", msgs)
	}
	if msgs[1].Content != "hello!" {
		t.Errorf("content not extracted, got %+v", msgs[1])
	}
}

func TestTabTogglesFocusBetweenInputAndMessages(t *testing.T) {
	m := newTestModel(t, nil)
	*m.msgs = append(*m.msgs, ChatMessage{Role: "user", Content: "hi"})

	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.focusMode != FocusMessages {
		t.Fatalf("first tab should focus messages")
	}
	if m.selectedMsg != 0 {
		t.Errorf("last message should be selected, got %d", m.selectedMsg)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.focusMode != FocusInput {
		t.Errorf("second tab should return to input")
	}
	if m.selectedMsg != -1 {
		t.Errorf("selection should be cleared, got %d", m.selectedMsg)
	}
}

func TestEscDuringGenerationCancels(t *testing.T) {
	m := newTestModel(t, nil)
	m.loading = true
	cancelled := false
	m.cancelGen = func() { cancelled = true }

	m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if !cancelled {
		t.Errorf("esc during generation should invoke the cancel function")
	}
	if !m.loading {
		t.Errorf("loading only ends when the cancelled response lands")
	}
}
