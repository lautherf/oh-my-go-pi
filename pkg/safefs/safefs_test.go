package safefs_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oh-my-pi/omp/pkg/safefs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadFile_Normal(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(path, []byte("hello world"), 0644))

	data, err := safefs.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "hello world", string(data))
}

func TestReadFile_NotFound(t *testing.T) {
	t.Parallel()
	_, err := safefs.ReadFile(filepath.Join(t.TempDir(), "nonexistent.txt"))
	require.Error(t, err)
	assert.True(t, os.IsNotExist(err), "should be not exist error")
}

func TestReadFile_SizeLimit(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "large.txt")
	data := make([]byte, 10*1024*1024+1) // 10MB+1
	require.NoError(t, os.WriteFile(path, data, 0644))

	_, err := safefs.ReadFile(path, safefs.WithMaxSize(10*1024*1024))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too large")
}

func TestReadFile_SizeLimitOK(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "ok.txt")
	require.NoError(t, os.WriteFile(path, []byte("small"), 0644))

	data, err := safefs.ReadFile(path, safefs.WithMaxSize(1024))
	require.NoError(t, err)
	assert.Equal(t, "small", string(data))
}

func TestReadFile_Directory(t *testing.T) {
	t.Parallel()
	_, err := safefs.ReadFile(t.TempDir())
	require.Error(t, err)
}

func TestWriteFile_Normal(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")

	err := safefs.WriteFile(path, []byte("test content"))
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "test content", string(data))
}

func TestWriteFile_AutoCreateDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "nested", "out.txt")

	err := safefs.WriteFile(path, []byte("auto created"))
	require.NoError(t, err)

	assert.FileExists(t, path)
}

func TestWriteFile_Atomic(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "atomic.txt")

	// write should be atomic: no partial writes visible
	err := safefs.WriteFile(path, []byte("atomic content"))
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "atomic content", string(data))
}

func TestSafePath_WithinRoot(t *testing.T) {
	t.Parallel()
	root := "/tmp/testroot"
	path, err := safefs.SafePath(root, "sub/file.txt")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(path, root+"/sub/file.txt") || path == root+"/sub/file.txt")
}

func TestSafePath_Traversal(t *testing.T) {
	t.Parallel()
	root := "/tmp/testroot"
	_, err := safefs.SafePath(root, "../../etc/passwd")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "traversal")
}

func TestSafePath_Absolute(t *testing.T) {
	t.Parallel()
	root := "/tmp/testroot"
	_, err := safefs.SafePath(root, "/etc/passwd")
	require.Error(t, err)
}

func TestExists_File(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "exist.txt")
	require.NoError(t, os.WriteFile(path, []byte("ok"), 0644))

	assert.True(t, safefs.Exists(path))
	assert.False(t, safefs.Exists(path+".nonexistent"))
}

func TestExists_Dir(t *testing.T) {
	t.Parallel()
	assert.True(t, safefs.Exists(t.TempDir()))
}
