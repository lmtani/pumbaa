package entities

import "time"

type Workflow struct {
	ID     string
	Name   string
	Start  time.Time
	End    time.Time
	Status string
	Calls  map[string][]Step
}
type Step struct {
	Name    string
	Spot    bool
	Start   string
	End     string
	Status  string
	Command string
}
