package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

type SearchOptions struct {
	SearchURL string
	Client    *http.Client
}

type SearchOption func(*SearchOptions)

func WithSearchURL(u string) SearchOption {
	return func(o *SearchOptions) {
		o.SearchURL = u
	}
}

func defaultSearchOpts() *SearchOptions {
	return &SearchOptions{
		SearchURL: "https://api.duckduckgo.com/?q=%s&format=json",
		Client: &http.Client{Timeout: 10 * time.Second},
	}
}

func Fetch(ctx context.Context, rawURL string) (string, error) {
	if rawURL == "" {
		return "", fmt.Errorf("empty URL")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", fmt.Errorf("invalid URL: %s", rawURL)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("fetch request: %w", err)
	}
	req.Header.Set("User-Agent", "omp/0.1")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("fetch %s: HTTP %d", rawURL, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("fetch read: %w", err)
	}

	return strings.TrimSpace(string(body)), nil
}

func Search(ctx context.Context, query string, opts ...SearchOption) ([]SearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("empty search query")
	}

	cfg := defaultSearchOpts()
	for _, o := range opts {
		o(cfg)
	}

	searchURL := cfg.SearchURL
	if strings.Contains(searchURL, "%s") {
		searchURL = strings.Replace(searchURL, "%s", url.QueryEscape(query), 1)
	} else {
		sep := "?"
		if strings.Contains(searchURL, "?") {
			sep = "&"
		}
		searchURL = fmt.Sprintf("%s%sq=%s", searchURL, sep, url.QueryEscape(query))
	}
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("search request: %w", err)
	}
	req.Header.Set("User-Agent", "omp/0.1")

	resp, err := cfg.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("search: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("search read: %w", err)
	}

	var result struct {
		Results []SearchResult `json:"results"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		// try reading as array directly
		var results []SearchResult
		if err2 := json.Unmarshal(body, &results); err2 != nil {
			return nil, fmt.Errorf("search parse: %w", err)
		}
		return results, nil
	}

	return result.Results, nil
}
