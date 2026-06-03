package shell_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/oh-my-pi/omp/pkg/shell"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecute_Echo(t *testing.T) {
	t.Parallel()
	out, err := shell.Execute(context.Background(), "echo hello omp")
	require.NoError(t, err)
	assert.Contains(t, out, "hello omp")
}

func TestExecute_Stderr(t *testing.T) {
	t.Parallel()
	out, err := shell.Execute(context.Background(), "echo stderr >&2 && echo stdout")
	require.NoError(t, err)
	assert.Contains(t, out, "stdout")
	assert.Contains(t, out, "stderr")
}

func TestExecute_ExitError(t *testing.T) {
	t.Parallel()
	_, err := shell.Execute(context.Background(), "exit 42")
	require.Error(t, err)
	var exitErr *shell.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, 42, exitErr.ExitCode())
}

func TestExecute_NotFound(t *testing.T) {
	t.Parallel()
	_, err := shell.Execute(context.Background(), "nonexistent_cmd_12345")
	require.Error(t, err)
}

func TestExecute_WithEnv(t *testing.T) {
	t.Parallel()
	out, err := shell.Execute(context.Background(), "echo $MY_VAR",
		shell.WithEnv("MY_VAR=tdd_test"))
	require.NoError(t, err)
	assert.Contains(t, out, "tdd_test")
}

func TestExecute_WithDir(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "marker.txt"), []byte("ok"), 0644))

	out, err := shell.Execute(context.Background(), "cat marker.txt",
		shell.WithDir(tmp))
	require.NoError(t, err)
	assert.Contains(t, out, "ok")
}

func TestExecute_Timeout(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := shell.Execute(ctx, "sleep 10")
	require.Error(t, err)
	// exec.CommandContext kills with SIGKILL; either context error or ExitError is fine
	if !errors.Is(err, context.DeadlineExceeded) {
		var exitErr *shell.ExitError
		if errors.As(err, &exitErr) {
			assert.Less(t, exitErr.ExitCode(), 0, "killed by signal on timeout")
		} else {
			t.Fatalf("expected context error or ExitError, got: %v (%T)", err, err)
		}
	}
}

func TestExecute_TrimTrailingNewline(t *testing.T) {
	t.Parallel()
	out, err := shell.Execute(context.Background(), "printf 'line1\nline2'")
	require.NoError(t, err)
	assert.Equal(t, "line1\nline2", out)
}

func TestExecute_MultiLine(t *testing.T) {
	t.Parallel()
	script := "echo line1 && echo line2 && echo line3"
	out, err := shell.Execute(context.Background(), script)
	require.NoError(t, err)
	assert.Contains(t, out, "line1")
	assert.Contains(t, out, "line2")
	assert.Contains(t, out, "line3")
}

func TestExecute_WhitespaceArgs(t *testing.T) {
	t.Parallel()
	out, err := shell.Execute(context.Background(), `echo "arg with spaces"`)
	require.NoError(t, err)
	assert.Contains(t, out, "arg with spaces")
}

func TestExecute_EmptyCommand(t *testing.T) {
	t.Parallel()
	_, err := shell.Execute(context.Background(), "")
	require.Error(t, err)
}

func TestExecute_Pipe(t *testing.T) {
	t.Parallel()
	out, err := shell.Execute(context.Background(), "echo 'abc\n123\nabc' | grep abc")
	require.NoError(t, err)
	assert.Contains(t, out, "abc")
	assert.NotContains(t, out, "123")
}

func TestExecute_WorkingDirectoryDefault(t *testing.T) {
	t.Parallel()
	cwd, _ := os.Getwd()
	out, err := shell.Execute(context.Background(), "pwd")
	require.NoError(t, err)
	assert.Contains(t, out, filepath.Base(cwd))
}
