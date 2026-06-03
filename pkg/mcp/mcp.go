package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type Client struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	scanner *bufio.Scanner
	closed  bool
	name    string
	version string
	msgID   int
}

type ClientOption func(*Client)

func WithClientName(name string) ClientOption {
	return func(c *Client) { c.name = name }
}

func WithClientVersion(v string) ClientOption {
	return func(c *Client) { c.version = v }
}

type ToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func NewClient(ctx context.Context, cmd *exec.Cmd, opts ...ClientOption) (*Client, error) {
	c := &Client{
		cmd:     cmd,
		name:    "omp",
		version: "0.1.0",
	}
	for _, o := range opts {
		o(c)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("mcp stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("mcp stdout: %w", err)
	}
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("mcp start: %w", err)
	}

	c.stdin = stdin
	c.stdout = stdout
	c.scanner = bufio.NewScanner(stdout)

	// initialize
	if _, err := c.sendRequest(ctx, "initialize", map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]string{
			"name":    c.name,
			"version": c.version,
		},
	}); err != nil {
		c.Close()
		return nil, fmt.Errorf("mcp init: %w", err)
	}

	return c, nil
}

func (c *Client) sendRequest(ctx context.Context, method string, params any) (json.RawMessage, error) {
	c.msgID++
	id := c.msgID

	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
	}
	if params != nil {
		req["params"] = params
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	if _, err := c.stdin.Write(append(payload, '\n')); err != nil {
		return nil, fmt.Errorf("mcp write: %w", err)
	}

	for c.scanner.Scan() {
		line := strings.TrimSpace(c.scanner.Text())
		if line == "" {
			continue
		}

		var resp struct {
			ID     int              `json:"id"`
			Result json.RawMessage  `json:"result"`
			Error  *json.RawMessage `json:"error,omitempty"`
		}
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			continue
		}

		if resp.ID == id {
			if resp.Error != nil {
				return nil, fmt.Errorf("mcp error: %s", string(*resp.Error))
			}
			return resp.Result, nil
		}
	}

	if err := c.scanner.Err(); err != nil {
		return nil, fmt.Errorf("mcp read: %w", err)
	}
	return nil, fmt.Errorf("mcp: no response for request %d", id)
}

func (c *Client) ListTools(ctx context.Context) ([]ToolInfo, error) {
	result, err := c.sendRequest(ctx, "tools/list", nil)
	if err != nil {
		return nil, err
	}

	var list struct {
		Tools []ToolInfo `json:"tools"`
	}
	if err := json.Unmarshal(result, &list); err != nil {
		return nil, fmt.Errorf("mcp list tools parse: %w", err)
	}
	return list.Tools, nil
}

func (c *Client) CallTool(ctx context.Context, name string, args json.RawMessage) (string, error) {
	params := map[string]any{
		"name": name,
	}
	if args != nil {
		params["arguments"] = args
	}

	result, err := c.sendRequest(ctx, "tools/call", params)
	if err != nil {
		return "", err
	}

	var callResult struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(result, &callResult); err != nil {
		return "", fmt.Errorf("mcp call parse: %w", err)
	}

	var texts []string
	for _, item := range callResult.Content {
		if item.Type == "text" {
			texts = append(texts, item.Text)
		}
	}
	return strings.Join(texts, "\n"), nil
}

func (c *Client) Close() error {
	if c.closed {
		return nil
	}
	c.closed = true
	if c.stdin != nil {
		c.stdin.Close()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		c.cmd.Process.Kill()
		c.cmd.Wait()
	}
	return nil
}
