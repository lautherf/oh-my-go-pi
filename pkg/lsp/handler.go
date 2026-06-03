package lsp

import (
	"encoding/json"
)

type NotificationHandler interface {
	HandleNotification(method string, params json.RawMessage)
}

type NotificationHandlerFunc func(method string, params json.RawMessage)

func (f NotificationHandlerFunc) HandleNotification(method string, params json.RawMessage) {
	f(method, params)
}

type ServerRequestHandler interface {
	HandleRequest(method string, params json.RawMessage) (any, error)
}

type ServerRequestHandlerFunc func(method string, params json.RawMessage) (any, error)

func (f ServerRequestHandlerFunc) HandleRequest(method string, params json.RawMessage) (any, error) {
	return f(method, params)
}

type DiagnosticsHandler func(uri string, diagnostics []Diagnostic)

func HandleDiagnostics(h DiagnosticsHandler) NotificationHandlerFunc {
	return func(method string, params json.RawMessage) {
		if method != "textDocument/publishDiagnostics" {
			return
		}
		var p PublishDiagnosticsParams
		if err := json.Unmarshal(params, &p); err != nil {
			return
		}
		h(p.URI, p.Diagnostics)
	}
}

func HandleLogMessage(h func(typ int, msg string)) NotificationHandlerFunc {
	return func(method string, params json.RawMessage) {
		if method != "window/logMessage" && method != "window/showMessage" {
			return
		}
		var p struct {
			Type    int    `json:"type"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return
		}
		h(p.Type, p.Message)
	}
}
