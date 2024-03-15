package types

import (
	"fmt"
	"time"
)

func dashIfZero(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	if v == 0.0 {
		s = "-"
	}
	return s
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
