package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/oh-my-pi/omp/pkg/memory"
)

type MemoryRememberTool struct {
	store *memory.Store
}

func NewMemoryRememberTool(store *memory.Store) *MemoryRememberTool {
	return &MemoryRememberTool{store: store}
}

func (t *MemoryRememberTool) Name() string { return "memory_remember" }

type rememberArgs struct {
	Content string `json:"content"`
}

func (t *MemoryRememberTool) Execute(ctx context.Context, argsJSON string) (string, error) {
	var args rememberArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("memory_remember: invalid args: %w", err)
	}
	if strings.TrimSpace(args.Content) == "" {
		return "", fmt.Errorf("memory_remember: content is required")
	}

	if err := t.store.Remember(ctx, args.Content); err != nil {
		return "", fmt.Errorf("memory_remember: %w", err)
	}
	return "remembered", nil
}

type MemoryRecallTool struct {
	store *memory.Store
}

func NewMemoryRecallTool(store *memory.Store) *MemoryRecallTool {
	return &MemoryRecallTool{store: store}
}

func (t *MemoryRecallTool) Name() string { return "memory_recall" }

type recallArgs struct {
	Query string `json:"query"`
}

func (t *MemoryRecallTool) Execute(ctx context.Context, argsJSON string) (string, error) {
	var args recallArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("memory_recall: invalid args: %w", err)
	}
	if strings.TrimSpace(args.Query) == "" {
		return "", fmt.Errorf("memory_recall: query is required")
	}

	results, err := t.store.Recall(ctx, args.Query)
	if err != nil {
		return "", fmt.Errorf("memory_recall: %w", err)
	}

	if len(results) == 0 {
		return "no memories found", nil
	}

	var b strings.Builder
	for _, m := range results {
		fmt.Fprintf(&b, "- %s\n", m.Content)
	}
	return b.String(), nil
}
