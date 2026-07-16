package chat

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/lmtani/pumbaa/internal/application/ports"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// Update handles every tea.Msg for the chat screen: key handling for both
// focus modes, streaming/tool messages pushed from the generation goroutine,
// and session lifecycle results.
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
				if store, ok := m.sessionService.(ports.ChatSessionStore); ok {
					sessionID := m.session.ID()
					inputTokens, outputTokens := m.inputTokens, m.outputTokens
					go func() { _ = store.UpdateTokenUsage(context.Background(), sessionID, inputTokens, outputTokens) }()
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
