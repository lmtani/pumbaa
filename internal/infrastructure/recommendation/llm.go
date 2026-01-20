// Package recommendation provides implementations of the RecommendationGenerator interface.
package recommendation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/adk/model"
	"google.golang.org/adk/tool"
	"google.golang.org/genai"

	"github.com/lmtani/pumbaa/internal/config"
	"github.com/lmtani/pumbaa/internal/domain/ports"
	"github.com/lmtani/pumbaa/internal/infrastructure/chat/agent/tools"
	"github.com/lmtani/pumbaa/internal/infrastructure/chat/agent/tools/wdl"
	"github.com/lmtani/pumbaa/internal/infrastructure/chat/llm"
)

// NewLLMGenerator creates a new LLM-based recommendation generator.
// Returns a generator with available=false if LLM is not configured.
func NewLLMGenerator(cfg *config.Config, wdlRepo wdl.Repository) *LLMGenerator {
	if cfg == nil || cfg.LLMProvider == "" {
		return &LLMGenerator{available: false}
	}

	// Create LLM
	llmModel, err := llm.NewLLM(cfg)
	if err != nil {
		return &LLMGenerator{available: false}
	}

	// Build model info string
	var modelInfo string
	switch cfg.LLMProvider {
	case "vertex":
		modelInfo = fmt.Sprintf("vertex/%s", cfg.VertexModel)
	case "gemini":
		modelInfo = fmt.Sprintf("gemini/%s", cfg.GeminiModel)
	case "ollama":
		modelInfo = fmt.Sprintf("ollama/%s", cfg.OllamaModel)
	default:
		modelInfo = cfg.LLMProvider
	}

	// Create tools registry with WDL tools only
	registry := tools.NewRegistry()
	if wdlRepo != nil {
		registry.Register("wdl_list", wdl.NewListHandler(wdlRepo))
		registry.Register("wdl_search", wdl.NewSearchHandler(wdlRepo))
		registry.Register("wdl_info", wdl.NewInfoHandler(wdlRepo))
	}

	return &LLMGenerator{
		llm:       llmModel,
		tools:     []tool.Tool{tools.GetPumbaaTool(registry)},
		available: true,
		modelInfo: modelInfo,
	}
}

// IsAvailable returns true if the generator is properly configured.
func (g *LLMGenerator) IsAvailable() bool {
	return g.available && g.llm != nil
}

// ModelInfo returns information about the model being used (e.g., "vertex/gemini-2.5-flash").
func (g *LLMGenerator) ModelInfo() string {
	return g.modelInfo
}

// SetDebugWriter sets an optional debug writer for logging LLM interactions.
func (g *LLMGenerator) SetDebugWriter(w ports.LLMDebugWriter) {
	g.debugWriter = w
}

// GenerateRecommendations uses the LLM to analyze task data and generate recommendations.
func (g *LLMGenerator) GenerateRecommendations(ctx context.Context, tasks []ports.TaskAnalysisData, batchSize int) (*ports.RecommendationResult, error) {
	if !g.IsAvailable() {
		return nil, fmt.Errorf("LLM generator not available")
	}

	if len(tasks) == 0 {
		return &ports.RecommendationResult{}, nil
	}

	if batchSize <= 0 {
		batchSize = 25 // Default batch size
	}

	var allRecommendations []ports.TaskRecommendation

	// 1. Process in batches (severity and textual recommendations)
	for i := 0; i < len(tasks); i += batchSize {
		end := i + batchSize
		if end > len(tasks) {
			end = len(tasks)
		}
		batchCalls := tasks[i:end]

		batchRecs, err := g.processBatch(ctx, batchCalls)
		if err != nil {
			// Log error but continue with what we have?
			// For now, let's propagate the error as it might be an API issue
			return nil, fmt.Errorf("batch processing failed at index %d: %w", i, err)
		}
		allRecommendations = append(allRecommendations, batchRecs...)
	}

	// 2. Generate formulas (dedicated call for tasks with >= 3 samples)
	formulas, err := g.generateFormulas(ctx, tasks)
	if err != nil {
		// Formula generation is optional; continue without formulas if it fails
		formulas = nil
	}

	// Apply formulas to recommendations
	if formulas != nil {
		for i := range allRecommendations {
			if f, ok := formulas[allRecommendations[i].TaskName]; ok {
				allRecommendations[i].DiskFormula = f.DiskFormula
				allRecommendations[i].DiskReasoning = f.DiskReasoning
				allRecommendations[i].MemoryFormula = f.MemoryFormula
				allRecommendations[i].MemoryReasoning = f.MemoryReasoning
			}
		}
	}

	// 3. Generate Global Summary
	// We use the aggregated results to create a high-level summary
	summary, err := g.generateGlobalSummary(ctx, tasks, allRecommendations)
	if err != nil {
		// If summary generation fails, we still return the recommendations with a generic message
		summary = "Executive summary generation failed, but detailed recommendations are available below."
	}

	return &ports.RecommendationResult{
		Summary:         summary,
		Recommendations: allRecommendations,
	}, nil
}

// convertToolsToGenAI converts ADK tools to genai format
func convertToolsToGenAI(adkTools []tool.Tool) []*genai.Tool {
	var genaiTools []*genai.Tool
	for _, t := range adkTools {
		if td, ok := t.(interface {
			Declaration() *genai.FunctionDeclaration
		}); ok {
			genaiTools = append(genaiTools, &genai.Tool{
				FunctionDeclarations: []*genai.FunctionDeclaration{td.Declaration()},
			})
		}
	}
	return genaiTools
}

func parseRecommendations(response string, tasks []ports.TaskAnalysisData) (*ports.RecommendationResult, error) {
	// Find JSON in response (may be wrapped in markdown code blocks)
	jsonStr := extractJSON(response)
	if jsonStr == "" {
		return &ports.RecommendationResult{}, nil
	}

	var llmResp llmResponse
	if err := json.Unmarshal([]byte(jsonStr), &llmResp); err != nil {
		return &ports.RecommendationResult{}, nil
	}

	// Build task lookup map
	taskMap := make(map[string]ports.TaskAnalysisData)
	for _, t := range tasks {
		taskMap[t.TaskName] = t
	}

	// Convert to TaskRecommendation
	var recommendations []ports.TaskRecommendation
	for _, rec := range llmResp.Recommendations {
		task, ok := taskMap[rec.TaskName]
		if !ok {
			continue
		}

		// Convert recommendations with severity
		var items []ports.RecommendationItem
		for _, r := range rec.Recommendations {
			severity := ports.SeverityWarning // default
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

		// Parse overall status from LLM
		overallStatus := ports.SeverityWarning // default
		switch rec.OverallStatus {
		case "good":
			overallStatus = ports.SeverityGood
		case "critical":
			overallStatus = ports.SeverityCritical
		}

		recommendations = append(recommendations, ports.TaskRecommendation{
			TaskName:        rec.TaskName,
			SampleCount:     task.SampleCount,
			OverallStatus:   overallStatus,
			ResourceCost:    task.ResourceCost,
			Recommendations: items,
		})
	}

	return &ports.RecommendationResult{
		Summary:         llmResp.Summary,
		Recommendations: recommendations,
	}, nil
}

func extractJSON(s string) string {
	start := strings.Index(s, "{")
	if start == -1 {
		return ""
	}

	depth := 0
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return ""
}

func (g *LLMGenerator) processBatch(ctx context.Context, tasks []ports.TaskAnalysisData) ([]ports.TaskRecommendation, error) {
	prompt := buildPrompt(tasks)
	responseText, err := g.callLLM(ctx, prompt, systemInstruction)
	if err != nil {
		return nil, err
	}

	// Write debug log if writer is configured
	if g.debugWriter != nil {
		_ = g.debugWriter.WriteInteraction("BATCH_RECOMMENDATIONS", systemInstruction, prompt, responseText)
	}

	result, err := parseRecommendations(responseText, tasks)
	if err != nil {
		return nil, err
	}
	return result.Recommendations, nil
}

func (g *LLMGenerator) callLLM(ctx context.Context, prompt string, sysInst string) (string, error) {
	history := []*genai.Content{
		{
			Role: "user",
			Parts: []*genai.Part{
				genai.NewPartFromText(prompt),
			},
		},
	}

	req := &model.LLMRequest{
		Contents: history,
		Config: &genai.GenerateContentConfig{
			Tools: convertToolsToGenAI(g.tools),
			SystemInstruction: &genai.Content{
				Parts: []*genai.Part{
					genai.NewPartFromText(sysInst),
				},
			},
		},
	}

	respSeq := g.llm.GenerateContent(ctx, req, false)

	var responseText strings.Builder
	for resp, err := range respSeq {
		if err != nil {
			return "", fmt.Errorf("LLM generation failed: %w", err)
		}
		if resp.Content != nil {
			for _, part := range resp.Content.Parts {
				if part.Text != "" {
					responseText.WriteString(part.Text)
				}
			}
		}
	}
	return responseText.String(), nil
}

func (g *LLMGenerator) generateGlobalSummary(ctx context.Context, tasks []ports.TaskAnalysisData, recommendations []ports.TaskRecommendation) (string, error) {
	prompt := buildSummaryPrompt(tasks, recommendations)
	responseText, err := g.callLLM(ctx, prompt, summarySystemInstruction)
	if err != nil {
		return "", err
	}

	// Write debug log if writer is configured
	if g.debugWriter != nil {
		_ = g.debugWriter.WriteInteraction("GLOBAL_SUMMARY", summarySystemInstruction, prompt, responseText)
	}

	jsonStr := extractJSON(responseText)
	if jsonStr == "" {
		return "", fmt.Errorf("no JSON found in summary response")
	}

	var resp struct {
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		return "", err
	}
	return resp.Summary, nil
}

// generateFormulas generates disk and memory formulas for tasks with sufficient data.
// It uses a dedicated LLM call with specialized prompt for formula derivation.
func (g *LLMGenerator) generateFormulas(ctx context.Context, tasks []ports.TaskAnalysisData) (map[string]formulaItem, error) {
	// Filter tasks with sufficient samples (>= 3)
	var eligibleTasks []ports.TaskAnalysisData
	for _, task := range tasks {
		if task.SampleCount >= 3 {
			eligibleTasks = append(eligibleTasks, task)
		}
	}

	if len(eligibleTasks) == 0 {
		return nil, nil
	}

	prompt := buildFormulaPrompt(eligibleTasks)

	responseText, err := g.callLLM(ctx, prompt, formulaSystemInstruction)
	if err != nil {
		return nil, err
	}

	// Write debug log if writer is configured
	if g.debugWriter != nil {
		_ = g.debugWriter.WriteInteraction("FORMULA_GENERATION", formulaSystemInstruction, prompt, responseText)
	}

	jsonStr := extractJSON(responseText)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON found in formula response")
	}

	var resp formulaResponse
	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		return nil, err
	}

	// Build map for easy lookup
	result := make(map[string]formulaItem)
	for _, f := range resp.Formulas {
		result[f.TaskName] = f
	}
	return result, nil
}
