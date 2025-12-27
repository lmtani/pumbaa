package chat

import (
	"google.golang.org/adk/tool"
	"google.golang.org/genai"
)

// FocusMode indicates which panel has focus
type FocusMode int

const (
	FocusInput FocusMode = iota
	FocusMessages
)

// ChatMessage represents a single message in the chat.
type ChatMessage struct {
	Role     string
	Content  string
	Rendered string // Pre-rendered markdown (cached)
}

// toolWithDefinition is an interface to access the hidden definition method of functiontool.
type toolWithDefinition interface {
	Declaration() *genai.FunctionDeclaration
	Run(ctx tool.Context, args interface{}) (map[string]interface{}, error)
}
