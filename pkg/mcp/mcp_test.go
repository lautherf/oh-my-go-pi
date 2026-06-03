package mcp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/oh-my-pi/omp/pkg/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mcpServerHelper() string {
	// Build a small Go MCP server helper
	src := `package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		var req struct {
			ID int ` + "`" + `json:"id"` + "`" + `
		}
		json.Unmarshal([]byte(line), &req)
		id := req.ID

		var result json.RawMessage
		var errObj json.RawMessage
		if strings.Contains(line, "initialize") {
			result = json.RawMessage(` + "`" + `{"protocolVersion":"2024-11-05","capabilities":{"tools":{}},"serverInfo":{"name":"test-mcp","version":"1.0"}}` + "`" + `)
		} else if strings.Contains(line, "tools/list") {
			result = json.RawMessage(` + "`" + `{"tools":[{"name":"echo","description":"echo tool","inputSchema":{"type":"object","properties":{"msg":{"type":"string"}}}}]}` + "`" + `)
		} else if strings.Contains(line, "tools/call") {
			result = json.RawMessage(` + "`" + `{"content":[{"type":"text","text":"hello from mcp"}]}` + "`" + `)
		} else {
			errObj = json.RawMessage(` + "`" + `{"code":-32601,"message":"Method not found"}` + "`" + `)
		}

		resp := map[string]any{"jsonrpc": "2.0", "id": id}
		if errObj != nil {
			resp["error"] = errObj
		} else {
			resp["result"] = result
		}
		b, _ := json.Marshal(resp)
		fmt.Println(string(b))
	}
}`

	f, _ := os.CreateTemp("", "mcp-server-*.go")
	os.WriteFile(f.Name(), []byte(src), 0644)
	binary := strings.TrimSuffix(f.Name(), ".go")
	cmd := exec.Command("go", "build", "-o", binary, f.Name())
	if out, err := cmd.CombinedOutput(); err != nil {
		panic(fmt.Sprintf("build failed: %v\n%s", err, out))
	}
	return binary
}

func TestMCP_Initialize(t *testing.T) {
	binary := mcpServerHelper()
	defer os.Remove(binary)

	cmd := exec.Command(binary)
	client, err := mcp.NewClient(context.Background(), cmd)
	require.NoError(t, err)
	defer client.Close()
}

func TestMCP_ListTools(t *testing.T) {
	binary := mcpServerHelper()
	defer os.Remove(binary)

	cmd := exec.Command(binary)
	client, err := mcp.NewClient(context.Background(), cmd)
	require.NoError(t, err)
	defer client.Close()

	tools, err := client.ListTools(context.Background())
	require.NoError(t, err)
	require.Len(t, tools, 1)
	assert.Equal(t, "echo", tools[0].Name)
}

func TestMCP_CallTool(t *testing.T) {
	binary := mcpServerHelper()
	defer os.Remove(binary)

	cmd := exec.Command(binary)
	client, err := mcp.NewClient(context.Background(), cmd)
	require.NoError(t, err)
	defer client.Close()

	result, err := client.CallTool(context.Background(), "echo", json.RawMessage(`{"msg":"hi"}`))
	require.NoError(t, err)
	assert.Contains(t, result, "hello from mcp")
}

func TestMCP_Close(t *testing.T) {
	binary := mcpServerHelper()
	defer os.Remove(binary)

	cmd := exec.Command(binary)
	client, err := mcp.NewClient(context.Background(), cmd)
	require.NoError(t, err)

	assert.NoError(t, client.Close())
}

func TestMCP_ClientName(t *testing.T) {
	binary := mcpServerHelper()
	defer os.Remove(binary)

	cmd := exec.Command(binary)
	client, err := mcp.NewClient(context.Background(), cmd,
		mcp.WithClientName("test-agent"),
		mcp.WithClientVersion("0.1.0"))
	require.NoError(t, err)
	defer client.Close()
}

func TestMCP_StdioError(t *testing.T) {
	_, err := mcp.NewClient(context.Background(), exec.Command("nonexistent_cmd_xyz"))
	require.Error(t, err)
}
