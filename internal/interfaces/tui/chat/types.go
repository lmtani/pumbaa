package chat

import (
	"google.golang.org/adk/session"
	"google.golang.org/genai"

	"github.com/lmtani/pumbaa/internal/application/ports"
)

// Chat message types and the tea.Msg definitions exchanged between the
// generation goroutine and the Update loop.
type ChatMessage struct {
	Role     string
	Content  string
	Rendered string // Pre-rendered markdown (cached)
}

type ResponseMsg struct {
	Content      string
	Err          error
	InputTokens  int // Input tokens used in this response
	OutputTokens int // Output tokens generated in this response

	// owner identifies the conversation that produced this response (the
	// model's msgs pointer is unique per instance), so an in-flight response
	// from a previous chat can never be appended to a newer conversation.
	owner *[]ChatMessage
}

// ToolNotificationMsg is sent when a tool is being called
type ToolNotificationMsg struct {
	ToolName string
	Action   string
	Params   map[string]any // Additional parameters

	owner *[]ChatMessage // See ResponseMsg.owner
}

// streamChunkMsg carries the accumulated text of the current turn while the
// response is streaming.
type streamChunkMsg struct {
	owner *[]ChatMessage
	text  string
}

// toolRecordMsg appends a persistent record of an executed tool call to the
// transcript (unlike ToolNotificationMsg, which is transient).
type toolRecordMsg struct {
	owner *[]ChatMessage
	line  string
}

// sessionCreatedMsg carries the result of the lazy session creation.
type sessionCreatedMsg struct {
	owner   *[]ChatMessage
	session session.Session
	err     error
}

// resumableFoundMsg reports a previous session for the same task context.
type resumableFoundMsg struct {
	owner *[]ChatMessage
	info  ports.ChatSessionInfo
}

// ClearNotificationMsg is sent to clear the tool notification
type ClearNotificationMsg struct{}

// clipboardCopiedMsg is sent when clipboard copy completes
type clipboardCopiedMsg struct {
	success bool
	err     error
}

// clearStatusMsg is sent to clear the status message
type clearStatusMsg struct{}

// sessionListLoadedMsg is sent when session list is loaded
type sessionListLoadedMsg struct {
	sessions []ports.ChatSessionInfo
}

// sessionListErrorMsg is sent when session loading fails
type sessionListErrorMsg struct {
	err error
}

// sessionSwitchedMsg is sent when a session is switched
type sessionSwitchedMsg struct {
	session session.Session
	history []*genai.Content
}

// sessionSummaryUpdatedMsg is sent when the session summary is updated
type sessionSummaryUpdatedMsg struct {
	sessionID string
	summary   string
}
