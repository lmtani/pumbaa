package recommendation

import (
	"google.golang.org/adk/model"
	"google.golang.org/adk/tool"

	"github.com/lmtani/pumbaa/internal/application/ports"
)

// LLMGenerator uses an LLM to generate resource optimization recommendations.
// It has access to WDL tools to look up task definitions for context.
type LLMGenerator struct {
	llm         model.LLM
	tools       []tool.Tool
	available   bool
	modelInfo   string // e.g., "vertex/gemini-2.5-flash"
	debugWriter ports.LLMDebugWriter
}

// llmRecommendationItem represents a recommendation with severity from LLM
type llmRecommendationItem struct {
	Message  string `json:"message"`
	Severity string `json:"severity"` // good, warning, critical
}

// llmResponse represents the expected JSON structure from the LLM
type llmResponse struct {
	Summary         string `json:"summary,omitempty"`
	Recommendations []struct {
		TaskName        string                  `json:"taskName"`
		OverallStatus   string                  `json:"overallStatus,omitempty"` // good, warning, critical
		Recommendations []llmRecommendationItem `json:"recommendations,omitempty"`
	} `json:"recommendations"`
}

// formulaItem represents a single task's formula from the LLM
type formulaItem struct {
	TaskName        string `json:"taskName"`
	DiskFormula     string `json:"diskFormula"`
	DiskReasoning   string `json:"diskReasoning"`
	MemoryFormula   string `json:"memoryFormula"`
	MemoryReasoning string `json:"memoryReasoning"`
}

// formulaResponse represents the expected JSON structure for formula generation
type formulaResponse struct {
	Formulas []formulaItem `json:"formulas"`
}
