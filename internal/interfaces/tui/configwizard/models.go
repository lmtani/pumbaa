package configwizard

import "github.com/charmbracelet/huh"

// ModelInfo represents an LLM model option for the config wizard.
type ModelInfo struct {
	ID            string
	DisplayName   string
	IsRecommended bool
	IsPreview     bool
}

// GeminiModels is the centralized list of available Gemini models.
// To add a new model, simply append to this slice.
var GeminiModels = []ModelInfo{
	{ID: "gemini-3-flash-preview", DisplayName: "Gemini 3 Flash Preview", IsRecommended: true},
	{ID: "gemini-2.5-flash", DisplayName: "Gemini 2.5 Flash"},
	{ID: "gemini-2.5-pro", DisplayName: "Gemini 2.5 Pro"},
}

// DefaultGeminiModel returns the recommended model ID.
func DefaultGeminiModel() string {
	for _, m := range GeminiModels {
		if m.IsRecommended {
			return m.ID
		}
	}
	if len(GeminiModels) > 0 {
		return GeminiModels[0].ID
	}
	return "gemini-2.5-flash"
}

// GetGeminiModelOptions generates huh.Option slice for the form select.
func GetGeminiModelOptions() []huh.Option[string] {
	options := make([]huh.Option[string], 0, len(GeminiModels))
	for _, m := range GeminiModels {
		label := m.DisplayName
		if m.IsRecommended {
			label += " ⭐"
		}
		options = append(options, huh.NewOption(label, m.ID))
	}
	return options
}
