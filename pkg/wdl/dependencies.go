package wdl

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
)

// DependencyReport is the result of checking a workflow's imports against a
// dependencies zip.
type DependencyReport struct {
	// ZipRead reports whether the zip could be read. When false, only a
	// warning is present and the check is inconclusive.
	ZipRead bool
	// WDLFiles is the number of .wdl entries found in the zip.
	WDLFiles int
	Findings []Finding
}

// HasErrors reports whether any import fails to resolve.
func (r *DependencyReport) HasErrors() bool {
	for _, f := range r.Findings {
		if f.Severity == SeverityError {
			return true
		}
	}
	return false
}

// CheckDependencies verifies that every WDL import — in the main workflow and,
// transitively, in each WDL inside the dependencies zip — resolves to a file
// present in the zip. It is what catches "you forgot to bundle a file" before
// Cromwell fails to parse the workflow.
//
// Matching is by basename, the convention `pumbaa bundle` produces and the
// most lenient one: an import resolves as long as a file with that name is in
// the zip, regardless of directory layout. This keeps the check from blocking
// a valid submission over a structural difference, while still catching a
// genuinely missing file. Remote (http/https) imports are left to Cromwell.
//
// It performs no IO: the zip is passed as bytes.
func CheckDependencies(mainSource, zipData []byte) *DependencyReport {
	report := &DependencyReport{}

	entries, err := readZipWDL(zipData)
	if err != nil {
		report.Findings = append(report.Findings, Finding{
			Severity: SeverityWarning,
			Message:  fmt.Sprintf("could not read the dependencies zip, so imports were not checked: %v", err),
		})
		return report
	}
	report.ZipRead = true
	report.WDLFiles = len(entries)

	available := make(map[string]bool, len(entries))
	for name := range entries {
		available[filepath.Base(name)] = true
	}

	// The main WDL is submitted alongside the zip; its imports resolve
	// against the zip's contents.
	report.Findings = append(report.Findings, unresolvedImports("the workflow", mainSource, available)...)

	// Each bundled WDL may import other bundled files (transitive deps).
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		report.Findings = append(report.Findings, unresolvedImports(filepath.Base(name), entries[name], available)...)
	}

	return report
}

// unresolvedImports returns a finding for each non-remote import in source
// whose target is not among the available files.
func unresolvedImports(importer string, source []byte, available map[string]bool) []Finding {
	var findings []Finding
	seen := make(map[string]bool)
	for _, imp := range extractImportPaths(source) {
		if isRemoteImport(imp) {
			continue
		}
		base := filepath.Base(imp)
		if available[base] || seen[base] {
			continue
		}
		seen[base] = true
		findings = append(findings, Finding{
			Severity: SeverityError,
			Message:  fmt.Sprintf("%s imports %q, which is not in the dependencies zip", importer, imp),
		})
	}
	return findings
}

// extractImportPaths pulls the import targets out of WDL source with the same
// regex the bundler uses, so it is robust to WDL this parser cannot fully read.
func extractImportPaths(source []byte) []string {
	matches := importRegex.FindAllStringSubmatch(string(source), -1)
	paths := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) >= 2 {
			paths = append(paths, m[1])
		}
	}
	return paths
}

// isRemoteImport reports whether an import points at a URL rather than a file
// expected in the zip.
func isRemoteImport(path string) bool {
	return strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://")
}

// readZipWDL reads the .wdl entries of a zip into memory, keyed by their name
// in the archive.
func readZipWDL(data []byte) (map[string][]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	entries := make(map[string][]byte)
	for _, f := range zr.File {
		if f.FileInfo().IsDir() || !strings.EqualFold(filepath.Ext(f.Name), ".wdl") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("could not open %s in the zip: %w", f.Name, err)
		}
		content, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			return nil, fmt.Errorf("could not read %s in the zip: %w", f.Name, err)
		}
		entries[f.Name] = content
	}
	return entries, nil
}
