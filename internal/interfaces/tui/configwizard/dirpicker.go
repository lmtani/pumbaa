package configwizard

import (
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/filepicker"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// DirectoryPickerModel is a Bubble Tea model for picking directories.
type DirectoryPickerModel struct {
	filepicker   filepicker.Model
	selectedPath string
	quitting     bool
	err          error
}

// NewDirectoryPicker creates a new directory picker starting at the given path.
func NewDirectoryPicker(startPath string) DirectoryPickerModel {
	fp := filepicker.New()
	fp.DirAllowed = true
	fp.FileAllowed = false
	fp.ShowHidden = false
	fp.Height = 15

	// Start at home directory if no path specified
	if startPath == "" {
		startPath, _ = os.UserHomeDir()
	}
	fp.CurrentDirectory = startPath

	// Style the filepicker
	fp.Styles.Cursor = lipgloss.NewStyle().Foreground(common.PrimaryColor)
	fp.Styles.Selected = lipgloss.NewStyle().Foreground(common.PrimaryColor).Bold(true)

	return DirectoryPickerModel{
		filepicker: fp,
	}
}

func (m DirectoryPickerModel) Init() tea.Cmd {
	return m.filepicker.Init()
}

func (m DirectoryPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			// Select current directory
			m.selectedPath = m.filepicker.CurrentDirectory
			m.quitting = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.filepicker, cmd = m.filepicker.Update(msg)

	// Check if a directory was selected via filepicker
	if didSelect, path := m.filepicker.DidSelectFile(msg); didSelect {
		m.selectedPath = path
		m.quitting = true
		return m, tea.Quit
	}

	if didSelect, path := m.filepicker.DidSelectDisabledFile(msg); didSelect {
		// This shouldn't happen since we only allow directories
		m.selectedPath = path
		m.quitting = true
		return m, tea.Quit
	}

	return m, cmd
}

func (m DirectoryPickerModel) View() string {
	if m.quitting {
		return ""
	}

	var s strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(common.PrimaryColor).
		MarginBottom(1)

	pathStyle := lipgloss.NewStyle().
		Foreground(common.MutedColor).
		Italic(true)

	helpStyle := lipgloss.NewStyle().
		Foreground(common.MutedColor)

	s.WriteString(titleStyle.Render("📁 Select WDL Directory"))
	s.WriteString("\n")
	s.WriteString(pathStyle.Render("Current: " + m.filepicker.CurrentDirectory))
	s.WriteString("\n\n")
	s.WriteString(m.filepicker.View())
	s.WriteString("\n\n")
	s.WriteString(helpStyle.Render("↑/↓: navigate items • →/l: enter folder • ←/h: go back"))
	s.WriteString("\n")
	s.WriteString(helpStyle.Render("enter: select this folder • q: cancel"))

	return s.String()
}

// SelectedPath returns the selected directory path.
func (m DirectoryPickerModel) SelectedPath() string {
	return m.selectedPath
}

// RunDirectoryPicker runs the directory picker and returns the selected path.
func RunDirectoryPicker(startPath string) (string, error) {
	m := NewDirectoryPicker(startPath)
	p := tea.NewProgram(m) // Pass value, not pointer

	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	// Type assert as value, not pointer
	result := finalModel.(DirectoryPickerModel)
	return result.SelectedPath(), nil
}
