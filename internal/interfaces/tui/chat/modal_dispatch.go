package chat

import (
	tea "github.com/charmbracelet/bubbletea"
)

type modalDispatch struct {
	active func(Model) bool
	view   func(Model) string
	handle func(Model, tea.KeyMsg) (tea.Model, tea.Cmd)
}

func (m *Model) modalDispatches() []modalDispatch {
	return []modalDispatch{
		{
			active: func(m Model) bool { return m.activeModal == ModalSessions },
			view: func(m Model) string {
				return (&m).renderSessionsModal()
			},
			handle: func(m Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
				return (&m).handleSessionsModalKeys(msg)
			},
		},
	}
}

func (m *Model) renderActiveModal() (string, bool) {
	for _, modal := range m.modalDispatches() {
		if modal.active(*m) {
			return modal.view(*m), true
		}
	}
	return "", false
}

func (m *Model) handleActiveModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	for _, modal := range m.modalDispatches() {
		if modal.active(*m) {
			model, cmd := modal.handle(*m, msg)
			return model, cmd, true
		}
	}
	return m, nil, false
}
