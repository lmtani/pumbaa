package chat

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool"
	"google.golang.org/genai"

	"github.com/lmtani/pumbaa/internal/infrastructure/agents/tools"
	infraSession "github.com/lmtani/pumbaa/internal/infrastructure/session"
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

// Styles for chat
var (
	userStyle = lipgloss.NewStyle().
			Foreground(common.InfoColor).
			Bold(true)

	contextBadgeStyle = lipgloss.NewStyle().
				Foreground(common.BadgeFg).
				Background(common.BadgeWarnBg).
				Padding(0, 1)

	llmBadgeStyle = lipgloss.NewStyle().
			Foreground(common.BadgeFg).
			Background(common.BadgeInfoBg).
			Padding(0, 1)

	tokenBadgeStyle = lipgloss.NewStyle().
			Foreground(common.BadgeFg).
			Background(common.BadgeSuccessBg).
			Padding(0, 1)

	sessionSummaryStyle = lipgloss.NewStyle().
				Foreground(common.MutedColor).
				Italic(true)

	agentStyle = lipgloss.NewStyle().
			Foreground(common.PrimaryColor).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(common.StatusFailed).
			Bold(true)

	messageStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			MarginBottom(1)

	infoStyle = common.MutedStyle.Copy().
			Bold(true)

	infoMessageStyle = messageStyle.Copy().
				Foreground(common.MutedColor)

	inputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(common.BorderColor).
			Padding(0, 1)

	focusedInputStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(common.FocusBorder).
				Padding(0, 1)
)

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
	sessionsList        []infraSession.SessionInfo
	sessionsCursor      int
	sessionsLoading     bool
	sessionsError       string
	sessionsSearch      string
	sessionsSearching   bool
	sessionsSearchInput textinput.Model

	// standalone is true when running directly from CLI (pumbaa chat), not embedded in TUI
	standalone bool
}

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
	info  infraSession.SessionInfo
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
	sessions []infraSession.SessionInfo
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

func extractText(content *genai.Content) string {
	var texts []string
	for _, part := range content.Parts {
		if part.Text != "" {
			texts = append(texts, part.Text)
		}
	}
	return strings.Join(texts, "\n")
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

// submitInput sends the current textarea content to the LLM.
func (m *Model) submitInput() (tea.Model, tea.Cmd) {
	if m.loading {
		return m, nil
	}
	input := m.textarea.Value()
	if strings.TrimSpace(input) == "" {
		return m, nil
	}

	*m.msgs = append(*m.msgs, ChatMessage{Role: "user", Content: input})
	m.viewport.SetContent(m.renderMessages())
	m.textarea.Reset()
	m.viewport.GotoBottom()
	m.loading = true

	ctx, cancel := context.WithCancel(context.Background())
	m.cancelGen = cancel

	cmds := []tea.Cmd{m.spinner.Tick, m.generateResponse(ctx, input)}

	// Lazy session creation: the first message brings the session to life
	if m.session == nil && m.sessionService != nil && !m.sessionCreating {
		m.sessionCreating = true
		cmds = append(cmds, m.createSessionCmd())
	}

	return m, tea.Batch(cmds...)
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

// findResumableCmd looks up a previous session for the same task context, so
// the user can resume it with ctrl+r instead of starting over.
func (m *Model) findResumableCmd() tea.Cmd {
	if m.session != nil || m.contextLabel == "" {
		return nil
	}
	svc, ok := m.sessionService.(*infraSession.SQLiteService)
	if !ok {
		return nil
	}
	label := m.contextLabel
	owner := m.msgs
	return func() tea.Msg {
		info, err := svc.FindLatestByContextLabel(context.Background(), infraSession.DefaultAppName, infraSession.DefaultUserID, label)
		if err != nil || info == nil {
			return nil
		}
		return resumableFoundMsg{owner: owner, info: *info}
	}
}

// createSessionCmd creates the persistent session on first use and tags it
// with the task context for later resume-by-task lookups.
func (m *Model) createSessionCmd() tea.Cmd {
	svc := m.sessionService
	label := m.contextLabel
	owner := m.msgs
	return func() tea.Msg {
		ctx := context.Background()
		resp, err := svc.Create(ctx, &session.CreateRequest{
			AppName: infraSession.DefaultAppName,
			UserID:  infraSession.DefaultUserID,
		})
		if err != nil {
			return sessionCreatedMsg{owner: owner, err: err}
		}
		if label != "" {
			if sqliteSvc, ok := svc.(*infraSession.SQLiteService); ok {
				_ = sqliteSvc.SetContextLabel(ctx, resp.Session.ID(), label)
			}
		}
		return sessionCreatedMsg{owner: owner, session: resp.Session}
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		spCmd tea.Cmd
	)

	// Only update textarea when focused on input
	if m.focusMode == FocusInput {
		m.textarea, tiCmd = m.textarea.Update(msg)
	}
	m.spinner, spCmd = m.spinner.Update(msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		inputHeight := 5
		availableHeight := m.height - headerHeight - common.FooterHeight - inputHeight - common.PanelChrome

		if !m.ready {
			m.viewport = viewport.New(m.width-4, availableHeight)
			m.viewport.SetContent(m.renderMessages())
			m.ready = true
		} else {
			m.viewport.Width = m.width - 4
			m.viewport.Height = availableHeight
			m.viewport.SetContent(m.renderMessages())
		}
		m.textarea.SetWidth(m.width - 6)

	case tea.KeyMsg:
		// Check for active modal first
		if model, cmd, handled := m.handleActiveModalKeys(msg); handled {
			return model, cmd
		}

		switch msg.Type {
		case tea.KeyEsc:
			// While generating, ESC cancels the in-flight response.
			if m.loading && m.cancelGen != nil {
				m.cancelGen()
				return m, nil
			}
			// First ESC leaves the input; second ESC leaves the screen.
			if m.focusMode == FocusInput {
				m.SetFocusMode(FocusMessages)
				return m, nil
			}
			if m.standalone {
				return m, tea.Quit
			}
			return m, common.NavigateCmd(common.NavigateBackMsg{})

		case tea.KeyEnter:
			if m.focusMode == FocusInput {
				return m.submitInput()
			}

		case tea.KeyRunes:
			// Message-navigation keys, only when browsing messages
			if m.focusMode == FocusMessages && len(msg.Runes) > 0 && m.msgs != nil && len(*m.msgs) > 0 {
				switch msg.Runes[0] {
				case 'y': // copy selected message
					if m.selectedMsg >= 0 && m.selectedMsg < len(*m.msgs) {
						content := (*m.msgs)[m.selectedMsg].Content
						return m, copyToClipboard(content)
					}
				case 'g': // first message
					m.selectedMsg = 0
					m.viewport.SetContent(m.renderMessages())
					m.viewport.GotoTop()
					return m, nil
				case 'G': // last message
					m.selectedMsg = len(*m.msgs) - 1
					m.viewport.SetContent(m.renderMessages())
					m.viewport.GotoBottom()
					return m, nil
				}
			}

		case tea.KeyPgUp:
			m.viewport.PageUp()
			return m, nil

		case tea.KeyPgDown:
			m.viewport.PageDown()
			return m, nil

		case tea.KeyHome:
			if m.focusMode == FocusMessages {
				m.viewport.GotoTop()
				return m, nil
			}

		case tea.KeyEnd:
			if m.focusMode == FocusMessages {
				m.viewport.GotoBottom()
				return m, nil
			}

		case tea.KeyTab:
			// Toggle focus mode
			if m.focusMode == FocusInput {
				m.focusMode = FocusMessages
				m.textarea.Blur()
				// Select last message if none selected
				if m.selectedMsg < 0 && m.msgs != nil && len(*m.msgs) > 0 {
					m.selectedMsg = len(*m.msgs) - 1
				}
				m.viewport.SetContent(m.renderMessages())
				m.scrollToSelectedMsg()
			} else {
				m.focusMode = FocusInput
				m.selectedMsg = -1
				m.textarea.Focus()
				m.viewport.SetContent(m.renderMessages())
			}
			return m, nil

		case tea.KeyUp:
			if m.focusMode == FocusMessages && m.msgs != nil && len(*m.msgs) > 0 {
				if m.selectedMsg > 0 {
					m.selectedMsg--
					m.viewport.SetContent(m.renderMessages())
					m.scrollToSelectedMsg()
				}
				return m, nil
			} else if m.focusMode == FocusInput {
				// Scroll viewport up when focus is on input
				m.viewport.ScrollUp(3)
				return m, nil
			}

		case tea.KeyDown:
			if m.focusMode == FocusMessages && m.msgs != nil && len(*m.msgs) > 0 {
				if m.selectedMsg < len(*m.msgs)-1 {
					m.selectedMsg++
					m.viewport.SetContent(m.renderMessages())
					m.scrollToSelectedMsg()
				}
				return m, nil
			} else if m.focusMode == FocusInput {
				// Scroll viewport down when focus is on input
				m.viewport.ScrollDown(3)
				return m, nil
			}

		case tea.KeyCtrlD:
			// Kept as an alias for Enter for muscle memory
			if m.focusMode == FocusInput {
				return m.submitInput()
			}

		case tea.KeyCtrlS:
			// Handle Ctrl+S for sessions modal when not loading
			if !m.loading {
				return m.openSessionsModal()
			}

		case tea.KeyCtrlR:
			// Resume the previous conversation for this task context
			if m.resumableID != "" && !m.loading {
				resumeID := m.resumableID
				m.resumableID = ""
				return m.switchToSession(resumeID)
			}
		}

	case ResponseMsg:
		if msg.owner != nil && msg.owner != m.msgs {
			// Response from a previous conversation; drop it.
			return m, nil
		}
		m.loading = false
		m.toolNotification = ""
		m.cancelGen = nil
		if m.pendingFlush {
			// Session attached mid-generation; history is stable now
			m.flushHistory()
		}
		partialText := m.streamingText
		m.streamingText = ""
		if errors.Is(msg.Err, context.Canceled) {
			// User cancelled: keep whatever streamed and say so.
			if partialText != "" {
				rendered := renderMarkdown(partialText, m.width-8)
				*m.msgs = append(*m.msgs, ChatMessage{Role: "agent", Content: partialText, Rendered: rendered})
			}
			*m.msgs = append(*m.msgs, ChatMessage{Role: "notice", Content: "Generation cancelled"})
		} else if msg.Err != nil {
			*m.msgs = append(*m.msgs, ChatMessage{Role: "error", Content: fmt.Sprintf("%v", msg.Err)})
		} else {
			rendered := renderMarkdown(msg.Content, m.width-8)
			*m.msgs = append(*m.msgs, ChatMessage{Role: "agent", Content: msg.Content, Rendered: rendered})

			// Accumulate token usage
			m.inputTokens += msg.InputTokens
			m.outputTokens += msg.OutputTokens

			// Persist token usage to session
			if m.session != nil && m.sessionService != nil {
				if sqliteSvc, ok := m.sessionService.(interface {
					UpdateTokenUsage(ctx context.Context, sessionID string, inputTokens, outputTokens int) error
				}); ok {
					go sqliteSvc.UpdateTokenUsage(context.Background(), m.session.ID(), m.inputTokens, m.outputTokens)
				}
			}

			// Generate session summary after first few exchanges
			if m.session != nil && m.msgs != nil && len(*m.msgs) >= 2 {
				sessionID := m.session.ID()
				msgsCopy := make([]ChatMessage, len(*m.msgs))
				copy(msgsCopy, *m.msgs)
				go m.generateAndSaveSummaryForSession(context.Background(), sessionID, msgsCopy)
			}
		}
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		m.textarea.Focus()
		m.focusMode = FocusInput
		return m, textarea.Blink

	case sessionCreatedMsg:
		if msg.owner != m.msgs {
			return m, nil
		}
		m.sessionCreating = false
		if msg.err != nil {
			m.AppendNotice("Session persistence unavailable: " + msg.err.Error())
			return m, nil
		}
		m.resumableID = "" // this conversation now has its own session
		m.SetSession(msg.session)
		return m, nil

	case resumableFoundMsg:
		if msg.owner != m.msgs || m.session != nil {
			return m, nil
		}
		m.resumableID = msg.info.ID
		desc := msg.info.Summary
		if desc == "" {
			desc = msg.info.ContextLabel
		}
		m.AppendNotice(fmt.Sprintf(
			"Previous conversation for this task: %s (%d events, %s) — press ctrl+r to resume",
			common.Truncate(desc, 50), msg.info.EventCount, formatAge(time.Since(msg.info.UpdatedAt)),
		))
		return m, nil

	case streamChunkMsg:
		if msg.owner != m.msgs || !m.loading {
			return m, nil
		}
		m.streamingText = msg.text
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, nil

	case toolRecordMsg:
		if msg.owner != m.msgs {
			return m, nil
		}
		*m.msgs = append(*m.msgs, ChatMessage{Role: "tool", Content: msg.line})
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, nil

	case ToolNotificationMsg:
		if msg.owner != nil && msg.owner != m.msgs {
			return m, nil
		}
		if msg.Action != "" {
			// Format params if present
			paramsStr := ""
			if len(msg.Params) > 0 {
				var parts []string
				for k, v := range msg.Params {
					parts = append(parts, fmt.Sprintf("%s=%v", k, v))
				}
				paramsStr = ", " + strings.Join(parts, ", ")
			}
			m.toolNotification = fmt.Sprintf("%s%s", msg.Action, paramsStr)
		} else {
			m.toolNotification = msg.ToolName
		}
		return m, m.spinner.Tick

	case ClearNotificationMsg:
		m.toolNotification = ""
		return m, nil

	case clipboardCopiedMsg:
		if msg.success {
			m.statusMessage = "✓ Copied!"
		} else {
			m.statusMessage = "✗ Copy failed"
		}
		m.statusExpires = time.Now().Add(2 * time.Second)
		return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
			return clearStatusMsg{}
		})

	case clearStatusMsg:
		if time.Now().After(m.statusExpires) {
			m.statusMessage = ""
		}
		return m, nil

	case sessionListLoadedMsg:
		m.sessionsLoading = false
		m.sessionsList = msg.sessions
		return m, nil

	case sessionListErrorMsg:
		m.sessionsLoading = false
		m.sessionsError = msg.err.Error()
		return m, nil

	case sessionSwitchedMsg:
		m.activeModal = ModalNone
		m.sessionsLoading = false
		m.resumableID = ""
		m.session = msg.session
		m.history = &msg.history
		// Rebuild msgs from history
		msgs := make([]ChatMessage, 0)
		for _, content := range msg.history {
			role := content.Role
			if role == "model" {
				role = "agent"
			}
			text := extractText(content)
			if text != "" {
				msgs = append(msgs, ChatMessage{Role: role, Content: text})
			}
		}
		m.msgs = &msgs
		m.selectedMsg = -1
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, nil

	case sessionSummaryUpdatedMsg:
		// Only update if it's for the current session
		if m.session != nil && m.session.ID() == msg.sessionID {
			m.sessionSummary = msg.summary
		}
		return m, nil
	}

	return m, tea.Batch(tiCmd, spCmd)
}

func (m *Model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	// Check for active modal first
	if modalView, ok := m.renderActiveModal(); ok {
		return modalView
	}

	header := m.renderHeader()
	content := m.renderContent()
	input := m.renderInput()
	footer := m.renderFooter()

	return lipgloss.JoinVertical(lipgloss.Left, header, content, input, footer)
}

// headerHeight is the fixed height of the chat header: the bar plus the
// session line. Keeping it constant lets the viewport size stay stable even
// when the session summary arrives asynchronously.
const headerHeight = 2

func (m Model) renderHeader() string {
	brand := common.HeaderBrandStyle.Render("Pumbaa")

	// Breadcrumb reflects how the chat was opened
	var screens []common.Screen
	if m.standalone {
		screens = []common.Screen{{Name: "Chat", Active: true}}
	} else {
		screens = []common.Screen{
			{Name: "Dashboard", Active: false},
			{Name: "Debug", Active: false},
			{Name: "Chat", Active: true},
		}
	}
	breadcrumbs := common.RenderBreadcrumbs(screens)

	left := brand + " " + breadcrumbs
	if m.contextLabel != "" {
		left += " " + contextBadgeStyle.Render(m.contextLabel)
	}

	// Right side: LLM provider and token usage
	var right []string
	if m.llm != nil {
		right = append(right, llmBadgeStyle.Render(m.llm.Name()))
	}
	if m.inputTokens > 0 || m.outputTokens > 0 {
		right = append(right, tokenBadgeStyle.Render(fmt.Sprintf("%s↑ %s↓", formatTokenCount(m.inputTokens), formatTokenCount(m.outputTokens))))
	}

	bar := common.RenderHeaderBar(m.width, left, strings.Join(right, " "))

	// Session line: summary when available, otherwise the session ID
	sessionLine := common.MutedStyle.Render("No session")
	if m.sessionSummary != "" {
		sessionLine = sessionSummaryStyle.Render(common.Truncate(m.sessionSummary, m.width-2))
	} else if m.session != nil {
		sessionID := m.session.ID()
		if len(sessionID) > 12 {
			sessionID = sessionID[:12] + "…"
		}
		sessionLine = common.MutedStyle.Render("Session: " + sessionID)
	}

	return lipgloss.JoinVertical(lipgloss.Left, bar, " "+sessionLine)
}

func (m Model) renderContent() string {
	return common.PanelStyle.
		Width(m.width - 2).
		Height(m.viewport.Height).
		Render(m.viewport.View())
}

func (m Model) renderInput() string {
	var inputBox string
	if m.loading {
		var loadingText string
		if m.toolNotification != "" {
			loadingText = fmt.Sprintf("%s Executing: %s", m.spinner.View(), m.toolNotification)
		} else {
			loadingText = fmt.Sprintf("%s Thinking...", m.spinner.View())
		}
		inputBox = inputStyle.
			Width(m.width - 4).
			Render(loadingText)
	} else {
		inputBox = focusedInputStyle.
			Width(m.width - 4).
			Render(m.textarea.View())
	}
	return inputBox
}

// scrollToSelectedMsg scrolls the viewport so the selected message is
// visible, using the real rendered line offsets tracked by renderMessages.
func (m *Model) scrollToSelectedMsg() {
	if m.msgs == nil || m.selectedMsg < 0 || m.selectedMsg >= len(*m.msgs) || m.selectedMsg >= len(m.msgOffsets) {
		return
	}

	// Place the message a third from the top: it reads naturally and leaves
	// room for the following context.
	offset := m.msgOffsets[m.selectedMsg] - m.viewport.Height/3
	if offset < 0 {
		offset = 0
	}
	m.viewport.SetYOffset(offset)
}

func (m Model) renderFooter() string {
	// Determine if ESC should show "back" or "quit"
	escAction := "back"
	if m.standalone {
		escAction = "quit"
	}

	hint := func(key, desc string) string {
		return common.KeyStyle.Render(key) + " " + common.DescStyle.Render(desc)
	}
	var hints []string
	switch {
	case m.loading:
		hints = []string{
			hint("esc", "cancel"),
			hint("↑↓", "scroll"),
		}
	case m.focusMode == FocusMessages:
		hints = []string{
			hint("↑↓", "navigate"),
			hint("pgup/pgdn", "page"),
			hint("g/G", "first/last"),
			hint("y", "copy"),
			hint("tab", "type"),
			hint("esc", escAction),
		}
	default:
		hints = []string{
			hint("enter", "send"),
			hint("ctrl+j", "newline"),
		}
		if m.resumableID != "" {
			hints = append(hints, hint("ctrl+r", "resume previous"))
		}
		hints = append(hints,
			hint("ctrl+s", "sessions"),
			hint("↑↓", "scroll"),
			hint("tab", "navigate msgs"),
			hint("esc", "messages"),
		)
	}

	// Show status message if present
	prefix := ""
	if m.statusMessage != "" {
		prefix = common.SuccessStyle.Render(m.statusMessage) + "  "
	}

	// Only as many hints as fit on one line, so the footer never wraps
	help := common.FitParts(m.width-2-lipgloss.Width(prefix), "  ", hints)

	return common.HelpBarStyle.
		Width(m.width).
		Render(prefix + help)
}

func (m *Model) renderMessages() string {
	hasMsgs := m.msgs != nil && len(*m.msgs) > 0
	if !hasMsgs && m.streamingText == "" {
		m.msgOffsets = nil
		return common.MutedStyle.Render("Welcome to Pumbaa Chat! 🐗\n\nType your message and press Enter to send (ctrl+j for a new line).")
	}

	var sb strings.Builder
	maxWidth := m.width - 8
	if maxWidth <= 0 {
		maxWidth = 80
	}

	// Track each message's rendered line offset for selection scrolling
	lineCount := 0
	offsets := make([]int, 0, 16)

	if hasMsgs {
		for i, msg := range *m.msgs {
			block := m.renderMessageBlock(i, msg, maxWidth)
			offsets = append(offsets, lineCount)
			sb.WriteString(block)
			lineCount += strings.Count(block, "\n")
		}
	}
	m.msgOffsets = offsets

	// Live response being streamed: plain text with a cursor block; the
	// final ResponseMsg replaces it with the markdown-rendered message.
	if m.loading && m.streamingText != "" {
		sb.WriteString(agentStyle.Render("Pumbaa") + "\n")
		sb.WriteString(messageStyle.Render(wrapText(m.streamingText, maxWidth) + " ▌"))
		sb.WriteString("\n\n")
	}

	return sb.String()
}

// renderMessageBlock renders a single transcript message, including its
// trailing spacing, so renderMessages can measure real line offsets.
func (m *Model) renderMessageBlock(i int, msg ChatMessage, maxWidth int) string {
	isSelected := m.focusMode == FocusMessages && i == m.selectedMsg
	selectedStyle := lipgloss.NewStyle().
		Background(common.SubtleColor).
		Padding(0, 1)

	// Compact one-line roles: tool records and notices have no header
	switch msg.Role {
	case "tool":
		line := common.MutedStyle.Render("🔧 " + msg.Content)
		if isSelected {
			line = selectedStyle.Render("▶ " + line)
		}
		return line + "\n\n"
	case "notice":
		line := common.MutedStyle.Italic(true).Render("· " + msg.Content)
		if isSelected {
			line = selectedStyle.Render("▶ " + line)
		}
		return line + "\n\n"
	}

	var roleStyle lipgloss.Style
	var roleName string
	contentStyle := messageStyle

	switch msg.Role {
	case "user":
		roleStyle = userStyle
		roleName = "You"
	case "agent":
		roleStyle = agentStyle
		roleName = "Pumbaa"
	case "info":
		roleStyle = infoStyle
		roleName = "Context"
		contentStyle = infoMessageStyle
	default:
		roleStyle = errorStyle
		roleName = "Error"
	}

	var sb strings.Builder

	// Render role with selection indicator
	if isSelected {
		sb.WriteString(selectedStyle.Render("▶ "+roleStyle.Render(roleName)) + "\n")
	} else {
		sb.WriteString(roleStyle.Render(roleName) + "\n")
	}

	// Render content
	var content string
	if msg.Role == "agent" && msg.Rendered != "" {
		content = msg.Rendered
	} else {
		content = contentStyle.Render(wrapText(msg.Content, maxWidth))
	}

	if isSelected {
		sb.WriteString(selectedStyle.Render(content))
	} else {
		sb.WriteString(content)
	}
	sb.WriteString("\n\n")

	return sb.String()
}

// renderMarkdown renders markdown content using glamour
func renderMarkdown(content string, width int) string {
	if width <= 20 {
		width = 80
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(width),
		glamour.WithEmoji(),
	)
	if err != nil {
		return content
	}

	rendered, err := renderer.Render(content)
	if err != nil {
		return content
	}

	return strings.TrimSpace(rendered)
}

// copyToClipboard creates a tea.Cmd that copies text to the system clipboard
func copyToClipboard(text string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd

		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("pbcopy")
		case "linux":
			if _, err := exec.LookPath("xclip"); err == nil {
				cmd = exec.Command("xclip", "-selection", "clipboard")
			} else if _, err := exec.LookPath("xsel"); err == nil {
				cmd = exec.Command("xsel", "--clipboard", "--input")
			} else if _, err := exec.LookPath("wl-copy"); err == nil {
				cmd = exec.Command("wl-copy")
			} else {
				return clipboardCopiedMsg{success: false, err: fmt.Errorf("no clipboard tool found")}
			}
		default:
			return clipboardCopiedMsg{success: false, err: fmt.Errorf("unsupported OS: %s", runtime.GOOS)}
		}

		stdin, err := cmd.StdinPipe()
		if err != nil {
			return clipboardCopiedMsg{success: false, err: err}
		}

		if err := cmd.Start(); err != nil {
			return clipboardCopiedMsg{success: false, err: err}
		}

		_, err = stdin.Write([]byte(text))
		if err != nil {
			return clipboardCopiedMsg{success: false, err: err}
		}
		stdin.Close()

		if err := cmd.Wait(); err != nil {
			return clipboardCopiedMsg{success: false, err: err}
		}

		return clipboardCopiedMsg{success: true}
	}
}

// wrapText wraps text to the given width
func wrapText(text string, width int) string {
	if width <= 0 {
		width = 80
	}

	var result strings.Builder
	lines := strings.Split(text, "\n")

	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}

		// Handle lines that are too long
		for len(line) > width {
			// Find a good break point
			breakPoint := width
			for breakPoint > 0 && line[breakPoint] != ' ' {
				breakPoint--
			}
			if breakPoint == 0 {
				breakPoint = width // Force break if no space found
			}

			result.WriteString(line[:breakPoint])
			result.WriteString("\n")
			line = strings.TrimLeft(line[breakPoint:], " ")
		}
		result.WriteString(line)
	}

	return result.String()
}

// formatAge renders a duration as a compact age, e.g. "2d ago".
func formatAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

// formatTokenCount formats a token count in a human-readable format (e.g., 1.2K, 3.4M)
func formatTokenCount(count int) string {
	if count >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(count)/1000000)
	}
	if count >= 1000 {
		return fmt.Sprintf("%.1fK", float64(count)/1000)
	}
	return fmt.Sprintf("%d", count)
}

func (m Model) generateResponse(ctx context.Context, input string) tea.Cmd {
	return func() tea.Msg {
		userContent := &genai.Content{
			Role: "user",
			Parts: []*genai.Part{
				genai.NewPartFromText(input),
			},
		}

		*m.history = append(*m.history, userContent)

		if m.sessionService != nil && m.session != nil {
			ev := session.NewEvent("")
			ev.Content = userContent
			ev.Author = "user"
			m.sessionService.AppendEvent(ctx, m.session, ev)
		}

		maxTurns := 15
		currentTurn := 0
		totalInputTokens := 0
		totalOutputTokens := 0

		for currentTurn < maxTurns {
			if ctx.Err() != nil {
				return ResponseMsg{Err: ctx.Err(), owner: m.msgs}
			}

			req := &model.LLMRequest{
				Contents: *m.history,
				Config: &genai.GenerateContentConfig{
					Tools: convertToolsToGenAI(m.tools),
				},
			}

			if m.systemInstruction != "" {
				req.Config.SystemInstruction = &genai.Content{
					Parts: []*genai.Part{
						genai.NewPartFromText(m.systemInstruction),
					},
				}
			}

			respSeq := m.llm.GenerateContent(ctx, req, true)

			var lastResp *model.LLMResponse
			var turnText strings.Builder

			for r, e := range respSeq {
				if e != nil {
					return ResponseMsg{Err: e, owner: m.msgs}
				}
				if r.Partial {
					// Forward the accumulated turn text so the UI renders
					// the response as it streams.
					turnText.WriteString(extractText(r.Content))
					if m.program != nil {
						m.program.Send(streamChunkMsg{owner: m.msgs, text: turnText.String()})
					}
					continue
				}
				lastResp = r
			}

			if lastResp == nil || lastResp.Content == nil {
				return ResponseMsg{Err: fmt.Errorf("empty response from model"), owner: m.msgs}
			}

			// Accumulate token usage from this response
			if lastResp.UsageMetadata != nil {
				totalInputTokens += int(lastResp.UsageMetadata.PromptTokenCount)
				totalOutputTokens += int(lastResp.UsageMetadata.CandidatesTokenCount)
			}

			*m.history = append(*m.history, lastResp.Content)

			if m.sessionService != nil && m.session != nil {
				ev := session.NewEvent("")
				ev.Content = lastResp.Content
				ev.Author = "model"
				m.sessionService.AppendEvent(ctx, m.session, ev)
			}

			toolCalls := getToolCalls(lastResp.Content)
			if len(toolCalls) > 0 {
				var toolParts []*genai.Part

				for _, tc := range toolCalls {
					// Extract action from tool args if available
					action := ""
					if tc.Args != nil {
						if actionVal, ok := tc.Args["action"]; ok {
							if actionStr, ok := actionVal.(string); ok {
								action = actionStr
							}
						}
					}

					// Collect other relevant params to show
					otherParams := make(map[string]any)
					for k, v := range tc.Args {
						if k != "action" && v != nil && v != "" {
							otherParams[k] = v
						}
					}

					// Send notification to UI about tool being called
					if m.program != nil {
						m.program.Send(ToolNotificationMsg{ToolName: tc.Name, Action: action, Params: otherParams, owner: m.msgs})
					}

					toolStart := time.Now()
					result, err := m.executeTool(ctx, tc)
					if m.program != nil {
						// Persistent transcript record of what the agent did
						m.program.Send(toolRecordMsg{
							owner: m.msgs,
							line:  formatToolRecord(tc.Name, action, otherParams, time.Since(toolStart), toolFailure(result, err)),
						})
					}
					if err != nil {
						toolParts = append(toolParts, &genai.Part{
							FunctionResponse: &genai.FunctionResponse{
								Name: tc.Name,
								Response: map[string]any{
									"error": err.Error(),
								},
							},
						})
					} else {
						toolParts = append(toolParts, &genai.Part{
							FunctionResponse: &genai.FunctionResponse{
								Name:     tc.Name,
								Response: result,
							},
						})
					}
				}

				toolContent := &genai.Content{
					Role:  "tool",
					Parts: toolParts,
				}

				*m.history = append(*m.history, toolContent)

				if m.sessionService != nil && m.session != nil {
					ev := session.NewEvent("")
					ev.Content = toolContent
					ev.Author = "tool"
					m.sessionService.AppendEvent(ctx, m.session, ev)
				}

				currentTurn++
				continue
			}

			text := ""
			for _, part := range lastResp.Content.Parts {
				if part.Text != "" {
					text += part.Text
				}
			}
			return ResponseMsg{Content: text, InputTokens: totalInputTokens, OutputTokens: totalOutputTokens, owner: m.msgs}
		}

		// Max turns reached
		var summary strings.Builder
		summary.WriteString("⚠ I reached the tool iteration limit.\n\n")
		summary.WriteString("**Information collected so far:**\n")

		toolResultCount := 0
		for _, content := range *m.history {
			if content.Role == "tool" {
				toolResultCount++
			}
		}

		if toolResultCount > 0 {
			summary.WriteString(fmt.Sprintf("- Executed %d tool calls\n", toolResultCount))
		}

		summary.WriteString("\nIf you need more information, please ask a more specific question or ask me to continue from where I left off.")

		return ResponseMsg{Content: summary.String(), owner: m.msgs}
	}
}

// toolFailure extracts a short failure description from a tool execution:
// either a transport error or a handler-level error output (success=false).
// Returns "" when the call succeeded.
func toolFailure(result map[string]any, err error) string {
	if err != nil {
		return err.Error()
	}
	if result != nil {
		if success, ok := result["success"].(bool); ok && !success {
			if msg, ok := result["error"].(string); ok && msg != "" {
				return msg
			}
			return "failed"
		}
	}
	return ""
}

// formatToolRecord builds the one-line transcript record of a tool call,
// e.g. `pumbaa query (status=Failed) ✓ 0.8s`. Failures include a short
// reason so the transcript explains itself.
func formatToolRecord(name, action string, params map[string]any, dur time.Duration, failure string) string {
	label := name
	if action != "" {
		label += " " + action
	}

	if len(params) > 0 {
		keys := make([]string, 0, len(params))
		for k := range params {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		pairs := make([]string, 0, len(keys))
		for _, k := range keys {
			pairs = append(pairs, fmt.Sprintf("%s=%v", k, params[k]))
		}
		label += " (" + strings.Join(pairs, ", ") + ")"
	}

	if failure != "" {
		return fmt.Sprintf("%s ✗ %.1fs — %s", label, dur.Seconds(), common.Truncate(failure, 80))
	}
	return fmt.Sprintf("%s ✓ %.1fs", label, dur.Seconds())
}

func getToolCalls(content *genai.Content) []*genai.FunctionCall {
	var calls []*genai.FunctionCall
	for _, part := range content.Parts {
		if part.FunctionCall != nil {
			calls = append(calls, part.FunctionCall)
		}
	}
	return calls
}

func (m Model) executeTool(ctx context.Context, fc *genai.FunctionCall) (map[string]any, error) {
	for _, t := range m.tools {
		if td, ok := t.(toolWithDefinition); ok {
			def := td.Declaration()
			if def.Name == fc.Name {
				// functiontool.Run dereferences the tool context (ADK v1.0.0
				// panics on nil), so pass the minimal no-op context.
				return td.Run(tools.NoopToolContext(), fc.Args)
			}
		}
	}
	return nil, fmt.Errorf("tool not found: %s", fc.Name)
}

func convertToolsToGenAI(tools []tool.Tool) []*genai.Tool {
	var genaiTools []*genai.Tool
	for _, t := range tools {
		if td, ok := t.(toolWithDefinition); ok {
			genaiTools = append(genaiTools, &genai.Tool{
				FunctionDeclarations: []*genai.FunctionDeclaration{
					td.Declaration(),
				},
			})
		}
	}
	return genaiTools
}
