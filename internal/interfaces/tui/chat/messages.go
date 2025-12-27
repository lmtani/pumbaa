package chat

// ResponseMsg is sent when the LLM responds.
type ResponseMsg struct {
	Content      string
	Err          error
	InputTokens  int // Input tokens used in this response
	OutputTokens int // Output tokens generated in this response
}

// ToolNotificationMsg is sent when a tool is being called.
type ToolNotificationMsg struct {
	ToolName string
	Action   string
	Params   map[string]interface{} // Additional parameters
}

// ClearNotificationMsg is sent to clear the tool notification.
type ClearNotificationMsg struct{}

// clipboardCopiedMsg is sent when clipboard copy completes.
type clipboardCopiedMsg struct {
	success bool
	err     error
}

// clearStatusMsg is sent to clear the status message.
type clearStatusMsg struct{}
