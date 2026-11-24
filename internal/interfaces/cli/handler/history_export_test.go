package handler

import (
	"io"
	"strings"
	"testing"

	"github.com/urfave/cli/v2"

	"github.com/lmtani/pumbaa/internal/interfaces/cli/presenter"
)

// urfave/cli stops reading flags at the first argument, so a flag written
// after the workflow id would silently become the directory, or be dropped.
// The export must refuse instead: --strip-secrets dropped means credentials
// in a directory meant to be shared.
func TestExportRefusesFlagsAfterTheWorkflowID(t *testing.T) {
	for _, args := range [][]string{
		{"abc", "--force"},
		{"abc", "./share", "--strip-secrets"},
		{"abc", "--json"},
		{"abc", "-o", "dir"},
	} {
		err := runHistoryExport(args...)
		if err == nil || !strings.Contains(err.Error(), "put flags first") {
			t.Errorf("export %v: err = %v, want flags after the id refused", args, err)
		}
	}

	if err := runHistoryExport("abc", "one", "two"); err == nil || !strings.Contains(err.Error(), "too many arguments") {
		t.Errorf("export with three arguments: err = %v, want too many arguments", err)
	}
}

// runHistoryExport runs `pumbaa history export <args>` with no exporter
// wired: only argument validation, which runs first, can be exercised.
func runHistoryExport(args ...string) error {
	h := NewHistoryHandler(nil, nil, presenter.New(io.Discard))
	app := &cli.App{Name: "pumbaa", Commands: []*cli.Command{h.Command()}, Writer: io.Discard, ErrWriter: io.Discard}
	return app.Run(append([]string{"pumbaa", "history", "export"}, args...))
}
