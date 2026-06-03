package hashline

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
)

type Op string

const (
	OpReplace Op = "="
	OpInsert  Op = "+"
	OpRemove  Op = "<"
)

type H struct {
	Operation Op
	Payload   string
	Anchor    string
}

func Hash(line string) string {
	h := sha256.Sum256([]byte(line))
	return fmt.Sprintf("%02x%02x", h[0], h[1])
}

func Parse(input string) (H, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return H{}, fmt.Errorf("empty hashline")
	}

	op := Op(input[0])
	switch op {
	case OpReplace, OpInsert, OpRemove:
	default:
		return H{}, fmt.Errorf("invalid operation %q", string(op))
	}

	rest := strings.TrimSpace(input[1:])
	if rest == "" && op != OpRemove {
		return H{}, fmt.Errorf("missing payload for %q operation", string(op))
	}

	return H{Operation: op, Payload: rest}, nil
}

func Apply(path string, h H) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("hashline read: %w", err)
	}

	if len(data) == 0 {
		// empty file: only insert makes sense
		if h.Operation != OpInsert {
			return fmt.Errorf("hashline: cannot %q on empty file", string(h.Operation))
		}
		return os.WriteFile(path, []byte(h.Payload+"\n"), 0644)
	}

	lines := strings.Split(string(data), "\n")
	// remove trailing empty line from split
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	anchorIdx := -1
	for i, line := range lines {
		if Hash(line) == h.Anchor {
			anchorIdx = i
			break
		}
	}

	if h.Anchor != "" && anchorIdx == -1 {
		return fmt.Errorf("hashline: anchor %q not found", h.Anchor)
	}

	switch h.Operation {
	case OpReplace:
		lines[anchorIdx] = h.Payload
	case OpInsert:
		newLines := make([]string, 0, len(lines)+1)
		newLines = append(newLines, lines[:anchorIdx]...)
		newLines = append(newLines, h.Payload)
		newLines = append(newLines, lines[anchorIdx:]...)
		lines = newLines
	case OpRemove:
		lines = append(lines[:anchorIdx], lines[anchorIdx+1:]...)
	}

	out := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(path, []byte(out), 0644)
}


