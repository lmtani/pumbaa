// Package wdl provides WDL indexing infrastructure.
package wdl

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lmtani/pumbaa/internal/domain/wdlindex"
	"github.com/lmtani/pumbaa/pkg/wdl"
	"github.com/lmtani/pumbaa/pkg/wdl/ast"
)

// Indexer indexes WDL files and implements ports.WDLRepository.
type Indexer struct {
	index     *wdlindex.Index
	indexPath string
}

// NewIndexer creates an indexer, loading from cache if available or building a new index.
func NewIndexer(directory, indexPath string, forceRebuild bool) (*Indexer, error) {
	indexer := &Indexer{indexPath: indexPath}

	// Try loading from cache if not forcing rebuild
	if !forceRebuild {
		cached, err := indexer.loadFromCache()
		if err == nil && cached != nil && cached.Directory == directory {
			indexer.index = cached
			return indexer, nil
		}
	}

	// Build new index
	if err := indexer.buildIndex(directory); err != nil {
		return nil, err
	}

	// Save to cache
	if err := indexer.saveToCache(); err != nil {
		// Non-fatal: log but continue
		fmt.Fprintf(os.Stderr, "Warning: failed to save WDL index cache: %v\n", err)
	}

	return indexer, nil
}

// buildIndex walks the directory and indexes all WDL files.
func (i *Indexer) buildIndex(directory string) error {
	absDir, err := filepath.Abs(directory)
	if err != nil {
		return fmt.Errorf("failed to resolve directory: %w", err)
	}

	i.index = wdlindex.NewIndex(absDir)

	// Track which files have been indexed (to handle imports)
	indexed := make(map[string]bool)

	// First pass: find all WDL files in directory
	var wdlFiles []string
	err = filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if !info.IsDir() && strings.HasSuffix(path, ".wdl") {
			wdlFiles = append(wdlFiles, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	// Index each file and follow imports
	for _, wdlFile := range wdlFiles {
		if err := i.indexFile(wdlFile, indexed); err != nil {
			// Log warning but continue
			fmt.Fprintf(os.Stderr, "Warning: failed to index %s: %v\n", wdlFile, err)
		}
	}

	return nil
}

// indexFile parses and indexes a single WDL file, following imports recursively.
func (i *Indexer) indexFile(filePath string, indexed map[string]bool) error {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return err
	}

	if indexed[absPath] {
		return nil // Already indexed
	}
	indexed[absPath] = true

	doc, err := wdl.Parse(absPath)
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	// Index tasks
	for _, task := range doc.Tasks {
		i.indexTask(task, absPath)
	}

	// Index workflow
	if doc.Workflow != nil {
		i.indexWorkflow(doc.Workflow, absPath)
	}

	// Follow imports recursively
	baseDir := filepath.Dir(absPath)
	for _, imp := range doc.Imports {
		importPath := i.resolveImportPath(imp.URI, baseDir)
		if importPath != "" {
			if err := i.indexFile(importPath, indexed); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to index import %s: %v\n", imp.URI, err)
			}
		}
	}

	return nil
}

// resolveImportPath resolves an import URI to an absolute path.
func (i *Indexer) resolveImportPath(uri, baseDir string) string {
	// Skip HTTP/HTTPS imports
	if strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://") {
		return ""
	}

	// Resolve relative path
	if !filepath.IsAbs(uri) {
		uri = filepath.Join(baseDir, uri)
	}

	absPath, err := filepath.Abs(uri)
	if err != nil {
		return ""
	}

	// Check if file exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return ""
	}

	return absPath
}

// indexTask extracts task information and adds to index.
func (i *Indexer) indexTask(task *ast.Task, source string) {
	indexed := &wdlindex.IndexedTask{
		Name:    task.Name,
		Source:  source,
		Command: task.Command,
		Runtime: make(map[string]string),
	}

	// Extract inputs
	for _, input := range task.Inputs {
		indexed.Inputs = append(indexed.Inputs, wdlindex.Declaration{
			Name:     input.Name,
			Type:     input.Type.String(),
			Optional: input.Type.Optional,
		})
	}

	// Extract outputs
	for _, output := range task.Outputs {
		indexed.Outputs = append(indexed.Outputs, wdlindex.Declaration{
			Name:     output.Name,
			Type:     output.Type.String(),
			Optional: output.Type.Optional,
		})
	}

	// Extract runtime as strings
	for key, expr := range task.Runtime {
		if lit, ok := expr.(*ast.Literal); ok {
			indexed.Runtime[key] = fmt.Sprintf("%v", lit.Value)
		}
	}

	// Extract description from meta
	if desc, ok := task.Meta["description"]; ok {
		indexed.Description = fmt.Sprintf("%v", desc)
	}

	i.index.Tasks[task.Name] = indexed
}

// indexWorkflow extracts workflow information and adds to index.
func (i *Indexer) indexWorkflow(wf *ast.Workflow, source string) {
	indexed := &wdlindex.IndexedWorkflow{
		Name:   wf.Name,
		Source: source,
	}

	// Extract inputs
	for _, input := range wf.Inputs {
		indexed.Inputs = append(indexed.Inputs, wdlindex.Declaration{
			Name:     input.Name,
			Type:     input.Type.String(),
			Optional: input.Type.Optional,
		})
	}

	// Extract outputs
	for _, output := range wf.Outputs {
		indexed.Outputs = append(indexed.Outputs, wdlindex.Declaration{
			Name:     output.Name,
			Type:     output.Type.String(),
			Optional: output.Type.Optional,
		})
	}

	// Extract call targets
	for _, call := range wf.Calls {
		indexed.Calls = append(indexed.Calls, call.Target)
	}

	// Extract description from meta
	if desc, ok := wf.Meta["description"]; ok {
		indexed.Description = fmt.Sprintf("%v", desc)
	}

	i.index.Workflows[wf.Name] = indexed
}

// loadFromCache loads the index from the cache file.
func (i *Indexer) loadFromCache() (*wdlindex.Index, error) {
	data, err := os.ReadFile(i.indexPath)
	if err != nil {
		return nil, err
	}

	var index wdlindex.Index
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, err
	}

	return &index, nil
}

// saveToCache saves the index to the cache file.
func (i *Indexer) saveToCache() error {
	// Ensure directory exists
	dir := filepath.Dir(i.indexPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(i.index, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(i.indexPath, data, 0644)
}

// ============================================================================
// Repository Implementation
// ============================================================================

// List returns the complete index.
func (i *Indexer) List() (*wdlindex.Index, error) {
	return i.index, nil
}

// SearchTasks finds tasks matching the query (case-insensitive).
func (i *Indexer) SearchTasks(query string) ([]*wdlindex.IndexedTask, error) {
	var results []*wdlindex.IndexedTask
	queryLower := strings.ToLower(query)

	for _, task := range i.index.Tasks {
		// Match name or command content
		if strings.Contains(strings.ToLower(task.Name), queryLower) ||
			strings.Contains(strings.ToLower(task.Command), queryLower) ||
			strings.Contains(strings.ToLower(task.Description), queryLower) {
			results = append(results, task)
		}
	}

	return results, nil
}

// SearchWorkflows finds workflows matching the query (case-insensitive).
func (i *Indexer) SearchWorkflows(query string) ([]*wdlindex.IndexedWorkflow, error) {
	var results []*wdlindex.IndexedWorkflow
	queryLower := strings.ToLower(query)

	for _, wf := range i.index.Workflows {
		// Match name, calls, or description
		if strings.Contains(strings.ToLower(wf.Name), queryLower) ||
			strings.Contains(strings.ToLower(wf.Description), queryLower) {
			results = append(results, wf)
			continue
		}
		// Check calls
		for _, call := range wf.Calls {
			if strings.Contains(strings.ToLower(call), queryLower) {
				results = append(results, wf)
				break
			}
		}
	}

	return results, nil
}

// GetTask returns a specific task by name.
func (i *Indexer) GetTask(name string) (*wdlindex.IndexedTask, error) {
	// Case-insensitive lookup
	nameLower := strings.ToLower(name)
	for taskName, task := range i.index.Tasks {
		if strings.ToLower(taskName) == nameLower {
			return task, nil
		}
	}
	return nil, fmt.Errorf("task not found: %s", name)
}

// GetWorkflow returns a specific workflow by name.
func (i *Indexer) GetWorkflow(name string) (*wdlindex.IndexedWorkflow, error) {
	// Case-insensitive lookup
	nameLower := strings.ToLower(name)
	for wfName, wf := range i.index.Workflows {
		if strings.ToLower(wfName) == nameLower {
			return wf, nil
		}
	}
	return nil, fmt.Errorf("workflow not found: %s", name)
}
