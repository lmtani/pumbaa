// Package tools provides implementations of tools for use with Google Agents ADK.
//
// This package follows the Registry pattern to allow easy extension with new actions.
// To add a new action:
//
//  1. Create a new handler struct in the appropriate subpackage (cromwell/, gcs/, wdl/)
//  2. Implement the ActionHandler interface
//  3. Register the handler in NewDefaultRegistry() in factory.go
//  4. Update GetParametersSchema() in schema.go if the action needs new parameters
//
// Example handler:
//
//	type MyHandler struct { /* dependencies */ }
//
//	func NewMyHandler() *MyHandler { return &MyHandler{} }
//
//	func (h *MyHandler) Handle(ctx context.Context, input tools.PumbaaInput) (tools.PumbaaOutput, error) {
//	    // implementation
//	    return tools.NewSuccessOutput("my_action", data), nil
//	}
package tools
