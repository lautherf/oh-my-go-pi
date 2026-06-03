package lsp

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestManager_StartAndGet(t *testing.T) {
	manager := NewManager()
	ctx := context.Background()
	binary := writeTestServer()
	t.Cleanup(func() { os.Remove(binary) })

	config := ServerConfig{
		Name:    "test-lsp",
		Command: binary,
	}

	client, err := manager.Start(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	t.Cleanup(func() { manager.Stop("test-lsp") })

	got := manager.Get("test-lsp")
	if got != client {
		t.Fatal("Get returned different client")
	}

	// Start duplicate should fail
	_, err = manager.Start(ctx, config)
	if err == nil || !strings.Contains(err.Error(), "already running") {
		t.Fatalf("expected 'already running' error, got %v", err)
	}
}

func TestManager_GetNilForUnknown(t *testing.T) {
	manager := NewManager()
	if got := manager.Get("nonexistent"); got != nil {
		t.Fatal("expected nil for unknown server")
	}
}

func TestManager_Stop(t *testing.T) {
	manager := NewManager()
	ctx := context.Background()
	binary := writeTestServer()
	t.Cleanup(func() { os.Remove(binary) })

	config := ServerConfig{Name: "test-lsp", Command: binary}
	client, err := manager.Start(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	_ = client

	if err := manager.Stop("test-lsp"); err != nil {
		t.Fatal(err)
	}

	// Should be removed
	if got := manager.Get("test-lsp"); got != nil {
		t.Fatal("expected nil after stop")
	}

	// Stop unknown should fail
	if err := manager.Stop("nonexistent"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected 'not found' error, got %v", err)
	}
}

func TestManager_GetOrStart(t *testing.T) {
	manager := NewManager()
	ctx := context.Background()
	binary := writeTestServer()
	t.Cleanup(func() { os.Remove(binary) })

	config := ServerConfig{Name: "test-lsp", Command: binary}

	// First call starts the server
	client1, err := manager.GetOrStart(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { manager.Stop("test-lsp") })

	// Second call returns existing client
	client2, err := manager.GetOrStart(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	if client1 != client2 {
		t.Fatal("GetOrStart should return the same client")
	}
}

func TestManager_StopAll(t *testing.T) {
	manager := NewManager()
	ctx := context.Background()
	binary := writeTestServer()
	t.Cleanup(func() { os.Remove(binary) })

	manager.Start(ctx, ServerConfig{Name: "server1", Command: binary})
	manager.Start(ctx, ServerConfig{Name: "server2", Command: binary})
	manager.Start(ctx, ServerConfig{Name: "server3", Command: binary})

	manager.StopAll()

	if got := manager.Get("server1"); got != nil {
		t.Fatal("expected all servers to be stopped")
	}
	if got := manager.Get("server2"); got != nil {
		t.Fatal("expected all servers to be stopped")
	}
	if got := manager.Get("server3"); got != nil {
		t.Fatal("expected all servers to be stopped")
	}
}

func TestDetectServerForFile(t *testing.T) {
	tests := []struct {
		file     string
		wantName string
		wantNil  bool
	}{
		{"main.go", "gopls", false},
		{"test.ts", "typescript-language-server", false},
		{"test.tsx", "typescript-language-server", false},
		{"test.js", "typescript-language-server", false},
		{"test.jsx", "typescript-language-server", false},
		{"app.py", "pyright", false},
		{"lib.rs", "rust-analyzer", false},
		{"Main.java", "jdtls", false},
		{"main.c", "", true},
		{"main.rb", "", true},
		{"Makefile", "", true},
		{"noext", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			cfg := DetectServerForFile(tt.file)
			if tt.wantNil {
				if cfg != nil {
					t.Fatalf("expected nil, got %+v", cfg)
				}
				return
			}
			if cfg == nil {
				t.Fatal("expected non-nil")
			}
			if cfg.Name != tt.wantName {
				t.Fatalf("expected name %s, got %s", tt.wantName, cfg.Name)
			}
		})
	}
}

func TestManager_ReuseAfterStop(t *testing.T) {
	manager := NewManager()
	ctx := context.Background()
	binary := writeTestServer()
	t.Cleanup(func() { os.Remove(binary) })

	config := ServerConfig{Name: "test-lsp", Command: binary}

	// Start, stop, then start again
	client1, err := manager.Start(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	_ = client1

	if err := manager.Stop("test-lsp"); err != nil {
		t.Fatal(err)
	}

	client2, err := manager.Start(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { manager.Stop("test-lsp") })

	if client1 == client2 {
		t.Fatal("expected a new client after restart")
	}
}

func TestManager_StartFailsWithBadBinary(t *testing.T) {
	manager := NewManager()
	ctx := context.Background()

	config := ServerConfig{
		Name:    "bad-lsp",
		Command: "/nonexistent/binary",
	}

	_, err := manager.Start(ctx, config)
	if err == nil {
		t.Fatal("expected error for bad binary")
	}
}

func TestManager_StartFailsWithBadServer(t *testing.T) {
	manager := NewManager()
	ctx := context.Background()

	src := `package main; func main() {}`
	f, _ := os.CreateTemp("", "fake-lsp-*.go")
	os.WriteFile(f.Name(), []byte(src), 0644)
	binary := strings.TrimSuffix(f.Name(), ".go")
	cmd := exec.Command("go", "build", "-o", binary, f.Name())
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	t.Cleanup(func() { os.Remove(binary) })
	t.Cleanup(func() { os.Remove(f.Name()) })

	config := ServerConfig{Name: "bad-lsp", Command: binary}
	_, err = manager.Start(ctx, config)
	if err == nil {
		t.Fatal("expected error for bad LSP server (exits immediately)")
	}
}
