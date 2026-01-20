package recommendation

import (
	"fmt"
	"sort"
	"strings"

	"github.com/lmtani/pumbaa/internal/application/ports"
)

const formulaSystemInstruction = `You are an expert in WDL (Workflow Description Language) resource optimization, specializing in deriving resource formulas from execution data.

## Your Task
Analyze the provided execution data and derive formulas for disk and memory allocation based on input file sizes.

## CRITICAL: Why Dynamic Formulas Are Essential
The execution samples you see are just a SMALL SUBSET of real-world data. In production:
- **Genomics workflows** process files ranging from 1GB to 200GB+
- **A fixed value that works for 5GB input will FAIL for 50GB input**
- **Disk usage almost ALWAYS scales with input size** (temporary files, sorted outputs, indexes)

**ALWAYS prefer dynamic formulas over fixed integers**, especially for disk.

## Methodology
For each task, you will receive a table of execution samples showing:
- input_gb: Size of the largest input file in GB
- disk_peak_gb: Actual disk usage peak in GB
- memory_peak_gb: Actual memory usage peak in GB

**Step 1: Calculate the relationship**
- slope = (max_usage - min_usage) / (max_input - min_input)
- If inputs have low variance, estimate slope from ratio: disk_peak / input_size

**Step 2: Derive the formula**
- Formula format: ceil(slope * size(input_name, "GB") + intercept)
- Add 10-20% safety margin to the intercept

## Formula Guidelines

### DISK (Almost always needs a dynamic formula)
- Disk usage scales with input in most bioinformatics tasks
- Even if samples show little variation, EXTRAPOLATE: what happens if input is 10x larger?
- Calculate ratio: avg(disk_peak) / avg(input_size) to estimate multiplier
- Only use fixed value if task genuinely doesn't read input files (e.g., download tasks)

### MEMORY
- Memory often has a base requirement + scaling component
- Can use fixed values if peaks are consistent regardless of input size
- But prefer formulas when memory clearly scales with input

### Minimum values (GCP constraints)
- disk >= 10 GB
- memory >= 1 GB

## Output Format
Your output MUST be valid JSON in this exact format:
{
  "formulas": [
    {
      "taskName": "AlignReads",
      "diskFormula": "ceil(2.5 * size(input_bam, \"GB\") + 15)",
      "diskReasoning": "Disk usage ~2.5x input size (observed 12GB disk for 5GB input); formula scales for larger genomes",
      "memoryFormula": "ceil(0.8 * size(input_bam, \"GB\") + 8)",
      "memoryReasoning": "Memory scales ~0.8x input size with 8GB base overhead"
    },
    {
      "taskName": "DownloadReference",
      "diskFormula": "50",
      "diskReasoning": "Fixed 50GB - downloads fixed reference genome, no input file dependency",
      "memoryFormula": "4",
      "memoryReasoning": "Fixed 4GB - simple download task with consistent memory usage"
    }
  ]
}

## When to Use Fixed Values (RARE)
Only use fixed integers when:
1. Task does NOT process variable-size input files (downloads, uploads, fixed operations)
2. Resource usage is genuinely constant regardless of any input

**If in doubt, use a formula.** A formula that overestimates is better than a fixed value that fails on larger inputs.`

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
  "recommendations": [
    {
      "taskName": "TaskName",
      "overallStatus": "warning",
      "recommendations": [
        {"message": "CPU is well-utilized at 80%, maintain current allocation", "severity": "good"},
        {"message": "Memory peaks are high, consider increasing by 20%", "severity": "warning"},
        {"message": "Disk request is 3x more than needed, reduce immediately", "severity": "critical"}
      ]
    }
  ]
}

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
4. COST PRIORITY: Focus your most detailed recommendations on tasks with HIGHEST cost contribution. A task with 50% of total cost deserves more optimization attention than one with 2%.`

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
	sb.WriteString("**Global Stats**:\n")
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

func buildFormulaPrompt(tasks []ports.TaskAnalysisData, recommendations []ports.TaskRecommendation) string {
	var sb strings.Builder

	sb.WriteString("Derive disk and memory formulas for the following tasks based on their execution data.\n\n")
	sb.WriteString("**IMPORTANT**: Choose the input that VARIES between samples for the formula. Inputs with constant size (like reference genomes) should NOT be used - look for BAM/CRAM files or other sample-specific inputs.\n\n")

	// Map recommendations for easy lookup
	recMap := make(map[string]ports.TaskRecommendation)
	for _, r := range recommendations {
		recMap[r.TaskName] = r
	}

	for _, task := range tasks {
		// Only include tasks with sufficient samples
		if task.SampleCount < 3 {
			continue
		}

		sb.WriteString(fmt.Sprintf("## Task: %s\n", task.TaskName))
		sb.WriteString(fmt.Sprintf("Samples: %d\n", task.SampleCount))

		// Inject Optimization Context
		if rec, ok := recMap[task.TaskName]; ok {
			sb.WriteString(fmt.Sprintf("\n### Optimization Context (from previous analysis):\n"))
			sb.WriteString(fmt.Sprintf("- Overall Status: %s\n", rec.OverallStatus))
			for _, item := range rec.Recommendations {
				sb.WriteString(fmt.Sprintf("- [%s] %s\n", item.Severity, item.Message))
			}
		}
		sb.WriteString("\n")

		// Collect all inputs and calculate their variance
		type inputInfo struct {
			name     string
			sizes    []int64
			minGB    float64
			maxGB    float64
			avgGB    float64
			variance float64
		}
		var inputs []inputInfo

		for inputName, sizes := range task.InputSizes {
			if len(sizes) == 0 {
				continue
			}
			var sum float64
			var minVal, maxVal float64 = float64(sizes[0]), float64(sizes[0])
			for _, s := range sizes {
				gb := float64(s) / (1024 * 1024 * 1024)
				sum += gb
				if gb < minVal {
					minVal = gb
				}
				if gb > maxVal {
					maxVal = gb
				}
			}
			avgGB := sum / float64(len(sizes))

			// Calculate variance
			var varianceSum float64
			for _, s := range sizes {
				gb := float64(s) / (1024 * 1024 * 1024)
				varianceSum += (gb - avgGB) * (gb - avgGB)
			}
			variance := varianceSum / float64(len(sizes))

			inputs = append(inputs, inputInfo{
				name:     inputName,
				sizes:    sizes,
				minGB:    minVal,
				maxGB:    maxVal,
				avgGB:    avgGB,
				variance: variance,
			})
		}

		if len(inputs) > 0 {
			// Sort inputs by variance (highest first) to highlight variable inputs
			sort.Slice(inputs, func(i, j int) bool {
				return inputs[i].variance > inputs[j].variance
			})

			// Show input summary
			sb.WriteString("### Available Inputs (sorted by variance - use the one that varies!):\n")
			for _, inp := range inputs {
				varianceLabel := "CONSTANT"
				if inp.variance > 0.01 {
					varianceLabel = "VARIES"
				}
				sb.WriteString(fmt.Sprintf("- **%s**: min=%.2f GB, max=%.2f GB, avg=%.2f GB [%s]\n",
					inp.name, inp.minGB, inp.maxGB, inp.avgGB, varianceLabel))
			}
			sb.WriteString("\n")

			// Build execution table with ALL inputs
			sb.WriteString("### Execution Data Table\n")

			// Build header
			sb.WriteString("| sample |")
			for _, inp := range inputs {
				sb.WriteString(fmt.Sprintf(" %s_gb |", inp.name))
			}
			sb.WriteString(" disk_peak_gb | memory_peak_gb |\n")

			// Build separator
			sb.WriteString("|--------|")
			for range inputs {
				sb.WriteString("----------|")
			}
			sb.WriteString("--------------|----------------|\n")

			// Find minimum sample count
			numSamples := len(task.DiskPeaksGB)
			if numSamples > len(task.MemoryPeaksMB) {
				numSamples = len(task.MemoryPeaksMB)
			}
			for _, inp := range inputs {
				if len(inp.sizes) < numSamples {
					numSamples = len(inp.sizes)
				}
			}

			// Build data rows
			for i := 0; i < numSamples; i++ {
				sb.WriteString(fmt.Sprintf("| %d |", i+1))
				for _, inp := range inputs {
					inputGB := float64(inp.sizes[i]) / (1024 * 1024 * 1024)
					sb.WriteString(fmt.Sprintf(" %.2f |", inputGB))
				}
				diskGB := task.DiskPeaksGB[i]
				memGB := task.MemoryPeaksMB[i] / 1024
				sb.WriteString(fmt.Sprintf(" %.2f | %.2f |\n", diskGB, memGB))
			}
		} else {
			// No input sizes available
			sb.WriteString("### Execution Data (no input sizes available)\n")
			sb.WriteString("| sample | disk_peak_gb | memory_peak_gb |\n")
			sb.WriteString("|--------|--------------|----------------|\n")

			numSamples := len(task.DiskPeaksGB)
			if numSamples > len(task.MemoryPeaksMB) {
				numSamples = len(task.MemoryPeaksMB)
			}

			for i := 0; i < numSamples; i++ {
				diskGB := task.DiskPeaksGB[i]
				memGB := task.MemoryPeaksMB[i] / 1024
				sb.WriteString(fmt.Sprintf("| %d | %.2f | %.2f |\n", i+1, diskGB, memGB))
			}
			sb.WriteString("\n*Note: Use fixed values based on peak usage since no input correlation is available.*\n")
		}

		sb.WriteString("\n---\n\n")
	}

	sb.WriteString("Output JSON with formulas for each task. Remember: disk minimum is 10 GB, memory minimum is 1 GB.\n")
	sb.WriteString("**Use the input that VARIES between samples for your formula. If all inputs are constant, use a fixed value based on peak usage.**")
	return sb.String()
}
