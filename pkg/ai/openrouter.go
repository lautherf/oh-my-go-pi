package ai

import (
	"context"
	"fmt"
	"net/http"
	"os"
)

type OpenRouterProvider struct {
	model   string
	baseURL string
	apiKey  string
	client  *http.Client
}

func NewOpenRouterProvider(model, apiKey string) *OpenRouterProvider {
	if model == "" {
		model = "openrouter/free"
	}
	if apiKey == "" {
		apiKey = os.Getenv("OPENROUTER_API_KEY")
	}
	return &OpenRouterProvider{
		model:   model,
		baseURL: "https://openrouter.ai/api/v1",
		apiKey:  apiKey,
		client:  http.DefaultClient,
	}
}

func (p *OpenRouterProvider) Name() string { return "openrouter" }

func (p *OpenRouterProvider) Stream(ctx context.Context, req Request) (<-chan StreamEvent, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("openrouter: OPENROUTER_API_KEY not set")
	}
	// Delegate to OpenAI provider with OpenRouter base URL
	o := NewOpenAIProvider(p.model, p.baseURL, p.apiKey)
	o.client = p.client
	return o.Stream(ctx, req)
}
