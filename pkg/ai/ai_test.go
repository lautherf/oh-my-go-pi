package ai_test

import (
	"context"
	"errors"
	"testing"

	"github.com/oh-my-pi/omp/pkg/ai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock provider ---

type mockProvider struct {
	events []ai.StreamEvent
	err    error
}

func (m *mockProvider) Stream(ctx context.Context, req ai.Request) (<-chan ai.StreamEvent, error) {
	if m.err != nil {
		return nil, m.err
	}
	ch := make(chan ai.StreamEvent, len(m.events))
	if ctx.Err() != nil {
		close(ch)
		return ch, nil
	}
	go func() {
		defer close(ch)
		for _, e := range m.events {
			select {
			case ch <- e:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch, nil
}

func (m *mockProvider) Name() string { return "mock" }

// --- Tests ---

func TestMessage_String(t *testing.T) {
	m := ai.Message{Role: ai.RoleUser, Content: "hello"}
	assert.Contains(t, m.String(), "user")
	assert.Contains(t, m.String(), "hello")
}

func TestNewUserMessage(t *testing.T) {
	m := ai.NewUserMessage("hello world")
	assert.Equal(t, ai.RoleUser, m.Role)
	assert.Equal(t, "hello world", m.Content)
}

func TestNewSystemMessage(t *testing.T) {
	m := ai.NewSystemMessage("be helpful")
	assert.Equal(t, ai.RoleSystem, m.Role)
	assert.Equal(t, "be helpful", m.Content)
}

func TestNewAssistantMessage(t *testing.T) {
	m := ai.NewAssistantMessage("hi")
	assert.Equal(t, ai.RoleAssistant, m.Role)
	assert.Equal(t, "hi", m.Content)
}

func TestNewToolMessage(t *testing.T) {
	m := ai.NewToolMessage("result", "call_123")
	assert.Equal(t, ai.RoleTool, m.Role)
	assert.Equal(t, "result", m.Content)
	assert.Equal(t, "call_123", m.ToolCallID)
}

func TestProvider_StreamText(t *testing.T) {
	p := &mockProvider{
		events: []ai.StreamEvent{
			{Type: ai.EventText, Content: "Hello "},
			{Type: ai.EventText, Content: "World"},
			{Type: ai.EventDone, FinishReason: "stop"},
		},
	}

	ch, err := p.Stream(context.Background(), ai.Request{Messages: []ai.Message{ai.NewUserMessage("hi")}})
	require.NoError(t, err)

	var events []ai.StreamEvent
	for e := range ch {
		events = append(events, e)
	}
	require.Len(t, events, 3)
	assert.Equal(t, "Hello ", events[0].Content)
	assert.Equal(t, "World", events[1].Content)
	assert.Equal(t, ai.EventDone, events[2].Type)
}

func TestProvider_StreamToolCall(t *testing.T) {
	p := &mockProvider{
		events: []ai.StreamEvent{
			{Type: ai.EventToolCall, ToolCall: &ai.ToolCall{ID: "call_1", Function: ai.FunctionCall{Name: "echo", Arguments: `{"msg":"hi"}`}}},
			{Type: ai.EventDone, FinishReason: "tool_calls"},
		},
	}

	ch, err := p.Stream(context.Background(), ai.Request{})
	require.NoError(t, err)

	var calls []ai.ToolCall
	for e := range ch {
		if e.Type == ai.EventToolCall && e.ToolCall != nil {
			calls = append(calls, *e.ToolCall)
		}
	}
	require.Len(t, calls, 1)
	assert.Equal(t, "echo", calls[0].Function.Name)
}

func TestProvider_StreamError(t *testing.T) {
	p := &mockProvider{err: errors.New("provider down")}
	_, err := p.Stream(context.Background(), ai.Request{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider down")
}

func TestProvider_StreamCancel(t *testing.T) {
	p := &mockProvider{
		events: []ai.StreamEvent{
			{Type: ai.EventText, Content: "before cancel"},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ch, err := p.Stream(ctx, ai.Request{})
	require.NoError(t, err)

	_, ok := <-ch
	assert.False(t, ok, "channel should be closed immediately on cancelled context")
}

func TestToolCall_String(t *testing.T) {
	tc := ai.ToolCall{ID: "call_1", Function: ai.FunctionCall{Name: "read_file", Arguments: `{"path":"/tmp/x"}`}}
	s := tc.String()
	assert.Contains(t, s, "read_file")
	assert.Contains(t, s, "/tmp/x")
}

func TestRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     ai.Request
		wantErr bool
	}{
		{"empty messages", ai.Request{}, true},
		{"no model", ai.Request{Messages: []ai.Message{ai.NewUserMessage("hi")}}, true},
		{"valid", ai.Request{Messages: []ai.Message{ai.NewUserMessage("hi")}, Model: "gpt-4"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
