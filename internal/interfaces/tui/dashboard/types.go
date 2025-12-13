package dashboard

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// KeyMap defines the key bindings specific to the dashboard.
type KeyMap struct {
	common.NavigationKeys
	Refresh      key.Binding
	Open         key.Binding
	Abort        key.Binding
	Filter       key.Binding
	LabelFilter  key.Binding
	ClearFilter  key.Binding
	StatusFilter key.Binding
}

// DefaultKeyMap returns the default key bindings for the dashboard.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		NavigationKeys: common.DefaultNavigationKeys(),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Open: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "open/debug"),
		),
		Abort: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "abort workflow"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter by name"),
		),
		LabelFilter: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "filter by label"),
		),
		ClearFilter: key.NewBinding(
			key.WithKeys("ctrl+x"),
			key.WithHelp("ctrl+x", "clear filters"),
		),
		StatusFilter: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "toggle status filter"),
		),
	}
}

// FilterState holds the current filter configuration
type FilterState struct {
	Status []workflow.Status
	Name   string
	Label  string // Format: key:value
}
