package recommendation

import (
	"fmt"
	"strings"

	"github.com/lmtani/pumbaa/internal/domain/ports"
)

const summarySystemInstruction = `You are an expert in WDL resource optimization.
Your task is to write a concise, balanced Executive Summary based on the provided aggregate statistics.

Guidelines:
- Be BALANCED: mention both what's working well AND what needs improvement
- If most tasks are "Good", lead with that positive finding
- Only emphasize problems if Critical or Warning count is significant (>30% of tasks)
- Focus on actionable insights for the top cost drivers
- Keep it under 200 words

Output format: JSON {"summary": "your summary text"}`

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
- "good": Well-utilized, no action needed. (e.g. usage is > 75% of request)
- "warning": Optimization opportunity exists. (e.g. usage is < 60% of request)
- "critical": Significant waste or misconfiguration. (e.g. usage is < 20% of request)

## Tolerance Guidelines
1. BUFFER/SAFETY MARGIN: It is normal to have some buffer. If a task requests 12 GB and uses 10 GB (83%), this is GOOD. Do NOT flag it as a warning.
2. 20% THRESHOLD: Only suggest reducing resources if usage is consistently below 80% of the request.
3. PEAKS: Always respect the peak usage. If peak is 10 GB, request should probably be at least 11-12 GB.

## Cloud Provider Constraints (GCP)
1. MINIMUM DISK SIZE: GCP has a minimum disk of 10 GB. Do NOT recommend reducing disk below 10 GB.
2. MINIMUM MEMORY: GCP has a minimum memory of 1 GB. Do NOT recommend reducing memory below 1 GB.
3. PREEMPTIBLE VMs: Tasks may run on preemptible VMs which are cheaper but can be interrupted.

## Data Quality Notes
1. SHORT TASKS: Tasks with duration < 60 seconds may show 0% CPU or inaccurate memory metrics due to sampling frequency. Be cautious when making recommendations for very short tasks.
2. CPU 0%: A CPU mean of 0% does NOT necessarily mean the task was idle. It often indicates the task completed very quickly (before monitoring could sample CPU usage) or that monitoring data was not collected. In these cases, do NOT recommend reducing CPU - the current allocation may be appropriate. Instead, note that metrics are unreliable for this task.
3. MEMORY/DISK 0: Similarly, if memory_peak=0 or disk_peak=0, it usually means monitoring failed to capture metrics, not that the task used no resources. Do not recommend reducing these resources based on 0 values.
4. COST PRIORITY: Focus your most detailed recommendations on tasks with HIGHEST cost contribution. A task with 50% of total cost deserves more optimization attention than one with 2%.

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

	sb.WriteString("Output JSON recommendations only. Include severity for each recommendation. Remember GCP constraints: min disk is 10 GB, min memory is 1 GB.")
	return sb.String()
}

func buildSummaryPrompt(tasks []ports.TaskAnalysisData, recommendations []ports.TaskRecommendation) string {
	var sb strings.Builder

	// Calculate global stats
	var totalCost float64
	var criticalCount, warningCount, goodCount int

	for _, t := range tasks {
		totalCost += t.ResourceCost
	}

	// Build a map of tasks that have recommendations
	tasksWithRecs := make(map[string]bool)
	for _, r := range recommendations {
		tasksWithRecs[r.TaskName] = true
		switch r.OverallStatus {
		case ports.SeverityCritical:
			criticalCount++
		case ports.SeverityWarning:
			warningCount++
		case ports.SeverityGood:
			goodCount++
		}
	}

	// Tasks without explicit recommendations are considered "good" (well-optimized)
	tasksWithoutRecs := len(tasks) - len(tasksWithRecs)
	goodCount += tasksWithoutRecs

	sb.WriteString("Generate an Executive Summary for the workflow resource analysis based on the following aggregate data.\n\n")
	sb.WriteString(fmt.Sprintf("**Global Stats**:\n"))
	sb.WriteString(fmt.Sprintf("- Total Tasks Analyzed: %d\n", len(tasks)))
	sb.WriteString(fmt.Sprintf("- Optimization Status: %d Critical, %d Warnings, %d Good (including %d tasks with no issues found)\n", criticalCount, warningCount, goodCount, tasksWithoutRecs))
	sb.WriteString("\n**Top 10 Tasks by Resource Cost**:\n")

	// Sort tasks by cost (they should be already sorted, but let's be safe or just take top 10 if input is sorted)
	// Input tasks are supposed to be sorted.
	limit := 10
	if len(tasks) < limit {
		limit = len(tasks)
	}

	for i := 0; i < limit; i++ {
		t := tasks[i]
		costPct := 0.0
		if totalCost > 0 {
			costPct = (t.ResourceCost / totalCost) * 100
		}
		sb.WriteString(fmt.Sprintf("%d. %s (%.1f%% of cost)\n", i+1, t.TaskName, costPct))
	}

	sb.WriteString("\n**Instructions**:\n")
	sb.WriteString("Write a BALANCED executive summary (max 200 words) that:\n")
	sb.WriteString("1. Starts with the overall health - if most tasks are 'Good', lead with that positive finding\n")
	sb.WriteString("2. Mentions the main cost drivers (top tasks by cost)\n")
	sb.WriteString("3. Lists specific actions to take, if any\n")
	sb.WriteString("4. If most tasks are well-optimized, acknowledge that and focus on the few that need attention\n")
	sb.WriteString("Output ONLY a JSON object: {\"summary\": \"...\"}")

	return sb.String()
}
