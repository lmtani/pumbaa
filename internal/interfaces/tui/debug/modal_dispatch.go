package debug

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type modalDispatch struct {
	active func(Model) bool
	view   func(Model) string
	handle func(Model, tea.KeyMsg) (tea.Model, tea.Cmd)
}

func (m Model) modalDispatches() []modalDispatch {
	return []modalDispatch{
		{
			active: func(m Model) bool { return m.showChatSelectionModal },
			view:   Model.renderChatSelectionModal,
			handle: Model.handleChatSelectionModalKeys,
		},
		{
			active: func(m Model) bool { return m.showHelp },
			view:   Model.renderHelpOverlay,
			handle: Model.handleHelpKeys,
		},
		{
			active: func(m Model) bool { return m.showLogModal },
			view:   Model.renderLogModal,
			handle: Model.handleLogModalKeys,
		},
		{
			active: func(m Model) bool { return m.showInputsModal },
			view:   Model.renderInputsModal,
			handle: Model.handleInputsModalKeys,
		},
		{
			active: func(m Model) bool { return m.showOutputsModal },
			view:   Model.renderOutputsModal,
			handle: Model.handleOutputsModalKeys,
		},
		{
			active: func(m Model) bool { return m.showOptionsModal },
			view:   Model.renderOptionsModal,
			handle: Model.handleOptionsModalKeys,
		},
		{
			active: func(m Model) bool { return m.showGlobalTimelineModal },
			view:   Model.renderGlobalTimelineModal,
			handle: Model.handleGlobalTimelineModalKeys,
		},
		{
			active: func(m Model) bool { return m.showCallInputsModal },
			view:   Model.renderCallInputsModal,
			handle: Model.handleCallInputsModalKeys,
		},
		{
			active: func(m Model) bool { return m.showCallOutputsModal },
			view:   Model.renderCallOutputsModal,
			handle: Model.handleCallOutputsModalKeys,
		},
		{
			active: func(m Model) bool { return m.showCallCommandModal },
			view:   Model.renderCallCommandModal,
			handle: Model.handleCallCommandModalKeys,
		},
		{
			active: func(m Model) bool { return m.showBatchLogsModal },
			view:   Model.renderBatchLogsModal,
			handle: Model.handleBatchLogsModalKeys,
		},
	}
}

func (m Model) renderActiveModal() (string, bool) {
	for _, modal := range m.modalDispatches() {
		if modal.active(m) {
			return modal.view(m), true
		}
	}
	return "", false
}

func (m Model) handleActiveModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	for _, modal := range m.modalDispatches() {
		if modal.active(m) {
			model, cmd := modal.handle(m, msg)
			return model, cmd, true
		}
	}
	return m, nil, false
}

func (m Model) handleHelpKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.Help) || key.Matches(msg, m.keys.Escape) || key.Matches(msg, m.keys.Quit) {
		m.showHelp = false
	}
	return m, nil
}
