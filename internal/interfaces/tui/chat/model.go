package chat

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
	"google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool"
	"google.golang.org/genai"
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
	Run(ctx tool.Context, args interface{}) (map[string]interface{}, error)
}

// Styles for chat
var (
	userStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00BFFF")).
			Bold(true)

	agentStyle = lipgloss.NewStyle().
			Foreground(common.PrimaryColor).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(common.StatusFailed).
			Bold(true)

	messageStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			MarginBottom(1)

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

	// Display messages (pointer for persistence)
	msgs *[]ChatMessage
}

type ChatMessage struct {
	Role     string
	Content  string
	Rendered string // Pre-rendered markdown (cached)
}

type ResponseMsg struct {
	Content string
	Err     error
}

// ToolNotificationMsg is sent when a tool is being called
type ToolNotificationMsg struct {
	ToolName string
	Action   string
	Params   map[string]interface{} // Additional parameters
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

func (m *Model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, tea.EnterAltScreen)
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

		headerHeight := 3
		footerHeight := 2
		inputHeight := 5
		contentBorderHeight := 2

		availableHeight := m.height - headerHeight - footerHeight - inputHeight - contentBorderHeight

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
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

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
				m.viewport.LineUp(3)
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
				m.viewport.LineDown(3)
				return m, nil
			}

		case tea.KeyCtrlD:
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

			return m, tea.Batch(m.spinner.Tick, m.generateResponse(input))

		case tea.KeyRunes:
			// Handle 'y' for copy when in messages mode
			if m.focusMode == FocusMessages && len(msg.Runes) > 0 && msg.Runes[0] == 'y' {
				if m.msgs != nil && m.selectedMsg >= 0 && m.selectedMsg < len(*m.msgs) {
					content := (*m.msgs)[m.selectedMsg].Content
					return m, copyToClipboard(content)
				}
			}
		}

	case ResponseMsg:
		m.loading = false
		m.toolNotification = ""
		if msg.Err != nil {
			*m.msgs = append(*m.msgs, ChatMessage{Role: "error", Content: fmt.Sprintf("%v", msg.Err)})
		} else {
			rendered := renderMarkdown(msg.Content, m.width-8)
			*m.msgs = append(*m.msgs, ChatMessage{Role: "agent", Content: msg.Content, Rendered: rendered})
		}
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		m.textarea.Focus()
		m.focusMode = FocusInput
		return m, textarea.Blink

	case ToolNotificationMsg:
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
	}

	return m, tea.Batch(tiCmd, spCmd)
}

func (m *Model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	header := m.renderHeader()
	content := m.renderContent()
	input := m.renderInput()
	footer := m.renderFooter()

	return lipgloss.JoinVertical(lipgloss.Left, header, content, input, footer)
}

func (m Model) renderHeader() string {
	title := common.HeaderTitleStyle.Render("🐗 Pumbaa Chat")

	sessionInfo := ""
	if m.session != nil {
		sessionInfo = common.MutedStyle.Render(fmt.Sprintf("Session: %s", m.session.ID()))
	}

	headerContent := lipgloss.JoinHorizontal(lipgloss.Center, title, "  ", sessionInfo)

	return common.HeaderStyle.
		Width(m.width - 2).
		Render(headerContent)
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
			loadingText = fmt.Sprintf("%s 🔧 Executing: %s", m.spinner.View(), m.toolNotification)
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

// scrollToSelectedMsg scrolls the viewport to center the selected message
func (m *Model) scrollToSelectedMsg() {
	if m.msgs == nil || m.selectedMsg < 0 || m.selectedMsg >= len(*m.msgs) {
		return
	}

	// Calculate approximate position based on message index
	// Each message is roughly: role line (1) + content lines + spacing (2)
	// We estimate ~6 lines per message on average for a good approximation
	linesPerMsg := 6
	targetLine := m.selectedMsg * linesPerMsg

	// Center the message in the viewport
	viewportHeight := m.viewport.Height
	centeredOffset := targetLine - (viewportHeight / 2)

	// Clamp to valid range
	if centeredOffset < 0 {
		centeredOffset = 0
	}

	m.viewport.SetYOffset(centeredOffset)
}

func (m Model) renderFooter() string {
	var help string
	if m.focusMode == FocusMessages {
		help = fmt.Sprintf(
			"%s %s  %s %s  %s %s  %s %s",
			common.KeyStyle.Render("↑↓"),
			common.DescStyle.Render("navigate"),
			common.KeyStyle.Render("y"),
			common.DescStyle.Render("copy"),
			common.KeyStyle.Render("tab"),
			common.DescStyle.Render("type"),
			common.KeyStyle.Render("esc"),
			common.DescStyle.Render("exit"),
		)
	} else {
		help = fmt.Sprintf(
			"%s %s  %s %s  %s %s  %s %s",
			common.KeyStyle.Render("ctrl+d"),
			common.DescStyle.Render("send"),
			common.KeyStyle.Render("↑↓"),
			common.DescStyle.Render("scroll"),
			common.KeyStyle.Render("tab"),
			common.DescStyle.Render("navigate msgs"),
			common.KeyStyle.Render("esc"),
			common.DescStyle.Render("exit"),
		)
	}

	// Show status message if present
	if m.statusMessage != "" {
		help = common.SuccessStyle.Render(m.statusMessage) + "  " + help
	}

	return common.HelpBarStyle.
		Width(m.width - 2).
		Render(help)
}

func (m Model) renderMessages() string {
	if m.msgs == nil || len(*m.msgs) == 0 {
		return common.MutedStyle.Render("Welcome to Pumbaa Chat!\n\nType your message and press Ctrl+D to send.")
	}

	var sb strings.Builder
	maxWidth := m.width - 8
	if maxWidth <= 0 {
		maxWidth = 80
	}

	// Style for selected message
	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#3a3a3a")).
		Padding(0, 1)

	for i, msg := range *m.msgs {
		var roleStyle lipgloss.Style
		var roleName string

		switch msg.Role {
		case "user":
			roleStyle = userStyle
			roleName = "You"
		case "agent":
			roleStyle = agentStyle
			roleName = "Pumbaa"
		default:
			roleStyle = errorStyle
			roleName = "Error"
		}

		// Highlight if selected
		isSelected := m.focusMode == FocusMessages && i == m.selectedMsg

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
			content = messageStyle.Render(wrapText(msg.Content, maxWidth))
		}

		if isSelected {
			sb.WriteString(selectedStyle.Render(content))
		} else {
			sb.WriteString(content)
		}
		sb.WriteString("\n\n")
	}

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

func (m Model) generateResponse(input string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

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

		for currentTurn < maxTurns {
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

			respSeq := m.llm.GenerateContent(ctx, req, false)

			var lastResp *model.LLMResponse

			for r, e := range respSeq {
				if e != nil {
					return ResponseMsg{Err: e}
				}
				lastResp = r
			}

			if lastResp == nil || lastResp.Content == nil {
				return ResponseMsg{Err: fmt.Errorf("empty response from model")}
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
					otherParams := make(map[string]interface{})
					for k, v := range tc.Args {
						if k != "action" && v != nil && v != "" {
							otherParams[k] = v
						}
					}

					// Send notification to UI about tool being called
					if m.program != nil {
						m.program.Send(ToolNotificationMsg{ToolName: tc.Name, Action: action, Params: otherParams})
					}

					result, err := m.executeTool(ctx, tc)
					if err != nil {
						toolParts = append(toolParts, &genai.Part{
							FunctionResponse: &genai.FunctionResponse{
								Name: tc.Name,
								Response: map[string]interface{}{
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
			return ResponseMsg{Content: text}
		}

		// Max turns reached
		var summary strings.Builder
		summary.WriteString("⚠️ I reached the tool iteration limit.\n\n")
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

		return ResponseMsg{Content: summary.String()}
	}
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
				return td.Run(nil, fc.Args)
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
