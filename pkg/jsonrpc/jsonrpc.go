package jsonrpc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

type Error struct {
	Code    int              `json:"code"`
	Message string           `json:"message"`
	Data    json.RawMessage  `json:"data,omitempty"`
}

func NewRequest(id int, method string, params any) *Message {
	var raw json.RawMessage
	if params != nil {
		raw, _ = json.Marshal(params)
	}
	return &Message{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  raw,
	}
}

func NewResponse(id int, result json.RawMessage, err *Error) *Message {
	return &Message{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
		Error:   err,
	}
}

func NewNotification(method string, params any) *Message {
	var raw json.RawMessage
	if params != nil {
		raw, _ = json.Marshal(params)
	}
	return &Message{
		JSONRPC: "2.0",
		Method:  method,
		Params:  raw,
	}
}

func (m *Message) ToMap() map[string]any {
	var result map[string]any
	data, _ := json.Marshal(m)
	json.Unmarshal(data, &result)
	return result
}

func EncodeMessage(w io.Writer, msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}
	_, err = fmt.Fprintf(w, "Content-Length: %d\r\n\r\n%s", len(data), data)
	return err
}

func DecodeMessage(r io.Reader) (*Message, error) {
	br, ok := r.(*bufio.Reader)
	if !ok {
		br = bufio.NewReader(r)
	}
	return decodeWithReader(br)
}

func DecodeFromBufio(br *bufio.Reader) (*Message, error) {
	return decodeWithReader(br)
}

func decodeWithReader(br *bufio.Reader) (*Message, error) {

	contentLength := 0
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("read header: %w", err)
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if strings.HasPrefix(line, "Content-Length: ") {
			n, err := strconv.Atoi(line[len("Content-Length: "):])
			if err != nil {
				return nil, fmt.Errorf("invalid content-length: %w", err)
			}
			contentLength = n
		}
	}

	if contentLength <= 0 {
		return nil, fmt.Errorf("missing or zero Content-Length")
	}

	body := make([]byte, contentLength)
	if _, err := io.ReadFull(br, body); err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var msg Message
	if err := json.Unmarshal(body, &msg); err != nil {
		return nil, fmt.Errorf("json decode: %w", err)
	}
	return &msg, nil
}
