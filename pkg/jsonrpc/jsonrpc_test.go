package jsonrpc

import (
	"bufio"
	"bytes"
	"encoding/json"
	"testing"
)

func TestNewRequest(t *testing.T) {
	params := map[string]string{"foo": "bar"}
	req := NewRequest(1, "test/method", params)
	if req.JSONRPC != "2.0" {
		t.Fatalf("expected jsonrpc 2.0, got %s", req.JSONRPC)
	}
	if req.ID != 1 {
		t.Fatalf("expected id 1, got %v", req.ID)
	}
	if req.Method != "test/method" {
		t.Fatalf("expected method test/method, got %s", req.Method)
	}
	var p map[string]string
	json.Unmarshal(req.Params, &p)
	if p["foo"] != "bar" {
		t.Fatalf("expected params.foo=bar")
	}
}

func TestNewResponse(t *testing.T) {
	resp := NewResponse(1, json.RawMessage(`"ok"`), nil)
	if resp.ID != 1 {
		t.Fatalf("expected id 1")
	}
	var s string
	json.Unmarshal(resp.Result, &s)
	if s != "ok" {
		t.Fatalf("expected result ok, got %s", s)
	}
	if resp.Error != nil {
		t.Fatal("expected no error")
	}
}

func TestNewErrorResponse(t *testing.T) {
	errResp := NewResponse(1, nil, &Error{Code: -32601, Message: "Method not found"})
	if errResp.Error == nil {
		t.Fatal("expected error")
	}
	if errResp.Error.Code != -32601 {
		t.Fatalf("expected code -32601, got %d", errResp.Error.Code)
	}
}

func TestNewNotification(t *testing.T) {
	notif := NewNotification("$/progress", map[string]any{"value": 50})
	if notif.ID != nil {
		t.Fatal("notification should have nil id")
	}
	if notif.Method != "$/progress" {
		t.Fatalf("expected method $/progress")
	}
}

func TestEncodeDecodeMessage(t *testing.T) {
	req := NewRequest(1, "test/method", json.RawMessage(`{"key":"val"}`))
	var buf bytes.Buffer
	if err := EncodeMessage(&buf, req); err != nil {
		t.Fatal(err)
	}

	decoded, err := DecodeMessage(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Method != "test/method" {
		t.Fatalf("expected method test/method, got %s", decoded.Method)
	}
	id, _ := decoded.ID.(float64)
	if id != 1 {
		t.Fatalf("expected id 1, got %v", decoded.ID)
	}
}

func TestEncodeDecodeResponse(t *testing.T) {
	resp := NewResponse(42, json.RawMessage(`{"done":true}`), nil)
	var buf bytes.Buffer
	if err := EncodeMessage(&buf, resp); err != nil {
		t.Fatal(err)
	}

	decoded, err := DecodeMessage(&buf)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := decoded.ID.(float64)
	if id != 42 {
		t.Fatalf("expected id 42, got %v", decoded.ID)
	}
	var result map[string]bool
	json.Unmarshal(decoded.Result, &result)
	if !result["done"] {
		t.Fatal("expected result.done=true")
	}
}

func TestEncodeDecodeNotification(t *testing.T) {
	notif := NewNotification("textDocument/didOpen", json.RawMessage(`{}`))
	var buf bytes.Buffer
	if err := EncodeMessage(&buf, notif); err != nil {
		t.Fatal(err)
	}

	decoded, err := DecodeMessage(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Method != "textDocument/didOpen" {
		t.Fatalf("expected method textDocument/didOpen")
	}
	if decoded.ID != nil {
		t.Fatal("notification should have nil id")
	}
}

func TestDecodeInvalidJSON(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString("Content-Length: 8\r\n\r\n{invalid}")
	_, err := DecodeMessage(&buf)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestDecodeTruncated(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString("Content-Length: 100\r\n\r\n{")
	_, err := DecodeMessage(&buf)
	if err == nil {
		t.Fatal("expected error for truncated content")
	}
}

func TestMultipleMessages(t *testing.T) {
	var buf bytes.Buffer

	msg1 := NewRequest(1, "method1", nil)
	msg2 := NewRequest(2, "method2", nil)

	EncodeMessage(&buf, msg1)
	EncodeMessage(&buf, msg2)

	br := bufio.NewReader(&buf)

	decoded1, err := DecodeFromBufio(br)
	if err != nil {
		t.Fatal(err)
	}
	id1, _ := decoded1.ID.(float64)
	if id1 != 1 {
		t.Fatalf("expected id 1, got %v", decoded1.ID)
	}

	decoded2, err := DecodeFromBufio(br)
	if err != nil {
		t.Fatal(err)
	}
	id2, _ := decoded2.ID.(float64)
	if id2 != 2 {
		t.Fatalf("expected id 2, got %v", decoded2.ID)
	}
}

func TestMessageToMap(t *testing.T) {
	req := NewRequest(5, "test/method", json.RawMessage(`{"x":1}`))
	m := req.ToMap()
	if m["method"] != "test/method" {
		t.Fatalf("expected method in map")
	}
}
