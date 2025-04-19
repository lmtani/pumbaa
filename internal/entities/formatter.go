package entities

// FormatType represents the type of formatter to use
type FormatType string

const (
	// TableFormat represents the table formatter (default)
	TableFormat FormatType = "table"
	// JSONFormat represents the JSON formatter
	JSONFormat FormatType = "json"
)
