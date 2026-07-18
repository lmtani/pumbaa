package wdl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lmtani/pumbaa/pkg/wdl/ast"
)

// maxImportDepth bounds import and subworkflow recursion. Valid WDL cannot be
// cyclic, but a hand-edited or partially-resolved set can be, and the walk must
// terminate either way.
const maxImportDepth = 16

// SourceSet holds the WDL sources available to resolve imports, keyed by file
// basename.
//
// Basename matching is the same convention `pumbaa bundle` produces and
// CheckDependencies validates: an import resolves as long as a file with that
// name is present, whatever the directory layout. It is the most lenient rule
// that still resolves, and matches how Cromwell consumes a dependencies zip.
type SourceSet map[string][]byte

// Add registers a source under its basename.
func (s SourceSet) Add(path string, content []byte) {
	s[filepath.Base(path)] = content
}

// Get resolves an import URI to a source. Remote imports (http/https) are not
// fetched — Cromwell resolves those itself.
func (s SourceSet) Get(uri string) ([]byte, bool) {
	if strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://") {
		return nil, false
	}
	content, ok := s[filepath.Base(uri)]
	return content, ok
}

// SourcesFromZip builds a SourceSet from a dependencies zip. It performs no IO
// beyond decoding the bytes it is given.
func SourcesFromZip(zipData []byte) (SourceSet, error) {
	entries, err := readZipWDL(zipData)
	if err != nil {
		return nil, err
	}
	out := make(SourceSet, len(entries))
	for name, content := range entries {
		out.Add(name, content)
	}
	return out, nil
}

// SourcesFromDir collects every .wdl file under dir, so a workflow run from a
// checkout resolves its imports without having to be bundled first. Later
// files win on a basename collision, matching the zip's leniency.
func SourcesFromDir(dir string) (SourceSet, error) {
	out := make(SourceSet)
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(path), ".wdl") {
			return nil
		}
		content, readErr := os.ReadFile(path) //nolint:gosec // paths come from the caller's own tree
		if readErr != nil {
			return readErr
		}
		out.Add(path, content)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("collecting WDL sources from %s: %w", dir, err)
	}
	return out, nil
}

// documentSet parses sources on demand and caches them, so a diamond of
// imports parses each file once.
type documentSet struct {
	sources SourceSet
	parsed  map[string]*ast.Document
}

func newDocumentSet(sources SourceSet) *documentSet {
	if sources == nil {
		sources = SourceSet{}
	}
	return &documentSet{sources: sources, parsed: make(map[string]*ast.Document)}
}

// document resolves an import URI to a parsed document. It reports false when
// the source is absent or does not parse; callers must degrade rather than
// assume the import is empty.
func (d *documentSet) document(uri string) (*ast.Document, bool) {
	key := filepath.Base(uri)
	if doc, ok := d.parsed[key]; ok {
		return doc, doc != nil
	}
	content, ok := d.sources.Get(uri)
	if !ok {
		d.parsed[key] = nil
		return nil, false
	}
	doc, err := ParseBytes(content)
	if err != nil {
		d.parsed[key] = nil
		return nil, false
	}
	d.parsed[key] = doc
	return doc, true
}

// namespaces maps the names a document addresses its imports by to the import
// URI. WDL uses the alias when one is given, otherwise the file's basename
// without the extension.
func namespaces(doc *ast.Document) map[string]string {
	out := make(map[string]string, len(doc.Imports))
	for _, imp := range doc.Imports {
		if imp == nil {
			continue
		}
		ns := imp.As
		if ns == "" {
			base := filepath.Base(imp.URI)
			ns = strings.TrimSuffix(base, filepath.Ext(base))
		}
		out[ns] = imp.URI
	}
	return out
}

// splitTarget separates a call target into its namespace and name:
// "lib.AlignReads" → ("lib", "AlignReads"), "AlignReads" → ("", "AlignReads").
func splitTarget(target string) (namespace, name string) {
	if i := strings.LastIndex(target, "."); i >= 0 {
		return target[:i], target[i+1:]
	}
	return "", target
}
