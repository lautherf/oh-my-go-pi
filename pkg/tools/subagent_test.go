package tools_test

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/oh-my-pi/omp/pkg/agent"
	"github.com/oh-my-pi/omp/pkg/ai"
	"github.com/oh-my-pi/omp/pkg/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockSubProvider struct {
	callCount int32
}

func (m *mockSubProvider) Name() string { return "mock" }
func (m *mockSubProvider) Stream(ctx context.Context, req ai.Request) (<-chan ai.StreamEvent, error) {
	atomic.AddInt32(&m.callCount, 1)
	ch := make(chan ai.StreamEvent, 4)
	ch <- ai.StreamEvent{Type: ai.EventText, Content: "worker done: "}
	for _, m := range req.Messages {
		if m.Role == ai.RoleUser {
			ch <- ai.StreamEvent{Type: ai.EventText, Content: m.Content}
		}
	}
	ch <- ai.StreamEvent{Type: ai.EventDone, FinishReason: "stop"}
	close(ch)
	return ch, nil
}

func TestSubAgentTool_Execute(t *testing.T) {
	p := &mockSubProvider{}
	req := ai.Request{Model: "test"}
	sub := tools.NewSubAgentTool(p, req, []agent.Tool{})

	result, err := sub.Execute(context.Background(), `{"task":"find all funcs in main.go"}`)
	require.NoError(t, err)
	assert.Contains(t, result, "worker done: find all funcs in main.go")
}

func TestSubAgentTool_EmptyTask(t *testing.T) {
	p := &mockSubProvider{}
	sub := tools.NewSubAgentTool(p, ai.Request{Model: "test"}, nil)

	_, err := sub.Execute(context.Background(), `{"task":""}`)
	require.Error(t, err)
}

func TestSubAgentTool_Name(t *testing.T) {
	assert.Equal(t, "subagent", (tools.NewSubAgentTool(nil, ai.Request{}, nil)).Name())
}
