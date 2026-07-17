package wdl

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

// makeZip builds an in-memory zip from name→content pairs.
func makeZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

const mainImports = `version 1.0
import "module1.wdl"
import "sub.wdl"
workflow Hello { call sub.Sub {} }
`

// sub.wdl pulls in module2.wdl — a transitive dependency of the main workflow.
const subImportsModule2 = `version 1.0
import "module2.wdl"
workflow Sub {}
`

func TestCheckDependenciesAllPresent(t *testing.T) {
	zipData := makeZip(t, map[string]string{
		"module1.wdl": "version 1.0\ntask t1 {}\n",
		"sub.wdl":     subImportsModule2,
		"module2.wdl": "version 1.0\ntask t2 {}\n",
	})

	report := CheckDependencies([]byte(mainImports), zipData)

	if !report.ZipRead {
		t.Fatal("zip should have been read")
	}
	if report.WDLFiles != 3 {
		t.Errorf("WDLFiles = %d, want 3", report.WDLFiles)
	}
	if report.HasErrors() || len(report.Findings) != 0 {
		t.Errorf("a complete bundle should pass: %+v", report.Findings)
	}
}

func TestCheckDependenciesMissingDirectImport(t *testing.T) {
	// sub.wdl is not bundled, though the main workflow imports it.
	zipData := makeZip(t, map[string]string{
		"module1.wdl": "version 1.0\ntask t1 {}\n",
	})

	report := CheckDependencies([]byte(mainImports), zipData)

	if !report.HasErrors() {
		t.Fatalf("a missing direct import must be an error: %+v", report.Findings)
	}
	if len(report.Findings) != 1 || !strings.Contains(report.Findings[0].Message, "sub.wdl") {
		t.Errorf("finding should name the missing import: %+v", report.Findings)
	}
	if !strings.Contains(report.Findings[0].Message, "the workflow imports") {
		t.Errorf("finding should attribute it to the main workflow: %q", report.Findings[0].Message)
	}
}

func TestCheckDependenciesMissingTransitiveImport(t *testing.T) {
	// The main workflow's direct imports are present, but sub.wdl needs
	// module2.wdl, which was forgotten. Cromwell would fail; preflight must
	// catch it.
	zipData := makeZip(t, map[string]string{
		"module1.wdl": "version 1.0\ntask t1 {}\n",
		"sub.wdl":     subImportsModule2,
	})

	report := CheckDependencies([]byte(mainImports), zipData)

	if !report.HasErrors() {
		t.Fatalf("a missing transitive import must be an error: %+v", report.Findings)
	}
	msg := report.Findings[0].Message
	if !strings.Contains(msg, "module2.wdl") || !strings.Contains(msg, "sub.wdl") {
		t.Errorf("finding should name both the missing file and its importer: %q", msg)
	}
}

func TestCheckDependenciesIgnoresRemoteImports(t *testing.T) {
	main := `version 1.0
import "https://example.com/lib.wdl"
import "local.wdl"
workflow W {}
`
	zipData := makeZip(t, map[string]string{"local.wdl": "version 1.0\ntask t {}\n"})

	report := CheckDependencies([]byte(main), zipData)

	if report.HasErrors() {
		t.Errorf("remote imports are left to Cromwell, not flagged: %+v", report.Findings)
	}
}

func TestCheckDependenciesMatchesByBasename(t *testing.T) {
	// The import carries a path; the bundle flattens to a basename. Matching
	// by basename must still resolve it, so a valid bundle is not blocked.
	main := `version 1.0
import "tasks/module1.wdl"
workflow W {}
`
	zipData := makeZip(t, map[string]string{"module1.wdl": "version 1.0\ntask t {}\n"})

	report := CheckDependencies([]byte(main), zipData)

	if report.HasErrors() {
		t.Errorf("a pathed import should resolve to the flattened basename: %+v", report.Findings)
	}
}

func TestCheckDependenciesUnreadableZipIsWarning(t *testing.T) {
	report := CheckDependencies([]byte(mainImports), []byte("this is not a zip"))

	if report.ZipRead {
		t.Error("an unreadable zip should be recorded as not read")
	}
	if report.HasErrors() {
		t.Errorf("an unreadable zip must not block (Cromwell decides): %+v", report.Findings)
	}
	if len(report.Findings) != 1 || report.Findings[0].Severity != SeverityWarning {
		t.Errorf("expected a single warning, got %+v", report.Findings)
	}
}

func TestCheckDependenciesReportsEachMissingOnce(t *testing.T) {
	// Two files import the same missing helper; report it once per importer,
	// not once per import line.
	main := `version 1.0
import "a.wdl"
import "a.wdl"
workflow W {}
`
	zipData := makeZip(t, map[string]string{"other.wdl": "version 1.0\ntask t {}\n"})

	report := CheckDependencies([]byte(main), zipData)

	count := 0
	for _, f := range report.Findings {
		if strings.Contains(f.Message, "a.wdl") {
			count++
		}
	}
	if count != 1 {
		t.Errorf("duplicate imports of the same file should report once, got %d: %+v", count, report.Findings)
	}
}
