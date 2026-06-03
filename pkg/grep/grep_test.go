package grep_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oh-my-pi/omp/pkg/grep"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"hello.txt":    "hello world\nhow are you\nhello again\n",
		"main.go":      "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n",
		"numbers.txt":  "123\n456\n789\n",
		"empty.txt":    "",
		"sub/match.txt": "matching line\nanother line\n",
		"sub/deep/nest.txt": "deeply nested\nhello from deep\n",
	}
	for path, content := range files {
		fp := filepath.Join(dir, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(fp), 0755))
		require.NoError(t, os.WriteFile(fp, []byte(content), 0644))
	}
	return dir
}

func TestSearch_Basic(t *testing.T) {
	t.Parallel()
	dir := setupTestDir(t)
	results, err := grep.Search(dir, "hello", nil)
	require.NoError(t, err)
	// hello.txt(2 lines: "hello world", "hello again") + main.go(1: println("hello")) + nest.txt(1: "hello from deep")
	assert.Len(t, results, 4)
}

func TestSearch_CaseSensitive(t *testing.T) {
	t.Parallel()
	dir := setupTestDir(t)

	results, err := grep.Search(dir, "Hello", &grep.Options{CaseSensitive: true})
	require.NoError(t, err)
	assert.Len(t, results, 0) // no match

	results, err = grep.Search(dir, "Hello", &grep.Options{CaseSensitive: false})
	require.NoError(t, err)
	assert.Len(t, results, 4) // "hello" appears in hello.txt(2) + main.go(1) + nest.txt(1)
}

func TestSearch_IncludeGlob(t *testing.T) {
	t.Parallel()
	dir := setupTestDir(t)

	results, err := grep.Search(dir, "hello", &grep.Options{Include: "*.go"})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Contains(t, results[0].Path, "main.go")
}

func TestSearch_ExcludeGlob(t *testing.T) {
	t.Parallel()
	dir := setupTestDir(t)

	results, err := grep.Search(dir, "hello", &grep.Options{Exclude: "*.go"})
	require.NoError(t, err)
	// excludes main.go, leaves hello.txt(2) + nest.txt(1)
	assert.Len(t, results, 3)
	for _, m := range results {
		assert.NotContains(t, m.Path, "main.go")
	}
}

func TestSearch_ContextLines(t *testing.T) {
	t.Parallel()
	dir := setupTestDir(t)

	results, err := grep.Search(dir, "how are you", &grep.Options{Context: 1})
	require.NoError(t, err)
	require.Len(t, results, 1)
	r := results[0]
	assert.GreaterOrEqual(t, len(r.Lines), 1)
}

func TestSearch_NoMatch(t *testing.T) {
	t.Parallel()
	dir := setupTestDir(t)

	results, err := grep.Search(dir, "zzz_nonexistent_zzz", nil)
	require.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestSearch_Regex(t *testing.T) {
	t.Parallel()
	dir := setupTestDir(t)

	results, err := grep.Search(dir, "he[l]+o", nil)
	require.NoError(t, err)
	// matches "hello" across 3 files: hello.txt(2) + main.go(1) + nest.txt(1)
	assert.Len(t, results, 4)
}

func TestSearch_EmptyPattern(t *testing.T) {
	t.Parallel()
	_, err := grep.Search("/tmp", "", nil)
	require.Error(t, err)
}

func TestSearch_LineNumbers(t *testing.T) {
	t.Parallel()
	dir := setupTestDir(t)

	results, err := grep.Search(dir, "again", nil)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, 3, results[0].LineNumber)
}

func TestSearch_Subdirs(t *testing.T) {
	t.Parallel()
	dir := setupTestDir(t)

	results, err := grep.Search(dir, "matching", nil)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Contains(t, results[0].Path, "match.txt")

	results, err = grep.Search(dir, "deeply", nil)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Contains(t, results[0].Path, "nest.txt")
}

func TestSearch_MultipleMatchesSameFile(t *testing.T) {
	t.Parallel()
	dir := setupTestDir(t)

	results, err := grep.Search(dir, "hello", &grep.Options{Include: "hello.txt"})
	require.NoError(t, err)
	require.Len(t, results, 2) // two lines match
}
