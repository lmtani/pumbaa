// Package wdlindex provides domain models for WDL workflow indexing.
package wdlindex

import "time"

// Index holds the complete WDL index (also used for JSON serialization).
type Index struct {
	Version   int                         `json:"version"`
	Directory string                      `json:"directory"`
	IndexedAt time.Time                   `json:"indexed_at"`
	Tasks     map[string]*IndexedTask     `json:"tasks"`
	Workflows map[string]*IndexedWorkflow `json:"workflows"`
}

// IndexedTask represents a task in the index.
type IndexedTask struct {
	Name        string            `json:"name"`
	Source      string            `json:"source"`
	Inputs      []Declaration     `json:"inputs"`
	Outputs     []Declaration     `json:"outputs"`
	Command     string            `json:"command"`
	Runtime     map[string]string `json:"runtime,omitempty"`
	Description string            `json:"description,omitempty"`
}

// IndexedWorkflow represents a workflow in the index.
type IndexedWorkflow struct {
	Name        string        `json:"name"`
	Source      string        `json:"source"`
	Inputs      []Declaration `json:"inputs"`
	Outputs     []Declaration `json:"outputs"`
	Calls       []string      `json:"calls"`
	Description string        `json:"description,omitempty"`
}

// Declaration represents a WDL input/output declaration.
type Declaration struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Optional bool   `json:"optional"`
}

// NewIndex creates a new empty Index.
func NewIndex(directory string) *Index {
	return &Index{
		Version:   1,
		Directory: directory,
		IndexedAt: time.Now(),
		Tasks:     make(map[string]*IndexedTask),
		Workflows: make(map[string]*IndexedWorkflow),
	}
}
