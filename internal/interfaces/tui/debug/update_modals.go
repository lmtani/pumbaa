package debug

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type viewportNavigator interface {
	ScrollUp(int) []string
	ScrollDown(int) []string
	PageUp() []string
	PageDown() []string
	GotoTop() []string
	GotoBottom() []string
}

type viewportModalActions struct {
	onClose func(*Model)
	onCopy  func(*Model) tea.Cmd
	onHome  func(*Model)
	onEnd   func(*Model)
	onLeft  func(*Model)
	onRight func(*Model)
}

func (m *Model) handleViewportModalKeys(msg tea.KeyMsg, navigator viewportNavigator, actions viewportModalActions) (tea.Cmd, bool) {
	switch {
	case key.Matches(msg, m.keys.Escape), key.Matches(msg, m.keys.Quit):
		if actions.onClose != nil {
			actions.onClose(m)
		}
		return nil, true
	case key.Matches(msg, m.keys.Copy):
		if actions.onCopy != nil {
			return actions.onCopy(m), true
		}
		return nil, true
	case key.Matches(msg, m.keys.Up):
		navigator.ScrollUp(1)
		return nil, true
	case key.Matches(msg, m.keys.Down):
		navigator.ScrollDown(1)
		return nil, true
	case key.Matches(msg, m.keys.PageUp):
		navigator.PageUp()
		return nil, true
	case key.Matches(msg, m.keys.PageDown):
		navigator.PageDown()
		return nil, true
	case key.Matches(msg, m.keys.Home):
		if actions.onHome != nil {
			actions.onHome(m)
		} else {
			navigator.GotoTop()
		}
		return nil, true
	case key.Matches(msg, m.keys.End):
		if actions.onEnd != nil {
			actions.onEnd(m)
		} else {
			navigator.GotoBottom()
		}
		return nil, true
	case key.Matches(msg, m.keys.Left):
		if actions.onLeft != nil {
			actions.onLeft(m)
			return nil, true
		}
	case key.Matches(msg, m.keys.Right):
		if actions.onRight != nil {
			actions.onRight(m)
			return nil, true
		}
	}
	return nil, false
}

func (m Model) handleLogModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	viewportWidth := m.logModalViewport.Width
	actions := viewportModalActions{
		onClose: func(m *Model) {
			m.showLogModal = false
			m.logModalContent = ""
			m.logModalRawContent = ""
			m.logModalError = ""
			m.logModalHScrollOffset = 0
		},
		onCopy: func(m *Model) tea.Cmd {
			if m.logModalRawContent != "" {
				return copyToClipboard(m.logModalRawContent, m.logModalTitle+" content")
			}
			return nil
		},
		onHome: func(m *Model) {
			m.logModalViewport.GotoTop()
			m.logModalHScrollOffset = 0
			truncatedContent := truncateLinesToWidth(m.logModalContent, viewportWidth)
			m.logModalViewport.SetContent(truncatedContent)
		},
		onLeft: func(m *Model) {
			if m.logModalHScrollOffset > 0 {
				m.logModalHScrollOffset -= 10
				if m.logModalHScrollOffset < 0 {
					m.logModalHScrollOffset = 0
				}
				scrolledContent := applyHorizontalScroll(m.logModalContent, m.logModalHScrollOffset, viewportWidth)
				truncatedContent := truncateLinesToWidth(scrolledContent, viewportWidth)
				m.logModalViewport.SetContent(truncatedContent)
			}
		},
		onRight: func(m *Model) {
			m.logModalHScrollOffset += 10
			scrolledContent := applyHorizontalScroll(m.logModalContent, m.logModalHScrollOffset, viewportWidth)
			truncatedContent := truncateLinesToWidth(scrolledContent, viewportWidth)
			m.logModalViewport.SetContent(truncatedContent)
		},
	}
	cmd, handled := m.handleViewportModalKeys(msg, &m.logModalViewport, actions)
	if handled {
		return m, cmd
	}
	return m, nil
}

func (m Model) handleInputsModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	actions := viewportModalActions{
		onClose: func(m *Model) {
			m.showInputsModal = false
		},
		onCopy: func(m *Model) tea.Cmd {
			return copyToClipboard(m.getRawInputsJSON(), "workflow inputs")
		},
	}
	cmd, handled := m.handleViewportModalKeys(msg, &m.inputsModalViewport, actions)
	if handled {
		return m, cmd
	}
	return m, nil
}

func (m Model) handleOutputsModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	actions := viewportModalActions{
		onClose: func(m *Model) {
			m.showOutputsModal = false
		},
		onCopy: func(m *Model) tea.Cmd {
			return copyToClipboard(m.getRawOutputsJSON(), "workflow outputs")
		},
	}
	cmd, handled := m.handleViewportModalKeys(msg, &m.outputsModalViewport, actions)
	if handled {
		return m, cmd
	}
	return m, nil
}

func (m Model) handleOptionsModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	actions := viewportModalActions{
		onClose: func(m *Model) {
			m.showOptionsModal = false
		},
		onCopy: func(m *Model) tea.Cmd {
			return copyToClipboard(m.getRawOptionsJSON(), "workflow options")
		},
	}
	cmd, handled := m.handleViewportModalKeys(msg, &m.optionsModalViewport, actions)
	if handled {
		return m, cmd
	}
	return m, nil
}

func (m Model) handleCallInputsModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	actions := viewportModalActions{
		onClose: func(m *Model) {
			m.showCallInputsModal = false
		},
		onCopy: func(m *Model) tea.Cmd {
			if m.cursor < len(m.nodes) {
				node := m.nodes[m.cursor]
				if node.CallData != nil {
					return copyToClipboard(m.getRawCallInputsJSON(node), "task inputs")
				}
			}
			return nil
		},
	}
	cmd, handled := m.handleViewportModalKeys(msg, &m.callInputsViewport, actions)
	if handled {
		return m, cmd
	}
	return m, nil
}

func (m Model) handleCallOutputsModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	actions := viewportModalActions{
		onClose: func(m *Model) {
			m.showCallOutputsModal = false
		},
		onCopy: func(m *Model) tea.Cmd {
			if m.cursor < len(m.nodes) {
				node := m.nodes[m.cursor]
				if node.CallData != nil {
					return copyToClipboard(m.getRawCallOutputsJSON(node), "task outputs")
				}
			}
			return nil
		},
	}
	cmd, handled := m.handleViewportModalKeys(msg, &m.callOutputsViewport, actions)
	if handled {
		return m, cmd
	}
	return m, nil
}

func (m Model) handleCallCommandModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	actions := viewportModalActions{
		onClose: func(m *Model) {
			m.showCallCommandModal = false
		},
		onCopy: func(m *Model) tea.Cmd {
			if m.cursor < len(m.nodes) {
				node := m.nodes[m.cursor]
				if node.CallData != nil && node.CallData.CommandLine != "" {
					return copyToClipboard(node.CallData.CommandLine, "command")
				}
			}
			return nil
		},
	}
	cmd, handled := m.handleViewportModalKeys(msg, &m.callCommandViewport, actions)
	if handled {
		return m, cmd
	}
	return m, nil
}

func (m Model) handleGlobalTimelineModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	actions := viewportModalActions{
		onClose: func(m *Model) {
			m.showGlobalTimelineModal = false
		},
	}
	cmd, handled := m.handleViewportModalKeys(msg, &m.globalTimelineViewport, actions)
	if handled {
		return m, cmd
	}
	return m, nil
}
