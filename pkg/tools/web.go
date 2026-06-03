package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/oh-my-pi/omp/pkg/web"
)

type WebSearchTool struct{}

func (t *WebSearchTool) Name() string { return "web_search" }

type webSearchArgs struct {
	Query string `json:"query"`
}

func (t *WebSearchTool) Execute(ctx context.Context, argsJSON string) (string, error) {
	var args webSearchArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("web_search: invalid args: %w", err)
	}
	if strings.TrimSpace(args.Query) == "" {
		return "", fmt.Errorf("web_search: query is required")
	}

	results, err := web.Search(ctx, args.Query)
	if err != nil {
		return "", fmt.Errorf("web_search: %w", err)
	}

	if len(results) == 0 {
		return "no search results", nil
	}

	var b strings.Builder
	for _, r := range results {
		fmt.Fprintf(&b, "- %s\n  %s\n  %s\n\n", r.Title, r.URL, r.Snippet)
	}
	return b.String(), nil
}

type WebFetchTool struct{}

func (t *WebFetchTool) Name() string { return "web_fetch" }

type webFetchArgs struct {
	URL string `json:"url"`
}

func (t *WebFetchTool) Execute(ctx context.Context, argsJSON string) (string, error) {
	var args webFetchArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("web_fetch: invalid args: %w", err)
	}
	if args.URL == "" {
		return "", fmt.Errorf("web_fetch: url is required")
	}

	content, err := web.Fetch(ctx, args.URL)
	if err != nil {
		return "", fmt.Errorf("web_fetch: %w", err)
	}

	if len(content) > 8000 {
		content = content[:8000] + "... (truncated)"
	}
	return content, nil
}
