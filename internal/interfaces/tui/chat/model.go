// Package chat provides the chat TUI interface for interacting with LLMs.
package chat

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool"
	"google.golang.org/genai"

	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// Model is the Bubble Tea model for the chat interface.
type Model struct {
	llm               model.LLM
	tools             []tool.Tool
	systemInstruction string

	// Session management
	sessionService session.Service
	session        session.Session

	// History is a pointer so it persists across value copies
	history *[]*genai.Content

	// Bubble Tea components
	viewport viewport.Model
	textarea textarea.Model
	spinner  spinner.Model
	program  *tea.Program // Program pointer for sending messages from goroutines

	// Dimensions
	width  int
	height int

	// State
	loading          bool
	ready            bool
	toolNotification string    // Temporary notification for tool calls
	focusMode        FocusMode // Current focus: input or messages
	selectedMsg      int       // Currently selected message index (-1 = none)
	statusMessage    string    // Temporary status message (e.g., "Copied!")
	statusExpires    time.Time // When status message expires

	// Token usage tracking
	inputTokens  int // Accumulated input tokens for the session
	outputTokens int // Accumulated output tokens for the session

	// Display messages (pointer for persistence)
	msgs *[]ChatMessage
}

// NewModel creates a new chat model with the given LLM, tools, system instruction, and session.
func NewModel(llm model.LLM, tools []tool.Tool, systemInstruction string, svc session.Service, sess session.Session) Model {
	ta := textarea.New()
	ta.Placeholder = "Type your message..."
	ta.Focus()
	ta.Prompt = ""
	ta.CharLimit = 2000
	ta.SetWidth(80)
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.BlurredStyle.CursorLine = lipgloss.NewStyle()

	vp := viewport.New(80, 20)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(common.PrimaryColor)

	// Load history from session events
	history := make([]*genai.Content, 0)
	msgs := make([]ChatMessage, 0)

	if sess != nil && sess.Events() != nil && sess.Events().Len() > 0 {
		for ev := range sess.Events().All() {
			if ev.Content != nil {
				history = append(history, ev.Content)
				role := ev.Content.Role
				if role == "model" {
					role = "agent"
				}
				text := extractText(ev.Content)
				if text != "" {
					msgs = append(msgs, ChatMessage{Role: role, Content: text})
				}
			}
		}
	}

	// Load token counts from session if available
	var inputTokens, outputTokens int
	if sess != nil {
		// Try to get tokens from session (using interface assertion for our extended session)
		if tokenSession, ok := sess.(interface{ InputTokens() int }); ok {
			inputTokens = tokenSession.InputTokens()
		}
		if tokenSession, ok := sess.(interface{ OutputTokens() int }); ok {
			outputTokens = tokenSession.OutputTokens()
		}
	}

	return Model{
		llm:               llm,
		tools:             tools,
		systemInstruction: systemInstruction,
		sessionService:    svc,
		session:           sess,
		textarea:          ta,
		viewport:          vp,
		spinner:           s,
		history:           &history,
		msgs:              &msgs,
		ready:             false,
		inputTokens:       inputTokens,
		outputTokens:      outputTokens,
	}
}

// extractText extracts text parts from a genai.Content.
func extractText(content *genai.Content) string {
	var texts []string
	for _, part := range content.Parts {
		if part.Text != "" {
			texts = append(texts, part.Text)
		}
	}
	return strings.Join(texts, "\n")
}

// SetProgram sets the tea.Program pointer to enable real-time updates from goroutines.
func (m *Model) SetProgram(p *tea.Program) {
	m.program = p
}

// Init initializes the model.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, tea.EnterAltScreen)
}
