package tui

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/charmbracelet/glamour"
	"github.com/oh-my-pi/omp/pkg/ai"
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (m ChatMessage) Render() string {
	prefix := fmt.Sprintf("[%s] ", m.Role)

	if m.Role == "assistant" {
		rendered, err := glamour.Render(m.Content, "dark")
		if err == nil {
			return prefix + strings.TrimSpace(rendered)
		}
	}
	return prefix + m.Content
}

type Session struct {
	provider ai.Provider
	req      ai.Request
	messages []ChatMessage
	mu       sync.Mutex
}

func NewSession(provider ai.Provider, req ai.Request) (*Session, error) {
	return &Session{
		provider: provider,
		req:      req,
	}, nil
}

func (s *Session) MessageCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.messages)
}

func (s *Session) Messages() []ChatMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	msgs := make([]ChatMessage, len(s.messages))
	copy(msgs, s.messages)
	return msgs
}

func (s *Session) AddMessage(role, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append(s.messages, ChatMessage{Role: role, Content: content})
}

func (s *Session) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = nil
}

func (s *Session) Send(ctx context.Context, input string) (string, error) {
	if strings.TrimSpace(input) == "" {
		return "", fmt.Errorf("empty input")
	}

	s.mu.Lock()
	s.messages = append(s.messages, ChatMessage{Role: "user", Content: input})
	s.mu.Unlock()

	aiMsgs := s.toAIMessages()

	req := s.req
	req.Messages = aiMsgs
	req.Stream = true

	ch, err := s.provider.Stream(ctx, req)
	if err != nil {
		return "", fmt.Errorf("tui: %w", err)
	}

	var resp strings.Builder
	for event := range ch {
		switch event.Type {
		case ai.EventText:
			resp.WriteString(event.Content)
		case ai.EventError:
			return "", fmt.Errorf("tui error: %s", event.Content)
		case ai.EventDone:
			text := resp.String()
			s.mu.Lock()
			s.messages = append(s.messages, ChatMessage{Role: "assistant", Content: text})
			s.mu.Unlock()
			return text, nil
		}
	}

	return "", fmt.Errorf("tui: stream ended without completion")
}

func (s *Session) toAIMessages() []ai.Message {
	s.mu.Lock()
	defer s.mu.Unlock()

	var msgs []ai.Message
	for _, m := range s.messages {
		role := ai.Role(m.Role)
		switch role {
		case ai.RoleUser:
			msgs = append(msgs, ai.NewUserMessage(m.Content))
		case ai.RoleAssistant:
			msgs = append(msgs, ai.NewAssistantMessage(m.Content))
		default:
			msgs = append(msgs, ai.NewUserMessage(m.Content))
		}
	}
	return msgs
}
