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

	// State
	loading bool
	ready   bool

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

// NewModel creates a new chat model with the given LLM, tools, system instruction, and session.
func NewModel(llm model.LLM, tools []tool.Tool, systemInstruction string, svc session.Service, sess session.Session) Model {
	ta := textarea.New()
	ta.Placeholder = "Type your message... (Ctrl+D to send, Esc to quit)"
	ta.Focus()
	ta.Prompt = "┃ "
	ta.CharLimit = 2000
	ta.SetWidth(80)
	ta.SetHeight(3)
	ta.ShowLineNumbers = false

	vp := viewport.New(80, 20)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	// Load history from session events
	history := make([]*genai.Content, 0)
	msgs := make([]ChatMessage, 0)

	// If session has events, load them
	if sess != nil && sess.Events() != nil && sess.Events().Len() > 0 {
		for ev := range sess.Events().All() {
			if ev.Content != nil {
				history = append(history, ev.Content)
				// Also populate display messages
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

	m := Model{
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

	// Set initial content
	if len(msgs) > 0 {
		vp.SetContent(m.renderMessages())
	} else {
		vp.SetContent("Welcome to Pumbaa Chat! Ask me about your Cromwell workflows.\n\nPress Ctrl+D to send a message, Esc to quit.\n")
	}

	return m
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

func (m Model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, tea.EnterAltScreen)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		headerHeight := 0
		footerHeight := 6
		verticalMarginHeight := headerHeight + footerHeight

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.SetContent(m.renderMessages())
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}
		m.textarea.SetWidth(msg.Width - 2)

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
		if msg.Err != nil {
			*m.msgs = append(*m.msgs, ChatMessage{Role: "error", Content: fmt.Sprintf("Error: %v", msg.Err)})
		} else {
			*m.msgs = append(*m.msgs, ChatMessage{Role: "agent", Content: msg.Content})
		}
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		m.textarea.Focus()
		return m, textarea.Blink
	}

	return m, tea.Batch(tiCmd, vpCmd, spCmd)
}

func (m Model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	var sb strings.Builder
	sb.WriteString(m.viewport.View())
	sb.WriteString("\n")

	if m.loading {
		sb.WriteString(fmt.Sprintf("\n%s Generating response...\n", m.spinner.View()))
	} else {
		sb.WriteString("\n")
	}

	sb.WriteString(m.textarea.View())

	return sb.String()
}

func (m Model) renderMessages() string {
	if m.msgs == nil || len(*m.msgs) == 0 {
		return "Welcome to Pumbaa Chat! Ask me about your Cromwell workflows.\n\nPress Ctrl+D to send a message, Esc to quit.\n"
	}

	var s strings.Builder
	for _, msg := range *m.msgs {
		role := strings.ToUpper(msg.Role)
		style := lipgloss.NewStyle().Bold(true)
		if msg.Role == "user" {
			style = style.Foreground(lipgloss.Color("39"))
		} else if msg.Role == "agent" {
			style = style.Foreground(lipgloss.Color("205"))
		} else {
			style = style.Foreground(lipgloss.Color("196"))
		}

		s.WriteString(style.Render(role) + ": " + msg.Content + "\n\n")
	}
	return s.String()
}

func (m Model) generateResponse(input string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Create user content
		userContent := &genai.Content{
			Role: "user",
			Parts: []*genai.Part{
				genai.NewPartFromText(input),
			},
		}

		// Update history
		*m.history = append(*m.history, userContent)

		// Save user event to session
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

			// Add model response to history
			*m.history = append(*m.history, lastResp.Content)

			// Save model event to session
			if m.sessionService != nil && m.session != nil {
				ev := session.NewEvent("")
				ev.Content = lastResp.Content
				ev.Author = "model"
				m.sessionService.AppendEvent(ctx, m.session, ev)
			}

			// Check for tool calls
			toolCalls := getToolCalls(lastResp.Content)
			if len(toolCalls) > 0 {
				var toolParts []*genai.Part

				for _, tc := range toolCalls {
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

				// Save tool response event
				if m.sessionService != nil && m.session != nil {
					ev := session.NewEvent("")
					ev.Content = toolContent
					ev.Author = "tool"
					m.sessionService.AppendEvent(ctx, m.session, ev)
				}

				currentTurn++
				continue
			}

			// Final text response
			text := ""
			for _, part := range lastResp.Content.Parts {
				if part.Text != "" {
					text += part.Text
				}
			}
			return ResponseMsg{Content: text}
		}

		// Max turns reached - provide a helpful message
		// Summarize what was gathered from the conversation
		var summary strings.Builder
		summary.WriteString("⚠️ Atingi o limite de iterações de ferramentas.\n\n")
		summary.WriteString("**Informações coletadas até agora:**\n")

		// Look through history for tool responses
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
