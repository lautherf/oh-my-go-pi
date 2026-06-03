package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/oh-my-pi/omp/pkg/ai"
	"github.com/rs/zerolog/log"
)

type Tool interface {
	Name() string
	Execute(ctx context.Context, argsJSON string) (string, error)
}

type Agent struct {
	provider ai.Provider
	req      ai.Request
	messages []ai.Message
	tools    map[string]Tool
	mu       sync.Mutex
}

type Response struct {
	Text string
}

func New(provider ai.Provider, req ai.Request) *Agent {
	return &Agent{
		provider: provider,
		req:      req,
		tools:    make(map[string]Tool),
	}
}

func (a *Agent) SetSystemPrompt(prompt string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.req.System = prompt
}

func (a *Agent) RegisterTool(t Tool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.tools[t.Name()] = t
}

func (a *Agent) Run(ctx context.Context, input string) (*Response, error) {
	if strings.TrimSpace(input) == "" {
		return nil, fmt.Errorf("empty input")
	}

	a.mu.Lock()
	if a.req.System != "" {
		a.messages = append(a.messages, ai.NewSystemMessage(a.req.System))
		a.req.System = ""
	}
	a.messages = append(a.messages, ai.NewUserMessage(input))
	currentReq := a.req
	a.mu.Unlock()

	// build tool defs for LLM
	a.mu.Lock()
	toolDefs := make([]ai.ToolDef, 0, len(a.tools))
	for name := range a.tools {
		toolDefs = append(toolDefs, ai.ToolDef{
			Type:     "function",
			Function: ai.ToolFunction{Name: name, Description: name},
		})
	}
	currentReq.Tools = toolDefs
	currentReq.Stream = true
	a.mu.Unlock()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		currentReq.Messages = a.getMessages()

		log.Debug().Int("messages", len(currentReq.Messages)).Int("tools", len(currentReq.Tools)).Msg("agent: calling provider")
		ch, err := a.provider.Stream(ctx, currentReq)
		if err != nil {
			log.Error().Err(err).Msg("agent: provider stream failed")
			return nil, fmt.Errorf("provider: %w", err)
		}

		var textBuilder strings.Builder
		var toolCalls []ai.ToolCall

		for event := range ch {
			switch event.Type {
			case ai.EventText:
				textBuilder.WriteString(event.Content)
			case ai.EventToolCall:
				if event.ToolCall != nil {
					toolCalls = append(toolCalls, *event.ToolCall)
				}
			case ai.EventError:
				log.Error().Str("error", event.Content).Msg("agent: provider error")
				return nil, fmt.Errorf("provider error: %s", event.Content)
			case ai.EventDone:
				if textBuilder.Len() > 0 {
					a.addMessage(ai.NewAssistantMessage(textBuilder.String()))
					log.Debug().Int("tokens", textBuilder.Len()).Msg("agent: received response")
				}
				if len(toolCalls) > 0 {
					names := make([]string, len(toolCalls))
					for i, tc := range toolCalls {
						names[i] = tc.Function.Name
					}
					log.Debug().Strs("tools", names).Msg("agent: executing tool calls")
					if err := a.executeToolCalls(ctx, toolCalls); err != nil {
						return nil, err
					}
					toolCalls = nil
					textBuilder.Reset()
					goto nextIteration
				}
				log.Debug().Msg("agent: done")
				return &Response{Text: textBuilder.String()}, nil
			}
		}

		// stream ended without done event
		if textBuilder.Len() == 0 && len(toolCalls) == 0 {
			return nil, fmt.Errorf("provider returned empty response")
		}

	nextIteration:
	}
}

func (a *Agent) executeToolCalls(ctx context.Context, calls []ai.ToolCall) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, call := range calls {
		tool, ok := a.tools[call.Function.Name]
		if !ok {
			return fmt.Errorf("tool %q not found (registered: %v)", call.Function.Name, toolNames(a.tools))
		}

		result, err := tool.Execute(ctx, call.Function.Arguments)
		if err != nil {
			result = fmt.Sprintf("error: %v", err)
		}

		a.messages = append(a.messages, ai.NewToolMessage(result, call.ID))
	}
	return nil
}

func (a *Agent) addMessage(m ai.Message) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.messages = append(a.messages, m)
}

func (a *Agent) getMessages() []ai.Message {
	a.mu.Lock()
	defer a.mu.Unlock()
	msgs := make([]ai.Message, len(a.messages))
	copy(msgs, a.messages)
	return msgs
}

func toolNames(tools map[string]Tool) string {
	names := make([]string, 0, len(tools))
	for n := range tools {
		names = append(names, n)
	}
	b, _ := json.Marshal(names)
	return string(b)
}
