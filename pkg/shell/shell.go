package shell

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type ExitError struct {
	code int
	err  error
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("exit status %d: %v", e.code, e.err)
}

func (e *ExitError) ExitCode() int {
	return e.code
}

func (e *ExitError) Unwrap() error {
	return e.err
}

type Options struct {
	Env []string
	Dir string
}

type Option func(*Options)

func WithEnv(kv ...string) Option {
	return func(o *Options) {
		o.Env = append(o.Env, kv...)
	}
}

func WithDir(dir string) Option {
	return func(o *Options) {
		o.Dir = dir
	}
}

func Execute(ctx context.Context, cmd string, opts ...Option) (string, error) {
	if strings.TrimSpace(cmd) == "" {
		return "", errors.New("empty command")
	}

	o := &Options{}
	for _, fn := range opts {
		fn(o)
	}

	c := exec.CommandContext(ctx, "sh", "-c", cmd)
	c.Stdin = os.Stdin

	if o.Env != nil {
		c.Env = append(os.Environ(), o.Env...)
	}
	if o.Dir != "" {
		c.Dir = o.Dir
	}

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()
	out := stdout.String()
	if stderr.Len() > 0 {
		if out != "" {
			out += "\n"
		}
		out += stderr.String()
	}

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return out, &ExitError{code: exitErr.ExitCode(), err: err}
		}
		return out, err
	}

	return out, nil
}
