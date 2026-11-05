package workflow

import "time"

// Submission is what a run was submitted with, as the server kept it: the
// workflow source, the inputs and options documents and the labels. It is
// the part of the metadata that lets a run be reproduced, and a fraction of
// the size of the whole — a run's full metadata grows with every call.
//
// The dependencies zip is not here: Cromwell discards it once the run leaves
// the workflow store, so only the machine that submitted the run still has
// it.
type Submission struct {
	ID          string
	Name        string
	Labels      map[string]string
	SubmittedAt time.Time
	// Workflow is the main WDL source. Empty when the run was submitted by
	// URL, in which case WorkflowURL says where it came from.
	Workflow    string
	WorkflowURL string
	Inputs      string
	Options     string
}
