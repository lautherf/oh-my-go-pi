package ai_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/oh-my-pi/omp/pkg/ai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnthropic_StreamText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/messages", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.Header.Get("x-api-key"), "sk-ant-test")

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		w.Write([]byte("event: content_block_delta\n"))
		w.Write([]byte("data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"Hello\"}}\n\n"))
		w.Write([]byte("event: content_block_delta\n"))
		w.Write([]byte("data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\" World\"}}\n\n"))
		w.Write([]byte("event: message_delta\n"))
		w.Write([]byte("data: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"}}\n\n"))
		w.Write([]byte("event: message_stop\n"))
		w.Write([]byte("data: {\"type\":\"message_stop\"}\n\n"))
	}))
	defer srv.Close()

	p := ai.NewAnthropicProvider("claude-sonnet-4-20250514", srv.URL, "sk-ant-test")
	ch, err := p.Stream(context.Background(), ai.Request{
		Model:    "claude-sonnet-4-20250514",
		Messages: []ai.Message{ai.NewUserMessage("hi")},
	})
	require.NoError(t, err)

	var text string
	for e := range ch {
		if e.Type == ai.EventText {
			text += e.Content
		}
	}
	assert.Equal(t, "Hello World", text)
}

func TestAnthropic_StreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{"error":{"message":"Invalid API key"}}`))
	}))
	defer srv.Close()

	p := ai.NewAnthropicProvider("claude-sonnet-4-20250514", srv.URL, "sk-bad")
	_, err := p.Stream(context.Background(), ai.Request{
		Model:    "claude-sonnet-4-20250514",
		Messages: []ai.Message{ai.NewUserMessage("hi")},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid API key")
}

func TestAnthropic_NoKey(t *testing.T) {
	p := ai.NewAnthropicProvider("claude-sonnet-4-20250514", "", "")
	_, err := p.Stream(context.Background(), ai.Request{
		Model:    "claude-sonnet-4-20250514",
		Messages: []ai.Message{ai.NewUserMessage("hi")},
	})
	require.Error(t, err)
}

func TestAnthropic_Name(t *testing.T) {
	p := ai.NewAnthropicProvider("claude-sonnet-4-20250514", "", "sk-ant-test")
	assert.Equal(t, "anthropic", p.Name())
}

func TestAnthropic_StreamWithSystem(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/messages", r.URL.Path)

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		w.Write([]byte("event: content_block_delta\n"))
		w.Write([]byte("data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"OK\"}}\n\n"))
		w.Write([]byte("event: message_stop\n"))
		w.Write([]byte("data: {\"type\":\"message_stop\"}\n\n"))
	}))
	defer srv.Close()

	p := ai.NewAnthropicProvider("claude-sonnet-4-20250514", srv.URL, "sk-ant-test")
	ch, err := p.Stream(context.Background(), ai.Request{
		Model:    "claude-sonnet-4-20250514",
		Messages: []ai.Message{ai.NewSystemMessage("be concise"), ai.NewUserMessage("hi")},
	})
	require.NoError(t, err)

	var text string
	for e := range ch {
		if e.Type == ai.EventText {
			text += e.Content
		}
	}
	assert.Equal(t, "OK", text)
}
