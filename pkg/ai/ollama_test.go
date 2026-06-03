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

func TestOllama_StreamText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/chat", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		w.Write([]byte(`data: {"message":{"role":"assistant","content":"Hello"},"done":false}` + "\n\n"))
		w.Write([]byte(`data: {"message":{"role":"assistant","content":" World"},"done":false}` + "\n\n"))
		w.Write([]byte(`data: {"message":{"role":"assistant","content":""},"done":true}` + "\n\n"))
	}))
	defer srv.Close()

	p := ai.NewOllamaProvider("test-model", srv.URL)
	ch, err := p.Stream(context.Background(), ai.Request{
		Model:    "test-model",
		Messages: []ai.Message{ai.NewUserMessage("hi")},
	})
	require.NoError(t, err)

	var text string
	for e := range ch {
		if e.Type == ai.EventText {
			text += e.Content
		}
		if e.Type == ai.EventDone {
			assert.Equal(t, "stop", e.FinishReason)
		}
	}
	assert.Equal(t, "Hello World", text)
}

func TestOllama_StreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		w.Write([]byte(`{"error":"bad request"}`))
	}))
	defer srv.Close()

	p := ai.NewOllamaProvider("test-model", srv.URL)
	_, err := p.Stream(context.Background(), ai.Request{
		Model:    "test-model",
		Messages: []ai.Message{ai.NewUserMessage("hi")},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad request")
}

func TestOllama_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	p := ai.NewOllamaProvider("test-model", srv.URL)
	_, err := p.Stream(context.Background(), ai.Request{
		Model:    "test-model",
		Messages: []ai.Message{ai.NewUserMessage("hi")},
	})
	require.Error(t, err)
}

func TestOllama_ModelName(t *testing.T) {
	p := ai.NewOllamaProvider("llama3.2:latest", "http://localhost:11434")
	assert.Equal(t, "ollama", p.Name())
}
