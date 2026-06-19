package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/gem-squared/gem2-lfs/internal/store"
)

func sendMsg(w io.Writer, msg any) {
	body, _ := json.Marshal(msg)
	fmt.Fprintf(w, "Content-Length: %d\r\n\r\n%s", len(body), body)
}

func readMsg(r *Reader) (*Response, error) {
	raw, err := r.Read()
	if err != nil {
		return nil, err
	}
	body, _ := json.Marshal(raw)
	var resp Response
	json.Unmarshal(body, &resp)
	return &resp, nil
}

func setupServer(t *testing.T) (*store.DB, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	stdin := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	return db, stdin, stdout
}

func runServerOnce(t *testing.T, db *store.DB, input []byte) []byte {
	t.Helper()
	stdin := bytes.NewReader(input)
	stdout := &bytes.Buffer{}

	srv := NewServer(db, nil, "sqlite-only", stdin, stdout)
	srv.RegisterAllTools()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	srv.Run(ctx)
	return stdout.Bytes()
}

func buildRequest(id int, method string, params any) []byte {
	p, _ := json.Marshal(params)
	msg := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
	}
	if params != nil {
		msg["params"] = json.RawMessage(p)
	}
	body, _ := json.Marshal(msg)
	return []byte(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body))
}

func buildNotification(method string) []byte {
	msg := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
	}
	body, _ := json.Marshal(msg)
	return []byte(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body))
}

func parseResponses(t *testing.T, raw []byte) []Response {
	t.Helper()
	reader := NewReader(bytes.NewReader(raw))
	var responses []Response
	for {
		req, err := reader.Read()
		if err != nil {
			break
		}
		b, _ := json.Marshal(req)
		var resp Response
		json.Unmarshal(b, &resp)
		responses = append(responses, resp)
	}
	return responses
}

func TestInitializeHandshake(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	input := buildRequest(1, "initialize", map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "test", "version": "1.0"},
	})

	output := runServerOnce(t, db, input)
	reader := NewReader(bytes.NewReader(output))
	req, err := reader.Read()
	if err != nil {
		t.Fatalf("read response: %v", err)
	}

	// Parse the response from the raw request format.
	respBody, _ := json.Marshal(req)
	var raw map[string]json.RawMessage
	json.Unmarshal(respBody, &raw)

	// The response was read as a Request (since Reader reads Request), but
	// the actual response is in the wire format. Let's re-parse the output directly.
	respReader := NewReader(bytes.NewReader(output))
	respReq, _ := respReader.Read()

	// respReq.Params contains the result (since we're reading via Request type).
	// Actually, let's just parse the raw output as JSON.
	parts := strings.SplitN(string(output), "\r\n\r\n", 2)
	if len(parts) < 2 {
		t.Fatalf("no body in response")
	}
	var resp Response
	if err := json.Unmarshal([]byte(parts[1]), &resp); err != nil {
		t.Fatalf("unmarshal response: %v (body: %s, id: %v)", err, parts[1], respReq)
	}

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	result, _ := json.Marshal(resp.Result)
	resultStr := string(result)
	if !strings.Contains(resultStr, "2024-11-05") {
		t.Errorf("expected protocolVersion 2024-11-05, got: %s", resultStr)
	}
	if !strings.Contains(resultStr, "gem2-lfs") {
		t.Errorf("expected server name gem2-lfs, got: %s", resultStr)
	}
}

func TestToolsList(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	input := buildRequest(1, "tools/list", nil)
	output := runServerOnce(t, db, input)

	parts := strings.SplitN(string(output), "\r\n\r\n", 2)
	if len(parts) < 2 {
		t.Fatalf("no body")
	}
	var resp Response
	json.Unmarshal([]byte(parts[1]), &resp)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	resultMap, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("result is not a map")
	}
	tools, ok := resultMap["tools"].([]any)
	if !ok {
		t.Fatalf("tools is not an array")
	}

	if len(tools) != 29 {
		t.Errorf("expected 29 tools, got %d", len(tools))
	}

	// Check a few known tools exist.
	toolNames := map[string]bool{}
	for _, tool := range tools {
		tm, _ := tool.(map[string]any)
		name, _ := tm["name"].(string)
		toolNames[name] = true
	}
	for _, expected := range []string{"gem2_task_create", "gem2_msg_search", "gem2_session_context", "gem2_project_list"} {
		if !toolNames[expected] {
			t.Errorf("missing tool: %s", expected)
		}
	}
}

func TestToolCallTaskCreateAndSearch(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Create a task.
	createInput := buildRequest(1, "tools/call", map[string]any{
		"name": "gem2_task_create",
		"arguments": map[string]any{
			"title":        "Test task",
			"project_slug": "test-project",
			"role":         "ARCHITECT",
			"priority":     "HIGH",
		},
	})

	createOutput := runServerOnce(t, db, createInput)
	parts := strings.SplitN(string(createOutput), "\r\n\r\n", 2)
	var createResp Response
	json.Unmarshal([]byte(parts[1]), &createResp)

	if createResp.Error != nil {
		t.Fatalf("create error: %v", createResp.Error)
	}

	resultMap := createResp.Result.(map[string]any)
	content := resultMap["content"].([]any)
	textObj := content[0].(map[string]any)
	text := textObj["text"].(string)

	if !strings.Contains(text, "Test task") {
		t.Errorf("expected task title in response, got: %s", text)
	}

	// Search for the task.
	searchInput := buildRequest(2, "tools/call", map[string]any{
		"name": "gem2_task_search",
		"arguments": map[string]any{
			"project_slug": "test-project",
		},
	})

	searchOutput := runServerOnce(t, db, searchInput)
	parts = strings.SplitN(string(searchOutput), "\r\n\r\n", 2)
	var searchResp Response
	json.Unmarshal([]byte(parts[1]), &searchResp)

	if searchResp.Error != nil {
		t.Fatalf("search error: %v", searchResp.Error)
	}

	searchResult := searchResp.Result.(map[string]any)
	searchContent := searchResult["content"].([]any)
	searchText := searchContent[0].(map[string]any)["text"].(string)

	if !strings.Contains(searchText, "Test task") {
		t.Errorf("expected task in search results, got: %s", searchText)
	}
}

func TestToolCallUnknownTool(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	input := buildRequest(1, "tools/call", map[string]any{
		"name":      "nonexistent_tool",
		"arguments": map[string]any{},
	})

	output := runServerOnce(t, db, input)
	parts := strings.SplitN(string(output), "\r\n\r\n", 2)
	var resp Response
	json.Unmarshal([]byte(parts[1]), &resp)

	if resp.Error == nil {
		t.Fatal("expected error for unknown tool")
	}
	if resp.Error.Code != ErrCodeMethodNotFound {
		t.Errorf("expected method not found error code %d, got %d", ErrCodeMethodNotFound, resp.Error.Code)
	}
}

func TestPing(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	input := buildRequest(1, "ping", nil)
	output := runServerOnce(t, db, input)

	parts := strings.SplitN(string(output), "\r\n\r\n", 2)
	var resp Response
	json.Unmarshal([]byte(parts[1]), &resp)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}
