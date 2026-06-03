package lsp

import (
	"encoding/json"
	"testing"
)

func TestHandleDiagnostics(t *testing.T) {
	var gotURI string
	var gotDiags []Diagnostic

	h := HandleDiagnostics(func(uri string, diagnostics []Diagnostic) {
		gotURI = uri
		gotDiags = diagnostics
	})

	t.Run("calls handler on publishDiagnostics", func(t *testing.T) {
		params := PublishDiagnosticsParams{
			URI: "file:///test.go",
			Diagnostics: []Diagnostic{
				{Message: "test error", Severity: 1},
			},
		}
		b, _ := json.Marshal(params)
		h.HandleNotification("textDocument/publishDiagnostics", b)

		if gotURI != "file:///test.go" {
			t.Fatalf("expected file:///test.go, got %s", gotURI)
		}
		if len(gotDiags) != 1 || gotDiags[0].Message != "test error" {
			t.Fatalf("unexpected diagnostics: %+v", gotDiags)
		}
	})

	t.Run("ignores other methods", func(t *testing.T) {
		gotURI = ""
		gotDiags = nil

		params := json.RawMessage(`{"uri":"file:///test.go","diagnostics":[]}`)
		h.HandleNotification("textDocument/didOpen", params)

		if gotURI != "" {
			t.Fatal("expected handler not to be called")
		}
	})

	t.Run("ignores invalid params", func(t *testing.T) {
		gotURI = ""
		gotDiags = nil

		h.HandleNotification("textDocument/publishDiagnostics", json.RawMessage(`invalid`))

		if gotURI != "" {
			t.Fatal("expected handler not to be called with invalid json")
		}
	})
}

func TestHandleLogMessage(t *testing.T) {
	var gotType int
	var gotMsg string

	h := HandleLogMessage(func(typ int, msg string) {
		gotType = typ
		gotMsg = msg
	})

	t.Run("calls handler on window/logMessage", func(t *testing.T) {
		b := json.RawMessage(`{"type":3,"message":"hello"}`)
		h.HandleNotification("window/logMessage", b)

		if gotType != 3 || gotMsg != "hello" {
			t.Fatalf("expected (3, hello), got (%d, %s)", gotType, gotMsg)
		}
	})

	t.Run("calls handler on window/showMessage", func(t *testing.T) {
		b := json.RawMessage(`{"type":1,"message":"show"}`)
		h.HandleNotification("window/showMessage", b)

		if gotType != 1 || gotMsg != "show" {
			t.Fatalf("expected (1, show), got (%d, %s)", gotType, gotMsg)
		}
	})

	t.Run("ignores other methods", func(t *testing.T) {
		gotType = 0
		gotMsg = ""

		h.HandleNotification("telemetry/event", json.RawMessage(`{"type":1,"message":"x"}`))

		if gotMsg != "" {
			t.Fatal("expected handler not to be called")
		}
	})

	t.Run("ignores invalid params", func(t *testing.T) {
		gotType = 0
		gotMsg = ""

		h.HandleNotification("window/logMessage", json.RawMessage(`invalid`))

		if gotMsg != "" {
			t.Fatal("expected handler not to be called with invalid json")
		}
	})
}

func TestNotificationHandlerFunc(t *testing.T) {
	var called bool
	h := NotificationHandlerFunc(func(method string, params json.RawMessage) {
		called = true
		if method != "test/method" {
			t.Fatalf("expected test/method, got %s", method)
		}
	})
	h.HandleNotification("test/method", nil)
	if !called {
		t.Fatal("expected handler to be called")
	}
}

func TestServerRequestHandlerFunc(t *testing.T) {
	h := ServerRequestHandlerFunc(func(method string, params json.RawMessage) (any, error) {
		return "result", nil
	})
	result, err := h.HandleRequest("test/method", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != "result" {
		t.Fatalf("expected result, got %v", result)
	}
}
