package tools_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/oh-my-pi/omp/pkg/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoDeclTool_ListDeclarations(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")
	src := `package main

import "fmt"

const greeting = "hello"

var count int

type Config struct {
	Port int
}

func main() {
	fmt.Println(greeting)
}

func (c *Config) String() string {
	return fmt.Sprintf("%d", c.Port)
}
`
	require.NoError(t, os.WriteFile(path, []byte(src), 0644))

	gt := &tools.GoDeclTool{}
	result, err := gt.Execute(context.Background(), `{"path":"`+path+`"}`)
	require.NoError(t, err)
	assert.Contains(t, result, "const greeting")
	assert.Contains(t, result, "var count")
	assert.Contains(t, result, "struct Config")
	assert.Contains(t, result, "func main")
	assert.Contains(t, result, "method String (*Config)")
}

func TestGoDeclTool_FindByName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")
	src := `package main

func foo() int {
	return 42
}

func bar() string {
	return "hello"
}
`
	require.NoError(t, os.WriteFile(path, []byte(src), 0644))

	gt := &tools.GoDeclTool{}
	result, err := gt.Execute(context.Background(), `{"path":"`+path+`","name":"foo"}`)
	require.NoError(t, err)
	assert.Contains(t, result, "name: foo")
	assert.Contains(t, result, "kind: func")
	assert.Contains(t, result, "return 42")
	assert.NotContains(t, result, "bar")
}

func TestGoDeclTool_FindStruct(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")
	src := `package main

type Server struct {
	Addr string
	Port int
}
`
	require.NoError(t, os.WriteFile(path, []byte(src), 0644))

	gt := &tools.GoDeclTool{}
	result, err := gt.Execute(context.Background(), `{"path":"`+path+`","name":"Server"}`)
	require.NoError(t, err)
	assert.Contains(t, result, "kind: struct")
	assert.Contains(t, result, "Addr string")
	assert.Contains(t, result, "Port int")
}

func TestGoDeclTool_NotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")
	require.NoError(t, os.WriteFile(path, []byte("package main\nfunc foo() {}\n"), 0644))

	gt := &tools.GoDeclTool{}
	_, err := gt.Execute(context.Background(), `{"path":"`+path+`","name":"nonexistent"}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGoDeclTool_InvalidFile(t *testing.T) {
	gt := &tools.GoDeclTool{}
	_, err := gt.Execute(context.Background(), `{"path":"/nonexistent/file.go"}`)
	require.Error(t, err)
}

func TestGoDeclTool_NoDeclarations(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.go")
	require.NoError(t, os.WriteFile(path, []byte("package empty\n"), 0644))

	gt := &tools.GoDeclTool{}
	result, err := gt.Execute(context.Background(), `{"path":"`+path+`"}`)
	require.NoError(t, err)
	assert.Contains(t, result, "no declarations")
}

func TestGoDeclTool_Name(t *testing.T) {
	assert.Equal(t, "go_decl", (&tools.GoDeclTool{}).Name())
}

func TestGoDeclTool_MissingPath(t *testing.T) {
	gt := &tools.GoDeclTool{}
	_, err := gt.Execute(context.Background(), `{"path":""}`)
	require.Error(t, err)
}
