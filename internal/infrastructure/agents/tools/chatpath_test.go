package tools

import (
	"testing"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/toolconfirmation"
	"google.golang.org/genai"
)

// chatNoopToolContext mirrors the chat's private noopToolContext
// (internal/interfaces/tui/chat/toolcontext.go): a minimal tool.Context for
// running function tools outside an ADK runner.
type chatNoopToolContext struct{ tool.Context }

func (chatNoopToolContext) ToolConfirmation() *toolconfirmation.ToolConfirmation { return nil }

// chatToolDefinition mirrors the private toolWithDefinition interface the
// chat uses (internal/interfaces/tui/chat/model.go) to execute tools.
type chatToolDefinition interface {
	Declaration() *genai.FunctionDeclaration
	Run(ctx tool.Context, args any) (map[string]any, error)
}

// TestChatExecutePathWDL exercises the exact call path the chat uses for a
// tool call — functiontool.Run with a nil tool.Context and raw map args —
// so a regression in this layer (not just the registry) fails loudly.
func TestChatExecutePathWDL(t *testing.T) {
	all := GetAllTools(Deps{WDLRepo: stubWDLRepo{}})

	var td chatToolDefinition
	for _, tl := range all {
		if c, ok := tl.(chatToolDefinition); ok && c.Declaration().Name == "pumbaa" {
			td = c
		}
	}
	if td == nil {
		t.Fatal("pumbaa tool does not satisfy the chat's toolWithDefinition interface")
	}

	for _, args := range []map[string]any{
		{"action": "wdl_list"},
		{"action": "wdl_search", "query": "a"},
	} {
		result, err := td.Run(chatNoopToolContext{}, args)
		if err != nil {
			t.Fatalf("Run(%v) returned error (chat would show ✗): %v", args, err)
		}
		if success, _ := result["success"].(bool); !success {
			t.Fatalf("Run(%v) output not successful: %v", args, result["error"])
		}
		t.Logf("%v ok", args)
	}
}
