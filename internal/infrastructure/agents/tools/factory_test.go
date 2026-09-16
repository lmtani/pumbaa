package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"

	"github.com/lmtani/pumbaa/internal/domain/wdlindex"
	"github.com/lmtani/pumbaa/internal/infrastructure/agents/tools/types"

	"github.com/lmtani/pumbaa/internal/application/ports"
)

// stubWDLRepo satisfies wdl.Repository for registry construction in tests.
type stubWDLRepo struct{}

func (stubWDLRepo) List() (*wdlindex.Index, error)                              { return &wdlindex.Index{}, nil }
func (stubWDLRepo) SearchTasks(string) ([]*wdlindex.IndexedTask, error)         { return nil, nil }
func (stubWDLRepo) SearchWorkflows(string) ([]*wdlindex.IndexedWorkflow, error) { return nil, nil }
func (stubWDLRepo) GetTask(string) (*wdlindex.IndexedTask, error)               { return nil, nil }
func (stubWDLRepo) GetWorkflow(string) (*wdlindex.IndexedWorkflow, error)       { return nil, nil }

func TestNewDefaultRegistryOmitsWDLActionsWithoutRepo(t *testing.T) {
	r := NewDefaultRegistry(Deps{})

	for _, action := range []string{"query", "status", "metadata", "outputs", "logs", "gcs_download"} {
		if _, ok := r.Get(action); !ok {
			t.Errorf("expected action %q to be registered", action)
		}
	}
	for _, action := range []string{"wdl_list", "wdl_search", "wdl_info"} {
		if _, ok := r.Get(action); ok {
			t.Errorf("expected WDL action %q to be omitted without a repository", action)
		}
	}
}

func TestNewDefaultRegistryIncludesWDLActionsWithRepo(t *testing.T) {
	r := NewDefaultRegistry(Deps{WDLRepo: stubWDLRepo{}})

	for _, action := range []string{"wdl_list", "wdl_search", "wdl_info"} {
		if _, ok := r.Get(action); !ok {
			t.Errorf("expected WDL action %q to be registered", action)
		}
	}
}

func TestBuildDescriptionListsRegisteredActions(t *testing.T) {
	r := NewDefaultRegistry(Deps{})
	r.Register("custom_action", "Does something custom. Required: foo.", types.HandlerFunc(
		func(ctx context.Context, input types.Input) (types.Output, error) {
			return types.Output{}, nil
		},
	))

	desc := buildDescription(r)

	for _, want := range []string{`"query"`, `"custom_action"`, "Does something custom. Required: foo."} {
		if !strings.Contains(desc, want) {
			t.Errorf("description missing %q:\n%s", want, desc)
		}
	}
	if strings.Contains(desc, "wdl_list") {
		t.Errorf("description should not document unregistered WDL actions:\n%s", desc)
	}
}

func TestSchemaEnumCoversAllRegisteredActions(t *testing.T) {
	enum := builtinActionNames()
	inEnum := make(map[string]bool, len(enum))
	for _, name := range enum {
		inEnum[name] = true
	}

	// Every action a fully-configured registry exposes must be in the enum,
	// otherwise providers using the explicit schema (Ollama) reject it.
	r := NewDefaultRegistry(Deps{WDLRepo: stubWDLRepo{}})
	for _, action := range r.Actions() {
		if !inEnum[action] {
			t.Errorf("action %q registered but missing from schema enum", action)
		}
	}
}

func TestGetAllToolsAppendsExtras(t *testing.T) {
	extra, err := functiontool.New(
		functiontool.Config{Name: "my_tool", Description: "test tool"},
		func(ctx tool.Context, input struct{}) (map[string]any, error) {
			return nil, nil
		},
	)
	if err != nil {
		t.Fatalf("failed to build extra tool: %v", err)
	}

	all := GetAllTools(Deps{}, extra)

	if len(all) != 2 {
		t.Fatalf("expected pumbaa tool + 1 extra, got %d tools", len(all))
	}
	if all[1] != extra {
		t.Errorf("extra tool not passed through")
	}
}

func TestRegistryHandleUnknownActionListsValidOnes(t *testing.T) {
	r := NewDefaultRegistry(Deps{})

	out, err := r.Handle(context.Background(), types.Input{Action: "nope"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Success {
		t.Fatalf("expected error output for unknown action")
	}
	if !strings.Contains(out.Error, "query") {
		t.Errorf("error should list valid actions, got: %s", out.Error)
	}
}

// stubFileProvider is a no-op ports.FileProvider for registry wiring tests.
type stubFileProvider struct{}

func (stubFileProvider) Read(context.Context, string) (string, error)      { return "", nil }
func (stubFileProvider) ReadBytes(context.Context, string) ([]byte, error) { return nil, nil }
func (stubFileProvider) GetSize(context.Context, string) (int64, error)    { return 0, nil }
func (stubFileProvider) GetContentDigests(context.Context, string) (ports.FileDigests, error) {
	return ports.FileDigests{}, nil
}

func TestScaffoldAlwaysAvailablePreflightNeedsFileProvider(t *testing.T) {
	// scaffold reads a local WDL and needs nothing else, so it is always on.
	bare := NewDefaultRegistry(Deps{})
	if _, ok := bare.Get("scaffold"); !ok {
		t.Error("scaffold should be registered without any dependency")
	}
	if _, ok := bare.Get("preflight"); ok {
		t.Error("preflight should be omitted without a file provider")
	}

	withFP := NewDefaultRegistry(Deps{FileProvider: stubFileProvider{}})
	if _, ok := withFP.Get("preflight"); !ok {
		t.Error("preflight should be registered when a file provider is wired")
	}
}

func TestScaffoldActionDispatchesThroughRegistry(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })

	wdlSrc := "version 1.0\n\nworkflow W {\n  input {\n    File x\n  }\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "w.wdl"), []byte(wdlSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	r := NewDefaultRegistry(Deps{})
	out, err := r.Handle(context.Background(), types.Input{Action: "scaffold", WorkflowFile: "w.wdl"})
	if err != nil || !out.Success {
		t.Fatalf("scaffold through the registry failed: err=%v out=%+v", err, out)
	}
	if data, ok := out.Data.(map[string]any); !ok || data["workflow"] != "W" {
		t.Errorf("unexpected scaffold output: %+v", out.Data)
	}
}

// stubRunHistory satisfies ports.RunHistoryReader for registry construction.
type stubRunHistory struct{}

func (stubRunHistory) Get(context.Context, ports.RunRef) (ports.RunRecord, error) {
	return ports.RunRecord{}, ports.ErrRunNotRemembered
}

func (stubRunHistory) List(context.Context, ports.RunHistoryFilter) ([]ports.RunRecord, error) {
	return nil, nil
}

func (stubRunHistory) Lookup(context.Context, string, []string) (map[string]ports.RunRecord, error) {
	return nil, nil
}

type stubHostProvider struct{}

func (stubHostProvider) CurrentHost() (string, string) { return "http://localhost:8000", "local" }

func TestHistoryActionNeedsBothStoreAndHost(t *testing.T) {
	if _, ok := NewDefaultRegistry(Deps{}).Get("history"); ok {
		t.Error("history should be omitted without a local store")
	}
	if _, ok := NewDefaultRegistry(Deps{History: stubRunHistory{}}).Get("history"); ok {
		t.Error("history should be omitted without a host to key records by")
	}

	full := NewDefaultRegistry(Deps{History: stubRunHistory{}, Host: stubHostProvider{}})
	if _, ok := full.Get("history"); !ok {
		t.Error("history should be registered once the store and host are wired")
	}
}

func TestHistoryActionDispatchesThroughRegistry(t *testing.T) {
	r := NewDefaultRegistry(Deps{History: stubRunHistory{}, Host: stubHostProvider{}})

	out, err := r.Handle(context.Background(), types.Input{Action: "history", WorkflowID: "ghost"})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if out.Action != "history" {
		t.Errorf("action = %q, want history", out.Action)
	}
	if out.Success {
		t.Error("Success = true for a run with no local record, want the absence reported")
	}
}
