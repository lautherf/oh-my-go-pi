package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
)

type OllamaProvider struct {
	model  string
	baseURL string
	client *http.Client
}

func NewOllamaProvider(model, baseURL string) *OllamaProvider {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return &OllamaProvider{
		model:   model,
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  http.DefaultClient,
	}
}

func (p *OllamaProvider) Name() string { return "ollama" }

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

type ollamaResponse struct {
	Message   *ollamaMessage `json:"message,omitempty"`
	Done      bool           `json:"done"`
	Error     string         `json:"error,omitempty"`
}

func (p *OllamaProvider) Stream(ctx context.Context, req Request) (<-chan StreamEvent, error) {
	body := ollamaRequest{
		Model:    p.model,
		Stream:   true,
	}
	for _, m := range req.Messages {
		body.Messages = append(body.Messages, ollamaMessage{
			Role:    string(m.Role),
			Content: m.Content,
		})
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("ollama marshal: %w", err)
	}

	log.Debug().Str("provider", "ollama").Str("model", p.model).Int("messages", len(req.Messages)).Msg("stream request")

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/chat", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("ollama request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		log.Error().Err(err).Str("provider", "ollama").Msg("stream failed")
		return nil, fmt.Errorf("ollama post: %w", err)
	}

	if resp.StatusCode != 200 {
		errMsg := readError(resp.Body)
		resp.Body.Close()
		log.Error().Int("status", resp.StatusCode).Str("body", errMsg).Str("provider", "ollama").Msg("stream error response")
		return nil, fmt.Errorf("ollama %d: %s", resp.StatusCode, errMsg)
	}

	log.Debug().Str("provider", "ollama").Msg("stream connected")
	ch := make(chan StreamEvent)
	go p.readStream(ctx, resp.Body, ch)
	return ch, nil
}

func (p *OllamaProvider) readStream(ctx context.Context, r io.ReadCloser, ch chan<- StreamEvent) {
	defer r.Close()
	defer close(ch)

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		var resp ollamaResponse
		if err := json.Unmarshal([]byte(data), &resp); err != nil {
			continue
		}

		if resp.Error != "" {
			ch <- StreamEvent{Type: EventError, Content: resp.Error}
			return
		}

		if resp.Message != nil && resp.Message.Content != "" {
			ch <- StreamEvent{Type: EventText, Content: resp.Message.Content}
		}

		if resp.Done {
			ch <- StreamEvent{Type: EventDone, FinishReason: "stop"}
			return
		}
	}
}

func readError(r io.Reader) string {
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return strings.TrimSpace(buf.String())
}
