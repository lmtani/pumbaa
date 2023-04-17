package cromwell_client

import "time"

type ParamsMetadataGet struct {
	IncludeKey         []string `url:"includeKey,omitempty"`
	ExcludeKey         []string `url:"excludeKey,omitempty"`
	ExpandSubWorkflows bool     `url:"expandSubWorkflows,omitempty"`
}

type ParamsQueryGet struct {
	Submission          time.Time `url:"submission,omitempty" layout:"2006-01-02T15:04:05.000Z"`
	Start               time.Time `url:"start,omitempty" layout:"2006-01-02T15:04:05.000Z"`
	End                 time.Time `url:"end,omitempty" layout:"2006-01-02T15:04:05.000Z"`
	Status              string    `url:"status,omitempty"`
	Name                string    `url:"name,omitempty"`
	Id                  string    `url:"id,omitempty"`
	IncludeSubworkflows bool      `url:"includeSubworkflows,omitempty"`
}
