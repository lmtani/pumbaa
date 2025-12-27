package chat

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// Update handles messages and updates the model.
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
