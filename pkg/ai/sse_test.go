package ai_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/oh-my-pi/omp/pkg/ai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSE_BasicEvent(t *testing.T) {
	input := "data: {\"key\":\"value\"}\n\n"
	events, err := ai.ParseSSE(context.Background(), strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, `{"key":"value"}`, events[0].Data)
}

func TestSSE_MultipleEvents(t *testing.T) {
	input := "data: first\n\ndata: second\n\ndata: third\n\n"
	events, err := ai.ParseSSE(context.Background(), strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 3)
	assert.Equal(t, "first", events[0].Data)
	assert.Equal(t, "second", events[1].Data)
	assert.Equal(t, "third", events[2].Data)
}

func TestSSE_EventWithID(t *testing.T) {
	input := "id: 1\ndata: hello\n\n"
	events, err := ai.ParseSSE(context.Background(), strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "hello", events[0].Data)
	assert.Equal(t, "1", events[0].ID)
}

func TestSSE_EventWithType(t *testing.T) {
	input := "event: completion\ndata: done\n\n"
	events, err := ai.ParseSSE(context.Background(), strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "completion", events[0].Type)
	assert.Equal(t, "done", events[0].Data)
}

func TestSSE_Comment(t *testing.T) {
	input := ": comment line\ndata: actual\n\n"
	events, err := ai.ParseSSE(context.Background(), strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "actual", events[0].Data)
}

func TestSSE_MultilineData(t *testing.T) {
	input := "data: line1\ndata: line2\n\n"
	events, err := ai.ParseSSE(context.Background(), strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "line1\nline2", events[0].Data)
}

func TestSSE_EmptyStream(t *testing.T) {
	events, err := ai.ParseSSE(context.Background(), strings.NewReader(""))
	require.NoError(t, err)
	assert.Len(t, events, 0)
}

func TestSSE_DoneEvent(t *testing.T) {
	input := "data: [DONE]\n\n"
	events, err := ai.ParseSSE(context.Background(), strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "[DONE]", events[0].Data)
}

func TestSSE_OnlyNewlines(t *testing.T) {
	input := "\n\n\n"
	events, err := ai.ParseSSE(context.Background(), strings.NewReader(input))
	require.NoError(t, err)
	assert.Len(t, events, 0)
}

func TestSSE_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := ai.ParseSSE(ctx, strings.NewReader("data: hello\n\n"))
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestSSE_ReadError(t *testing.T) {
	_, err := ai.ParseSSE(context.Background(), &brokenReader{})
	require.Error(t, err)
}

type brokenReader struct{}

func (b *brokenReader) Read(p []byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}
