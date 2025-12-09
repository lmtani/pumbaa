package debug

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all key bindings for the TUI.
type KeyMap struct {
	Up             key.Binding
	Down           key.Binding
	Left           key.Binding
	Right          key.Binding
	Enter          key.Binding
	Space          key.Binding
	Tab            key.Binding
	Quit           key.Binding
	Help           key.Binding
	Escape         key.Binding
	Details        key.Binding
	Command        key.Binding
	Logs           key.Binding
	Inputs         key.Binding
	Outputs        key.Binding
	Options        key.Binding
	GlobalTimeline key.Binding
	CopyStdout     key.Binding
	CopyStderr     key.Binding
	ExpandAll      key.Binding
	CollapseAll    key.Binding
	Home           key.Binding
	End            key.Binding
	PageUp         key.Binding
	PageDown       key.Binding
	OpenLog        key.Binding
	Copy           key.Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "collapse"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "expand"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "toggle/select"),
		),
		Space: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "toggle expand"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "switch panel"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Details: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "view details"),
		),
		Command: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "view command"),
		),
		Logs: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "view logs"),
		),
		Inputs: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "view inputs"),
		),
		Outputs: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "view outputs"),
		),
		Options: key.NewBinding(
			key.WithKeys("O"),
			key.WithHelp("O", "view options"),
		),
		GlobalTimeline: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "tasks duration"),
		),
		CopyStdout: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "copy stdout path"),
		),
		CopyStderr: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "copy stderr path"),
		),
		ExpandAll: key.NewBinding(
			key.WithKeys("E"),
			key.WithHelp("E", "expand all"),
		),
		CollapseAll: key.NewBinding(
			key.WithKeys("C"),
			key.WithHelp("C", "collapse all"),
		),
		Home: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("g/home", "go to top"),
		),
		End: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("G/end", "go to bottom"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+u"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+d"),
			key.WithHelp("pgdn", "page down"),
		),
		OpenLog: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "open log"),
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
		{k.Details, k.Command, k.Logs, k.Inputs},
		{k.Outputs, k.GlobalTimeline, k.Options},
		{k.ExpandAll, k.CollapseAll, k.Home, k.End},
		{k.PageUp, k.PageDown, k.CopyStdout, k.CopyStderr},
		{k.Help, k.Quit},
	}
}
