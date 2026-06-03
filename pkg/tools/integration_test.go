// Integration test: agent + real tools end-to-end
package tools_test

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/oh-my-pi/omp/pkg/ai"
	"github.com/oh-my-pi/omp/pkg/agent"
	"github.com/oh-my-pi/omp/pkg/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type integrationProvider struct {
	responses [][]ai.StreamEvent
	callCount int32
}

func (p *integrationProvider) Name() string { return "integration-mock" }
func (p *integrationProvider) Stream(ctx context.Context, req ai.Request) (<-chan ai.StreamEvent, error) {
	atomic.AddInt32(&p.callCount, 1)
	ch := make(chan ai.StreamEvent, 16)
	go func() {
		defer close(ch)
		idx := int(atomic.LoadInt32(&p.callCount)) - 1
		if idx >= len(p.responses) {
			return
		}
		for _, e := range p.responses[idx] {
			select {
			case ch <- e:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch, nil
}

func TestAgentWithBashTool(t *testing.T) {
	p := &integrationProvider{
		responses: [][]ai.StreamEvent{
			{
				{Type: ai.EventToolCall, ToolCall: &ai.ToolCall{
					ID: "call_1", Type: "function",
					Function: ai.FunctionCall{Name: "bash", Arguments: `{"command":"echo hello from bash"}`},
				}},
				{Type: ai.EventDone, FinishReason: "tool_calls"},
			},
			{
				{Type: ai.EventText, Content: "The bash tool said: "},
				{Type: ai.EventText, Content: "hello from bash"},
				{Type: ai.EventDone, FinishReason: "stop"},
			},
		},
	}

	a := agent.New(p, ai.Request{Model: "test"})
	a.RegisterTool(&tools.BashTool{})

	resp, err := a.Run(context.Background(), "run bash and tell me what it says")
	require.NoError(t, err)
	assert.Contains(t, resp.Text, "hello from bash")
	assert.Equal(t, int32(2), atomic.LoadInt32(&p.callCount))
}

func TestAgentWithGrepThenRead(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "target.txt"), []byte("find this text\n"), 0644))

	p := &integrationProvider{
		responses: [][]ai.StreamEvent{
			{
				{Type: ai.EventToolCall, ToolCall: &ai.ToolCall{
					ID: "call_1", Type: "function",
					Function: ai.FunctionCall{Name: "grep", Arguments: `{"pattern":"find this","root":"` + dir + `"}`},
				}},
				{Type: ai.EventDone, FinishReason: "tool_calls"},
			},
			{
				{Type: ai.EventToolCall, ToolCall: &ai.ToolCall{
					ID: "call_2", Type: "function",
					Function: ai.FunctionCall{Name: "read", Arguments: `{"path":"` + dir + `/target.txt"}`},
				}},
				{Type: ai.EventDone, FinishReason: "tool_calls"},
			},
			{
				{Type: ai.EventText, Content: "Found: find this text"},
				{Type: ai.EventDone, FinishReason: "stop"},
			},
		},
	}

	a := agent.New(p, ai.Request{Model: "test"})
	a.RegisterTool(&tools.GrepTool{})
	a.RegisterTool(&tools.ReadTool{})

	resp, err := a.Run(context.Background(), "find the file and read it")
	require.NoError(t, err)
	assert.Contains(t, resp.Text, "Found")
	assert.Equal(t, int32(3), atomic.LoadInt32(&p.callCount))
}

func TestAgentWithSubAgentTool(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "data.txt"), []byte("worker was here\n"), 0644))

	p := &integrationProvider{
		responses: [][]ai.StreamEvent{
			{
				{Type: ai.EventToolCall, ToolCall: &ai.ToolCall{
					ID: "call_1", Type: "function",
					Function: ai.FunctionCall{Name: "subagent", Arguments: `{"task":"read data.txt in ` + dir + `"}`},
				}},
				{Type: ai.EventDone, FinishReason: "tool_calls"},
			},
			{
				{Type: ai.EventText, Content: "Worker result: worker was here"},
				{Type: ai.EventDone, FinishReason: "stop"},
			},
			{
				{Type: ai.EventText, Content: "The subagent returned: worker was here"},
				{Type: ai.EventDone, FinishReason: "stop"},
			},
		},
	}

	subagentTool := tools.NewSubAgentTool(p, ai.Request{Model: "test"}, []agent.Tool{
		&tools.ReadTool{},
	})

	a := agent.New(p, ai.Request{Model: "test"})
	a.RegisterTool(subagentTool)

	resp, err := a.Run(context.Background(), "use subagent to read the file")
	require.NoError(t, err)
	assert.Contains(t, resp.Text, "The subagent returned: worker was here")
}

func TestAgentWithWriteThenRead(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "created.txt")

	p := &integrationProvider{
		responses: [][]ai.StreamEvent{
			{
				{Type: ai.EventToolCall, ToolCall: &ai.ToolCall{
					ID: "call_1", Type: "function",
					Function: ai.FunctionCall{Name: "write", Arguments: `{"path":"` + outPath + `","content":"written by agent"}`},
				}},
				{Type: ai.EventDone, FinishReason: "tool_calls"},
			},
			{
				{Type: ai.EventToolCall, ToolCall: &ai.ToolCall{
					ID: "call_2", Type: "function",
					Function: ai.FunctionCall{Name: "read", Arguments: `{"path":"` + outPath + `"}`},
				}},
				{Type: ai.EventDone, FinishReason: "tool_calls"},
			},
			{
				{Type: ai.EventText, Content: "written by agent"},
				{Type: ai.EventDone, FinishReason: "stop"},
			},
		},
	}

	a := agent.New(p, ai.Request{Model: "test"})
	a.RegisterTool(&tools.WriteTool{})
	a.RegisterTool(&tools.ReadTool{})

	resp, err := a.Run(context.Background(), "create a file and read it back")
	require.NoError(t, err)
	assert.Contains(t, resp.Text, "written by agent")
}
