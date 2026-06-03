package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/oh-my-pi/omp/pkg/jsonrpc"
)

func writeTestServer() string {
	src := `package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	br := bufio.NewReader(os.Stdin)
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimRight(line, "\r\n")

		if !strings.HasPrefix(line, "Content-Length:") {
			continue
		}
		n := 0
		fmt.Sscanf(line, "Content-Length: %d", &n)
		br.ReadString('\n') // blank line

		body := make([]byte, n)
		if _, err := io.ReadFull(br, body); err != nil {
			break
		}

		var req struct {
			ID     any              ` + "`" + `json:"id"` + "`" + `
			Method string           ` + "`" + `json:"method"` + "`" + `
			Params json.RawMessage  ` + "`" + `json:"params"` + "`" + `
		}
		json.Unmarshal(body, &req)

		var result json.RawMessage
		var errObj json.RawMessage

		switch {
		case strings.Contains(req.Method, "initialize"):
			result = json.RawMessage(` + "`" + `{"capabilities":{"textDocumentSync":{"openClose":true,"change":1},"hoverProvider":true,"definitionProvider":true,"referencesProvider":true},"serverInfo":{"name":"test-lsp","version":"1.0"}}` + "`" + `)
		case strings.Contains(req.Method, "shutdown"):
			result = json.RawMessage(` + "`" + `null` + "`" + `)
		case strings.Contains(req.Method, "textDocument/hover"):
			result = json.RawMessage(` + "`" + `{"contents":{"kind":"markdown","value":"**int** x"}}` + "`" + `)
		case strings.Contains(req.Method, "textDocument/definition"):
			result = json.RawMessage(` + "`" + `{"uri":"file:///test.go","range":{"start":{"line":0,"character":0},"end":{"line":0,"character":1}}}` + "`" + `)
		case strings.Contains(req.Method, "textDocument/references"):
			result = json.RawMessage(` + "`" + `[{"uri":"file:///test.go","range":{"start":{"line":1,"character":0},"end":{"line":1,"character":1}}}]` + "`" + `)
		case strings.Contains(req.Method, "textDocument/rename"):
			result = json.RawMessage(` + "`" + `{"changes":{"file:///test.go":[{"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":1}},"newText":"y"}]}}` + "`" + `)
		case strings.Contains(req.Method, "textDocument/completion"):
			result = json.RawMessage(` + "`" + `[{"label":"fmt.Println","kind":3,"detail":"func"}]` + "`" + `)
		case strings.Contains(req.Method, "textDocument/documentSymbol"):
			result = json.RawMessage(` + "`" + `[{"name":"main","kind":12,"range":{"start":{"line":0,"character":0},"end":{"line":2,"character":0}},"selectionRange":{"start":{"line":0,"character":0},"end":{"line":0,"character":0}}}]` + "`" + `)
		default:
			errObj = json.RawMessage(` + "`" + `{"code":-32601,"message":"Method not found"}` + "`" + `)
		}

		resp := map[string]any{"jsonrpc":"2.0","id":req.ID}
		if errObj != nil {
			resp["error"] = errObj
		} else {
			resp["result"] = result
		}
		b, _ := json.Marshal(resp)
		fmt.Printf("Content-Length: %d\r\n\r\n%s", len(b), b)
	}
}`
	f, _ := os.CreateTemp("", "lsp-server-*.go")
	os.WriteFile(f.Name(), []byte(src), 0644)
	binary := strings.TrimSuffix(f.Name(), ".go")
	cmd := exec.Command("go", "build", "-o", binary, f.Name())
	out, err := cmd.CombinedOutput()
	if err != nil {
		panic(fmt.Sprintf("build failed: %v\n%s", err, out))
	}
	return binary
}

func newTestClient(t *testing.T) *Client {
	t.Helper()
	binary := writeTestServer()
	t.Cleanup(func() { os.Remove(binary) })

	cmd := exec.Command(binary)
	transport, err := jsonrpc.NewStdioTransport(cmd)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { transport.Close() })
	return NewClient(transport)
}

func TestClient_Initialize(t *testing.T) {
	client := newTestClient(t)
	ctx := context.Background()

	result, err := client.Initialize(ctx, InitializeParams{
		ClientInfo: &ClientInfo{Name: "test-client", Version: "0.1.0"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ServerInfo.Name != "test-lsp" {
		t.Fatalf("expected test-lsp, got %s", result.ServerInfo.Name)
	}
	if !result.Capabilities.HoverProvider {
		t.Fatal("expected hover provider")
	}
}

func TestClient_ShutdownExit(t *testing.T) {
	client := newTestClient(t)
	ctx := context.Background()

	client.Initialize(ctx, InitializeParams{})
	if err := client.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	if err := client.Exit(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestClient_DidOpen(t *testing.T) {
	client := newTestClient(t)
	ctx := context.Background()

	client.Initialize(ctx, InitializeParams{})
	if err := client.DidOpen(ctx, "file:///test.go", "go", `package main`); err != nil {
		t.Fatal(err)
	}
}

func TestClient_Hover(t *testing.T) {
	client := newTestClient(t)
	ctx := context.Background()

	client.Initialize(ctx, InitializeParams{})
	hover, err := client.Hover(ctx, "file:///test.go", Position{Line: 0, Character: 5})
	if err != nil {
		t.Fatal(err)
	}
	if hover.Contents.Value != "**int** x" {
		t.Fatalf("expected hover text, got %s", hover.Contents.Value)
	}
}

func TestClient_GotoDefinition(t *testing.T) {
	client := newTestClient(t)
	ctx := context.Background()

	client.Initialize(ctx, InitializeParams{})
	loc, err := client.GotoDefinition(ctx, "file:///test.go", Position{Line: 0, Character: 0})
	if err != nil {
		t.Fatal(err)
	}
	if loc.URI != "file:///test.go" {
		t.Fatalf("expected file:///test.go, got %s", loc.URI)
	}
}

func TestClient_References(t *testing.T) {
	client := newTestClient(t)
	ctx := context.Background()

	client.Initialize(ctx, InitializeParams{})
	locs, err := client.References(ctx, "file:///test.go", Position{Line: 0, Character: 0}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(locs) != 1 {
		t.Fatalf("expected 1 reference, got %d", len(locs))
	}
}

func TestClient_Rename(t *testing.T) {
	client := newTestClient(t)
	ctx := context.Background()

	client.Initialize(ctx, InitializeParams{})
	edit, err := client.Rename(ctx, "file:///test.go", Position{Line: 0, Character: 0}, "y")
	if err != nil {
		t.Fatal(err)
	}
	if edit.Changes == nil {
		t.Fatal("expected changes")
	}
}

func TestClient_Completion(t *testing.T) {
	client := newTestClient(t)
	ctx := context.Background()

	client.Initialize(ctx, InitializeParams{})
	list, err := client.Completion(ctx, "file:///test.go", Position{Line: 0, Character: 0})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Items) == 0 {
		t.Fatal("expected completion items")
	}
}

func TestClient_Notifications(t *testing.T) {
	client := newTestClient(t)
	ctx := context.Background()
	client.Initialize(ctx, InitializeParams{})

	var gotMethod string
	var gotParams json.RawMessage
	client.SetNotificationHandler(NotificationHandlerFunc(func(method string, params json.RawMessage) {
		gotMethod = method
		gotParams = params
	}))

	// The test server never sends notifications, but the dispatcher should be running
	if client.ServerCapabilities().HoverProvider != true {
		t.Fatal("expected hover provider from capabilities")
	}
	_ = gotMethod
	_ = gotParams
}

func TestClient_DocumentSymbols(t *testing.T) {
	client := newTestClient(t)
	ctx := context.Background()

	client.Initialize(ctx, InitializeParams{})
	symbols, err := client.DocumentSymbols(ctx, "file:///test.go")
	if err != nil {
		t.Fatal(err)
	}
	if len(symbols) != 1 || symbols[0].Name != "main" {
		t.Fatalf("expected [main], got %+v", symbols)
	}
}
