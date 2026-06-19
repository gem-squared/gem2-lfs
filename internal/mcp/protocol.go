package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
)

const jsonRPCVersion = "2.0"

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

func (r *Request) IsNotification() bool {
	return len(r.ID) == 0
}

type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("jsonrpc error %d: %s", e.Code, e.Message)
}

const (
	ErrCodeParseError     = -32700
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternal       = -32603
)

func NewParseError(msg string) *RPCError {
	return &RPCError{Code: ErrCodeParseError, Message: msg}
}

func NewMethodNotFound(method string) *RPCError {
	return &RPCError{Code: ErrCodeMethodNotFound, Message: fmt.Sprintf("method not found: %s", method)}
}

func NewInvalidParams(msg string) *RPCError {
	return &RPCError{Code: ErrCodeInvalidParams, Message: msg}
}

func NewInternalError(msg string) *RPCError {
	return &RPCError{Code: ErrCodeInternal, Message: msg}
}

// Reader reads Content-Length framed JSON-RPC messages from an io.Reader.
type Reader struct {
	br *bufio.Reader
}

func NewReader(r io.Reader) *Reader {
	return &Reader{br: bufio.NewReader(r)}
}

func (r *Reader) Read() (*Request, error) {
	contentLen := -1
	for {
		line, err := r.br.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if strings.HasPrefix(line, "Content-Length:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("invalid Content-Length: %s", val)
			}
			contentLen = n
		}
	}
	if contentLen < 0 {
		return nil, fmt.Errorf("missing Content-Length header")
	}
	body := make([]byte, contentLen)
	if _, err := io.ReadFull(r.br, body); err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	var req Request
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("unmarshal request: %w", err)
	}
	return &req, nil
}

// Writer writes Content-Length framed JSON-RPC messages to an io.Writer.
type Writer struct {
	w  io.Writer
	mu sync.Mutex
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

func (w *Writer) Write(msg any) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	if _, err := io.WriteString(w.w, header); err != nil {
		return err
	}
	_, err = w.w.Write(body)
	return err
}

func (w *Writer) WriteResponse(id json.RawMessage, result any) error {
	return w.Write(&Response{
		JSONRPC: jsonRPCVersion,
		ID:      id,
		Result:  result,
	})
}

func (w *Writer) WriteError(id json.RawMessage, rpcErr *RPCError) error {
	return w.Write(&Response{
		JSONRPC: jsonRPCVersion,
		ID:      id,
		Error:   rpcErr,
	})
}
