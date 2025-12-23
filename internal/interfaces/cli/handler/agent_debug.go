package handler

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/lmtani/pumbaa/internal/config"
	"github.com/lmtani/pumbaa/internal/infrastructure/agent/tools"
	"github.com/lmtani/pumbaa/internal/infrastructure/cromwell"
	"github.com/lmtani/pumbaa/internal/infrastructure/ollama"
	"github.com/urfave/cli/v2"
	"google.golang.org/adk/model"
	"google.golang.org/adk/tool"
	"google.golang.org/genai"
)

// AgentTestHandler handles the agent-test command for debugging LLM interactions
type AgentTestHandler struct {
	config     *config.Config
	pumbaaTool tool.Tool // Store the tool for execution
}

func NewAgentTestHandler(cfg *config.Config) *AgentTestHandler {
	return &AgentTestHandler{config: cfg}
}

func (h *AgentTestHandler) Command() *cli.Command {
	return &cli.Command{
		Name:  "agent-test",
		Usage: "Test LLM agent tool calling (for debugging)",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "provider",
				Aliases: []string{"p"},
				Usage:   "LLM provider: ollama or vertex",
				Value:   "ollama",
			},
			&cli.StringFlag{
				Name:    "model",
				Aliases: []string{"m"},
				Usage:   "Model name",
				Value:   "",
			},
			&cli.StringFlag{
				Name:    "prompt",
				Aliases: []string{"q"},
				Usage:   "Prompt to send (if not provided, enters interactive mode)",
			},
			&cli.BoolFlag{
				Name:  "verbose",
				Usage: "Show detailed debug output",
				Value: true,
			},
			&cli.BoolFlag{
				Name:  "execute",
				Usage: "Actually execute tool calls (default: true)",
				Value: true,
			},
		},
		Action: h.Run,
	}
}

func (h *AgentTestHandler) Run(c *cli.Context) error {
	provider := c.String("provider")
	modelName := c.String("model")
	prompt := c.String("prompt")
	verbose := c.Bool("verbose")
	execute := c.Bool("execute")

	// Create LLM
	var llm model.LLM
	switch provider {
	case "ollama":
		if modelName == "" {
			modelName = h.config.OllamaModel
		}
		llm = ollama.NewModel(h.config.OllamaHost, modelName)
		fmt.Printf("Using Ollama model: %s at %s\n", modelName, h.config.OllamaHost)
	default:
		return fmt.Errorf("only 'ollama' provider supported in this test command")
	}

	// Create Cromwell repository for tools
	repo := cromwell.NewClient(cromwell.Config{Host: h.config.CromwellHost, Timeout: h.config.CromwellTimeout})
	h.pumbaaTool = tools.GetPumbaaTool(repo, nil)

	// Convert tool to genai format
	genaiTools := convertToolsToGenAI([]tool.Tool{h.pumbaaTool})

	if verbose {
		fmt.Println("\n=== TOOL DEFINITION ===")
		for _, t := range genaiTools {
			for _, fn := range t.FunctionDeclarations {
				fmt.Printf("Tool: %s\n", fn.Name)
				fmt.Printf("Description: %s\n", fn.Description[:min(200, len(fn.Description))])
				if fn.Parameters != nil {
					fmt.Printf("Parameters.Type: %s\n", fn.Parameters.Type)
					fmt.Printf("Parameters.Properties count: %d\n", len(fn.Parameters.Properties))
					fmt.Printf("Parameters.Required: %v\n", fn.Parameters.Required)
					// Print the whole Parameters struct
					paramsJSON, _ := json.MarshalIndent(fn.Parameters, "", "  ")
					fmt.Printf("Parameters (full):\n%s\n", string(paramsJSON))
				} else {
					fmt.Println("Parameters: nil")
				}
			}
		}
		fmt.Println("=== END TOOL DEFINITION ===\n")
	}

	// Interactive mode or single prompt mode
	if prompt == "" {
		return h.interactiveMode(llm, genaiTools, verbose, execute)
	}

	return h.singlePrompt(llm, genaiTools, prompt, verbose, execute)
}

func (h *AgentTestHandler) singlePrompt(llm model.LLM, genaiTools []*genai.Tool, prompt string, verbose bool, execute bool) error {
	ctx := context.Background()

	history := []*genai.Content{
		{
			Role: "user",
			Parts: []*genai.Part{
				genai.NewPartFromText(prompt),
			},
		},
	}

	return h.executeRequest(ctx, llm, genaiTools, history, verbose, execute)
}

func (h *AgentTestHandler) interactiveMode(llm model.LLM, genaiTools []*genai.Tool, verbose bool, execute bool) error {
	ctx := context.Background()
	reader := bufio.NewReader(os.Stdin)
	history := make([]*genai.Content, 0)

	fmt.Println("Interactive mode. Type 'quit' to exit.")
	fmt.Println("Try: 'busque pelos metadados do workflow <id>'")
	fmt.Println()

	for {
		fmt.Print("You: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		input = strings.TrimSpace(input)
		if input == "quit" || input == "exit" {
			break
		}
		if input == "" {
			continue
		}

		// Add user message to history
		history = append(history, &genai.Content{
			Role: "user",
			Parts: []*genai.Part{
				genai.NewPartFromText(input),
			},
		})

		// Execute request and update history
		newHistory, err := h.executeRequestWithHistory(ctx, llm, genaiTools, history, verbose, execute)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		history = newHistory
		fmt.Println()
	}

	return nil
}

func (h *AgentTestHandler) executeRequest(ctx context.Context, llm model.LLM, genaiTools []*genai.Tool, history []*genai.Content, verbose bool, execute bool) error {
	_, err := h.executeRequestWithHistory(ctx, llm, genaiTools, history, verbose, execute)
	return err
}

func (h *AgentTestHandler) executeRequestWithHistory(ctx context.Context, llm model.LLM, genaiTools []*genai.Tool, history []*genai.Content, verbose bool, execute bool) ([]*genai.Content, error) {
	maxTurns := 5
	currentTurn := 0

	for currentTurn < maxTurns {
		req := &model.LLMRequest{
			Contents: history,
			Config: &genai.GenerateContentConfig{
				Tools: genaiTools,
			},
		}

		if verbose {
			fmt.Printf("\n[Turn %d] Sending request with %d messages...\n", currentTurn+1, len(history))
		}

		respSeq := llm.GenerateContent(ctx, req, false)

		var lastResp *model.LLMResponse
		for r, e := range respSeq {
			if e != nil {
				return history, fmt.Errorf("LLM error: %w", e)
			}
			lastResp = r
		}

		if lastResp == nil || lastResp.Content == nil {
			return history, fmt.Errorf("empty response from model")
		}

		// Show response details
		if verbose {
			fmt.Printf("[Response] Role: %s, Parts: %d, TurnComplete: %v\n",
				lastResp.Content.Role, len(lastResp.Content.Parts), lastResp.TurnComplete)
		}

		// Add response to history
		history = append(history, lastResp.Content)

		// Check for tool calls
		var toolCalls []*genai.FunctionCall
		for _, part := range lastResp.Content.Parts {
			if part.FunctionCall != nil {
				toolCalls = append(toolCalls, part.FunctionCall)
			}
			if part.Text != "" {
				fmt.Printf("\nAgent: %s\n", part.Text)
			}
		}

		if len(toolCalls) > 0 {
			fmt.Printf("\n[Tool Calls] Model wants to call %d tool(s):\n", len(toolCalls))
			for _, tc := range toolCalls {
				argsJSON, _ := json.MarshalIndent(tc.Args, "", "  ")
				fmt.Printf("  - %s(%s)\n", tc.Name, string(argsJSON))
			}

			if !execute {
				fmt.Println("\n[Note] Tool calls detected but not executed (--execute=false).")
				fmt.Println("[Note] This confirms the model IS calling tools correctly!")
				return history, nil
			}

			// Execute each tool call and collect responses
			var toolResponses []*genai.Part
			for _, tc := range toolCalls {
				fmt.Printf("\n[Executing] %s...\n", tc.Name)

				// Execute the tool using the toolWithDef interface
				if td, ok := h.pumbaaTool.(toolWithDef); ok {
					result, err := td.Run(nil, tc.Args)

					var responseData map[string]any
					if err != nil {
						fmt.Printf("[Tool Error] %v\n", err)
						responseData = map[string]any{"error": err.Error()}
					} else {
						resultJSON, _ := json.MarshalIndent(result, "", "  ")
						fmt.Printf("[Tool Result]\n%s\n", string(resultJSON))
						responseData = map[string]any{"result": result}
					}

					toolResponses = append(toolResponses, genai.NewPartFromFunctionResponse(tc.Name, responseData))
				} else {
					fmt.Printf("[Error] Tool doesn't support Run method\n")
					toolResponses = append(toolResponses, genai.NewPartFromFunctionResponse(tc.Name, map[string]any{
						"error": "tool doesn't implement Run method",
					}))
				}
			}

			// Add tool responses to history
			history = append(history, &genai.Content{
				Role:  "tool",
				Parts: toolResponses,
			})

			// Continue to next turn to get model response
			currentTurn++
			continue
		}

		// No tool calls and turn complete - we're done
		if lastResp.TurnComplete {
			return history, nil
		}

		currentTurn++
	}

	return history, fmt.Errorf("max turns reached")
}

// Interface to access the hidden definition method of functiontool
type toolWithDef interface {
	Declaration() *genai.FunctionDeclaration
	Run(ctx tool.Context, args interface{}) (map[string]interface{}, error)
}

func convertToolsToGenAI(toolList []tool.Tool) []*genai.Tool {
	var genaiTools []*genai.Tool
	for _, t := range toolList {
		if td, ok := t.(toolWithDef); ok {
			genaiTools = append(genaiTools, &genai.Tool{
				FunctionDeclarations: []*genai.FunctionDeclaration{
					td.Declaration(),
				},
			})
		}
	}
	return genaiTools
}
