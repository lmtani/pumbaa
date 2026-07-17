package recommendation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/lmtani/pumbaa/internal/application/ports"
)

// MockLLMGenerator replays recommendations from a debug log file.
type MockLLMGenerator struct {
	debugFilePath string
}

// NewMockLLMGenerator creates a new mock generator.
func NewMockLLMGenerator(debugFilePath string) *MockLLMGenerator {
	return &MockLLMGenerator{
		debugFilePath: debugFilePath,
	}
}

func (m *MockLLMGenerator) IsAvailable() bool {
	_, err := os.Stat(m.debugFilePath)
	return err == nil
}

func (m *MockLLMGenerator) ModelInfo() string {
	return "mock/replay"
}

func (m *MockLLMGenerator) SetDebugWriter(w ports.LLMDebugWriter) {
	// No-op
}

func (m *MockLLMGenerator) GenerateRecommendations(ctx context.Context, tasks []ports.TaskAnalysisData, batchSize int) (*ports.RecommendationResult, error) {
	content, err := os.ReadFile(m.debugFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read debug file: %w", err)
	}
	text := string(content)

	// 1. Parse Batch Recommendations
	batchJSON := extractSectionJSON(text, "=== BATCH_RECOMMENDATIONS ===")
	if batchJSON == "" {
		return nil, fmt.Errorf("BATCH_RECOMMENDATIONS not found in debug file")
	}

	var batchResp llmResponse
	if err := json.Unmarshal([]byte(batchJSON), &batchResp); err != nil {
		return nil, fmt.Errorf("failed to parse batch recommendations: %w", err)
	}

	// 2. Parse Formula Generation
	formulaJSON := extractSectionJSON(text, "=== FORMULA_GENERATION ===")
	// Formula generation might be optional or missing in some logs
	var formulas map[string]formulaItem
	if formulaJSON != "" {
		var formulaResp formulaResponse
		if err := json.Unmarshal([]byte(formulaJSON), &formulaResp); err != nil {
			return nil, fmt.Errorf("failed to parse formulas: %w", err)
		}
		formulas = make(map[string]formulaItem)
		for _, f := range formulaResp.Formulas {
			formulas[f.TaskName] = f
		}
	}

	// 3. Parse Global Summary
	summaryJSON := extractSectionJSON(text, "=== GLOBAL_SUMMARY ===")
	summary := "Summary not found in logs."
	if summaryJSON != "" {
		var summaryResp struct {
			Summary string `json:"summary"`
		}
		if err := json.Unmarshal([]byte(summaryJSON), &summaryResp); err == nil {
			summary = summaryResp.Summary
		}
	}

	// 4. Merge and Build Result
	taskMap := make(map[string]ports.TaskAnalysisData)
	for _, t := range tasks {
		taskMap[t.TaskName] = t
	}

	var recommendations []ports.TaskRecommendation
	for _, rec := range batchResp.Recommendations {
		task, ok := taskMap[rec.TaskName]
		if !ok {
			// If we can't find the task in current input, we might skip or create a dummy
			// For this mock, let's assume input tasks match the log or we just use what's in the log
			// But we need TaskAnalysisData to fill missing fields if any.
			// Simpler approach: Create recommendation even if task lookup fails, using 0 for missing data
			task = ports.TaskAnalysisData{}
		}

		// Convert severity strings to enum
		var items []ports.RecommendationItem
		for _, r := range rec.Recommendations {
			severity := ports.SeverityWarning
			switch r.Severity {
			case "good":
				severity = ports.SeverityGood
			case "critical":
				severity = ports.SeverityCritical
			}
			items = append(items, ports.RecommendationItem{
				Message:  r.Message,
				Severity: severity,
			})
		}

		overallStatus := ports.SeverityWarning
		switch rec.OverallStatus {
		case "good":
			overallStatus = ports.SeverityGood
		case "critical":
			overallStatus = ports.SeverityCritical
		}

		outRec := ports.TaskRecommendation{
			TaskName:        rec.TaskName,
			SampleCount:     task.SampleCount,
			OverallStatus:   overallStatus,
			ResourceCost:    task.ResourceCost,
			Recommendations: items,
		}

		// Attach formula if available
		if f, ok := formulas[rec.TaskName]; ok {
			outRec.DiskFormula = f.DiskFormula
			outRec.DiskReasoning = f.DiskReasoning
			outRec.MemoryFormula = f.MemoryFormula
			outRec.MemoryReasoning = f.MemoryReasoning
		}

		recommendations = append(recommendations, outRec)
	}

	return &ports.RecommendationResult{
		Summary:         summary,
		Recommendations: recommendations,
	}, nil
}

func extractSectionJSON(fullText, sectionHeader string) string {
	startIdx := strings.Index(fullText, sectionHeader)
	if startIdx == -1 {
		return ""
	}

	// Find "--- LLM RESPONSE ---" after the header
	responseMarker := "--- LLM RESPONSE ---"
	responseIdx := strings.Index(fullText[startIdx:], responseMarker)
	if responseIdx == -1 {
		return ""
	}

	// Start searching for JSON after the marker
	jsonStartSearch := startIdx + responseIdx + len(responseMarker)

	// Find the first '{'
	jsonStart := strings.Index(fullText[jsonStartSearch:], "{")
	if jsonStart == -1 {
		return ""
	}
	realStart := jsonStartSearch + jsonStart

	// Extract JSON using brace counting
	depth := 0
	for i := realStart; i < len(fullText); i++ {
		switch fullText[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return fullText[realStart : i+1]
			}
		}
	}
	return ""
}
