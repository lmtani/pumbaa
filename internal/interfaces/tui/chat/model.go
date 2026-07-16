package chat

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool"
	"google.golang.org/genai"

	"github.com/lmtani/pumbaa/internal/application/ports"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// FocusMode indicates which panel has focus
type FocusMode int

const (
	FocusInput FocusMode = iota
	FocusMessages
)

// Interface to access the hidden definition method of functiontool
type toolWithDefinition interface {
	Declaration() *genai.FunctionDeclaration
	Run(ctx tool.Context, args any) (map[string]any, error)
}

// Model is the Bubble Tea model for the chat interface.
type Model struct {
	llm               model.LLM
	tools             []tool.Tool
	systemInstruction string
	contextLabel      string // Optional context label shown in header

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

	// Streaming state: text of the in-progress response, shown live at the
	// end of the transcript until the final ResponseMsg replaces it.
	streamingText string

	// cancelGen aborts the in-flight generation (ESC while loading).
	cancelGen context.CancelFunc

	// Lazy session lifecycle: the session is created on the first message so
	// abandoned chats never leave empty rows behind.
	sessionCreating bool
	pendingFlush    bool // persist history once the in-flight generation ends

	// resumableID points at a previous session for the same task context,
	// offered for resume via ctrl+r.
	resumableID string

	// msgOffsets holds the rendered line offset of each message, refreshed
	// by renderMessages, so selection scrolling targets real positions.
	msgOffsets []int

	// Token usage tracking
	inputTokens  int // Accumulated input tokens for the session
	outputTokens int // Accumulated output tokens for the session

	// Display messages (pointer for persistence)
	msgs *[]ChatMessage

	// Session summary for header display
	sessionSummary string

	// Modal state
	activeModal         ModalKind
	sessionsList        []ports.ChatSessionInfo
	sessionsCursor      int
	sessionsLoading     bool
	sessionsError       string
	sessionsSearch      string
	sessionsSearching   bool
	sessionsSearchInput textinput.Model

	// standalone is true when running directly from CLI (pumbaa chat), not embedded in TUI
	standalone bool
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
	// Enter submits (handled in Update); newlines are inserted with ctrl+j
	ta.KeyMap.InsertNewline = key.NewBinding(
		key.WithKeys("ctrl+j"),
		key.WithHelp("ctrl+j", "newline"),
	)

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

// SetProgram sets the tea.Program pointer to enable real-time updates from goroutines
func (m *Model) SetProgram(p *tea.Program) {
	m.program = p
}

// SetContextLabel sets the optional context label shown in the header.
func (m *Model) SetContextLabel(label string) {
	m.contextLabel = label
}

// SetStandalone sets whether the chat is running in standalone mode (directly from CLI).
// When true, exit commands will close the program. When false, they navigate back.
func (m *Model) SetStandalone(standalone bool) {
	m.standalone = standalone
}

// AddInfoMessage adds a muted informational message to the message list.
func (m *Model) AddInfoMessage(content string) {
	if m.msgs == nil || strings.TrimSpace(content) == "" {
		return
	}
	msg := ChatMessage{Role: "info", Content: content}
	if len(*m.msgs) == 0 {
		*m.msgs = append(*m.msgs, msg)
		return
	}
	*m.msgs = append([]ChatMessage{msg}, (*m.msgs)...)
}

// SetSession attaches a session created asynchronously after the screen
// opened, so session creation never blocks the UI thread. Turns exchanged
// while the session was still being created are persisted retroactively.
func (m *Model) SetSession(sess session.Session) {
	m.session = sess
	if sess == nil {
		return
	}
	if m.loading {
		// A generation is appending to history from its own goroutine;
		// copying it here would race. Flush once the response lands.
		m.pendingFlush = true
		return
	}
	m.flushHistory()
}

// flushHistory persists the whole in-memory history to the session. Used
// when the session attaches after turns already happened (lazy creation).
func (m *Model) flushHistory() {
	m.pendingFlush = false
	if m.session == nil || m.sessionService == nil || m.history == nil || len(*m.history) == 0 {
		return
	}

	backlog := make([]*genai.Content, len(*m.history))
	copy(backlog, *m.history)
	svc := m.sessionService
	sess := m.session
	go func() {
		ctx := context.Background()
		for _, content := range backlog {
			ev := session.NewEvent("")
			ev.Content = content
			ev.Author = content.Role
			svc.AppendEvent(ctx, sess, ev)
		}
	}()
}

// IsBusy reports whether a response is currently being generated.
func (m *Model) IsBusy() bool {
	return m.loading
}

// ResumeCmd re-arms the spinner when the screen becomes current again while
// a response is still being generated (spinner ticks are focus-only).
func (m *Model) ResumeCmd() tea.Cmd {
	if m.loading {
		return m.spinner.Tick
	}
	return nil
}

// SetSystemInstruction replaces the system instruction for future turns.
// Used when the user re-enters the chat for the same task with freshly
// collected context.
func (m *Model) SetSystemInstruction(instruction string) {
	m.systemInstruction = instruction
}

// AppendNotice appends a muted one-line notice to the end of the transcript.
func (m *Model) AppendNotice(content string) {
	if m.msgs == nil || strings.TrimSpace(content) == "" {
		return
	}
	*m.msgs = append(*m.msgs, ChatMessage{Role: "notice", Content: content})
	if m.ready {
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
	}
}

// SetFocusMode sets the focus mode and updates UI state accordingly.
func (m *Model) SetFocusMode(mode FocusMode) {
	m.focusMode = mode
	if mode == FocusInput {
		m.selectedMsg = -1
		m.textarea.Focus()
	} else {
		m.textarea.Blur()
		// Select last message if none selected
		if m.selectedMsg < 0 && m.msgs != nil && len(*m.msgs) > 0 {
			m.selectedMsg = len(*m.msgs) - 1
		}
	}
	m.viewport.SetContent(m.renderMessages())
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, tea.EnterAltScreen, m.findResumableCmd())
}
