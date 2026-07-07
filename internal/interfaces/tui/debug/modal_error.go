package debug

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// openErrorModal shows the full text of the last error. Footer status
// messages truncate errors to keep the bar on one line; this modal is the
// place to read (and copy) the whole thing.
func (m Model) openErrorModal() (tea.Model, tea.Cmd) {
	if m.lastError == "" {
		m.setStatusMessage("No recent errors")
		return m, getClearStatusCmd()
	}
	m.activeModal = ModalError
	m.errorModalViewport = viewport.New(m.width-10, m.height-8)
	m.errorModalViewport.SetContent(wrapText(m.lastError, m.width-12))
	return m, nil
}

func (m Model) renderErrorModal() string {
	title := titleStyle.Render("⚠  Last Error")
	content := renderModalViewportContent(m.errorModalViewport.View(), m.errorModalViewport.Width, false, "")
	return m.renderStandardModal(title, content, m.modalFooter())
}

func (m Model) handleErrorModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	actions := viewportModalActions{
		onClose: func(m *Model) {
			m.activeModal = ModalNone
		},
		onCopy: func(m *Model) tea.Cmd {
			return copyToClipboard(m.lastError, "error details")
		},
	}
	cmd, handled := m.handleViewportModalKeys(msg, &m.errorModalViewport, actions)
	if handled {
		return m, cmd
	}
	return m, nil
}
