// Package bundle contains domain entities for WDL bundle operations.
package bundle

// Bundle represents a self-contained WDL bundle.
type Bundle struct {
	MainWorkflow string            // Path to the main workflow
	Files        map[string][]byte // All files in the bundle (relative path -> content)
	Metadata     *Metadata
}

// Metadata contains information about the bundle.
type Metadata struct {
	Version      string   `json:"version"`
	MainWorkflow string   `json:"main_workflow"`
	WDLVersion   string   `json:"wdl_version"`
	Dependencies []string `json:"dependencies"`
	TotalFiles   int      `json:"total_files"`
}

// CreateRequest represents a request to create a bundle.
type CreateRequest struct {
	MainWorkflowPath string
	OutputPath       string
	RewriteImports   bool // If true, rewrite imports to be relative within the bundle
}

// CreateResponse represents the result of creating a bundle.
type CreateResponse struct {
	OutputPath   string
	MainWorkflow string
	Dependencies []string
	TotalFiles   int
}
