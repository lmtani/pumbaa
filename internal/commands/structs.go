package commands

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/lmtani/cromwell-cli/pkg/cromwell"
)

type ResourceTableResponse struct {
	Operation string
	Total     cromwell.TotalResources
}

func (rtr ResourceTableResponse) Header() []string {
	return []string{"Resource", "Normalized to", "Preemptive", "Normal"}
}

func (rtr ResourceTableResponse) Rows() [][]string {
	rows := make([][]string, 4)

	rows = append(rows, []string{
		"CPUs",
		"1 hour",
		dashIfZero(rtr.Total.PreemptCPU),
		dashIfZero(rtr.Total.CPU),
	})

	rows = append(rows, []string{
		"Memory (GB)",
		"1 hour",
		dashIfZero(rtr.Total.PreemptMemory),
		dashIfZero(rtr.Total.Memory),
	})

	rows = append(rows, []string{
		"HDD disk (GB)",
		"1 month",
		dashIfZero(rtr.Total.PreemptHdd),
		dashIfZero(rtr.Total.Hdd),
	})
	rows = append(rows, []string{
		"SSD disk (GB)",
		"1 month",
		dashIfZero(rtr.Total.PreemptSsd),
		dashIfZero(rtr.Total.Ssd),
	})
	return rows
}

type QueryTableResponse struct {
	Results           []cromwell.QueryResponseWorkflow
	TotalResultsCount int
}

func (qtr QueryTableResponse) Header() []string {
	return []string{"Operation", "Name", "Start", "Duration", "Status"}
}

func (qtr QueryTableResponse) Rows() [][]string {
	rows := make([][]string, len(qtr.Results))
	timePattern := "2006-01-02 15h04m"
	for _, r := range qtr.Results {
		if r.End.IsZero() {
			r.End = time.Now()
		}
		elapsedTime := r.End.Sub(r.Start)
		rows = append(rows, []string{
			r.ID,
			fmt.Sprintf("%.35s", r.Name),
			r.Start.Format(timePattern),
			elapsedTime.Round(time.Second).String(),
			r.Status,
		})
	}
	return rows
}

type MetadataTableResponse struct {
	Metadata cromwell.MetadataResponse
}

func (mtr MetadataTableResponse) Header() []string {
	return []string{"task", "attempt", "elapsed", "status"}
}

func (mtr MetadataTableResponse) Rows() [][]string {
	singleRows := mtr.collectSingleTasks()
	scatterRows := mtr.collectScatterTasks()
	rows := append(singleRows, scatterRows...)
	rs := rowSlice(rows)
	sort.Sort(rs)
	return rs
}

func (mtr MetadataTableResponse) collectSingleTasks() [][]string {
	rows := [][]string{}
	for call, elements := range mtr.Metadata.Calls {
		substrings := strings.Split(call, ".")
		for _, elem := range elements {
			if elem.ExecutionStatus == "" {
				continue
			}
			if elem.End.IsZero() {
				elem.End = time.Now()
			}
			if elem.ShardIndex != -1 { // skip if it is a shard
				continue
			}
			elapsedTime := elem.End.Sub(elem.Start)
			row := []string{substrings[len(substrings)-1], fmt.Sprintf("%d", elem.Attempt), elapsedTime.String(), elem.ExecutionStatus}
			rows = append(rows, row)
		}
	}
	return rows
}

func (mtr MetadataTableResponse) collectScatterTasks() [][]string {
	names := []string{}
	duration := []time.Duration{}
	status := []string{}
	for call, elements := range mtr.Metadata.Calls {
		substrings := strings.Split(call, ".")
		for _, elem := range elements {
			if elem.ExecutionStatus == "" {
				continue
			}
			if elem.End.IsZero() {
				elem.End = time.Now()
			}
			if elem.ShardIndex == -1 { // skip if it is not a shard
				continue
			}
			elapsedTime := elem.End.Sub(elem.Start)
			status = append(status, elem.ExecutionStatus)
			duration = append(duration, elapsedTime)
			names = append(names, substrings[len(substrings)-1])
		}
	}

	rows := [][]string{}
	nShards, nStatusDone, nStatusFailed, totalDuration := countOccurence(names, status, duration)

	for name, _ := range nShards {
		row := []string{fmt.Sprintf("%v (Scatter)", name), "-", totalDuration[name].String(), fmt.Sprintf("%v/%v Done | %v Failed", nStatusDone[name], nShards[name], nStatusFailed[name])}
		rows = append(rows, row)
	}

	return rows
}

func countOccurence(names, status []string, duration []time.Duration) (map[string]int, map[string]int, map[string]int, map[string]time.Duration) {
	wfShards := make(map[string]int)
	wfStatusDone := make(map[string]int)
	wfStatusFailed := make(map[string]int)
	wfTimeElapsed := make(map[string]time.Duration)
	for idx, v := range names {
		wfShards[v]++
		wfTimeElapsed[v] += duration[idx]
		if status[idx] == "Done" {
			wfStatusDone[v]++
		} else if status[idx] == "Failed" {
			wfStatusFailed[v]++
		}

	}
	return wfShards, wfStatusDone, wfStatusFailed, wfTimeElapsed
}
