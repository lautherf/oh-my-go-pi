package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/oh-my-pi/omp/pkg/jsonrpc"
)

type Client struct {
	transport    *jsonrpc.StdioTransport
	msgID        atomic.Int64
	cap          ServerCapabilities
	mu           sync.Mutex
	notifHandler NotificationHandler
	reqHandler   ServerRequestHandler
}

func NewClient(transport *jsonrpc.StdioTransport) *Client {
	return &Client{
		transport: transport,
	}
}

func (c *Client) SetNotificationHandler(h NotificationHandler) {
	c.notifHandler = h
}

func (c *Client) SetServerRequestHandler(h ServerRequestHandler) {
	c.reqHandler = h
}

func (c *Client) sendRequest(method string, params any) (*jsonrpc.Message, error) {
	id := int(c.msgID.Add(1))
	req := jsonrpc.NewRequest(id, method, params)
	if err := c.transport.Send(req); err != nil {
		return nil, fmt.Errorf("%s send: %w", method, err)
	}
	return c.waitResponse(id, method)
}

func (c *Client) waitResponse(id int, method string) (*jsonrpc.Message, error) {
	for {
		msg, err := c.transport.Receive()
		if err != nil {
			return nil, fmt.Errorf("%s receive: %w", method, err)
		}
		if msg.ID != nil {
			msgID := toInt(msg.ID)
			if msgID == id {
				if msg.Error != nil {
					return nil, fmt.Errorf("%s error: %s", method, msg.Error.Message)
				}
				return msg, nil
			}
		} else if msg.Method != "" && c.notifHandler != nil {
			c.notifHandler.HandleNotification(msg.Method, msg.Params)
		}
	}
}

func (c *Client) Initialize(ctx context.Context, params InitializeParams) (*InitializeResult, error) {
	resp, err := c.sendRequest("initialize", params)
	if err != nil {
		return nil, err
	}
	var result InitializeResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("initialize parse: %w", err)
	}
	c.mu.Lock()
	c.cap = result.Capabilities
	c.mu.Unlock()
	return &result, nil
}

func (c *Client) Initialized(ctx context.Context) error {
	return c.transport.Send(jsonrpc.NewNotification("initialized", nil))
}

func (c *Client) Shutdown(ctx context.Context) error {
	id := int(c.msgID.Add(1))
	req := jsonrpc.NewRequest(id, "shutdown", nil)
	if err := c.transport.Send(req); err != nil {
		return err
	}
	_, err := c.waitResponse(id, "shutdown")
	return err
}

func (c *Client) Exit(ctx context.Context) error {
	return c.transport.Send(jsonrpc.NewNotification("exit", nil))
}

func (c *Client) DidOpen(ctx context.Context, uri, languageID, text string) error {
	params := DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        uri,
			LanguageID: languageID,
			Version:    1,
			Text:       text,
		},
	}
	return c.transport.Send(jsonrpc.NewNotification("textDocument/didOpen", params))
}

func (c *Client) DidChange(ctx context.Context, uri string, version int, text string) error {
	params := DidChangeTextDocumentParams{
		TextDocument: VersionedTextDocumentIdentifier{
			URI:     uri,
			Version: version,
		},
		ContentChanges: []TextDocumentContentChangeEvent{
			{Text: text},
		},
	}
	return c.transport.Send(jsonrpc.NewNotification("textDocument/didChange", params))
}

func (c *Client) DidClose(ctx context.Context, uri string) error {
	params := DidCloseTextDocumentParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
	}
	return c.transport.Send(jsonrpc.NewNotification("textDocument/didClose", params))
}

func (c *Client) DidSave(ctx context.Context, uri, text string) error {
	params := DidSaveTextDocumentParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Text:         text,
	}
	return c.transport.Send(jsonrpc.NewNotification("textDocument/didSave", params))
}

func (c *Client) Hover(ctx context.Context, uri string, pos Position) (*Hover, error) {
	params := map[string]any{
		"textDocument": TextDocumentIdentifier{URI: uri},
		"position":     pos,
	}
	resp, err := c.sendRequest("textDocument/hover", params)
	if err != nil {
		return nil, err
	}
	var hover Hover
	if err := json.Unmarshal(resp.Result, &hover); err != nil {
		return nil, err
	}
	return &hover, nil
}

func (c *Client) GotoDefinition(ctx context.Context, uri string, pos Position) (*Location, error) {
	params := map[string]any{
		"textDocument": TextDocumentIdentifier{URI: uri},
		"position":     pos,
	}
	resp, err := c.sendRequest("textDocument/definition", params)
	if err != nil {
		return nil, err
	}
	var loc Location
	if err := json.Unmarshal(resp.Result, &loc); err != nil {
		return nil, err
	}
	return &loc, nil
}

func (c *Client) References(ctx context.Context, uri string, pos Position, includeDecl bool) ([]Location, error) {
	params := ReferenceParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     pos,
		Context:      ReferenceContext{IncludeDeclaration: includeDecl},
	}
	resp, err := c.sendRequest("textDocument/references", params)
	if err != nil {
		return nil, err
	}
	var locs []Location
	if err := json.Unmarshal(resp.Result, &locs); err != nil {
		return nil, err
	}
	return locs, nil
}

func (c *Client) Rename(ctx context.Context, uri string, pos Position, newName string) (*WorkspaceEdit, error) {
	params := RenameParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     pos,
		NewName:      newName,
	}
	resp, err := c.sendRequest("textDocument/rename", params)
	if err != nil {
		return nil, err
	}
	var edit WorkspaceEdit
	if err := json.Unmarshal(resp.Result, &edit); err != nil {
		return nil, err
	}
	return &edit, nil
}

func (c *Client) Completion(ctx context.Context, uri string, pos Position) (*CompletionList, error) {
	params := CompletionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     pos,
	}
	resp, err := c.sendRequest("textDocument/completion", params)
	if err != nil {
		return nil, err
	}
	var list CompletionList
	if err := json.Unmarshal(resp.Result, &list); err != nil {
		var items []CompletionItem
		if err2 := json.Unmarshal(resp.Result, &items); err2 != nil {
			return nil, err
		}
		list.Items = items
	}
	return &list, nil
}

func (c *Client) DocumentSymbols(ctx context.Context, uri string) ([]DocumentSymbol, error) {
	params := DocumentSymbolParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
	}
	resp, err := c.sendRequest("textDocument/documentSymbol", params)
	if err != nil {
		return nil, err
	}
	var symbols []DocumentSymbol
	if err := json.Unmarshal(resp.Result, &symbols); err != nil {
		var flat []SymbolInformation
		if err2 := json.Unmarshal(resp.Result, &flat); err2 != nil {
			return nil, err
		}
		for _, s := range flat {
			symbols = append(symbols, DocumentSymbol{
				Name:           s.Name,
				Kind:           s.Kind,
				Range:          s.Location.Range,
				SelectionRange: s.Location.Range,
				Children:       nil,
			})
		}
	}
	return symbols, nil
}

func (c *Client) ServerCapabilities() ServerCapabilities {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cap
}

func toInt(id any) int {
	switch v := id.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	default:
		return 0
	}
}
