package tui_test

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/oh-my-pi/omp/pkg/ai"
	"github.com/oh-my-pi/omp/pkg/tui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockTUIProvider struct {
	responses [][]ai.StreamEvent
	callCount int32
}

func (m *mockTUIProvider) Name() string { return "mock" }
func (m *mockTUIProvider) Stream(ctx context.Context, req ai.Request) (<-chan ai.StreamEvent, error) {
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

func TestChatMessage_Render(t *testing.T) {
	msg := tui.ChatMessage{Role: "user", Content: "hello"}
	rendered := msg.Render()
	assert.Contains(t, rendered, "hello")
	assert.Contains(t, rendered, "user")
}

func TestChatMessage_AssistantRender(t *testing.T) {
	msg := tui.ChatMessage{Role: "assistant", Content: "**bold** *italic*"}
	rendered := msg.Render()
	assert.Contains(t, rendered, "assistant")
}

func TestNewSession(t *testing.T) {
	p := &mockTUIProvider{
		responses: [][]ai.StreamEvent{
			{
				{Type: ai.EventText, Content: "Hello from TUI"},
				{Type: ai.EventDone, FinishReason: "stop"},
			},
		},
	}

	sess, err := tui.NewSession(p, ai.Request{Model: "test"})
	require.NoError(t, err)
	require.NotNil(t, sess)
	assert.Equal(t, 0, sess.MessageCount())
}

func TestSession_SendMessage(t *testing.T) {
	p := &mockTUIProvider{
		responses: [][]ai.StreamEvent{
			{
				{Type: ai.EventText, Content: "Hello World"},
				{Type: ai.EventDone, FinishReason: "stop"},
			},
		},
	}

	sess, err := tui.NewSession(p, ai.Request{Model: "test"})
	require.NoError(t, err)

	resp, err := sess.Send(context.Background(), "say hi")
	require.NoError(t, err)
	assert.Contains(t, resp, "Hello World")
	// user + assistant message should be recorded
	assert.Equal(t, 2, sess.MessageCount())
}

func TestSession_MessageHistory(t *testing.T) {
	p := &mockTUIProvider{
		responses: [][]ai.StreamEvent{
			{
				{Type: ai.EventText, Content: "first reply"},
				{Type: ai.EventDone, FinishReason: "stop"},
			},
			{
				{Type: ai.EventText, Content: "second reply"},
				{Type: ai.EventDone, FinishReason: "stop"},
			},
		},
	}

	sess, err := tui.NewSession(p, ai.Request{Model: "test"})
	require.NoError(t, err)

	_, err = sess.Send(context.Background(), "msg1")
	require.NoError(t, err)

	_, err = sess.Send(context.Background(), "msg2")
	require.NoError(t, err)

	assert.Equal(t, 4, sess.MessageCount())

	msgs := sess.Messages()
	assert.Equal(t, "user", msgs[0].Role)
	assert.Contains(t, msgs[0].Content, "msg1")
	assert.Equal(t, "assistant", msgs[1].Role)
	assert.Equal(t, "user", msgs[2].Role)
	assert.Equal(t, "assistant", msgs[3].Role)
}

func TestSession_EmptyInput(t *testing.T) {
	sess, err := tui.NewSession(nil, ai.Request{Model: "test"})
	require.NoError(t, err)

	_, err = sess.Send(context.Background(), "")
	require.Error(t, err)
}

func TestSession_ClearMessages(t *testing.T) {
	p := &mockTUIProvider{
		responses: [][]ai.StreamEvent{
			{
				{Type: ai.EventText, Content: "reply"},
				{Type: ai.EventDone, FinishReason: "stop"},
			},
		},
	}

	sess, err := tui.NewSession(p, ai.Request{Model: "test"})
	require.NoError(t, err)

	_, err = sess.Send(context.Background(), "hi")
	require.NoError(t, err)
	assert.Equal(t, 2, sess.MessageCount())

	sess.Clear()
	assert.Equal(t, 0, sess.MessageCount())
}
