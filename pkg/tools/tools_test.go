package tools_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/oh-my-pi/omp/pkg/hashline"
	"github.com/oh-my-pi/omp/pkg/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadTool_ReadFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(path, []byte("file content"), 0644))

	rt := &tools.ReadTool{}
	result, err := rt.Execute(context.Background(), `{"path":"`+path+`"}`)
	require.NoError(t, err)
	assert.Contains(t, result, "file content")
}

func TestReadTool_NotFound(t *testing.T) {
	t.Parallel()
	rt := &tools.ReadTool{}
	_, err := rt.Execute(context.Background(), `{"path":"/nonexistent/path"}`)
	require.Error(t, err)
}

func TestReadTool_Name(t *testing.T) {
	assert.Equal(t, "read", (&tools.ReadTool{}).Name())
}

func TestWriteTool_WriteFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")

	wt := &tools.WriteTool{}
	result, err := wt.Execute(context.Background(),
		`{"path":"`+path+`","content":"hello world"}`)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "hello world", string(data))
	assert.Contains(t, result, "written")
}

func TestWriteTool_Name(t *testing.T) {
	assert.Equal(t, "write", (&tools.WriteTool{}).Name())
}

func TestBashTool_Execute(t *testing.T) {
	t.Parallel()
	bt := &tools.BashTool{}
	result, err := bt.Execute(context.Background(), `{"command":"echo hello bash"}`)
	require.NoError(t, err)
	assert.Contains(t, result, "hello bash")
}

func TestBashTool_ExitError(t *testing.T) {
	t.Parallel()
	bt := &tools.BashTool{}
	result, err := bt.Execute(context.Background(), `{"command":"exit 1"}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exit")
	t.Logf("result=%q err=%v", result, err)
}

func TestBashTool_Name(t *testing.T) {
	assert.Equal(t, "bash", (&tools.BashTool{}).Name())
}

func TestGrepTool_Search(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "search.txt"), []byte("find me\nother"), 0644))

	gt := &tools.GrepTool{}
	result, err := gt.Execute(context.Background(),
		`{"pattern":"find me","root":"`+dir+`"}`)
	require.NoError(t, err)
	assert.Contains(t, result, "find me")
	assert.Contains(t, result, "search.txt")
}

func TestGrepTool_NoMatch(t *testing.T) {
	t.Parallel()
	gt := &tools.GrepTool{}
	result, err := gt.Execute(context.Background(), `{"pattern":"zzz","root":"/tmp"}`)
	require.NoError(t, err)
	assert.Contains(t, result, "no matches")
}

func TestGrepTool_Name(t *testing.T) {
	assert.Equal(t, "grep", (&tools.GrepTool{}).Name())
}

func TestReadTool_SizeLimit(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "large.txt")
	data := make([]byte, 2*1024*1024+1)
	require.NoError(t, os.WriteFile(path, data, 0644))

	rt := &tools.ReadTool{}
	_, err := rt.Execute(context.Background(), `{"path":"`+path+`"}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too large")
}

func TestReadTool_InvalidJSON(t *testing.T) {
	t.Parallel()
	rt := &tools.ReadTool{}
	_, err := rt.Execute(context.Background(), `not json`)
	require.Error(t, err)
}

func TestEditTool_Replace(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := dir + "/f.txt"
	require.NoError(t, os.WriteFile(path, []byte("hello\nworld\n"), 0644))

	et := &tools.EditTool{}
	anchor := hashline.Hash("hello")
	result, err := et.Execute(context.Background(),
		`{"path":"`+path+`","anchor":"`+anchor+`","op":"=","payload":"hi"}`)
	require.NoError(t, err)
	assert.Contains(t, result, "applied")

	data, _ := os.ReadFile(path)
	assert.Contains(t, string(data), "hi\nworld")
}

func TestEditTool_Insert(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := dir + "/f.txt"
	require.NoError(t, os.WriteFile(path, []byte("line2\nline3\n"), 0644))

	et := &tools.EditTool{}
	anchor := hashline.Hash("line2")
	result, err := et.Execute(context.Background(),
		`{"path":"`+path+`","anchor":"`+anchor+`","op":"+","payload":"line1"}`)
	require.NoError(t, err)
	assert.Contains(t, result, "applied")

	data, _ := os.ReadFile(path)
	assert.Equal(t, "line1\nline2\nline3\n", string(data))
}

func TestEditTool_Remove(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := dir + "/f.txt"
	require.NoError(t, os.WriteFile(path, []byte("remove_me\nkeep_me\n"), 0644))

	et := &tools.EditTool{}
	anchor := hashline.Hash("remove_me")
	result, err := et.Execute(context.Background(),
		`{"path":"`+path+`","anchor":"`+anchor+`","op":"<"}`)
	require.NoError(t, err)
	assert.Contains(t, result, "applied")

	data, _ := os.ReadFile(path)
	assert.Equal(t, "keep_me\n", string(data))
}

func TestEditTool_MissingArgs(t *testing.T) {
	et := &tools.EditTool{}
	_, err := et.Execute(context.Background(), `{"path":""}`)
	require.Error(t, err)
}

func TestEditTool_Name(t *testing.T) {
	assert.Equal(t, "edit", (&tools.EditTool{}).Name())
}

func TestWebSearchTool_Name(t *testing.T) {
	assert.Equal(t, "web_search", (&tools.WebSearchTool{}).Name())
}

func TestWebFetchTool_Name(t *testing.T) {
	assert.Equal(t, "web_fetch", (&tools.WebFetchTool{}).Name())
}

func TestWebSearchTool_EmptyQuery(t *testing.T) {
	ws := &tools.WebSearchTool{}
	_, err := ws.Execute(context.Background(), `{"query":""}`)
	require.Error(t, err)
}

func TestWebFetchTool_EmptyURL(t *testing.T) {
	wf := &tools.WebFetchTool{}
	_, err := wf.Execute(context.Background(), `{"url":""}`)
	require.Error(t, err)
}
