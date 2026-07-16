package chat

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool"
	"google.golang.org/genai"

	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// The chat drives its own agent loop: generateResponse streams partials
// from the LLM, executes tool calls and pushes results into the Update loop
// via tea.Program.Send.
func (m Model) generateResponse(ctx context.Context, input string) tea.Cmd {
	return func() tea.Msg {
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
			_ = m.sessionService.AppendEvent(ctx, m.session, ev)
		}

		maxTurns := 15
		currentTurn := 0
		totalInputTokens := 0
		totalOutputTokens := 0

		for currentTurn < maxTurns {
			if ctx.Err() != nil {
				return ResponseMsg{Err: ctx.Err(), owner: m.msgs}
			}

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

			respSeq := m.llm.GenerateContent(ctx, req, true)

			var lastResp *model.LLMResponse
			var turnText strings.Builder

			for r, e := range respSeq {
				if e != nil {
					return ResponseMsg{Err: e, owner: m.msgs}
				}
				if r.Partial {
					// Forward the accumulated turn text so the UI renders
					// the response as it streams.
					turnText.WriteString(extractText(r.Content))
					if m.program != nil {
						m.program.Send(streamChunkMsg{owner: m.msgs, text: turnText.String()})
					}
					continue
				}
				lastResp = r
			}

			if lastResp == nil || lastResp.Content == nil {
				return ResponseMsg{Err: fmt.Errorf("empty response from model"), owner: m.msgs}
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
				_ = m.sessionService.AppendEvent(ctx, m.session, ev)
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
					otherParams := make(map[string]any)
					for k, v := range tc.Args {
						if k != "action" && v != nil && v != "" {
							otherParams[k] = v
						}
					}

					// Send notification to UI about tool being called
					if m.program != nil {
						m.program.Send(ToolNotificationMsg{ToolName: tc.Name, Action: action, Params: otherParams, owner: m.msgs})
					}

					toolStart := time.Now()
					result, err := m.executeTool(ctx, tc)
					if m.program != nil {
						// Persistent transcript record of what the agent did
						m.program.Send(toolRecordMsg{
							owner: m.msgs,
							line:  formatToolRecord(tc.Name, action, otherParams, time.Since(toolStart), toolFailure(result, err)),
						})
					}
					if err != nil {
						toolParts = append(toolParts, &genai.Part{
							FunctionResponse: &genai.FunctionResponse{
								Name: tc.Name,
								Response: map[string]any{
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
					_ = m.sessionService.AppendEvent(ctx, m.session, ev)
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
			return ResponseMsg{Content: text, InputTokens: totalInputTokens, OutputTokens: totalOutputTokens, owner: m.msgs}
		}

		// Max turns reached
		var summary strings.Builder
		summary.WriteString("⚠ I reached the tool iteration limit.\n\n")
		summary.WriteString("**Information collected so far:**\n")

		toolResultCount := 0
		for _, content := range *m.history {
			if content.Role == "tool" {
				toolResultCount++
			}
		}

		if toolResultCount > 0 {
			fmt.Fprintf(&summary, "- Executed %d tool calls\n", toolResultCount)
		}

		summary.WriteString("\nIf you need more information, please ask a more specific question or ask me to continue from where I left off.")

		return ResponseMsg{Content: summary.String(), owner: m.msgs}
	}
}

// toolFailure extracts a short failure description from a tool execution:
// either a transport error or a handler-level error output (success=false).
// Returns "" when the call succeeded.
func toolFailure(result map[string]any, err error) string {
	if err != nil {
		return err.Error()
	}
	if result != nil {
		if success, ok := result["success"].(bool); ok && !success {
			if msg, ok := result["error"].(string); ok && msg != "" {
				return msg
			}
			return "failed"
		}
	}
	return ""
}

// formatToolRecord builds the one-line transcript record of a tool call,
// e.g. `pumbaa query (status=Failed) ✓ 0.8s`. Failures include a short
// reason so the transcript explains itself.
func formatToolRecord(name, action string, params map[string]any, dur time.Duration, failure string) string {
	label := name
	if action != "" {
		label += " " + action
	}

	if len(params) > 0 {
		keys := make([]string, 0, len(params))
		for k := range params {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		pairs := make([]string, 0, len(keys))
		for _, k := range keys {
			pairs = append(pairs, fmt.Sprintf("%s=%v", k, params[k]))
		}
		label += " (" + strings.Join(pairs, ", ") + ")"
	}

	if failure != "" {
		return fmt.Sprintf("%s ✗ %.1fs — %s", label, dur.Seconds(), common.Truncate(failure, 80))
	}
	return fmt.Sprintf("%s ✓ %.1fs", label, dur.Seconds())
}

func getToolCalls(content *genai.Content) []*genai.FunctionCall {
	var calls []*genai.FunctionCall
	for _, part := range content.Parts {
		if part.FunctionCall != nil {
			calls = append(calls, part.FunctionCall)
		}
	}
	return calls
}

func (m Model) executeTool(ctx context.Context, fc *genai.FunctionCall) (map[string]any, error) {
	for _, t := range m.tools {
		if td, ok := t.(toolWithDefinition); ok {
			def := td.Declaration()
			if def.Name == fc.Name {
				// functiontool.Run dereferences the tool context (ADK v1.0.0
				// panics on nil), so pass the minimal no-op context.
				return td.Run(noopToolContext{}, fc.Args)
			}
		}
	}
	return nil, fmt.Errorf("tool not found: %s", fc.Name)
}

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
