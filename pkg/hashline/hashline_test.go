package hashline_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oh-my-pi/omp/pkg/hashline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func anchor(text string) string {
	// 2-char hash: first 2 bytes of xxhash32
	h := hashline.Hash(text)
	return h
}

func TestParse_ReplaceLine(t *testing.T) {
	h, err := hashline.Parse("= new content")
	require.NoError(t, err)
	assert.Equal(t, hashline.OpReplace, h.Operation)
	assert.Equal(t, "new content", h.Payload)
}

func TestParse_InsertBefore(t *testing.T) {
	h, err := hashline.Parse("+ inserted line")
	require.NoError(t, err)
	assert.Equal(t, hashline.OpInsert, h.Operation)
	assert.Equal(t, "inserted line", h.Payload)
}

func TestParse_RemoveLine(t *testing.T) {
	h, err := hashline.Parse("<")
	require.NoError(t, err)
	assert.Equal(t, hashline.OpRemove, h.Operation)
	assert.Equal(t, "", h.Payload)
}

func TestParse_InvalidOp(t *testing.T) {
	_, err := hashline.Parse("? invalid")
	require.Error(t, err)
}

func TestParse_Empty(t *testing.T) {
	_, err := hashline.Parse("")
	require.Error(t, err)
}

func TestParse_OnlyOp(t *testing.T) {
	_, err := hashline.Parse("=")
	require.Error(t, err)
}

func TestHash_Consistent(t *testing.T) {
	a := hashline.Hash("hello world")
	b := hashline.Hash("hello world")
	c := hashline.Hash("different")
	assert.Equal(t, a, b)
	assert.NotEqual(t, a, c)
	assert.Len(t, a, 4) // 2 bytes -> 4 hex chars
}

func TestApply_Replace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f.txt")
	require.NoError(t, os.WriteFile(path, []byte("line1\nline2\nline3\n"), 0644))

	h := hashline.H{Operation: hashline.OpReplace, Payload: "modified", Anchor: hashline.Hash("line2")}
	err := hashline.Apply(path, h)
	require.NoError(t, err)

	data, _ := os.ReadFile(path)
	assert.Equal(t, "line1\nmodified\nline3\n", string(data))
}

func TestApply_Insert(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f.txt")
	require.NoError(t, os.WriteFile(path, []byte("line1\nline3\n"), 0644))

	h := hashline.H{Operation: hashline.OpInsert, Payload: "line2", Anchor: hashline.Hash("line3")}
	err := hashline.Apply(path, h)
	require.NoError(t, err)

	data, _ := os.ReadFile(path)
	assert.Equal(t, "line1\nline2\nline3\n", string(data))
}

func TestApply_Remove(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f.txt")
	require.NoError(t, os.WriteFile(path, []byte("line1\nline2\nline3\n"), 0644))

	h := hashline.H{Operation: hashline.OpRemove, Anchor: hashline.Hash("line2")}
	err := hashline.Apply(path, h)
	require.NoError(t, err)

	data, _ := os.ReadFile(path)
	assert.Equal(t, "line1\nline3\n", string(data))
}

func TestApply_AnchorNotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f.txt")
	require.NoError(t, os.WriteFile(path, []byte("content\n"), 0644))

	h := hashline.H{Operation: hashline.OpReplace, Payload: "new", Anchor: "xx"}
	err := hashline.Apply(path, h)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "anchor")
}

func TestApply_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.txt")
	require.NoError(t, os.WriteFile(path, []byte{}, 0644))

	h := hashline.H{Operation: hashline.OpInsert, Payload: "first line", Anchor: ""}
	err := hashline.Apply(path, h)
	require.NoError(t, err)

	data, _ := os.ReadFile(path)
	assert.Equal(t, "first line\n", string(data))
}

func TestApply_ParseAndApply(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f.txt")
	require.NoError(t, os.WriteFile(path, []byte("old content\n"), 0644))

	h, err := hashline.Parse("= new content")
	require.NoError(t, err)
	h.Anchor = hashline.Hash("old content") // set anchor from file content

	err = hashline.Apply(path, h)
	require.NoError(t, err)

	data, _ := os.ReadFile(path)
	assert.Equal(t, "new content\n", string(data))
}
