package debug

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// Colors - aliases of the shared palette (single source of truth in common/colors.go)
var (
	primaryColor = common.PrimaryColor
	borderColor  = common.BorderColor
	textColor    = common.TextColor
	mutedColor   = common.MutedColor
)

// Styles shared with common - aliased so call sites in this package stay short.
var (
	headerStyle              = common.HeaderStyle
	treePanelStyle           = common.PanelStyle
	detailsPanelStyle        = common.PanelStyle
	headerTitleStyle         = common.HeaderTitleStyle
	durationBadgeStyle       = common.DurationBadgeStyle
	costBadgeStyle           = common.CostBadgeStyle
	titleStyle               = common.TitleStyle
	labelStyle               = common.LabelStyle
	valueStyle               = common.ValueStyle
	mutedStyle               = common.MutedStyle
	errorStyle               = common.ErrorStyle
	modalStyle               = common.ModalStyle
	breadcrumbStyle          = common.BreadcrumbInactiveStyle
	breadcrumbSeparatorStyle = common.BreadcrumbSeparatorStyle
	breadcrumbActiveStyle    = common.BreadcrumbActiveStyle
)

// Debug-specific styles
var (
	searchBadgeStyle = lipgloss.NewStyle().
				Foreground(common.BadgeFg).
				Background(common.BadgeSearchBg).
				Padding(0, 1).
				MarginLeft(1)

	// Path style (for GCS/file paths)
	pathStyle = lipgloss.NewStyle().
			Foreground(common.InfoColor).
			Italic(true)

	// Informational note style (subworkflow hints, etc.)
	infoNoteStyle = lipgloss.NewStyle().
			Foreground(common.InfoColor).
			Italic(true)

	// Modal label style (brighter for dark background)
	modalLabelStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Bold(true)

	// Modal value style (bright for dark background)
	modalValueStyle = lipgloss.NewStyle().
			Foreground(textColor)

	// Modal path style (for GCS/file paths in modals)
	modalPathStyle = lipgloss.NewStyle().
			Foreground(common.InfoColor)

	// Temporary status message style
	temporaryStatusStyle = lipgloss.NewStyle().
				Foreground(common.StatusRunning).
				Bold(true)

	// Docker tag style (highlighted for visibility)
	tagStyle = lipgloss.NewStyle().
			Foreground(common.StatusSucceeded).
			Bold(true)

	// Section separator style
	sectionSeparatorStyle = lipgloss.NewStyle().
				Foreground(borderColor)

	// Selected tree node style
	selectedStyle = lipgloss.NewStyle().
			Background(common.HighlightColor).
			Foreground(textColor).
			Bold(true)
)
