package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sort"

	"github.com/gem-squared/gem2-lfs/internal/embedding"
	"github.com/gem-squared/gem2-lfs/internal/store"
)

const (
	protocolVersion = "2024-11-05"
	serverName      = "gem2-lfs"
	serverVersion   = "0.1.0"
)

type ToolSchema struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	InputSchema any    `json:"inputSchema"`
}

type ToolHandler func(args json.RawMessage) (any, *RPCError)

type tool struct {
	schema  ToolSchema
	handler ToolHandler
}

type Server struct {
	db       *store.DB
	embedSvc *embedding.OllamaService
	mode     string
	tools    map[string]tool
	reader   *Reader
	writer   *Writer
}

func NewServer(db *store.DB, embedSvc *embedding.OllamaService, mode string, r io.Reader, w io.Writer) *Server {
	s := &Server{
		db:       db,
		embedSvc: embedSvc,
		mode:     mode,
		tools:    make(map[string]tool),
		reader:   NewReader(r),
		writer:   NewWriter(w),
	}
	return s
}

func (s *Server) RegisterTool(schema ToolSchema, handler ToolHandler) {
	s.tools[schema.Name] = tool{schema: schema, handler: handler}
}

func (s *Server) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		req, err := s.reader.Read()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("read: %w", err)
		}

		if req.IsNotification() {
			s.handleNotification(req)
			continue
		}

		resp := s.dispatch(req)
		if err := s.writer.Write(resp); err != nil {
			return fmt.Errorf("write: %w", err)
		}
	}
}

func (s *Server) handleNotification(req *Request) {
	switch req.Method {
	case "notifications/initialized":
		log.Printf("mcp: client initialized")
	case "notifications/cancelled":
		log.Printf("mcp: request cancelled")
	default:
		log.Printf("mcp: unknown notification: %s", req.Method)
	}
}

func (s *Server) dispatch(req *Request) *Response {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(req)
	case "ping":
		return &Response{JSONRPC: jsonRPCVersion, ID: req.ID, Result: map[string]any{}}
	default:
		return &Response{
			JSONRPC: jsonRPCVersion,
			ID:      req.ID,
			Error:   NewMethodNotFound(req.Method),
		}
	}
}

func (s *Server) handleInitialize(req *Request) *Response {
	return &Response{
		JSONRPC: jsonRPCVersion,
		ID:      req.ID,
		Result: map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    serverName,
				"version": serverVersion,
			},
		},
	}
}

func (s *Server) handleToolsList(req *Request) *Response {
	schemas := make([]ToolSchema, 0, len(s.tools))
	for _, t := range s.tools {
		schemas = append(schemas, t.schema)
	}
	sort.Slice(schemas, func(i, j int) bool {
		return schemas[i].Name < schemas[j].Name
	})
	return &Response{
		JSONRPC: jsonRPCVersion,
		ID:      req.ID,
		Result: map[string]any{
			"tools": schemas,
		},
	}
}

func (s *Server) handleToolsCall(req *Request) *Response {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &Response{
			JSONRPC: jsonRPCVersion,
			ID:      req.ID,
			Error:   NewInvalidParams(err.Error()),
		}
	}

	t, ok := s.tools[params.Name]
	if !ok {
		return &Response{
			JSONRPC: jsonRPCVersion,
			ID:      req.ID,
			Error:   NewMethodNotFound(params.Name),
		}
	}

	result, rpcErr := t.handler(params.Arguments)
	if rpcErr != nil {
		resultJSON, _ := json.Marshal(map[string]any{"error": rpcErr.Message})
		return &Response{
			JSONRPC: jsonRPCVersion,
			ID:      req.ID,
			Result: map[string]any{
				"content": []map[string]any{
					{"type": "text", "text": string(resultJSON)},
				},
				"isError": true,
			},
		}
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		resultJSON = []byte(fmt.Sprintf(`{"error":"marshal: %s"}`, err.Error()))
		return &Response{
			JSONRPC: jsonRPCVersion,
			ID:      req.ID,
			Result: map[string]any{
				"content": []map[string]any{
					{"type": "text", "text": string(resultJSON)},
				},
				"isError": true,
			},
		}
	}

	return &Response{
		JSONRPC: jsonRPCVersion,
		ID:      req.ID,
		Result: map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": string(resultJSON)},
			},
		},
	}
}
