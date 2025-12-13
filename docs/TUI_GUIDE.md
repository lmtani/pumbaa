# Pumbaa TUI - Reference Guide

This document describes the architecture and structure of the Pumbaa Terminal User Interface (TUI), a CLI tool for managing Cromwell workflows.

## Overview

The TUI is built using the [Bubble Tea](https://github.com/charmbracelet/bubbletea) framework, following the **Model-View-Update (MVU)** pattern inspired by Elm. Styling is done with [Lipgloss](https://github.com/charmbracelet/lipgloss).

### Directory Structure

```
internal/interfaces/tui/
├── app.go                    # Main coordinator (screen navigation)
├── common/                   # Shared components
│   ├── colors.go             # Color palette
│   ├── keys.go               # Global key bindings
│   └── styles.go             # Reusable Lipgloss styles
├── dashboard/                # Workflow listing screen (12 files)
│   ├── model.go              # Main model, Init, Update
│   ├── messages.go           # tea.Msg message types
│   ├── types.go              # KeyMap, FilterState
│   ├── helpers.go            # Utility functions
│   ├── update_keys.go        # Keyboard handlers
│   ├── update_async.go       # Async operations
│   ├── update_scroll.go      # Navigation/scroll
│   ├── view.go               # Main view router
│   ├── view_header.go        # Header with badges
│   ├── view_content.go       # Content area
│   ├── view_table.go         # Workflows table
│   └── view_footer.go        # Help bar
└── debug/                    # Workflow debug screen (21 files)
    ├── model.go              # Main model
    ├── types.go              # Enums and type aliases
    ├── keys.go               # Specific key bindings
    ├── styles.go             # Specific styles
    ├── helpers.go            # Helper functions
    ├── view.go               # View router
    ├── view_header.go        # Header with metadata
    ├── view_tree.go          # Calls tree
    ├── view_details.go       # Details panel
    ├── view_footer.go        # Status bar
    ├── view_failures.go      # Failures list
    ├── update.go             # Main update
    ├── update_async.go       # Async operations
    ├── update_modals.go      # Modal handlers
    ├── update_navigation.go  # Tree navigation
    ├── modals.go             # Modal rendering
    ├── modal_logs.go         # Logs modal
    ├── modal_call.go         # Call details modal
    ├── modal_workflow.go     # Inputs/outputs modal
    ├── modal_timeline.go     # Global timeline modal
    └── parser_test.go        # Parser tests
```

## Architecture

### Model-View-Update (MVU) Pattern

Each screen follows the Bubble Tea pattern:

```go
type Model struct {
    // Application state
}

func (m Model) Init() tea.Cmd {
    // Initial command (data fetch, etc)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Update state based on messages
}

func (m Model) View() string {
    // Render the interface
}
```

### Screen Coordination (app.go)

The `AppModel` coordinates navigation between screens:

```go
type Screen int

const (
    ScreenDashboard Screen = iota
    ScreenDebug
)

type AppModel struct {
    currentScreen Screen
    dashboard     dashboard.Model
    debug         debug.Model
    // ...
}
```

### Message Flow

```
┌─────────────────────────────────────────────────────────┐
│                     tea.Program                          │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐           │
│  │  Event   │───▶│  Update  │───▶│   View   │           │
│  │ (KeyMsg) │    │ (Model)  │    │ (string) │           │
│  └──────────┘    └──────────┘    └──────────┘           │
│       │               │               │                  │
│       │               ▼               ▼                  │
│       │         ┌──────────┐    ┌──────────┐           │
│       │         │   Cmd    │    │ Terminal │           │
│       │         │ (async)  │    │  Output  │           │
│       └─────────┴──────────┘    └──────────┘           │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

## Shared Components (common/)

### Colors (colors.go)

```go
// Status colors
var (
    StatusSucceeded = lipgloss.Color("#00ff00")  // Green
    StatusFailed    = lipgloss.Color("#ff0000")  // Red
    StatusRunning   = lipgloss.Color("#ffff00")  // Yellow
    StatusPending   = lipgloss.Color("#888888")  // Gray
)

// UI colors
var (
    PrimaryColor   = lipgloss.Color("#7D56F4")  // Main purple
    BorderColor    = lipgloss.Color("#444444")  // Borders
    TextColor      = lipgloss.Color("#FAFAFA")  // Text
    MutedColor     = lipgloss.Color("#888888")  // Secondary text
    HighlightColor = lipgloss.Color("#874BFD")  // Selection
)
```

### Key Bindings (keys.go)

```go
// Standard navigation (vim-style + arrows)
type NavigationKeys struct {
    Up, Down, Left, Right key.Binding  // ↑↓←→ or hjkl
    Enter, Space, Tab, Escape key.Binding
    Home, End key.Binding              // g/G or Home/End
    PageUp, PageDown key.Binding       // PgUp/PgDn or Ctrl+U/D
}

// Global keys (work on any screen)
type GlobalKeys struct {
    Quit key.Binding  // q or Ctrl+C
    Help key.Binding  // ?
}
```

### Styles (styles.go)

Reusable styles for visual consistency:

| Style | Usage |
|-------|-------|
| `PanelStyle` | Panels with rounded border |
| `FocusedPanelStyle` | Focused panel (highlighted border) |
| `TitleStyle` | Bold titles |
| `MutedStyle` | Secondary/disabled text |
| `ErrorStyle` | Error messages |
| `SuccessStyle` | Success messages |
| `ModalStyle` | Modal windows |
| `HeaderStyle` | Screen headers |
| `HelpBarStyle` | Bottom help bar |
| `KeyStyle` | Shortcut keys |
| `DescStyle` | Shortcut descriptions |
| `BadgeStyle` | Status badges |

## Dashboard

### Features

- **Workflow listing** with status, ID, name, date and labels
- **Filters**: by name (`/`), label (`l`), status (`s`)
- **Actions**: refresh (`r`), abort (`a`), debug (`enter`)
- **Scroll**: navigation with arrows, Page Up/Down

### File Structure

| File | Responsibility | Lines |
|------|----------------|-------|
| `model.go` | Model, Init, main Update | ~196 |
| `messages.go` | Message types | ~36 |
| `types.go` | KeyMap, FilterState | ~61 |
| `helpers.go` | truncateID, formatDuration, etc | ~91 |
| `update_keys.go` | handleMainKeys, handleFilterKeys | ~175 |
| `update_async.go` | fetchWorkflows, abortWorkflow | ~87 |
| `update_scroll.go` | ensureVisible, getVisibleRows | ~17 |
| `view.go` | View() router | ~22 |
| `view_header.go` | renderHeader | ~94 |
| `view_content.go` | renderContent, modals | ~109 |
| `view_table.go` | renderTable, renderRow | ~142 |
| `view_footer.go` | renderFooter (help bar) | ~83 |

### Main Model

```go
type Model struct {
    // Data
    workflows    []workflow.Workflow
    totalCount   int
    
    // UI state
    cursor       int
    scrollY      int
    width, height int
    
    // Filters
    activeFilters FilterState
    showFilter    bool
    filterInput   textinput.Model
    filterType    string  // "name" or "label"
    
    // Modals
    showConfirm   bool
    confirmID     string
    
    // Loading
    loading       bool
    loadingDebug  bool
    loadingDebugID string
    
    // Fetchers (interfaces)
    fetcher         WorkflowFetcher
    metadataFetcher MetadataFetcher
}
```

### Messages

```go
type workflowsLoadedMsg struct {
    workflows  []workflow.Workflow
    totalCount int
}

type workflowsErrorMsg struct{ error }
type abortResultMsg struct{ success bool; message string }
type debugMetadataLoadedMsg struct{ metadata *debug.WorkflowMetadata }
type debugMetadataErrorMsg struct{ error }
type NavigateToDebugMsg struct{ Metadata *debug.WorkflowMetadata }
```

### Dashboard Key Bindings

| Key | Action |
|-----|--------|
| `↑/k` | Move up |
| `↓/j` | Move down |
| `enter` | Open debug view |
| `r` | Refresh |
| `a` | Abort workflow |
| `/` | Filter by name |
| `l` | Filter by label |
| `s` | Cycle status filter |
| `ctrl+x` | Clear filters |
| `q` | Quit |

## Debug View

### Features

- **Hierarchical tree** of calls and subworkflows
- **Execution details** (timing, resources, status)
- **Modals**: logs, inputs/outputs, command, timeline
- **Navigation**: expand/collapse nodes, focus between panels

### File Structure

| Category | Files | Total Lines |
|----------|-------|-------------|
| Core | model.go, types.go, keys.go | ~356 |
| Views | view*.go (6 files) | ~843 |
| Updates | update*.go (4 files) | ~783 |
| Modals | modal*.go (4 files) | ~703 |
| Helpers | helpers.go, styles.go | ~628 |

### Main Model

```go
type Model struct {
    // Data
    metadata *WorkflowMetadata
    tree     *TreeNode
    nodes    []*TreeNode
    
    // Fetcher for on-demand subworkflows
    fetcher MetadataFetcher
    
    // UI state
    cursor       int
    focus        PanelFocus  // FocusTree or FocusDetails
    viewMode     ViewMode
    width, height int
    treeWidth, detailsWidth int
    
    // Loading
    isLoading      bool
    loadingMessage string
    loadingSpinner spinner.Model
    
    // Modals (each has its own viewport)
    showLogModal, showInputsModal, showOutputsModal bool
    showCallInputsModal, showCallOutputsModal bool
    showGlobalTimelineModal bool
    // ... viewports for each modal
    
    // Components
    keys           KeyMap
    help           help.Model
    detailViewport viewport.Model
}
```

### ViewMode e PanelFocus

```go
type ViewMode int
const (
    ViewModeTree ViewMode = iota
    ViewModeDetails
    ViewModeCommand
    ViewModeLogs
    ViewModeInputs
    ViewModeOutputs
    ViewModeHelp
)

type PanelFocus int
const (
    FocusTree PanelFocus = iota
    FocusDetails
)
```

### Debug Key Bindings

| Key | Action |
|-----|--------|
| `↑/k` | Navigate up |
| `↓/j` | Navigate down |
| `enter/→` | Expand node |
| `←/h` | Collapse node |
| `tab` | Toggle tree/details focus |
| `L` | View logs (stdout/stderr) |
| `i` | View workflow inputs |
| `o` | View workflow outputs |
| `I` | View call inputs |
| `O` | View call outputs |
| `c` | View call command |
| `t` | View global timeline |
| `?` | Help |
| `esc` | Close modal/back |
| `q` | Quit |

### Screen Layout

```
┌────────────────────────────────────────────────────────────┐
│ 🔍 Workflow Debug - MyWorkflow                    ⏱ 5m 32s │
│ ● Succeeded  📊 10 calls  ⚠ 0 failures                     │
├──────────────────────────────┬─────────────────────────────┤
│ ▼ MyWorkflow (Succeeded)     │ Call Details                │
│   ├─ task1 (Succeeded)       │                             │
│   │  └─ shard-0 (Succeeded)  │ Status: ● Succeeded         │
│   │  └─ shard-1 (Succeeded)  │ Duration: 2m 15s            │
│   ├─ task2 (Succeeded)       │ Machine: n1-standard-4      │
│   └─ task3 (Running)         │ Preemptible: Yes (2 tries)  │
│      └─ shard-0 (Running)    │                             │
│                              │ Inputs:                     │
│                              │   file1: gs://bucket/...    │
│                              │   param: 42                 │
├──────────────────────────────┴─────────────────────────────┤
│ ↑↓ navigate  enter expand  tab switch  L logs  ? help  q   │
└────────────────────────────────────────────────────────────┘
```

## Code Patterns

### Naming Conventions

| Padrão | Exemplo | Uso |
|--------|---------|-----|
| `render*` | `renderHeader()` | Rendering functions |
| `handle*Keys` | `handleMainKeys()` | Keyboard handlers |
| `*Msg` | `workflowsLoadedMsg` | Message types |
| `view_*.go` | `view_header.go` | View files |
| `update_*.go` | `update_keys.go` | Update files |
| `modal_*.go` | `modal_logs.go` | Modal files |

### Estrutura de Arquivo View

```go
package dashboard

import (...)

// Related rendering functions
func (m Model) renderX() string {
    // Uses lipgloss for styling
    return lipgloss.JoinVertical(...)
}
```

### Estrutura de Arquivo Update

```go
package dashboard

import (...)

// Handlers for a specific category
func (m Model) handleXKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
    switch {
    case key.Matches(msg, m.keys.SomeKey):
        // Update state
        return m, someCmd
    }
    return m, nil
}
```

### Operações Assíncronas

```go
// Command that returns a message
func (m Model) fetchData() tea.Cmd {
    return func() tea.Msg {
        data, err := m.fetcher.GetData(context.Background())
        if err != nil {
            return errorMsg{err}
        }
        return dataLoadedMsg{data}
    }
}

// In Update, handle the message
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case dataLoadedMsg:
        m.data = msg.data
        return m, nil
    case errorMsg:
        m.error = msg.Error()
        return m, nil
    }
    return m, nil
}
```

## Extensibility

### Adding a New Screen

1. Create a directory in `internal/interfaces/tui/new_screen/`
2. Implement `Model`, `Init()`, `Update()`, `View()`
3. Add a constant in `app.go`:
   ```go
   const (
       ScreenDashboard Screen = iota
       ScreenDebug
       ScreenNovaTela  // novo
   )
   ```
4. Add a field to `AppModel` and cases in `Update`/`View`

### Adding a New Modal

1. Create `modal_name.go` with function `renderNameModal()`
2. Add state to the Model:
   ```go
   showNomeModal    bool
   nomeModalViewport viewport.Model
   ```
3. Add check in `View()`:
   ```go
   if m.showNomeModal {
       return m.renderNomeModal()
   }
   ```
4. Add key binding and handler

### Adding a New View Component

1. Create `view_component.go`
2. Implement `render*()` method
3. Integrate into main `View()` or composition

## Tests

The TUI tests focus mainly on parsing and data transformation logic:

```go
// debug/parser_test.go
func TestParseMetadata(t *testing.T) {
    // Tests conversion from JSON to internal structures
}
```

For visual tests, use the TUI with test data:

```bash
# Dashboard with mock
pumbaa dashboard --mock

# Debug with metadata file
pumbaa debug test_data/metadata.json
```

## Referências

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - Framework TUI
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Styling
- [Bubbles](https://github.com/charmbracelet/bubbles) - Components (spinner, viewport, textinput)
- [The Elm Architecture](https://guide.elm-lang.org/architecture/) - Padrão MVU

---

*Last updated: December 2025*
