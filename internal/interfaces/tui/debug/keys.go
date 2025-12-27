package debug

import (
	"github.com/charmbracelet/bubbles/key"

	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// KeyMap defines all key bindings for the debug TUI.
// It embeds common key bindings for consistency across screens.
type KeyMap struct {
	common.NavigationKeys // Embedded navigation (Up, Down, Left, Right, Enter, Space, Tab, etc.)
	common.GlobalKeys     // Embedded globals (Quit, Help)

	// Debug-specific keys
	Details     key.Binding
	ExpandAll   key.Binding
	CollapseAll key.Binding
	Copy        key.Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		NavigationKeys: common.DefaultNavigationKeys(),
		GlobalKeys:     common.DefaultGlobalKeys(),

		Details: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "view details"),
		),
		ExpandAll: key.NewBinding(
			key.WithKeys("E"),
			key.WithHelp("E", "expand all"),
		),
		CollapseAll: key.NewBinding(
			key.WithKeys("C"),
			key.WithHelp("C", "collapse all"),
		),
		Copy: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "copy to clipboard"),
		),
	}
}

// ShortHelp returns keybindings to be shown in the mini help view.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Tab, k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded help view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.Enter, k.Space, k.Tab, k.Escape},
		{k.Details, k.ExpandAll, k.CollapseAll},
		{k.Home, k.End, k.PageUp, k.PageDown},
		{k.Copy, k.Help, k.Quit},
	}
}
