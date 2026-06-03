package ai

import (
	"context"
	"fmt"
)

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

type Message struct {
	Role       Role
	Content    string
	ToolCalls  []ToolCall
	ToolCallID string
}

func NewUserMessage(content string) Message {
	return Message{Role: RoleUser, Content: content}
}

func NewSystemMessage(content string) Message {
	return Message{Role: RoleSystem, Content: content}
}

func NewAssistantMessage(content string) Message {
	return Message{Role: RoleAssistant, Content: content}
}

func NewToolMessage(content, toolCallID string) Message {
	return Message{Role: RoleTool, Content: content, ToolCallID: toolCallID}
}

func (m Message) String() string {
	return fmt.Sprintf("[%s] %s", m.Role, m.Content)
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

func (t ToolCall) String() string {
	return fmt.Sprintf("%s(%s)", t.Function.Name, t.Function.Arguments)
}

type StreamEventType string

const (
	EventText     StreamEventType = "text"
	EventToolCall StreamEventType = "tool_call"
	EventDone     StreamEventType = "done"
	EventError    StreamEventType = "error"
)

type StreamEvent struct {
	Type         StreamEventType
	Content      string
	ToolCall     *ToolCall
	FinishReason string
	Usage        *Usage
}

type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type Request struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	System      string    `json:"system,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Tools       []ToolDef `json:"tools,omitempty"`
	Stream      bool      `json:"stream"`
}

type ToolDef struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
}

func (r Request) Validate() error {
	if len(r.Messages) == 0 {
		return fmt.Errorf("at least one message is required")
	}
	if r.Model == "" {
		return fmt.Errorf("model is required")
	}
	return nil
}

type Provider interface {
	Name() string
	Stream(ctx context.Context, req Request) (<-chan StreamEvent, error)
}
