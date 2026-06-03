package ai

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
)

type SSEEvent struct {
	ID   string
	Type string
	Data string
}

func ParseSSE(ctx context.Context, r io.Reader) ([]SSEEvent, error) {
	var events []SSEEvent
	scanner := bufio.NewScanner(r)

	var current SSEEvent
	var dataBuf strings.Builder

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		line := scanner.Text()

		switch {
		case line == "":
			// empty line = event delimiter
			if dataBuf.Len() > 0 {
				current.Data = strings.TrimSuffix(dataBuf.String(), "\n")
				events = append(events, current)
			}
			current = SSEEvent{}
			dataBuf.Reset()

		case strings.HasPrefix(line, ":"):
			// comment, skip

		case strings.HasPrefix(line, "id:"):
			current.ID = strings.TrimSpace(line[3:])

		case strings.HasPrefix(line, "event:"):
			current.Type = strings.TrimSpace(line[6:])

		case strings.HasPrefix(line, "data:"):
			val := strings.TrimPrefix(line, "data:")
			val = strings.TrimSpace(val)
			if dataBuf.Len() > 0 {
				dataBuf.WriteString("\n")
			}
			dataBuf.WriteString(val)
		}
	}

	// emit remaining if no trailing blank line
	if dataBuf.Len() > 0 {
		current.Data = strings.TrimSuffix(dataBuf.String(), "\n")
		events = append(events, current)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("sse read error: %w", err)
	}

	return events, nil
}
