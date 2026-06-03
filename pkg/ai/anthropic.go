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

type AnthropicProvider struct {
	model   string
	baseURL string
	apiKey  string
	client  *http.Client
}

func NewAnthropicProvider(model, baseURL, apiKey string) *AnthropicProvider {
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	return &AnthropicProvider{
		model:   model,
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		client:  http.DefaultClient,
	}
}

func (p *AnthropicProvider) Name() string { return "anthropic" }

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicRequest struct {
	Model       string             `json:"model"`
	Messages    []anthropicMessage `json:"messages"`
	System      string             `json:"system,omitempty"`
	MaxTokens   int                `json:"max_tokens"`
	Stream      bool               `json:"stream"`
}

type anthropicChunk struct {
	Type  string           `json:"type"`
	Delta *anthropicDelta  `json:"delta,omitempty"`
	Index *int             `json:"index,omitempty"`
}

type anthropicDelta struct {
	Text        string `json:"text"`
	StopReason  string `json:"stop_reason,omitempty"`
	StopString  string `json:"stop_string,omitempty"`
	Type        string `json:"type,omitempty"`
}

type anthropicContentBlockDelta struct {
	Type  string         `json:"type"`
	Index int            `json:"index"`
	Delta *anthropicDelta `json:"delta"`
}

type anthropicError struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (p *AnthropicProvider) Stream(ctx context.Context, req Request) (<-chan StreamEvent, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("anthropic: ANTHROPIC_API_KEY not set")
	}

	body := anthropicRequest{
		Model:     p.model,
		MaxTokens: 4096,
		Stream:    true,
	}
	if req.MaxTokens > 0 {
		body.MaxTokens = req.MaxTokens
	}

	// Separate system prompt
	var system string
	for _, m := range req.Messages {
		if m.Role == RoleSystem {
			if system != "" {
				system += "\n"
			}
			system += m.Content
		} else {
			body.Messages = append(body.Messages, anthropicMessage{
				Role:    string(m.Role),
				Content: m.Content,
			})
		}
	}
	if system != "" {
		body.System = system
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("anthropic marshal: %w", err)
	}

	log.Debug().Str("provider", "anthropic").Str("model", p.model).Int("messages", len(req.Messages)).Msg("stream request")

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/messages", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("anthropic request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		log.Error().Err(err).Str("provider", "anthropic").Msg("stream failed")
		return nil, fmt.Errorf("anthropic post: %w", err)
	}

	if resp.StatusCode != 200 {
		var errResp anthropicError
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if json.Unmarshal(body, &errResp) == nil && errResp.Error.Message != "" {
			log.Error().Int("status", resp.StatusCode).Str("error", errResp.Error.Message).Str("provider", "anthropic").Msg("stream error response")
			return nil, fmt.Errorf("anthropic %d: %s", resp.StatusCode, errResp.Error.Message)
		}
		log.Error().Int("status", resp.StatusCode).Str("body", strings.TrimSpace(string(body))).Str("provider", "anthropic").Msg("stream error response")
		return nil, fmt.Errorf("anthropic %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	log.Debug().Str("provider", "anthropic").Msg("stream connected")
	ch := make(chan StreamEvent)
	go p.readStream(ctx, resp.Body, ch)
	return ch, nil
}

func (p *AnthropicProvider) readStream(ctx context.Context, r io.ReadCloser, ch chan<- StreamEvent) {
	defer r.Close()
	defer close(ch)

	scanner := bufio.NewScanner(r)
	var currentEvent string

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Text()

		if strings.HasPrefix(line, "event: ") {
			currentEvent = strings.TrimPrefix(line, "event: ")
			continue
		}

		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			switch currentEvent {
			case "content_block_delta":
				var block anthropicContentBlockDelta
				if err := json.Unmarshal([]byte(data), &block); err != nil {
					continue
				}
				if block.Delta != nil && block.Delta.Text != "" {
					ch <- StreamEvent{Type: EventText, Content: block.Delta.Text}
				}

			case "message_delta":
				var msg anthropicChunk
				if err := json.Unmarshal([]byte(data), &msg); err != nil {
					continue
				}
				if msg.Delta != nil && msg.Delta.StopReason != "" {
					ch <- StreamEvent{Type: EventDone, FinishReason: msg.Delta.StopReason}
					return
				}

			case "message_stop":
				ch <- StreamEvent{Type: EventDone, FinishReason: "end_turn"}
				return

			case "error":
				var errResp anthropicError
				if json.Unmarshal([]byte(data), &errResp) == nil {
					ch <- StreamEvent{Type: EventError, Content: errResp.Error.Message}
				}
				return
			}
		}
	}

	ch <- StreamEvent{Type: EventDone, FinishReason: "stop"}
}
