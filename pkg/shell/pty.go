package shell

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/creack/pty"
)

type PTYSession struct {
	cmd      *exec.Cmd
	ptyFile  *os.File
	mu       sync.Mutex
	closed   bool
}

func NewPTY(ctx context.Context) (*PTYSession, error) {
	cmd := exec.CommandContext(ctx, "sh")
	cmd.Env = os.Environ()

	f, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 24, Cols: 80})
	if err != nil {
		return nil, err
	}

	return &PTYSession{
		cmd:     cmd,
		ptyFile: f,
	}, nil
}

func (s *PTYSession) Exec(command string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return "", os.ErrClosed
	}

	if _, err := s.ptyFile.Write([]byte(command + "\n")); err != nil {
		return "", err
	}

	// read until output settles — crude but works for simple commands
	var buf bytes.Buffer
	_ = s.ptyFile.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _ = io.Copy(&buf, s.ptyFile)
	return buf.String(), nil
}

func (s *PTYSession) Resize(rows, cols uint16) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return os.ErrClosed
	}

	return pty.Setsize(s.ptyFile, &pty.Winsize{Rows: rows, Cols: cols})
}

func (s *PTYSession) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true
	_ = s.cmd.Process.Kill()
	return s.ptyFile.Close()
}
