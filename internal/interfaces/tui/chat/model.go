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
	"google.golang.org/adk/tool"
	"google.golang.org/genai"
)

// Interface to access the hidden definition method of functiontool
type toolWithDefinition interface {
	Declaration() *genai.FunctionDeclaration
	Run(ctx tool.Context, args interface{}) (map[string]interface{}, error)
}

// Model is the Bubble Tea model for the chat interface.
// Uses pointers for mutable state that needs to persist across Updates.
type Model struct {
	llm               model.LLM
	tools             []tool.Tool
	systemInstruction string

	// History is a pointer so it persists across value copies
	history *[]*genai.Content

	// Bubble Tea components
	viewport viewport.Model
	textarea textarea.Model
	spinner  spinner.Model

	// State
	loading bool
	ready   bool // Track if we've received initial window size

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

// NewModel creates a new chat model with the given LLM, tools, and system instruction.
func NewModel(llm model.LLM, tools []tool.Tool, systemInstruction string) Model {
	ta := textarea.New()
	ta.Placeholder = "Type your message... (Ctrl+D to send, Esc to quit)"
	ta.Focus()
	ta.Prompt = "┃ "
	ta.CharLimit = 2000
	ta.SetWidth(80)
	ta.SetHeight(3)
	ta.ShowLineNumbers = false

	vp := viewport.New(80, 20)
	vp.SetContent("Welcome to Pumbaa Chat! Ask me about your Cromwell workflows.\n\nPress Ctrl+D to send a message, Esc to quit.\n")

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	// Initialize slices via pointers so they persist
	history := make([]*genai.Content, 0)
	msgs := make([]ChatMessage, 0)

	return Model{
		llm:               llm,
		tools:             tools,
		systemInstruction: systemInstruction,
		textarea:          ta,
		viewport:          vp,
		spinner:           s,
		history:           &history,
		msgs:              &msgs,
		ready:             false,
	}
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
		footerHeight := 6 // textarea height + margins
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
		case tea.KeyCtrlD: // Use Ctrl+D to send message
			if m.loading {
				return m, nil
			}
			input := m.textarea.Value()
			if strings.TrimSpace(input) == "" {
				return m, nil
			}

			// Update UI state (using pointers, so this persists)
			*m.msgs = append(*m.msgs, ChatMessage{Role: "user", Content: input})
			m.viewport.SetContent(m.renderMessages())
			m.textarea.Reset()
			m.viewport.GotoBottom()
			m.loading = true

			// Command to call agent
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

	// Chat viewport
	sb.WriteString(m.viewport.View())
	sb.WriteString("\n")

	// Status line
	if m.loading {
		sb.WriteString(fmt.Sprintf("\n%s Generating response...\n", m.spinner.View()))
	} else {
		sb.WriteString("\n")
	}

	// Input area
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
			style = style.Foreground(lipgloss.Color("39")) // Blue
		} else if msg.Role == "agent" {
			style = style.Foreground(lipgloss.Color("205")) // Pink
		} else {
			style = style.Foreground(lipgloss.Color("196")) // Red (System/Error)
		}

		s.WriteString(style.Render(role) + ": " + msg.Content + "\n\n")
	}
	return s.String()
}

func (m Model) generateResponse(input string) tea.Cmd {
	return func() tea.Msg {
		// Update history with user input (using pointer)
		*m.history = append(*m.history, &genai.Content{
			Role: "user",
			Parts: []*genai.Part{
				genai.NewPartFromText(input),
			},
		})

		ctx := context.Background()

		maxTurns := 5
		currentTurn := 0

		for currentTurn < maxTurns {
			// Prepare request with system instruction
			req := &model.LLMRequest{
				Contents: *m.history,
				Config: &genai.GenerateContentConfig{
					Tools: convertToolsToGenAI(m.tools),
				},
			}

			// Add system instruction if provided
			if m.systemInstruction != "" {
				req.Config.SystemInstruction = &genai.Content{
					Parts: []*genai.Part{
						genai.NewPartFromText(m.systemInstruction),
					},
				}
			}

			// Generate content
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

			// Check for tool calls
			toolCalls := getToolCalls(lastResp.Content)
			if len(toolCalls) > 0 {
				// Execute tools
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

				// Add tool responses to history
				*m.history = append(*m.history, &genai.Content{
					Role:  "tool",
					Parts: toolParts,
				})

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

		return ResponseMsg{Content: "Max turns reached"}
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
