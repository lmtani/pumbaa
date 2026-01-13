package debug

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type modalDispatch struct {
	active func(Model) bool
	view   func(Model) string
	handle func(Model, tea.KeyMsg) (tea.Model, tea.Cmd)
	resize func(*Model)
}

func (m Model) modalDispatches() []modalDispatch {
	return []modalDispatch{
		{
			active: func(m Model) bool { return m.activeModal == ModalChatSelection },
			view:   Model.renderChatSelectionModal,
			handle: Model.handleChatSelectionModalKeys,
		},
		{
			active: func(m Model) bool { return m.activeModal == ModalHelp },
			view:   Model.renderHelpOverlay,
			handle: Model.handleHelpKeys,
		},
		{
			active: func(m Model) bool { return m.activeModal == ModalLog },
			view:   Model.renderLogModal,
			handle: Model.handleLogModalKeys,
			resize: func(m *Model) { m.resizeLogModalViewport() },
		},
		{
			active: func(m Model) bool { return m.activeModal == ModalInputs },
			view:   Model.renderInputsModal,
			handle: Model.handleInputsModalKeys,
			resize: func(m *Model) { m.resizeStandardModalViewport(&m.inputsModalViewport) },
		},
		{
			active: func(m Model) bool { return m.activeModal == ModalOutputs },
			view:   Model.renderOutputsModal,
			handle: Model.handleOutputsModalKeys,
			resize: func(m *Model) { m.resizeStandardModalViewport(&m.outputsModalViewport) },
		},
		{
			active: func(m Model) bool { return m.activeModal == ModalOptions },
			view:   Model.renderOptionsModal,
			handle: Model.handleOptionsModalKeys,
			resize: func(m *Model) { m.resizeStandardModalViewport(&m.optionsModalViewport) },
		},
		{
			active: func(m Model) bool { return m.activeModal == ModalGlobalTimeline },
			view:   Model.renderGlobalTimelineModal,
			handle: Model.handleGlobalTimelineModalKeys,
			resize: func(m *Model) { m.resizeStandardModalViewport(&m.globalTimelineViewport) },
		},
		{
			active: func(m Model) bool { return m.activeModal == ModalCallInputs },
			view:   Model.renderCallInputsModal,
			handle: Model.handleCallInputsModalKeys,
			resize: func(m *Model) { m.resizeStandardModalViewport(&m.callInputsViewport) },
		},
		{
			active: func(m Model) bool { return m.activeModal == ModalCallOutputs },
			view:   Model.renderCallOutputsModal,
			handle: Model.handleCallOutputsModalKeys,
			resize: func(m *Model) { m.resizeStandardModalViewport(&m.callOutputsViewport) },
		},
		{
			active: func(m Model) bool { return m.activeModal == ModalCallCommand },
			view:   Model.renderCallCommandModal,
			handle: Model.handleCallCommandModalKeys,
			resize: func(m *Model) { m.resizeStandardModalViewport(&m.callCommandViewport) },
		},
		{
			active: func(m Model) bool { return m.activeModal == ModalBatchLogs },
			view:   Model.renderBatchLogsModal,
			handle: Model.handleBatchLogsModalKeys,
			resize: func(m *Model) { m.resizeBatchLogsModalViewport() },
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

func (m *Model) resizeActiveModal() {
	for _, modal := range m.modalDispatches() {
		if modal.active(*m) && modal.resize != nil {
			modal.resize(m)
			return
		}
	}
}

func (m Model) handleHelpKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.Help) || key.Matches(msg, m.keys.Escape) {
		m.activeModal = ModalNone
	}
	return m, nil
}
