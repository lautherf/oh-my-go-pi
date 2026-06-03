package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/oh-my-pi/omp/pkg/agent"
	"github.com/oh-my-pi/omp/pkg/ai"
	"github.com/rs/zerolog/log"
)

type SubAgentTool struct {
	provider ai.Provider
	req      ai.Request
	tools    []agent.Tool
}

func NewSubAgentTool(provider ai.Provider, req ai.Request, tools []agent.Tool) *SubAgentTool {
	return &SubAgentTool{provider: provider, req: req, tools: tools}
}

func (t *SubAgentTool) Name() string { return "subagent" }

type subAgentArgs struct {
	Task string `json:"task"`
}

func (t *SubAgentTool) Execute(ctx context.Context, argsJSON string) (string, error) {
	var args subAgentArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("subagent: invalid args: %w", err)
	}
	if strings.TrimSpace(args.Task) == "" {
		return "", fmt.Errorf("subagent: task is required")
	}

	log.Debug().Str("task", truncate(args.Task, 80)).Msg("subagent: spawning")

	sub := agent.New(t.provider, t.req)
	for _, tl := range t.tools {
		sub.RegisterTool(tl)
	}

	resp, err := sub.Run(ctx, args.Task)
	if err != nil {
		log.Error().Err(err).Msg("subagent: failed")
		return "", fmt.Errorf("subagent: %w", err)
	}

	log.Debug().Int("tokens", len(resp.Text)).Msg("subagent: done")
	return resp.Text, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
