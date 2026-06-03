package lsp

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/oh-my-pi/omp/pkg/jsonrpc"
)

type ServerConfig struct {
	Name    string
	Command string
	Args    []string
	LanguageIDs []string
}

type ManagedServer struct {
	config ServerConfig
	client *Client
	transport *jsonrpc.StdioTransport
	mu     sync.Mutex
}

type Manager struct {
	servers map[string]*ManagedServer
	mu      sync.Mutex
}

func NewManager() *Manager {
	return &Manager{
		servers: make(map[string]*ManagedServer),
	}
}

func (m *Manager) Start(ctx context.Context, config ServerConfig) (*Client, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.servers[config.Name]; ok {
		return nil, fmt.Errorf("server %s already running", config.Name)
	}

	cmd := exec.Command(config.Command, config.Args...)
	transport, err := jsonrpc.NewStdioTransport(cmd)
	if err != nil {
		return nil, fmt.Errorf("start %s: %w", config.Name, err)
	}

	client := NewClient(transport)

	result, err := client.Initialize(ctx, InitializeParams{
		ClientInfo: &ClientInfo{Name: "omplsp", Version: "0.1.0"},
		Capabilities: ClientCapabilities{
			TextDocument: &TextDocumentClientCapabilities{
				Hover:      &HoverCapability{ContentFormat: []string{"markdown", "plaintext"}},
				Completion: &CompletionCapability{},
				Definition: &DefinitionCapability{},
				References: &ReferencesCapability{},
				Rename:     &RenameCapability{},
				DocumentSymbol: &DocumentSymbolCapability{
					HierarchicalDocumentSymbolSupport: &[]bool{true}[0],
				},
			},
		},
	})
	if err != nil {
		transport.Close()
		return nil, fmt.Errorf("initialize %s: %w", config.Name, err)
	}
	_ = result

	if err := client.Initialized(ctx); err != nil {
		transport.Close()
		return nil, fmt.Errorf("initialized %s: %w", config.Name, err)
	}

	ms := &ManagedServer{
		config:    config,
		client:    client,
		transport: transport,
	}
	m.servers[config.Name] = ms
	return client, nil
}

func (m *Manager) Stop(name string) error {
	m.mu.Lock()
	ms, ok := m.servers[name]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("server %s not found", name)
	}
	delete(m.servers, name)
	m.mu.Unlock()

	ms.client.Shutdown(context.Background())
	ms.client.Exit(context.Background())
	return ms.transport.Close()
}

func (m *Manager) Get(name string) *Client {
	m.mu.Lock()
	defer m.mu.Unlock()
	ms, ok := m.servers[name]
	if !ok {
		return nil
	}
	return ms.client
}

func (m *Manager) GetOrStart(ctx context.Context, config ServerConfig) (*Client, error) {
	m.mu.Lock()
	if ms, ok := m.servers[config.Name]; ok {
		m.mu.Unlock()
		return ms.client, nil
	}
	m.mu.Unlock()
	return m.Start(ctx, config)
}

func (m *Manager) StopAll() {
	m.mu.Lock()
	names := make([]string, 0, len(m.servers))
	for name := range m.servers {
		names = append(names, name)
	}
	m.mu.Unlock()

	for _, name := range names {
		m.Stop(name)
	}
}

func DetectServerForFile(filePath string) *ServerConfig {
	ext := filepath.Ext(filePath)
	switch ext {
	case ".go":
		return &ServerConfig{
			Name:        "gopls",
			Command:     "gopls",
			LanguageIDs: []string{"go"},
		}
	case ".ts", ".tsx", ".js", ".jsx":
		return &ServerConfig{
			Name:    "typescript-language-server",
			Command: "typescript-language-server",
			Args:    []string{"--stdio"},
			LanguageIDs: []string{"typescript", "typescriptreact", "javascript", "javascriptreact"},
		}
	case ".py":
		return &ServerConfig{
			Name:    "pyright",
			Command: "pyright-langserver",
			Args:    []string{"--stdio"},
			LanguageIDs: []string{"python"},
		}
	case ".rs":
		return &ServerConfig{
			Name:    "rust-analyzer",
			Command: "rust-analyzer",
			LanguageIDs: []string{"rust"},
		}
	case ".java":
		return &ServerConfig{
			Name:    "jdtls",
			Command: "jdtls",
			LanguageIDs: []string{"java"},
		}
	}
	return nil
}
