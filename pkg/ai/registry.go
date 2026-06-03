package ai

import (
	"fmt"
	"net"
	"os"
	"sync"
)

type Registry struct {
	mu   sync.RWMutex
	providers map[string]Provider
}

func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

func (r *Registry) Register(name string, p Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[name] = p
}

func (r *Registry) Get(name string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider %q not found", name)
	}
	return p, nil
}

func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.providers))
	for n := range r.providers {
		names = append(names, n)
	}
	return names
}

func (r *Registry) AutoDiscover() int {
	count := 0

	// Ollama: check localhost:11434
	if portOpen("localhost:11434") {
		r.Register("ollama", NewOllamaProvider("llama3.2", "http://localhost:11434"))
		count++
	}

	// OpenAI: check OPENAI_API_KEY
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		r.Register("openai", NewOpenAIProvider("gpt-4o", "", key))
		count++
	}

	// Anthropic: check ANTHROPIC_API_KEY
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		r.Register("anthropic", NewAnthropicProvider("claude-sonnet-4-20250514", "", key))
		count++
	}

	// OpenRouter: check OPENROUTER_API_KEY
	if key := os.Getenv("OPENROUTER_API_KEY"); key != "" {
		r.Register("openrouter", NewOpenRouterProvider("openrouter/free", key))
		count++
	}

	return count
}

func portOpen(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr, 1000)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
