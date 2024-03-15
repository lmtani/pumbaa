package types

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type MetadataTableResponse struct {
	Metadata MetadataResponse
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
