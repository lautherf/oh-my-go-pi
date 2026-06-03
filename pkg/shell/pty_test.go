package shell_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/oh-my-pi/omp/pkg/shell"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPTY_Echo(t *testing.T) {
	if os.Getenv("CI") == "" && os.Getenv("OMP_TEST_PTY") == "" {
		t.Skip("PTY tests require terminal; set OMP_TEST_PTY=1")
	}
	sess, err := shell.NewPTY(context.Background())
	require.NoError(t, err)
	defer sess.Close()

	out, err := sess.Exec("echo hello pty")
	require.NoError(t, err)
	assert.Contains(t, out, "hello pty")
}

func TestPTY_InteractiveCommand(t *testing.T) {
	if os.Getenv("CI") == "" && os.Getenv("OMP_TEST_PTY") == "" {
		t.Skip("PTY tests require terminal; set OMP_TEST_PTY=1")
	}
	sess, err := shell.NewPTY(context.Background())
	require.NoError(t, err)
	defer sess.Close()

	out, err := sess.Exec("echo line1 && echo line2")
	require.NoError(t, err)
	assert.Contains(t, out, "line1")
	assert.Contains(t, out, "line2")
}

func TestPTY_Resize(t *testing.T) {
	if os.Getenv("CI") == "" && os.Getenv("OMP_TEST_PTY") == "" {
		t.Skip("PTY tests require terminal; set OMP_TEST_PTY=1")
	}
	sess, err := shell.NewPTY(context.Background())
	require.NoError(t, err)
	defer sess.Close()

	err = sess.Resize(80, 24)
	require.NoError(t, err)
}

func TestPTY_Timeout(t *testing.T) {
	if os.Getenv("CI") == "" && os.Getenv("OMP_TEST_PTY") == "" {
		t.Skip("PTY tests require terminal; set OMP_TEST_PTY=1")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := shell.NewPTY(ctx)
	if err != nil {
		t.Logf("PTY creation with timeout: %v", err)
	}
}

func TestPTY_Close(t *testing.T) {
	if os.Getenv("CI") == "" && os.Getenv("OMP_TEST_PTY") == "" {
		t.Skip("PTY tests require terminal; set OMP_TEST_PTY=1")
	}
	sess, err := shell.NewPTY(context.Background())
	require.NoError(t, err)

	assert.NoError(t, sess.Close())
	// double close should not panic
	assert.NoError(t, sess.Close())
}
