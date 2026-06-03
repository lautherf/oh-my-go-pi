package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/oh-my-pi/omp/pkg/hashline"
)

type EditTool struct{}

func (t *EditTool) Name() string { return "edit" }

type editArgs struct {
	Path    string `json:"path"`
	Anchor  string `json:"anchor"`
	Op      string `json:"op"`
	Payload string `json:"payload,omitempty"`
}

func (t *EditTool) Execute(ctx context.Context, argsJSON string) (string, error) {
	var args editArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("edit: invalid args: %w", err)
	}
	if args.Path == "" {
		return "", fmt.Errorf("edit: path is required")
	}
	if args.Anchor == "" {
		return "", fmt.Errorf("edit: anchor is required")
	}

	h := hashline.H{
		Operation: hashline.Op(args.Op),
		Anchor:    args.Anchor,
		Payload:   args.Payload,
	}

	if err := hashline.Apply(args.Path, h); err != nil {
		return "", fmt.Errorf("edit: %w", err)
	}
	return fmt.Sprintf("applied %s at anchor %s on %s", args.Op, args.Anchor, args.Path), nil
}
