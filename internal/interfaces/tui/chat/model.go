package chat

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
	"google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool"
	"google.golang.org/genai"
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
	toolNotification string // Temporary notification for tool calls

	// Display messages (pointer for persistence)
	msgs *[]ChatMessage
}

type ChatMessage struct {
	Role    string
	Content string
}

type ResponseMsg struct {
	Content string
	Err     error
}

// ToolNotificationMsg is sent when a tool is being called
type ToolNotificationMsg struct {
	ToolName string
	Action   string
}

// ClearNotificationMsg is sent to clear the tool notification
type ClearNotificationMsg struct{}

// NewModel creates a new chat model with the given LLM, tools, system instruction, and session.
func NewModel(llm model.LLM, tools []tool.Tool, systemInstruction string, svc session.Service, sess session.Session) Model {
	ta := textarea.New()
	ta.Placeholder = "Digite sua mensagem..."
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
		vpCmd tea.Cmd
		spCmd tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)
	m.spinner, spCmd = m.spinner.Update(msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Calculate available heights
		// Header: border (2) + padding (0) + content (1) = 3 lines
		// Footer: border (1) + padding (0) + content (1) = 2 lines
		// Input:  border (2) + padding (0) + content (3) = 5 lines
		// Content panel: border (2) adds to viewport height
		headerHeight := 3
		footerHeight := 2
		inputHeight := 5
		contentBorderHeight := 2 // Top and bottom border of content panel

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
		}

	case ResponseMsg:
		m.loading = false
		m.toolNotification = "" // Clear notification
		if msg.Err != nil {
			*m.msgs = append(*m.msgs, ChatMessage{Role: "error", Content: fmt.Sprintf("%v", msg.Err)})
		} else {
			*m.msgs = append(*m.msgs, ChatMessage{Role: "agent", Content: msg.Content})
		}
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		m.textarea.Focus()
		return m, textarea.Blink

	case ToolNotificationMsg:
		if msg.Action != "" {
			m.toolNotification = fmt.Sprintf("%s (%s)", msg.ToolName, msg.Action)
		} else {
			m.toolNotification = msg.ToolName
		}
		return m, m.spinner.Tick

	case ClearNotificationMsg:
		m.toolNotification = ""
		return m, nil
	}

	return m, tea.Batch(tiCmd, vpCmd, spCmd)
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
			loadingText = fmt.Sprintf("%s 🔧 Executando: %s", m.spinner.View(), m.toolNotification)
		} else {
			loadingText = fmt.Sprintf("%s Processando...", m.spinner.View())
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

func (m Model) renderFooter() string {
	help := fmt.Sprintf(
		"%s %s  %s %s  %s %s",
		common.KeyStyle.Render("ctrl+d"),
		common.DescStyle.Render("enviar"),
		common.KeyStyle.Render("↑↓"),
		common.DescStyle.Render("scroll"),
		common.KeyStyle.Render("esc"),
		common.DescStyle.Render("sair"),
	)

	return common.HelpBarStyle.
		Width(m.width - 2).
		Render(help)
}

func (m Model) renderMessages() string {
	if m.msgs == nil || len(*m.msgs) == 0 {
		return common.MutedStyle.Render("Bem-vindo ao Pumbaa Chat!\n\nDigite sua mensagem e pressione Ctrl+D para enviar.")
	}

	var sb strings.Builder
	maxWidth := m.width - 8 // Account for padding and borders

	for _, msg := range *m.msgs {
		var roleStyle lipgloss.Style
		var roleName string

		switch msg.Role {
		case "user":
			roleStyle = userStyle
			roleName = "Você"
		case "agent":
			roleStyle = agentStyle
			roleName = "Pumbaa"
		default:
			roleStyle = errorStyle
			roleName = "Erro"
		}

		// Render role
		sb.WriteString(roleStyle.Render(roleName) + "\n")

		// Wrap content to fit width
		wrappedContent := wrapText(msg.Content, maxWidth)
		sb.WriteString(messageStyle.Render(wrappedContent))
		sb.WriteString("\n\n")
	}

	return sb.String()
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

					// Send notification to UI about tool being called
					if m.program != nil {
						m.program.Send(ToolNotificationMsg{ToolName: tc.Name, Action: action})
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
		summary.WriteString("⚠️ Atingi o limite de iterações de ferramentas.\n\n")
		summary.WriteString("**Informações coletadas até agora:**\n")

		toolResultCount := 0
		for _, content := range *m.history {
			if content.Role == "tool" {
				toolResultCount++
			}
		}

		if toolResultCount > 0 {
			summary.WriteString(fmt.Sprintf("- Executei %d chamadas de ferramentas\n", toolResultCount))
		}

		summary.WriteString("\nSe precisar de mais informações, por favor faça uma pergunta mais específica ou peça para eu continuar de onde parei.")

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
