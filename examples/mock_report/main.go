// Package main allows generating a resource analysis report using mocked data from a debug log.
// This is useful for developing the report template and UI without needing to call the actual LLM.
//
// Usage:
//
//	go run examples/mock_report/main.go --debug-file=path/to/llm_debug.txt --output=report.html
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"github.com/lmtani/pumbaa/internal/application/ports"
	"github.com/lmtani/pumbaa/internal/infrastructure/recommendation"
	"github.com/lmtani/pumbaa/internal/infrastructure/templates"
)

func main() {
	// 1. Setup paths via flags
	cwd, _ := os.Getwd()
	defaultDebugFile := filepath.Join(cwd, "llm_debug2.txt")
	defaultOutputFile := filepath.Join(cwd, "mock_report.html")

	debugFile := flag.String("debug-file", defaultDebugFile, "Path to the LLM debug log file")
	outputFile := flag.String("output", defaultOutputFile, "Path to the output HTML report")
	flag.Parse()

	fmt.Printf("Reading debug log from: %s\n", *debugFile)

	// 2. Initialize Mock Generator
	mockGen := recommendation.NewMockLLMGenerator(*debugFile)
	if !mockGen.IsAvailable() {
		panic("Mock generator not available (check debug file path)")
	}

	// 3. Create Dummy Task Data
	// For the mock, we just need the Task names to match what's in the log to merge correctly.
	// But the Mock implementation I wrote also handles cases where tasks interact.
	// Let's create a minimal set of tasks based on the log content so we can exercise the full flow.
	tasks := []ports.TaskAnalysisData{
		{TaskName: "RunDeepVariant", SampleCount: 20, ResourceCost: 100.0},
		{TaskName: "FixItFelix", SampleCount: 20, ResourceCost: 20.0},
		{TaskName: "Downsample", SampleCount: 21, ResourceCost: 10.0},
	}

	// 4. Generate Recommendations
	ctx := context.Background()
	result, err := mockGen.GenerateRecommendations(ctx, tasks, 10)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Generated %d recommendations\n", len(result.Recommendations))
	fmt.Printf("Summary: %s\n", result.Summary)

	// 5. Generate Dummy Metrics to allow Modal to open
	type dummyDTO struct {
		TaskName             string  `json:"taskName"`
		ShardIndex           int     `json:"shardIndex"`
		CPUMean              float64 `json:"cpuMean"`
		MemoryPeakMB         float64 `json:"memoryPeakMB"`
		DiskPeakBytes        int64   `json:"diskPeakBytes"`
		DurationSeconds      float64 `json:"durationSeconds"`
		TotalInputBytes      int64   `json:"totalInputBytes"`
		MemoryRequestBytes   int64   `json:"memoryRequestBytes"`
		DiskSizeRequestBytes int64   `json:"diskSizeRequestBytes"`
	}

	var metrics []dummyDTO
	for _, rec := range result.Recommendations {
		// Create 3 dummy shards for each task to show some variation
		for i := 0; i < 3; i++ {
			metrics = append(metrics, dummyDTO{
				TaskName:             rec.TaskName,
				ShardIndex:           i + 1,
				CPUMean:              50.0 + float64(i)*5,
				MemoryPeakMB:         1024.0 + float64(i)*100,
				DiskPeakBytes:        10 * 1024 * 1024 * 1024,
				DurationSeconds:      300.0 + float64(i)*10,
				TotalInputBytes:      1024 * 1024 * 100,       // 100MB
				MemoryRequestBytes:   4 * 1024 * 1024 * 1024,  // 4GB
				DiskSizeRequestBytes: 20 * 1024 * 1024 * 1024, // 20GB
			})
		}
	}

	metricsJSON, err := json.Marshal(metrics)
	if err != nil {
		panic(err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		panic(err)
	}

	// 6. Render HTML
	fmt.Printf("Rendering report template...\n")
	htmlContent, err := templates.RenderReport(templates.ReportData{
		DataJSON:            template.JS(metricsJSON), // Populated metrics
		WorkflowsJSON:       template.JS("[]"),        // Empty workflows
		RecommendationsJSON: template.JS(resultJSON),
		LLMModelInfo:        "Mock/Replay",
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to render: %v", err))
	}

	if err := os.WriteFile(*outputFile, []byte(htmlContent), 0644); err != nil {
		panic(fmt.Sprintf("Failed to write file: %v", err))
	}

	fmt.Printf("Report generated at: %s\n", *outputFile)
}
