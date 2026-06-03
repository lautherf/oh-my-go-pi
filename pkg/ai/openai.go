package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
)

type OpenAIProvider struct {
	model   string
	baseURL string
	apiKey  string
	client  *http.Client
}

func NewOpenAIProvider(model, baseURL, apiKey string) *OpenAIProvider {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	return &OpenAIProvider{
		model:   model,
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		client:  http.DefaultClient,
	}
}

func (p *OpenAIProvider) Name() string { return "openai" }

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

type openAIChunk struct {
	Choices []openAIChoice `json:"choices"`
}

type openAIChoice struct {
	Delta struct {
		Content string `json:"content"`
	} `json:"delta"`
	FinishReason *string `json:"finish_reason"`
}

type openAIError struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (p *OpenAIProvider) Stream(ctx context.Context, req Request) (<-chan StreamEvent, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("openai: OPENAI_API_KEY not set")
	}

	body := openAIRequest{
		Model:  p.model,
		Stream: true,
	}
	for _, m := range req.Messages {
		body.Messages = append(body.Messages, openAIMessage{
			Role:    string(m.Role),
			Content: m.Content,
		})
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("openai marshal: %w", err)
	}

	log.Debug().Str("provider", "openai").Str("model", p.model).Int("messages", len(req.Messages)).Msg("stream request")

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("openai request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		log.Error().Err(err).Str("provider", "openai").Msg("stream failed")
		return nil, fmt.Errorf("openai post: %w", err)
	}

	if resp.StatusCode != 200 {
		var errResp openAIError
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if json.Unmarshal(body, &errResp) == nil && errResp.Error.Message != "" {
			log.Error().Int("status", resp.StatusCode).Str("error", errResp.Error.Message).Str("provider", "openai").Msg("stream error response")
			return nil, fmt.Errorf("openai %d: %s", resp.StatusCode, errResp.Error.Message)
		}
		log.Error().Int("status", resp.StatusCode).Str("body", strings.TrimSpace(string(body))).Str("provider", "openai").Msg("stream error response")
		return nil, fmt.Errorf("openai %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	log.Debug().Str("provider", "openai").Msg("stream connected")
	ch := make(chan StreamEvent)
	go p.readStream(ctx, resp.Body, ch)
	return ch, nil
}

func (p *OpenAIProvider) readStream(ctx context.Context, r io.ReadCloser, ch chan<- StreamEvent) {
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
		if line == "" {
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		if data == "[DONE]" {
			ch <- StreamEvent{Type: EventDone, FinishReason: "stop"}
			return
		}

		var chunk openAIChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		choice := chunk.Choices[0]

		if choice.Delta.Content != "" {
			ch <- StreamEvent{Type: EventText, Content: choice.Delta.Content}
		}

		if choice.FinishReason != nil {
			ch <- StreamEvent{Type: EventDone, FinishReason: *choice.FinishReason}
			return
		}
	}
}
