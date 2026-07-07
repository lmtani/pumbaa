package tools

import (
	"context"
	"strings"
	"testing"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"

	"github.com/lmtani/pumbaa/internal/domain/wdlindex"
	"github.com/lmtani/pumbaa/internal/infrastructure/agents/tools/types"
)

// stubWDLRepo satisfies wdl.Repository for registry construction in tests.
type stubWDLRepo struct{}

func (stubWDLRepo) List() (*wdlindex.Index, error)                              { return &wdlindex.Index{}, nil }
func (stubWDLRepo) SearchTasks(string) ([]*wdlindex.IndexedTask, error)         { return nil, nil }
func (stubWDLRepo) SearchWorkflows(string) ([]*wdlindex.IndexedWorkflow, error) { return nil, nil }
func (stubWDLRepo) GetTask(string) (*wdlindex.IndexedTask, error)               { return nil, nil }
func (stubWDLRepo) GetWorkflow(string) (*wdlindex.IndexedWorkflow, error)       { return nil, nil }

func TestNewDefaultRegistryOmitsWDLActionsWithoutRepo(t *testing.T) {
	r := NewDefaultRegistry(nil, nil)

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
	r := NewDefaultRegistry(nil, stubWDLRepo{})

	for _, action := range []string{"wdl_list", "wdl_search", "wdl_info"} {
		if _, ok := r.Get(action); !ok {
			t.Errorf("expected WDL action %q to be registered", action)
		}
	}
}

func TestBuildDescriptionListsRegisteredActions(t *testing.T) {
	r := NewDefaultRegistry(nil, nil)
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
	r := NewDefaultRegistry(nil, stubWDLRepo{})
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

	all := GetAllTools(nil, nil, extra)

	if len(all) != 2 {
		t.Fatalf("expected pumbaa tool + 1 extra, got %d tools", len(all))
	}
	if all[1] != extra {
		t.Errorf("extra tool not passed through")
	}
}

func TestRegistryHandleUnknownActionListsValidOnes(t *testing.T) {
	r := NewDefaultRegistry(nil, nil)

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
