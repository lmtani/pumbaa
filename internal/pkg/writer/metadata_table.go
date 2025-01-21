package writer

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/lmtani/pumbaa/internal/entities"
)

type MetadataTableResponse struct {
	Metadata entities.MetadataResponse
}

func (MetadataTableResponse) Header() []string {
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
	var rows [][]string
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
	var names []string
	var duration []time.Duration
	var status []string

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

	var rows [][]string
	nShards, nStatusDone, nStatusFailed, totalDuration := countOccurrence(names, status, duration)

	for name := range nShards {
		row := []string{fmt.Sprintf("%v (Scatter)", name), "-", totalDuration[name].String(), fmt.Sprintf("%v/%v Done | %v Failed", nStatusDone[name], nShards[name], nStatusFailed[name])}
		rows = append(rows, row)
	}

	return rows
}

func countOccurrence(names, status []string, duration []time.Duration) (map[string]int, map[string]int, map[string]int, map[string]time.Duration) {
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

type rowSlice [][]string

func (c rowSlice) Len() int           { return len(c) }
func (c rowSlice) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
func (c rowSlice) Less(i, j int) bool { return c[i][0] < c[j][0] }
