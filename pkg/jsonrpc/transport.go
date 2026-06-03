package jsonrpc

import (
	"bufio"
	"io"
	"os/exec"
)

type StdioTransport struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	reader *bufio.Reader
}

func NewStdioTransport(cmd *exec.Cmd) (*StdioTransport, error) {
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &StdioTransport{
		cmd:    cmd,
		stdin:  stdin,
		reader: bufio.NewReader(stdout),
	}, nil
}

func (t *StdioTransport) Send(msg *Message) error {
	return EncodeMessage(t.stdin, msg)
}

func (t *StdioTransport) Receive() (*Message, error) {
	return DecodeMessage(t.reader)
}

func (t *StdioTransport) Close() error {
	t.stdin.Close()
	if t.cmd != nil && t.cmd.Process != nil {
		t.cmd.Process.Kill()
		t.cmd.Wait()
	}
	return nil
}
