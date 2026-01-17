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

// LLMGenerator uses an LLM to generate resource optimization recommendations.
// It has access to WDL tools to look up task definitions for context.
type LLMGenerator struct {
	llm       model.LLM
	tools     []tool.Tool
	available bool
}

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
	}
}

// IsAvailable returns true if the generator is properly configured.
func (g *LLMGenerator) IsAvailable() bool {
	return g.available && g.llm != nil
}

// GenerateRecommendations uses the LLM to analyze task data and generate recommendations.
func (g *LLMGenerator) GenerateRecommendations(ctx context.Context, tasks []ports.TaskAnalysisData) (*ports.RecommendationResult, error) {
	if !g.IsAvailable() {
		return nil, fmt.Errorf("LLM generator not available")
	}

	if len(tasks) == 0 {
		return &ports.RecommendationResult{}, nil
	}

	// Build prompt with task data
	prompt := buildPrompt(tasks)

	// Create request
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
					genai.NewPartFromText(systemInstruction),
				},
			},
		},
	}

	// Generate response (single turn, no tool calling for simplicity)
	respSeq := g.llm.GenerateContent(ctx, req, false)

	var responseText strings.Builder
	for resp, err := range respSeq {
		if err != nil {
			return nil, fmt.Errorf("LLM generation failed: %w", err)
		}
		if resp.Content != nil {
			for _, part := range resp.Content.Parts {
				if part.Text != "" {
					responseText.WriteString(part.Text)
				}
			}
		}
	}

	// Parse the response into recommendations
	return parseRecommendations(responseText.String(), tasks)
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

const systemInstruction = `You are an expert in WDL (Workflow Description Language) resource optimization.
Your task is to analyze resource usage data from workflow executions and generate optimization recommendations.

## Input Data
For each task, you will receive:
- Task name (the WDL task name)
- Resource REQUESTS (configured in WDL runtime: CPU, Memory, Disk)
- Actual USAGE (what was actually used: CPU mean %, Memory peak, Disk peak)
- Cost Contribution (% of total workflow cost) - USE THIS TO PRIORITIZE RECOMMENDATIONS
- Average execution duration and sample count
- Input file sizes (per sample, in bytes)

## Output Format
Your output MUST be valid JSON in this exact format:
{
  "summary": "Brief executive summary (max 200 words). Include: tasks ignored due to insufficient data, main cost drivers, and key optimization opportunities. This summary will be shown to users.",
  "recommendations": [
    {
      "taskName": "TaskName",
      "overallStatus": "warning",
      "diskFormula": "Int disk_gb = ceil(2 * size(input_bam, \"GB\") + 5)",
      "memoryFormula": "Int memory_gb = ceil(0.5 * size(input_bam, \"GB\") + 4)",
      "recommendations": [
        {"message": "CPU is well-utilized at 80%, maintain current allocation", "severity": "good"},
        {"message": "Memory peaks are high, consider increasing by 20%", "severity": "warning"},
        {"message": "Disk request is 3x more than needed, reduce immediately", "severity": "critical"}
      ]
    }
  ]
}

## Summary Guidelines
The summary field should:
- Start by mentioning any tasks that were ignored or have unreliable data (short duration, insufficient samples)
- Highlight the top 1-2 tasks by cost contribution
- Summarize the overall optimization potential
- Keep it under 200 words, be concise

## Status Assignment Rules
- overallStatus: "good" = ALL recommendations are good (no action needed)
- overallStatus: "warning" = At least one warning (optimization opportunity)
- overallStatus: "critical" = At least one critical issue (significant waste)
If ANY recommendation is critical, overallStatus MUST be "critical".

## Severity Levels
- "good": Well-utilized, no action needed
- "warning": Optimization opportunity exists
- "critical": Significant waste or misconfiguration

## Cloud Provider Constraints
1. MINIMUM DISK SIZE: Cloud providers typically have a minimum disk of 10 GB. Do NOT recommend reducing disk below 10 GB.
2. PREEMPTIBLE VMs: Tasks may run on preemptible VMs which are cheaper but can be interrupted.

## Data Quality Notes
1. SHORT TASKS: Tasks with duration < 60 seconds may show 0% CPU or inaccurate memory metrics due to sampling frequency. Be cautious when making recommendations for very short tasks.
2. COST PRIORITY: Focus your most detailed recommendations on tasks with HIGHEST cost contribution. A task with 50% of total cost deserves more optimization attention than one with 2%.

## Formula Guidelines
1. Use the LARGEST variable input for the formula (gives smaller, more intuitive multipliers)
2. Round up intercepts to whole numbers for safety margin
3. Always ensure minimum 10 GB for disk formulas (e.g., ceil(...) + 10, or max(10, ...))
4. Use ceil() to ensure sufficient resources
5. Keep formulas simple and readable`

func buildPrompt(tasks []ports.TaskAnalysisData) string {
	var sb strings.Builder

	// Calculate total cost for percentage
	var totalCost float64
	for _, task := range tasks {
		totalCost += task.ResourceCost
	}

	sb.WriteString("Analyze the following task resource usage data and generate optimization recommendations.\n")
	sb.WriteString("Tasks are sorted by cost contribution (highest first). Prioritize recommendations for high-cost tasks.\n\n")

	for _, task := range tasks {
		costPct := 0.0
		if totalCost > 0 {
			costPct = (task.ResourceCost / totalCost) * 100
		}

		// Calculate mean duration
		var meanDuration float64
		if len(task.DurationSeconds) > 0 {
			for _, d := range task.DurationSeconds {
				meanDuration += d
			}
			meanDuration /= float64(len(task.DurationSeconds))
		}

		sb.WriteString(fmt.Sprintf("## Task: %s\n", task.TaskName))
		sb.WriteString(fmt.Sprintf("**Cost Contribution: %.1f%%** (prioritize if high)\n", costPct))
		sb.WriteString(fmt.Sprintf("- Samples: %d | Avg Duration: %.0f seconds\n", task.SampleCount, meanDuration))

		// Resource requests
		sb.WriteString(fmt.Sprintf("- CPU Request: %s cores\n", task.CPURequest))
		sb.WriteString(fmt.Sprintf("- Memory Request: %.1f GB\n", task.MemoryReqGB))
		sb.WriteString(fmt.Sprintf("- Disk Request: %.1f GB\n\n", task.DiskReqGB))

		// Actual usage
		sb.WriteString("### Actual Usage:\n")
		sb.WriteString(fmt.Sprintf("- CPU means (%%): %v\n", task.CPUMeans))
		sb.WriteString(fmt.Sprintf("- Memory peaks (MB): %v\n", task.MemoryPeaksMB))
		sb.WriteString(fmt.Sprintf("- Disk peaks (GB): %v\n", task.DiskPeaksGB))

		// Short task warning
		if meanDuration < 60 {
			sb.WriteString("⚠️ **SHORT TASK** - Metrics may be inaccurate due to short execution time.\n")
		}

		// Input sizes
		if len(task.InputSizes) > 0 {
			sb.WriteString("\n### Input Sizes (bytes per sample):\n")
			for name, sizes := range task.InputSizes {
				sb.WriteString(fmt.Sprintf("- %s: %v\n", name, sizes))
			}
		}

		sb.WriteString("\n---\n\n")
	}

	sb.WriteString("Output JSON recommendations only. Include severity for each recommendation. Remember: min disk is 10 GB.")
	return sb.String()
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
		DiskFormula     string                  `json:"diskFormula,omitempty"`
		MemoryFormula   string                  `json:"memoryFormula,omitempty"`
		Recommendations []llmRecommendationItem `json:"recommendations,omitempty"`
	} `json:"recommendations"`
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
			DiskFormula:     rec.DiskFormula,
			MemoryFormula:   rec.MemoryFormula,
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
