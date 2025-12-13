package wdl

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/lmtani/pumbaa/pkg/wdl/ast"
)

// DependencyGraph represents the complete dependency graph for a WDL document
type DependencyGraph struct {
	Root         string                     // Path to the root WDL file
	Dependencies map[string]*DependencyNode // All dependencies indexed by resolved path
	Imports      []string                   // Ordered list of all imports (topological order)
}

// DependencyNode represents a single node in the dependency graph
type DependencyNode struct {
	Path         string            // Resolved absolute path
	Document     *ast.Document     // Parsed AST
	DirectDeps   []string          // Direct dependencies (resolved paths)
	ImportedBy   []string          // Files that import this file
	Aliases      map[string]string // Import aliases (alias -> original)
	ResolvedFrom string            // The import URI as written in the source
}

// AnalyzeDependencies analyzes a WDL document and builds a complete dependency graph
// It resolves all imports (direct and transitive) and detects circular dependencies
func AnalyzeDependencies(doc *ast.Document, sourcePath string) (*DependencyGraph, error) {
	analyzer := newDependencyAnalyzer()
	return analyzer.analyze(doc, sourcePath)
}

// AnalyzeDependenciesFromFile parses a WDL file and analyzes its dependencies
func AnalyzeDependenciesFromFile(filePath string) (*DependencyGraph, error) {
	doc, err := Parse(filePath)
	if err != nil {
		return nil, err
	}
	return AnalyzeDependencies(doc, filePath)
}

// dependencyAnalyzer handles dependency analysis with caching
type dependencyAnalyzer struct {
	cache   map[string]*DependencyNode // Cache of parsed documents
	visited map[string]bool            // Tracks visited files during current traversal
	stack   []string                   // Stack for cycle detection
}

func newDependencyAnalyzer() *dependencyAnalyzer {
	return &dependencyAnalyzer{
		cache:   make(map[string]*DependencyNode),
		visited: make(map[string]bool),
		stack:   make([]string, 0),
	}
}

func (a *dependencyAnalyzer) analyze(doc *ast.Document, sourcePath string) (*DependencyGraph, error) {
	absPath, err := filepath.Abs(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	graph := &DependencyGraph{
		Root:         absPath,
		Dependencies: make(map[string]*DependencyNode),
		Imports:      make([]string, 0),
	}

	// Create root node
	rootNode := &DependencyNode{
		Path:       absPath,
		Document:   doc,
		DirectDeps: make([]string, 0),
		ImportedBy: make([]string, 0),
		Aliases:    make(map[string]string),
	}

	a.cache[absPath] = rootNode
	graph.Dependencies[absPath] = rootNode

	// Analyze imports recursively
	err = a.analyzeImports(rootNode, graph, filepath.Dir(absPath))
	if err != nil {
		return nil, err
	}

	// Build topologically sorted list of imports
	graph.Imports, err = a.topologicalSort(graph)
	if err != nil {
		return nil, err
	}

	return graph, nil
}

func (a *dependencyAnalyzer) analyzeImports(node *DependencyNode, graph *DependencyGraph, baseDir string) error {
	// Check for circular dependency
	for _, stackPath := range a.stack {
		if stackPath == node.Path {
			cycle := append(a.stack, node.Path)
			return fmt.Errorf("circular dependency detected: %s", strings.Join(cycle, " -> "))
		}
	}

	// Add to stack for cycle detection
	a.stack = append(a.stack, node.Path)
	defer func() {
		a.stack = a.stack[:len(a.stack)-1]
	}()

	if node.Document == nil {
		return nil
	}

	for _, imp := range node.Document.Imports {
		resolvedPath, err := a.resolveImportPath(imp.URI, baseDir)
		if err != nil {
			return fmt.Errorf("failed to resolve import %s: %w", imp.URI, err)
		}

		node.DirectDeps = append(node.DirectDeps, resolvedPath)

		// Store aliases
		if imp.As != "" {
			node.Aliases[imp.As] = filepath.Base(resolvedPath)
		}
		for _, alias := range imp.Aliases {
			node.Aliases[alias.Alias] = alias.Original
		}

		// Check if already processed
		if existingNode, ok := graph.Dependencies[resolvedPath]; ok {
			existingNode.ImportedBy = append(existingNode.ImportedBy, node.Path)
			continue
		}

		// Parse the imported file
		impDoc, err := Parse(resolvedPath)
		if err != nil {
			return fmt.Errorf("failed to parse import %s: %w", resolvedPath, err)
		}

		impNode := &DependencyNode{
			Path:         resolvedPath,
			Document:     impDoc,
			DirectDeps:   make([]string, 0),
			ImportedBy:   []string{node.Path},
			Aliases:      make(map[string]string),
			ResolvedFrom: imp.URI,
		}

		a.cache[resolvedPath] = impNode
		graph.Dependencies[resolvedPath] = impNode

		// Recursively analyze imports
		err = a.analyzeImports(impNode, graph, filepath.Dir(resolvedPath))
		if err != nil {
			return err
		}
	}

	return nil
}

// resolveImportPath resolves an import URI to an absolute path
func (a *dependencyAnalyzer) resolveImportPath(uri string, baseDir string) (string, error) {
	// Check if it's a URL
	if strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://") {
		// For now, we don't support HTTP imports
		// This could be extended to download and cache remote files
		return "", fmt.Errorf("HTTP imports not yet supported: %s", uri)
	}

	// Check if it's a file:// URL
	if strings.HasPrefix(uri, "file://") {
		parsedURL, err := url.Parse(uri)
		if err != nil {
			return "", err
		}
		uri = parsedURL.Path
	}

	// Handle relative paths
	if !filepath.IsAbs(uri) {
		uri = filepath.Join(baseDir, uri)
	}

	// Clean and make absolute
	absPath, err := filepath.Abs(uri)
	if err != nil {
		return "", err
	}

	return filepath.Clean(absPath), nil
}

// topologicalSort returns imports in topological order (dependencies first)
func (a *dependencyAnalyzer) topologicalSort(graph *DependencyGraph) ([]string, error) {
	visited := make(map[string]bool)
	tempMark := make(map[string]bool)
	result := make([]string, 0)

	var visit func(path string) error
	visit = func(path string) error {
		if tempMark[path] {
			return fmt.Errorf("circular dependency detected involving %s", path)
		}
		if visited[path] {
			return nil
		}

		tempMark[path] = true

		node := graph.Dependencies[path]
		if node != nil {
			for _, dep := range node.DirectDeps {
				if err := visit(dep); err != nil {
					return err
				}
			}
		}

		tempMark[path] = false
		visited[path] = true

		// Don't include root in imports list
		if path != graph.Root {
			result = append(result, path)
		}

		return nil
	}

	// Start from root
	if err := visit(graph.Root); err != nil {
		return nil, err
	}

	return result, nil
}

// GetAllDependencies returns all dependencies including transitive ones
func (g *DependencyGraph) GetAllDependencies() []string {
	return g.Imports
}

// GetDirectDependencies returns only direct dependencies of the root document
func (g *DependencyGraph) GetDirectDependencies() []string {
	if root, ok := g.Dependencies[g.Root]; ok {
		return root.DirectDeps
	}
	return nil
}

// HasCyclicDependency checks if there are any circular dependencies
// This is already checked during analysis, but this method can be used for explicit verification
func (g *DependencyGraph) HasCyclicDependency() bool {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var hasCycle func(path string) bool
	hasCycle = func(path string) bool {
		visited[path] = true
		recStack[path] = true

		node := g.Dependencies[path]
		if node != nil {
			for _, dep := range node.DirectDeps {
				if !visited[dep] {
					if hasCycle(dep) {
						return true
					}
				} else if recStack[dep] {
					return true
				}
			}
		}

		recStack[path] = false
		return false
	}

	return hasCycle(g.Root)
}

// PrintGraph prints a human-readable representation of the dependency graph
func (g *DependencyGraph) PrintGraph() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Root: %s\n", g.Root))
	sb.WriteString("Dependencies:\n")

	for _, path := range g.Imports {
		node := g.Dependencies[path]
		relPath, _ := filepath.Rel(filepath.Dir(g.Root), path)
		sb.WriteString(fmt.Sprintf("  - %s\n", relPath))
		if len(node.DirectDeps) > 0 {
			sb.WriteString("    imports:\n")
			for _, dep := range node.DirectDeps {
				depRel, _ := filepath.Rel(filepath.Dir(g.Root), dep)
				sb.WriteString(fmt.Sprintf("      - %s\n", depRel))
			}
		}
	}

	return sb.String()
}
