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

func TestOpenAI_StreamText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/chat/completions", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.Header.Get("Authorization"), "Bearer ")

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		w.Write([]byte("data: {\"id\":\"1\",\"choices\":[{\"delta\":{\"content\":\"Hello\"},\"finish_reason\":null}]}\n\n"))
		w.Write([]byte("data: {\"id\":\"2\",\"choices\":[{\"delta\":{\"content\":\" World\"},\"finish_reason\":null}]}\n\n"))
		w.Write([]byte("data: {\"id\":\"3\",\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer srv.Close()

	p := ai.NewOpenAIProvider("gpt-4o-mini", srv.URL, "sk-test")
	ch, err := p.Stream(context.Background(), ai.Request{
		Model:    "gpt-4o-mini",
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

func TestOpenAI_StreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{"error":{"message":"Incorrect API key"}}`))
	}))
	defer srv.Close()

	p := ai.NewOpenAIProvider("gpt-4o-mini", srv.URL, "sk-bad")
	_, err := p.Stream(context.Background(), ai.Request{
		Model:    "gpt-4o-mini",
		Messages: []ai.Message{ai.NewUserMessage("hi")},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Incorrect API key")
}

func TestOpenAI_NoKey(t *testing.T) {
	p := ai.NewOpenAIProvider("gpt-4o-mini", "", "")
	_, err := p.Stream(context.Background(), ai.Request{
		Model:    "gpt-4o-mini",
		Messages: []ai.Message{ai.NewUserMessage("hi")},
	})
	require.Error(t, err)
}

func TestOpenAI_Name(t *testing.T) {
	p := ai.NewOpenAIProvider("gpt-4o-mini", "", "sk-test")
	assert.Equal(t, "openai", p.Name())
}
