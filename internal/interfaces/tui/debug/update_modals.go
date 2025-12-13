package debug

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleLogModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape), key.Matches(msg, m.keys.Quit):
		m.showLogModal = false
		m.logModalContent = ""
		m.logModalError = ""
	case key.Matches(msg, m.keys.Copy):
		if m.logModalContent != "" {
			return m, copyToClipboard(m.logModalContent)
		}
	case key.Matches(msg, m.keys.Up):
		m.logModalViewport.ScrollUp(1)
	case key.Matches(msg, m.keys.Down):
		m.logModalViewport.ScrollDown(1)
	case key.Matches(msg, m.keys.PageUp):
		m.logModalViewport.PageUp()
	case key.Matches(msg, m.keys.PageDown):
		m.logModalViewport.PageDown()
	case key.Matches(msg, m.keys.Home):
		m.logModalViewport.GotoTop()
	case key.Matches(msg, m.keys.End):
		m.logModalViewport.GotoBottom()
	}
	return m, nil
}

func (m Model) handleInputsModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape), key.Matches(msg, m.keys.Quit):
		m.showInputsModal = false
	case key.Matches(msg, m.keys.Copy):
		return m, copyToClipboard(m.getRawInputsJSON())
	case key.Matches(msg, m.keys.Up):
		m.inputsModalViewport.ScrollUp(1)
	case key.Matches(msg, m.keys.Down):
		m.inputsModalViewport.ScrollDown(1)
	case key.Matches(msg, m.keys.PageUp):
		m.inputsModalViewport.PageUp()
	case key.Matches(msg, m.keys.PageDown):
		m.inputsModalViewport.PageDown()
	case key.Matches(msg, m.keys.Home):
		m.inputsModalViewport.GotoTop()
	case key.Matches(msg, m.keys.End):
		m.inputsModalViewport.GotoBottom()
	}
	return m, nil
}

func (m Model) handleOutputsModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape), key.Matches(msg, m.keys.Quit):
		m.showOutputsModal = false
	case key.Matches(msg, m.keys.Copy):
		return m, copyToClipboard(m.getRawOutputsJSON())
	case key.Matches(msg, m.keys.Up):
		m.outputsModalViewport.ScrollUp(1)
	case key.Matches(msg, m.keys.Down):
		m.outputsModalViewport.ScrollDown(1)
	case key.Matches(msg, m.keys.PageUp):
		m.outputsModalViewport.PageUp()
	case key.Matches(msg, m.keys.PageDown):
		m.outputsModalViewport.PageDown()
	case key.Matches(msg, m.keys.Home):
		m.outputsModalViewport.GotoTop()
	case key.Matches(msg, m.keys.End):
		m.outputsModalViewport.GotoBottom()
	}
	return m, nil
}

func (m Model) handleOptionsModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape), key.Matches(msg, m.keys.Quit):
		m.showOptionsModal = false
	case key.Matches(msg, m.keys.Copy):
		return m, copyToClipboard(m.getRawOptionsJSON())
	case key.Matches(msg, m.keys.Up):
		m.optionsModalViewport.ScrollUp(1)
	case key.Matches(msg, m.keys.Down):
		m.optionsModalViewport.ScrollDown(1)
	case key.Matches(msg, m.keys.PageUp):
		m.optionsModalViewport.PageUp()
	case key.Matches(msg, m.keys.PageDown):
		m.optionsModalViewport.PageDown()
	case key.Matches(msg, m.keys.Home):
		m.optionsModalViewport.GotoTop()
	case key.Matches(msg, m.keys.End):
		m.optionsModalViewport.GotoBottom()
	}
	return m, nil
}

func (m Model) handleCallInputsModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape), key.Matches(msg, m.keys.Quit):
		m.showCallInputsModal = false
	case key.Matches(msg, m.keys.Copy):
		if m.cursor < len(m.nodes) {
			node := m.nodes[m.cursor]
			if node.CallData != nil {
				return m, copyToClipboard(m.getRawCallInputsJSON(node))
			}
		}
	case key.Matches(msg, m.keys.Up):
		m.callInputsViewport.ScrollUp(1)
	case key.Matches(msg, m.keys.Down):
		m.callInputsViewport.ScrollDown(1)
	case key.Matches(msg, m.keys.PageUp):
		m.callInputsViewport.PageUp()
	case key.Matches(msg, m.keys.PageDown):
		m.callInputsViewport.PageDown()
	case key.Matches(msg, m.keys.Home):
		m.callInputsViewport.GotoTop()
	case key.Matches(msg, m.keys.End):
		m.callInputsViewport.GotoBottom()
	}
	return m, nil
}

func (m Model) handleCallOutputsModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape), key.Matches(msg, m.keys.Quit):
		m.showCallOutputsModal = false
	case key.Matches(msg, m.keys.Copy):
		if m.cursor < len(m.nodes) {
			node := m.nodes[m.cursor]
			if node.CallData != nil {
				return m, copyToClipboard(m.getRawCallOutputsJSON(node))
			}
		}
	case key.Matches(msg, m.keys.Up):
		m.callOutputsViewport.ScrollUp(1)
	case key.Matches(msg, m.keys.Down):
		m.callOutputsViewport.ScrollDown(1)
	case key.Matches(msg, m.keys.PageUp):
		m.callOutputsViewport.PageUp()
	case key.Matches(msg, m.keys.PageDown):
		m.callOutputsViewport.PageDown()
	case key.Matches(msg, m.keys.Home):
		m.callOutputsViewport.GotoTop()
	case key.Matches(msg, m.keys.End):
		m.callOutputsViewport.GotoBottom()
	}
	return m, nil
}

func (m Model) handleCallCommandModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape), key.Matches(msg, m.keys.Quit):
		m.showCallCommandModal = false
	case key.Matches(msg, m.keys.Copy):
		if m.cursor < len(m.nodes) {
			node := m.nodes[m.cursor]
			if node.CallData != nil && node.CallData.CommandLine != "" {
				return m, copyToClipboard(node.CallData.CommandLine)
			}
		}
	case key.Matches(msg, m.keys.Up):
		m.callCommandViewport.ScrollUp(1)
	case key.Matches(msg, m.keys.Down):
		m.callCommandViewport.ScrollDown(1)
	case key.Matches(msg, m.keys.PageUp):
		m.callCommandViewport.PageUp()
	case key.Matches(msg, m.keys.PageDown):
		m.callCommandViewport.PageDown()
	case key.Matches(msg, m.keys.Home):
		m.callCommandViewport.GotoTop()
	case key.Matches(msg, m.keys.End):
		m.callCommandViewport.GotoBottom()
	}
	return m, nil
}

func (m Model) handleGlobalTimelineModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape), key.Matches(msg, m.keys.Quit):
		m.showGlobalTimelineModal = false
	case key.Matches(msg, m.keys.Up):
		m.globalTimelineViewport.ScrollUp(1)
	case key.Matches(msg, m.keys.Down):
		m.globalTimelineViewport.ScrollDown(1)
	case key.Matches(msg, m.keys.PageUp):
		m.globalTimelineViewport.PageUp()
	case key.Matches(msg, m.keys.PageDown):
		m.globalTimelineViewport.PageDown()
	case key.Matches(msg, m.keys.Home):
		m.globalTimelineViewport.GotoTop()
	case key.Matches(msg, m.keys.End):
		m.globalTimelineViewport.GotoBottom()
	}
	return m, nil
}
