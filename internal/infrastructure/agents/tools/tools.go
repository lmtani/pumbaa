// Package tools provides implementations of tools for use with Google Agents ADK.
//
// There are two extension points for giving the chat agent new capabilities:
//
// # Adding an action to the pumbaa tool
//
// Actions are sub-commands of the unified "pumbaa" tool, dispatched through
// the Registry. The builtinActions table in factory.go is the single source
// of truth: registration, the LLM-facing tool description and the
// parameters-schema enum all derive from it. To add an action:
//
//  1. Create a handler in the appropriate subpackage (cromwell/, gcs/, wdl/)
//     implementing types.Handler
//  2. Add one entry to builtinActions() in factory.go with the action name
//     and a description stating its required/optional parameters
//  3. Only if the action introduces new parameters: add the property to
//     GetParametersSchema() in schema.go
//
// Callers building a custom registry can also register ad-hoc actions:
//
//	registry := tools.NewDefaultRegistry(repo, nil)
//	registry.Register("my_action", "Does something. Required: foo.", myHandler)
//	agentTools := []tool.Tool{tools.GetPumbaaTool(registry)}
//
// # Adding a standalone ADK tool
//
// Independent tools (their own name and schema, not a pumbaa action) are
// passed through GetAllTools' variadic parameter and flow into the chat
// agent untouched:
//
//	myTool, _ := functiontool.New(functiontool.Config{Name: "my_tool", ...}, run)
//	agentTools := tools.GetAllTools(repo, nil, myTool)
//
// Example handler:
//
//	type MyHandler struct { /* dependencies */ }
//
//	func NewMyHandler() *MyHandler { return &MyHandler{} }
//
//	func (h *MyHandler) Handle(ctx context.Context, input types.Input) (types.Output, error) {
//	    // implementation
//	    return types.NewSuccessOutput("my_action", data), nil
//	}
package tools
