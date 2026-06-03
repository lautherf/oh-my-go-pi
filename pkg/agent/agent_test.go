package agent_test

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/oh-my-pi/omp/pkg/ai"
	"github.com/oh-my-pi/omp/pkg/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- mock provider ---

type mockProvider struct {
	responses [][]ai.StreamEvent
	callCount int32
}

func (m *mockProvider) Name() string { return "mock" }
func (m *mockProvider) Stream(ctx context.Context, req ai.Request) (<-chan ai.StreamEvent, error) {
	atomic.AddInt32(&m.callCount, 1)
	ch := make(chan ai.StreamEvent, 16)
	go func() {
		defer close(ch)
		idx := int(atomic.LoadInt32(&m.callCount)) - 1
		if idx >= len(m.responses) {
			return
		}
		for _, e := range m.responses[idx] {
			select {
			case ch <- e:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch, nil
}

// --- mock tool ---

type mockTool struct {
	name   string
	result string
}

func (m *mockTool) Name() string { return m.name }
func (m *mockTool) Execute(ctx context.Context, argsJSON string) (string, error) {
	return m.result, nil
}

type echoTool struct{}

func (e *echoTool) Name() string { return "echo" }
func (e *echoTool) Execute(ctx context.Context, argsJSON string) (string, error) {
	return "executed: " + argsJSON, nil
}

// --- tests ---

func TestAgent_BasicTextResponse(t *testing.T) {
	p := &mockProvider{
		responses: [][]ai.StreamEvent{
			{
				{Type: ai.EventText, Content: "Hello from agent"},
				{Type: ai.EventDone, FinishReason: "stop"},
			},
		},
	}

	a := agent.New(p, ai.Request{Model: "test"})
	resp, err := a.Run(context.Background(), "say hi")
	require.NoError(t, err)
	assert.Equal(t, "Hello from agent", resp.Text)
}

func TestAgent_WithSystemMessage(t *testing.T) {
	p := &mockProvider{
		responses: [][]ai.StreamEvent{
			{
				{Type: ai.EventText, Content: "I am helpful"},
				{Type: ai.EventDone, FinishReason: "stop"},
			},
		},
	}

	a := agent.New(p, ai.Request{Model: "test"})
	a.SetSystemPrompt("You are a helpful assistant")
	resp, err := a.Run(context.Background(), "help me")
	require.NoError(t, err)
	assert.Contains(t, resp.Text, "helpful")
}

func TestAgent_ToolCallAndResponse(t *testing.T) {
	p := &mockProvider{
		responses: [][]ai.StreamEvent{
			{
				{Type: ai.EventToolCall, ToolCall: &ai.ToolCall{
					ID: "call_1", Type: "function",
					Function: ai.FunctionCall{Name: "echo", Arguments: `{"msg":"hello"}`},
				}},
				{Type: ai.EventDone, FinishReason: "tool_calls"},
			},
			{
				{Type: ai.EventText, Content: "Tool result received"},
				{Type: ai.EventDone, FinishReason: "stop"},
			},
		},
	}

	a := agent.New(p, ai.Request{Model: "test"})
	a.RegisterTool(&echoTool{})

	resp, err := a.Run(context.Background(), "use echo tool")
	require.NoError(t, err)
	assert.Equal(t, "Tool result received", resp.Text)
	assert.Equal(t, int32(2), atomic.LoadInt32(&p.callCount), "should call provider twice")
}

func TestAgent_NoToolsRegistered(t *testing.T) {
	p := &mockProvider{
		responses: [][]ai.StreamEvent{
			{
				{Type: ai.EventToolCall, ToolCall: &ai.ToolCall{
					ID: "call_1", Type: "function",
					Function: ai.FunctionCall{Name: "nonexistent_tool", Arguments: `{}`},
				}},
				{Type: ai.EventDone, FinishReason: "tool_calls"},
			},
		},
	}

	a := agent.New(p, ai.Request{Model: "test"})
	_, err := a.Run(context.Background(), "call tool")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent_tool")
}

func TestAgent_MessagesHistory(t *testing.T) {
	p := &mockProvider{
		responses: [][]ai.StreamEvent{
			{
				{Type: ai.EventText, Content: "first"},
				{Type: ai.EventDone, FinishReason: "stop"},
			},
			{
				{Type: ai.EventText, Content: "second"},
				{Type: ai.EventDone, FinishReason: "stop"},
			},
		},
	}

	a := agent.New(p, ai.Request{Model: "test"})
	resp1, err := a.Run(context.Background(), "msg1")
	require.NoError(t, err)
	assert.Equal(t, "first", resp1.Text)

	resp2, err := a.Run(context.Background(), "msg2")
	require.NoError(t, err)
	assert.Equal(t, "second", resp2.Text)

	assert.Equal(t, int32(2), atomic.LoadInt32(&p.callCount))
}

func TestAgent_EmptyInput(t *testing.T) {
	p := &mockProvider{}
	a := agent.New(p, ai.Request{Model: "test"})
	_, err := a.Run(context.Background(), "")
	require.Error(t, err)
}

func TestAgent_ProviderError(t *testing.T) {
	p := &mockProvider{} // no responses -> returns nothing
	a := agent.New(p, ai.Request{Model: "test"})
	_, err := a.Run(context.Background(), "hi")
	require.Error(t, err)
}

func TestAgent_ContextCancel(t *testing.T) {
	p := &mockProvider{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	a := agent.New(p, ai.Request{Model: "test"})
	_, err := a.Run(ctx, "hi")
	require.Error(t, err)
}
