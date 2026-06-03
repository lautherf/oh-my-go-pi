package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/oh-my-pi/omp/pkg/grep"
	"github.com/oh-my-pi/omp/pkg/safefs"
	"github.com/oh-my-pi/omp/pkg/shell"
	"github.com/rs/zerolog/log"
)

// --- ReadTool ---

type ReadTool struct{}

func (t *ReadTool) Name() string { return "read" }

type readArgs struct {
	Path string `json:"path"`
}

func (t *ReadTool) Execute(ctx context.Context, argsJSON string) (string, error) {
	var args readArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("read: invalid args: %w", err)
	}
	if args.Path == "" {
		return "", fmt.Errorf("read: path is required")
	}

	log.Debug().Str("tool", "read").Str("path", args.Path).Msg("tool: execute")
	data, err := safefs.ReadFile(args.Path, safefs.WithMaxSize(2*1024*1024))
	if err != nil {
		log.Error().Err(err).Str("tool", "read").Str("path", args.Path).Msg("tool: failed")
		return "", fmt.Errorf("read: %w", err)
	}
	log.Debug().Str("tool", "read").Str("path", args.Path).Int("bytes", len(data)).Msg("tool: ok")
	return string(data), nil
}

// --- WriteTool ---

type WriteTool struct{}

func (t *WriteTool) Name() string { return "write" }

type writeArgs struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func (t *WriteTool) Execute(ctx context.Context, argsJSON string) (string, error) {
	var args writeArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("write: invalid args: %w", err)
	}
	if args.Path == "" {
		return "", fmt.Errorf("write: path is required")
	}

	log.Debug().Str("tool", "write").Str("path", args.Path).Int("bytes", len(args.Content)).Msg("tool: execute")
	if err := safefs.WriteFile(args.Path, []byte(args.Content)); err != nil {
		log.Error().Err(err).Str("tool", "write").Str("path", args.Path).Msg("tool: failed")
		return "", fmt.Errorf("write: %w", err)
	}
	log.Debug().Str("tool", "write").Str("path", args.Path).Msg("tool: ok")
	return fmt.Sprintf("written %d bytes to %s", len(args.Content), args.Path), nil
}

// --- BashTool ---

type BashTool struct{}

func (t *BashTool) Name() string { return "bash" }

type bashArgs struct {
	Command string `json:"command"`
	Dir     string `json:"dir,omitempty"`
	Timeout int    `json:"timeout,omitempty"`
}

func (t *BashTool) Execute(ctx context.Context, argsJSON string) (string, error) {
	var args bashArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("bash: invalid args: %w", err)
	}
	if args.Command == "" {
		return "", fmt.Errorf("bash: command is required")
	}

	var opts []shell.Option
	if args.Dir != "" {
		opts = append(opts, shell.WithDir(args.Dir))
	}

	log.Debug().Str("tool", "bash").Str("dir", args.Dir).Str("cmd", args.Command).Msg("tool: execute")
	out, err := shell.Execute(ctx, args.Command, opts...)
	if err != nil {
		log.Error().Err(err).Str("tool", "bash").Str("cmd", args.Command).Msg("tool: failed")
		return out, fmt.Errorf("bash exit: %w", err)
	}
	log.Debug().Str("tool", "bash").Int("output", len(out)).Msg("tool: ok")
	return strings.TrimSpace(out), nil
}

// --- GrepTool ---

type GrepTool struct{}

func (t *GrepTool) Name() string { return "grep" }

type grepArgs struct {
	Pattern string `json:"pattern"`
	Root    string `json:"root"`
	Include string `json:"include,omitempty"`
	Exclude string `json:"exclude,omitempty"`
}

func (t *GrepTool) Execute(ctx context.Context, argsJSON string) (string, error) {
	var args grepArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("grep: invalid args: %w", err)
	}
	if args.Pattern == "" {
		return "", fmt.Errorf("grep: pattern is required")
	}

	root := args.Root
	if root == "" {
		root, _ = os.Getwd()
	}

	opts := &grep.Options{}
	if args.Include != "" {
		opts.Include = args.Include
	}
	if args.Exclude != "" {
		opts.Exclude = args.Exclude
	}

	log.Debug().Str("tool", "grep").Str("pattern", args.Pattern).Str("root", root).Msg("tool: execute")
	matches, err := grep.Search(root, args.Pattern, opts)
	if err != nil {
		log.Error().Err(err).Str("tool", "grep").Msg("tool: failed")
		return "", fmt.Errorf("grep: %w", err)
	}

	if len(matches) == 0 {
		log.Debug().Str("tool", "grep").Msg("tool: no matches")
		return "no matches found", nil
	}

	var b strings.Builder
	for _, m := range matches {
		fmt.Fprintf(&b, "%s:%d: %s\n", m.Path, m.LineNumber, m.Line)
	}
	log.Debug().Str("tool", "grep").Int("matches", len(matches)).Msg("tool: ok")
	return b.String(), nil
}
