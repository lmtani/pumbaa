package chat

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool"
	"google.golang.org/genai"
)

// generateResponse generates a response from the LLM.
func (m Model) generateResponse(input string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		userContent := &genai.Content{
			Role: "user",
			Parts: []*genai.Part{
				genai.NewPartFromText(input),
			},
		}

		*m.history = append(*m.history, userContent)

		if m.sessionService != nil && m.session != nil {
			ev := session.NewEvent("")
			ev.Content = userContent
			ev.Author = "user"
			m.sessionService.AppendEvent(ctx, m.session, ev)
		}

		maxTurns := 15
		currentTurn := 0
		totalInputTokens := 0
		totalOutputTokens := 0

		for currentTurn < maxTurns {
			req := &model.LLMRequest{
				Contents: *m.history,
				Config: &genai.GenerateContentConfig{
					Tools: convertToolsToGenAI(m.tools),
				},
			}

			if m.systemInstruction != "" {
				req.Config.SystemInstruction = &genai.Content{
					Parts: []*genai.Part{
						genai.NewPartFromText(m.systemInstruction),
					},
				}
			}

			respSeq := m.llm.GenerateContent(ctx, req, false)

			var lastResp *model.LLMResponse

			for r, e := range respSeq {
				if e != nil {
					return ResponseMsg{Err: e}
				}
				lastResp = r
			}

			if lastResp == nil || lastResp.Content == nil {
				return ResponseMsg{Err: fmt.Errorf("empty response from model")}
			}

			// Accumulate token usage from this response
			if lastResp.UsageMetadata != nil {
				totalInputTokens += int(lastResp.UsageMetadata.PromptTokenCount)
				totalOutputTokens += int(lastResp.UsageMetadata.CandidatesTokenCount)
			}

			*m.history = append(*m.history, lastResp.Content)

			if m.sessionService != nil && m.session != nil {
				ev := session.NewEvent("")
				ev.Content = lastResp.Content
				ev.Author = "model"
				m.sessionService.AppendEvent(ctx, m.session, ev)
			}

			toolCalls := getToolCalls(lastResp.Content)
			if len(toolCalls) > 0 {
				var toolParts []*genai.Part

				for _, tc := range toolCalls {
					// Extract action from tool args if available
					action := ""
					if tc.Args != nil {
						if actionVal, ok := tc.Args["action"]; ok {
							if actionStr, ok := actionVal.(string); ok {
								action = actionStr
							}
						}
					}

					// Collect other relevant params to show
					otherParams := make(map[string]interface{})
					for k, v := range tc.Args {
						if k != "action" && v != nil && v != "" {
							otherParams[k] = v
						}
					}

					// Send notification to UI about tool being called
					if m.program != nil {
						m.program.Send(ToolNotificationMsg{ToolName: tc.Name, Action: action, Params: otherParams})
					}

					result, err := m.executeTool(ctx, tc)
					if err != nil {
						toolParts = append(toolParts, &genai.Part{
							FunctionResponse: &genai.FunctionResponse{
								Name: tc.Name,
								Response: map[string]interface{}{
									"error": err.Error(),
								},
							},
						})
					} else {
						toolParts = append(toolParts, &genai.Part{
							FunctionResponse: &genai.FunctionResponse{
								Name:     tc.Name,
								Response: result,
							},
						})
					}
				}

				toolContent := &genai.Content{
					Role:  "tool",
					Parts: toolParts,
				}

				*m.history = append(*m.history, toolContent)

				if m.sessionService != nil && m.session != nil {
					ev := session.NewEvent("")
					ev.Content = toolContent
					ev.Author = "tool"
					m.sessionService.AppendEvent(ctx, m.session, ev)
				}

				currentTurn++
				continue
			}

			text := ""
			for _, part := range lastResp.Content.Parts {
				if part.Text != "" {
					text += part.Text
				}
			}
			return ResponseMsg{Content: text, InputTokens: totalInputTokens, OutputTokens: totalOutputTokens}
		}

		// Max turns reached
		var summary strings.Builder
		summary.WriteString("⚠️ I reached the tool iteration limit.\n\n")
		summary.WriteString("**Information collected so far:**\n")

		toolResultCount := 0
		for _, content := range *m.history {
			if content.Role == "tool" {
				toolResultCount++
			}
		}

		if toolResultCount > 0 {
			summary.WriteString(fmt.Sprintf("- Executed %d tool calls\n", toolResultCount))
		}

		summary.WriteString("\nIf you need more information, please ask a more specific question or ask me to continue from where I left off.")

		return ResponseMsg{Content: summary.String(), InputTokens: totalInputTokens, OutputTokens: totalOutputTokens}
	}
}

// getToolCalls extracts function calls from content.
func getToolCalls(content *genai.Content) []*genai.FunctionCall {
	var calls []*genai.FunctionCall
	for _, part := range content.Parts {
		if part.FunctionCall != nil {
			calls = append(calls, part.FunctionCall)
		}
	}
	return calls
}

// executeTool runs a tool by name.
func (m Model) executeTool(ctx context.Context, fc *genai.FunctionCall) (map[string]any, error) {
	for _, t := range m.tools {
		if td, ok := t.(toolWithDefinition); ok {
			def := td.Declaration()
			if def.Name == fc.Name {
				return td.Run(nil, fc.Args)
			}
		}
	}
	return nil, fmt.Errorf("tool not found: %s", fc.Name)
}

// convertToolsToGenAI converts tools to genai format.
func convertToolsToGenAI(tools []tool.Tool) []*genai.Tool {
	var genaiTools []*genai.Tool
	for _, t := range tools {
		if td, ok := t.(toolWithDefinition); ok {
			genaiTools = append(genaiTools, &genai.Tool{
				FunctionDeclarations: []*genai.FunctionDeclaration{
					td.Declaration(),
				},
			})
		}
	}
	return genaiTools
}
