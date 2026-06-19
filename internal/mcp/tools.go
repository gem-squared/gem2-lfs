package mcp

import (
	"encoding/json"
	"log"
	"sort"

	"github.com/gem-squared/gem2-lfs/internal/embedding"
	"github.com/gem-squared/gem2-lfs/internal/store"
)

func (s *Server) RegisterAllTools() {
	s.registerTaskTools()
	s.registerMessageTools()
	s.registerDecisionTools()
	s.registerKnowledgeTools()
	s.registerEdgeTools()
	s.registerSkillTools()
	s.registerSessionTools()
	s.registerProjectTools()
}

// --- helpers ---

func props(fields ...any) map[string]any {
	p := map[string]any{}
	for i := 0; i < len(fields)-1; i += 2 {
		p[fields[i].(string)] = fields[i+1]
	}
	return p
}

func str(desc string) map[string]any     { return map[string]any{"type": "string", "description": desc} }
func num(desc string) map[string]any     { return map[string]any{"type": "integer", "description": desc} }
func boolean(desc string) map[string]any { return map[string]any{"type": "boolean", "description": desc} }
func strArr(desc string) map[string]any {
	return map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": desc}
}

func schema(properties map[string]any, required []string) map[string]any {
	s := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		s["required"] = required
	}
	return s
}

// --- Tasks (4) ---

func (s *Server) registerTaskTools() {
	s.RegisterTool(ToolSchema{
		Name:        "gem2_task_create",
		Description: "Create a new task (work item)",
		InputSchema: schema(props(
			"role", str("Role creating the task"),
			"title", str("Task title"),
			"project_slug", str("Project slug"),
			"description", str("Task description"),
			"priority", str("Priority: LOW, MEDIUM, HIGH, CRITICAL"),
			"plan_doc", str("Plan document reference"),
			"parent_task_id", str("Parent task ID for subtasks"),
			"tags", strArr("Tags"),
			"metadata_json", str("Additional metadata as JSON string"),
		), []string{"title", "project_slug"}),
	}, func(args json.RawMessage) (any, *RPCError) {
		var req struct {
			Role         string   `json:"role"`
			Title        string   `json:"title"`
			ProjectSlug  string   `json:"project_slug"`
			Description  string   `json:"description"`
			Priority     string   `json:"priority"`
			PlanDoc      string   `json:"plan_doc"`
			ParentTaskID string   `json:"parent_task_id"`
			Tags         []string `json:"tags"`
			MetadataJSON string   `json:"metadata_json"`
		}
		if e := json.Unmarshal(args, &req); e != nil {
			return nil, NewInvalidParams(e.Error())
		}
		t := &store.Task{
			Role: req.Role, Title: req.Title, Description: req.Description,
			Priority: req.Priority, ProjectSlug: req.ProjectSlug, PlanDoc: req.PlanDoc,
			ParentTaskID: req.ParentTaskID, Tags: req.Tags, MetadataJSON: req.MetadataJSON,
		}
		if err := s.db.CreateTask(t); err != nil {
			return nil, NewInternalError(err.Error())
		}
		return map[string]any{"task": t}, nil
	})

	s.RegisterTool(ToolSchema{
		Name:        "gem2_task_get",
		Description: "Get a task by ID",
		InputSchema: schema(props("task_id", str("Task ID")), []string{"task_id"}),
	}, func(args json.RawMessage) (any, *RPCError) {
		var req struct {
			TaskID string `json:"task_id"`
		}
		if e := json.Unmarshal(args, &req); e != nil {
			return nil, NewInvalidParams(e.Error())
		}
		t, err := s.db.GetTask(req.TaskID)
		if err != nil {
			return nil, NewInternalError(err.Error())
		}
		return map[string]any{"task": t}, nil
	})

	s.RegisterTool(ToolSchema{
		Name:        "gem2_task_search",
		Description: "Search tasks by role, status, project, priority, query, or tags",
		InputSchema: schema(props(
			"role", str("Filter by role"),
			"status", str("Filter by status: PENDING, IN_PROGRESS, COMPLETED, ABORTED, BLOCKED"),
			"project_slug", str("Filter by project"),
			"priority", str("Filter by priority"),
			"query", str("Full-text search query"),
			"tags", strArr("Filter by tags"),
			"page_size", num("Max results to return"),
		), nil),
	}, func(args json.RawMessage) (any, *RPCError) {
		var req struct {
			Role        string   `json:"role"`
			Status      string   `json:"status"`
			ProjectSlug string   `json:"project_slug"`
			Priority    string   `json:"priority"`
			Query       string   `json:"query"`
			Tags        []string `json:"tags"`
			PageSize    int      `json:"page_size"`
		}
		if e := json.Unmarshal(args, &req); e != nil {
			return nil, NewInvalidParams(e.Error())
		}
		tasks, err := s.db.SearchTasks(req.Role, req.Status, req.ProjectSlug, req.Priority, req.Query, req.Tags, req.PageSize)
		if err != nil {
			return nil, NewInternalError(err.Error())
		}
		if tasks == nil {
			tasks = []store.Task{}
		}
		return map[string]any{"results": tasks}, nil
	})

	s.RegisterTool(ToolSchema{
		Name:        "gem2_task_update",
		Description: "Update a task's status, result, state, tags, or metadata",
		InputSchema: schema(props(
			"task_id", str("Task ID"),
			"status", str("New status"),
			"result_summary", str("Result summary"),
			"state", str("State: SUCCESS or FAILURE"),
			"tags", strArr("Updated tags"),
			"metadata_json", str("Updated metadata"),
		), []string{"task_id"}),
	}, func(args json.RawMessage) (any, *RPCError) {
		var req struct {
			TaskID        string   `json:"task_id"`
			Status        string   `json:"status"`
			ResultSummary string   `json:"result_summary"`
			State         string   `json:"state"`
			Tags          []string `json:"tags"`
			MetadataJSON  string   `json:"metadata_json"`
		}
		if e := json.Unmarshal(args, &req); e != nil {
			return nil, NewInvalidParams(e.Error())
		}
		t, err := s.db.UpdateTask(req.TaskID, req.Status, req.ResultSummary, req.State, req.Tags, req.MetadataJSON)
		if err != nil {
			return nil, NewInternalError(err.Error())
		}
		return map[string]any{"task": t}, nil
	})
}

// --- Messages (3) ---

func (s *Server) registerMessageTools() {
	s.RegisterTool(ToolSchema{
		Name:        "gem2_msg_create",
		Description: "Create a protocol message between roles",
		InputSchema: schema(props(
			"from_role", str("Sender role"),
			"to_role", str("Recipient role"),
			"message", str("Message summary"),
			"content", str("Full message content"),
			"project_slug", str("Project slug"),
			"session_id", str("Session ID"),
			"terminal_id", str("Terminal ID"),
			"terminal_title", str("Terminal title"),
			"metadata_json", str("Additional metadata"),
		), []string{"from_role", "to_role", "message"}),
	}, func(args json.RawMessage) (any, *RPCError) {
		var req struct {
			FromRole      string `json:"from_role"`
			ToRole        string `json:"to_role"`
			Message       string `json:"message"`
			Content       string `json:"content"`
			ProjectSlug   string `json:"project_slug"`
			SessionID     string `json:"session_id"`
			TerminalID    string `json:"terminal_id"`
			TerminalTitle string `json:"terminal_title"`
			MetadataJSON  string `json:"metadata_json"`
		}
		if e := json.Unmarshal(args, &req); e != nil {
			return nil, NewInvalidParams(e.Error())
		}
		m := &store.Message{
			FromRole: req.FromRole, ToRole: req.ToRole, MessageText: req.Message,
			Content: req.Content, ProjectSlug: req.ProjectSlug, SessionID: req.SessionID,
			TerminalID: req.TerminalID, TerminalTitle: req.TerminalTitle, MetadataJSON: req.MetadataJSON,
		}
		if err := s.db.CreateMessage(m); err != nil {
			return nil, NewInternalError(err.Error())
		}
		return map[string]any{"message": m}, nil
	})

	s.RegisterTool(ToolSchema{
		Name:        "gem2_msg_get",
		Description: "Get a message by ID",
		InputSchema: schema(props("msg_id", str("Message ID")), []string{"msg_id"}),
	}, func(args json.RawMessage) (any, *RPCError) {
		var req struct {
			MsgID string `json:"msg_id"`
		}
		if e := json.Unmarshal(args, &req); e != nil {
			return nil, NewInvalidParams(e.Error())
		}
		m, err := s.db.GetMessage(req.MsgID)
		if err != nil {
			return nil, NewInternalError(err.Error())
		}
		return map[string]any{"message": m}, nil
	})

	s.RegisterTool(ToolSchema{
		Name:        "gem2_msg_search",
		Description: "Search messages by role, project, session, or full-text query",
		InputSchema: schema(props(
			"from_role", str("Filter by sender role"),
			"to_role", str("Filter by recipient role"),
			"project_slug", str("Filter by project"),
			"session_id", str("Filter by session"),
			"query", str("Full-text search query"),
			"page_size", num("Max results"),
		), nil),
	}, func(args json.RawMessage) (any, *RPCError) {
		var req struct {
			FromRole    string `json:"from_role"`
			ToRole      string `json:"to_role"`
			ProjectSlug string `json:"project_slug"`
			SessionID   string `json:"session_id"`
			Query       string `json:"query"`
			PageSize    int    `json:"page_size"`
		}
		if e := json.Unmarshal(args, &req); e != nil {
			return nil, NewInvalidParams(e.Error())
		}
		msgs, err := s.db.SearchMessages(req.FromRole, req.ToRole, req.ProjectSlug, req.SessionID, req.Query, req.PageSize)
		if err != nil {
			return nil, NewInternalError(err.Error())
		}
		if msgs == nil {
			msgs = []store.Message{}
		}
		return map[string]any{"results": msgs}, nil
	})
}

// --- Decisions (3) ---

func (s *Server) registerDecisionTools() {
	s.RegisterTool(ToolSchema{
		Name:        "gem2_decision_create",
		Description: "Create an architectural decision record",
		InputSchema: schema(props(
			"role", str("Role making the decision"),
			"category", str("Decision category"),
			"title", str("Decision title"),
			"content", str("Decision content/rationale"),
			"status", str("Status: ACTIVE, SUPERSEDED, DEPRECATED"),
			"project_slug", str("Project slug"),
			"tags", strArr("Tags"),
			"metadata_json", str("Additional metadata"),
		), []string{"title", "content"}),
	}, func(args json.RawMessage) (any, *RPCError) {
		var req struct {
			Role         string   `json:"role"`
			Category     string   `json:"category"`
			Title        string   `json:"title"`
			Content      string   `json:"content"`
			Status       string   `json:"status"`
			ProjectSlug  string   `json:"project_slug"`
			Tags         []string `json:"tags"`
			MetadataJSON string   `json:"metadata_json"`
		}
		if e := json.Unmarshal(args, &req); e != nil {
			return nil, NewInvalidParams(e.Error())
		}
		d := &store.Decision{
			Role: req.Role, Category: req.Category, Title: req.Title, Content: req.Content,
			Status: req.Status, ProjectSlug: req.ProjectSlug, Tags: req.Tags, MetadataJSON: req.MetadataJSON,
		}
		if err := s.db.CreateDecision(d); err != nil {
			return nil, NewInternalError(err.Error())
		}
		return map[string]any{"decision": d}, nil
	})

	s.RegisterTool(ToolSchema{
		Name:        "gem2_decision_search",
		Description: "Search decisions by role, category, status, project, query, or tags",
		InputSchema: schema(props(
			"role", str("Filter by role"),
			"category", str("Filter by category"),
			"status", str("Filter by status"),
			"project_slug", str("Filter by project"),
			"query", str("Full-text search query"),
			"tags", strArr("Filter by tags"),
			"page_size", num("Max results"),
		), nil),
	}, func(args json.RawMessage) (any, *RPCError) {
		var req struct {
			Role        string   `json:"role"`
			Category    string   `json:"category"`
			Status      string   `json:"status"`
			ProjectSlug string   `json:"project_slug"`
			Query       string   `json:"query"`
			Tags        []string `json:"tags"`
			PageSize    int      `json:"page_size"`
		}
		if e := json.Unmarshal(args, &req); e != nil {
			return nil, NewInvalidParams(e.Error())
		}
		decs, err := s.db.SearchDecisions(req.Role, req.Category, req.Status, req.ProjectSlug, req.Query, req.Tags, req.PageSize)
		if err != nil {
			return nil, NewInternalError(err.Error())
		}
		if decs == nil {
			decs = []store.Decision{}
		}
		return map[string]any{"results": decs}, nil
	})

	s.RegisterTool(ToolSchema{
		Name:        "gem2_decision_update",
		Description: "Update a decision's status, content, or tags",
		InputSchema: schema(props(
			"decision_id", str("Decision ID"),
			"status", str("New status"),
			"superseded_by", str("ID of superseding decision"),
			"tags", strArr("Updated tags"),
			"content", str("Updated content"),
		), []string{"decision_id"}),
	}, func(args json.RawMessage) (any, *RPCError) {
		var req struct {
			DecisionID   string   `json:"decision_id"`
			Status       string   `json:"status"`
			SupersededBy string   `json:"superseded_by"`
			Tags         []string `json:"tags"`
			Content      string   `json:"content"`
		}
		if e := json.Unmarshal(args, &req); e != nil {
			return nil, NewInvalidParams(e.Error())
		}
		d, err := s.db.UpdateDecision(req.DecisionID, req.Status, req.SupersededBy, req.Tags, req.Content)
		if err != nil {
			return nil, NewInternalError(err.Error())
		}
		return map[string]any{"decision": d}, nil
	})
}

// --- Knowledge (5) + extras (2) ---

func (s *Server) registerKnowledgeTools() {
	s.RegisterTool(ToolSchema{
		Name:        "gem2_knowledge_create",
		Description: "Create a knowledge document",
		InputSchema: schema(props(
			"entity_type", str("Entity type (e.g., code, doc, decision)"),
			"title", str("Document title"),
			"content", str("Raw content"),
			"content_nl", str("Natural language description (used for embeddings)"),
			"project_slug", str("Project slug"),
			"tags", strArr("Tags"),
			"file_path", str("Source file path"),
			"line_count", num("Line count"),
			"metadata_json", str("Additional metadata as JSON object"),
		), []string{"title", "content"}),
	}, func(args json.RawMessage) (any, *RPCError) {
		var req struct {
			EntityType  string         `json:"entity_type"`
			Title       string         `json:"title"`
			ContentRaw  string         `json:"content"`
			ContentNL   string         `json:"content_nl"`
			ProjectSlug string         `json:"project_slug"`
			Tags        []string       `json:"tags"`
			FilePath    string         `json:"file_path"`
			LineCount   int            `json:"line_count"`
			Metadata    map[string]any `json:"metadata_json"`
		}
		if e := json.Unmarshal(args, &req); e != nil {
			return nil, NewInvalidParams(e.Error())
		}
		k := &store.Knowledge{
			EntityType: req.EntityType, Title: req.Title, ContentRaw: req.ContentRaw,
			ContentNL: req.ContentNL, ProjectSlug: req.ProjectSlug, Tags: req.Tags,
			FilePath: req.FilePath, LineCount: req.LineCount, Metadata: req.Metadata,
		}
		chunks, err := s.db.CreateKnowledge(k)
		if err != nil {
			return nil, NewInternalError(err.Error())
		}
		if s.embedSvc != nil && k.ContentNL != "" {
			go s.asyncEmbed(k.KnowledgeID, k.ContentNL)
		}
		return map[string]any{"knowledge": k, "chunks_created": chunks}, nil
	})

	s.RegisterTool(ToolSchema{
		Name:        "gem2_knowledge_get",
		Description: "Get a knowledge document by ID (includes chunks)",
		InputSchema: schema(props("knowledge_id", str("Knowledge ID")), []string{"knowledge_id"}),
	}, func(args json.RawMessage) (any, *RPCError) {
		var req struct {
			KnowledgeID string `json:"knowledge_id"`
		}
		if e := json.Unmarshal(args, &req); e != nil {
			return nil, NewInvalidParams(e.Error())
		}
		k, chunks, err := s.db.GetKnowledge(req.KnowledgeID)
		if err != nil {
			return nil, NewInternalError(err.Error())
		}
		return map[string]any{"knowledge": k, "chunks": chunks}, nil
	})

	s.RegisterTool(ToolSchema{
		Name:        "gem2_knowledge_search",
		Description: "Search knowledge by project, entity type, query, or tags",
		InputSchema: schema(props(
			"project_slug", str("Filter by project"),
			"entity_type", str("Filter by entity type"),
			"query", str("Full-text search query"),
			"tags", strArr("Filter by tags"),
			"page_size", num("Max results"),
		), nil),
	}, func(args json.RawMessage) (any, *RPCError) {
		var req struct {
			ProjectSlug string   `json:"project_slug"`
			EntityType  string   `json:"entity_type"`
			Query       string   `json:"query"`
			Tags        []string `json:"tags"`
			PageSize    int      `json:"page_size"`
		}
		if e := json.Unmarshal(args, &req); e != nil {
			return nil, NewInvalidParams(e.Error())
		}
		results, err := s.db.SearchKnowledge(req.ProjectSlug, req.EntityType, req.Query, req.Tags, req.PageSize)
		if err != nil {
			return nil, NewInternalError(err.Error())
		}
		if results == nil {
			results = []store.Knowledge{}
		}
		return map[string]any{"results": results}, nil
	})

	s.RegisterTool(ToolSchema{
		Name:        "gem2_knowledge_upsert",
		Description: "Create or update a knowledge document (deduplicates by title+entity_type+project)",
		InputSchema: schema(props(
			"entity_type", str("Entity type"),
			"title", str("Document title"),
			"content", str("Raw content"),
			"content_nl", str("Natural language description"),
			"project_slug", str("Project slug"),
			"tags", strArr("Tags"),
			"file_path", str("Source file path"),
			"line_count", num("Line count"),
			"user_id", str("User ID"),
			"metadata_json", str("Additional metadata as JSON object"),
		), []string{"title", "content"}),
	}, func(args json.RawMessage) (any, *RPCError) {
		var req struct {
			EntityType  string         `json:"entity_type"`
			Title       string         `json:"title"`
			ContentRaw  string         `json:"content"`
			ContentNL   string         `json:"content_nl"`
			ProjectSlug string         `json:"project_slug"`
			Tags        []string       `json:"tags"`
			FilePath    string         `json:"file_path"`
			LineCount   int            `json:"line_count"`
			Metadata    map[string]any `json:"metadata_json"`
		}
		if e := json.Unmarshal(args, &req); e != nil {
			return nil, NewInvalidParams(e.Error())
		}
		k := &store.Knowledge{
			EntityType: req.EntityType, Title: req.Title, ContentRaw: req.ContentRaw,
			ContentNL: req.ContentNL, ProjectSlug: req.ProjectSlug, Tags: req.Tags,
			FilePath: req.FilePath, LineCount: req.LineCount, Metadata: req.Metadata,
		}
		knowledgeID, created, version, chunksCreated, err := s.db.UpsertKnowledge(k)
		if err != nil {
			return nil, NewInternalError(err.Error())
		}
		if s.embedSvc != nil && k.ContentNL != "" {
			go s.asyncEmbed(knowledgeID, k.ContentNL)
		}
		return map[string]any{
			"knowledge_id": knowledgeID, "created": created,
			"version": version, "chunks_created": chunksCreated,
		}, nil
	})

	s.RegisterTool(ToolSchema{
		Name:        "gem2_semantic_search",
		Description: "Semantic similarity search over knowledge documents (requires sqlite-ollama mode)",
		InputSchema: schema(props(
			"query", str("Natural language search query"),
			"project_slug", str("Filter by project"),
			"model", str("Embedding model name"),
			"limit", num("Max results"),
		), []string{"query"}),
	}, func(args json.RawMessage) (any, *RPCError) {
		if s.embedSvc == nil {
			return nil, NewInternalError("semantic search requires sqlite-ollama mode")
		}
		var req struct {
			Query       string `json:"query"`
			ProjectSlug string `json:"project_slug"`
			Limit       int    `json:"limit"`
		}
		if e := json.Unmarshal(args, &req); e != nil {
			return nil, NewInvalidParams(e.Error())
		}
		if req.Limit <= 0 {
			req.Limit = 10
		}
		queryVec, err := s.embedSvc.Embed(req.Query)
		if err != nil {
			return nil, NewInternalError("embedding failed: " + err.Error())
		}
		knowledges, embBytes, err := s.db.AllKnowledgeWithEmbeddings()
		if err != nil {
			return nil, NewInternalError(err.Error())
		}
		type scored struct {
			k     store.Knowledge
			score float64
		}
		var matches []scored
		dims := s.embedSvc.Dimensions()
		for i, k := range knowledges {
			if req.ProjectSlug != "" && k.ProjectSlug != req.ProjectSlug {
				continue
			}
			storedVec := embedding.DecodeEmbedding(embBytes[i], dims)
			if storedVec == nil {
				continue
			}
			sim := embedding.CosineSimilarity(queryVec, storedVec)
			matches = append(matches, scored{k: k, score: sim})
		}
		sort.Slice(matches, func(i, j int) bool { return matches[i].score > matches[j].score })
		if len(matches) > req.Limit {
			matches = matches[:req.Limit]
		}
		var out []store.SemanticResult
		for _, m := range matches {
			out = append(out, store.SemanticResult{Knowledge: m.k, Score: m.score})
		}
		if out == nil {
			out = []store.SemanticResult{}
		}
		return map[string]any{"results": out}, nil
	})

	s.RegisterTool(ToolSchema{
		Name:        "gem2_similar_patterns",
		Description: "Find knowledge documents similar to a given document (requires sqlite-ollama mode)",
		InputSchema: schema(props(
			"knowledge_id", str("Source knowledge ID"),
			"limit", num("Max results"),
			"cross_project_only", boolean("Only return results from other projects"),
		), []string{"knowledge_id"}),
	}, func(args json.RawMessage) (any, *RPCError) {
		if s.embedSvc == nil {
			return nil, NewInternalError("similar patterns requires sqlite-ollama mode")
		}
		var req struct {
			KnowledgeID      string `json:"knowledge_id"`
			Limit            int    `json:"limit"`
			CrossProjectOnly bool   `json:"cross_project_only"`
		}
		if e := json.Unmarshal(args, &req); e != nil {
			return nil, NewInvalidParams(e.Error())
		}
		if req.Limit <= 0 {
			req.Limit = 10
		}
		sourceK, _, err := s.db.GetKnowledge(req.KnowledgeID)
		if err != nil {
			return nil, NewInternalError(err.Error())
		}
		knowledges, embBytes, err := s.db.AllKnowledgeWithEmbeddings()
		if err != nil {
			return nil, NewInternalError(err.Error())
		}
		dims := s.embedSvc.Dimensions()
		var sourceVec []float32
		for i, k := range knowledges {
			if k.KnowledgeID == req.KnowledgeID {
				sourceVec = embedding.DecodeEmbedding(embBytes[i], dims)
				break
			}
		}
		if sourceVec == nil {
			text := sourceK.ContentNL
			if text == "" {
				text = sourceK.ContentRaw
			}
			sourceVec, err = s.embedSvc.Embed(text)
			if err != nil {
				return nil, NewInternalError("embedding failed: " + err.Error())
			}
			s.db.StoreEmbedding(req.KnowledgeID, embedding.EncodeEmbedding(sourceVec), s.embedSvc.Model(), dims)
		}
		type scored struct {
			k     store.Knowledge
			score float64
		}
		var matches []scored
		for i, k := range knowledges {
			if k.KnowledgeID == req.KnowledgeID {
				continue
			}
			if req.CrossProjectOnly && k.ProjectSlug == sourceK.ProjectSlug {
				continue
			}
			storedVec := embedding.DecodeEmbedding(embBytes[i], dims)
			if storedVec == nil {
				continue
			}
			sim := embedding.CosineSimilarity(sourceVec, storedVec)
			matches = append(matches, scored{k: k, score: sim})
		}
		sort.Slice(matches, func(i, j int) bool { return matches[i].score > matches[j].score })
		if len(matches) > req.Limit {
			matches = matches[:req.Limit]
		}
		var out []store.SemanticResult
		for _, m := range matches {
			out = append(out, store.SemanticResult{Knowledge: m.k, Score: m.score})
		}
		if out == nil {
			out = []store.SemanticResult{}
		}
		return map[string]any{"results": out}, nil
	})

	s.RegisterTool(ToolSchema{
		Name:        "gem2_list_models",
		Description: "List available embedding models",
		InputSchema: schema(props(), nil),
	}, func(args json.RawMessage) (any, *RPCError) {
		models := []store.EmbeddingModel{}
		if s.embedSvc != nil {
			models = append(models, store.EmbeddingModel{
				ModelName:  s.embedSvc.Model(),
				Provider:   "ollama",
				Dimensions: s.embedSvc.Dimensions(),
				Status:     "active",
			})
		}
		return map[string]any{"models": models}, nil
	})
}

// --- Edges (3) ---

func (s *Server) registerEdgeTools() {
	s.RegisterTool(ToolSchema{
		Name:        "gem2_edge_create",
		Description: "Create a provenance edge between two knowledge documents",
		InputSchema: schema(props(
			"source_id", str("Source knowledge ID"),
			"target_id", str("Target knowledge ID"),
			"project_slug", str("Project slug"),
			"tags", strArr("Edge type tags"),
			"metadata_json", str("Additional metadata"),
		), []string{"source_id", "target_id"}),
	}, func(args json.RawMessage) (any, *RPCError) {
		var req struct {
			SourceID     string   `json:"source_id"`
			TargetID     string   `json:"target_id"`
			ProjectSlug  string   `json:"project_slug"`
			Tags         []string `json:"tags"`
			MetadataJSON string   `json:"metadata_json"`
		}
		if e := json.Unmarshal(args, &req); e != nil {
			return nil, NewInvalidParams(e.Error())
		}
		edge := &store.Edge{
			SourceID: req.SourceID, TargetID: req.TargetID,
			ProjectSlug: req.ProjectSlug, Tags: req.Tags, MetadataJSON: req.MetadataJSON,
		}
		if err := s.db.CreateEdge(edge); err != nil {
			return nil, NewInternalError(err.Error())
		}
		return map[string]any{"edge": edge}, nil
	})

	s.RegisterTool(ToolSchema{
		Name:        "gem2_edge_search",
		Description: "Search edges by source, target, project, or tags",
		InputSchema: schema(props(
			"source_id", str("Filter by source ID"),
			"target_id", str("Filter by target ID"),
			"project_slug", str("Filter by project"),
			"tags", strArr("Filter by tags"),
			"page_size", num("Max results"),
		), nil),
	}, func(args json.RawMessage) (any, *RPCError) {
		var req struct {
			SourceID    string   `json:"source_id"`
			TargetID    string   `json:"target_id"`
			ProjectSlug string   `json:"project_slug"`
			Tags        []string `json:"tags"`
			PageSize    int      `json:"page_size"`
		}
		if e := json.Unmarshal(args, &req); e != nil {
			return nil, NewInvalidParams(e.Error())
		}
		edges, err := s.db.SearchEdges(req.SourceID, req.TargetID, req.ProjectSlug, req.Tags, req.PageSize)
		if err != nil {
			return nil, NewInternalError(err.Error())
		}
		if edges == nil {
			edges = []store.Edge{}
		}
		return map[string]any{"results": edges}, nil
	})

	s.RegisterTool(ToolSchema{
		Name:        "gem2_edge_delete",
		Description: "Delete a provenance edge by ID",
		InputSchema: schema(props("edge_id", str("Edge ID")), []string{"edge_id"}),
	}, func(args json.RawMessage) (any, *RPCError) {
		var req struct {
			EdgeID string `json:"edge_id"`
		}
		if e := json.Unmarshal(args, &req); e != nil {
			return nil, NewInvalidParams(e.Error())
		}
		if err := s.db.DeleteEdge(req.EdgeID); err != nil {
			return nil, NewInternalError(err.Error())
		}
		return map[string]any{"deleted": true}, nil
	})
}

// --- Skills (4) ---

func (s *Server) registerSkillTools() {
	s.RegisterTool(ToolSchema{
		Name:        "gem2_skill_upsert",
		Description: "Create or update a skill (grounding prompt or formal contract)",
		InputSchema: schema(props(
			"skill_id", str("Skill ID (for update; auto-generated if omitted)"),
			"user_id", str("User/owner ID"),
			"skill_type", str("Skill type"),
			"title", str("Skill title"),
			"content", str("Skill content (TPMN contract or prompt)"),
			"tags", strArr("Tags"),
			"source_tool", str("Source tool that generated this skill"),
			"metadata", str("Additional metadata"),
		), []string{"title", "content"}),
	}, func(args json.RawMessage) (any, *RPCError) {
		var req struct {
			SkillID    string   `json:"skill_id"`
			UserID     string   `json:"user_id"`
			SkillType  string   `json:"skill_type"`
			Title      string   `json:"title"`
			Content    string   `json:"content"`
			Tags       []string `json:"tags"`
			SourceTool string   `json:"source_tool"`
			Metadata   string   `json:"metadata"`
		}
		if e := json.Unmarshal(args, &req); e != nil {
			return nil, NewInvalidParams(e.Error())
		}
		sk := &store.Skill{
			SkillID: req.SkillID, UserID: req.UserID, SkillType: req.SkillType,
			Title: req.Title, Content: req.Content, Tags: req.Tags,
			SourceTool: req.SourceTool, Metadata: req.Metadata,
		}
		if err := s.db.UpsertSkill(sk); err != nil {
			return nil, NewInternalError(err.Error())
		}
		return map[string]any{"skill": sk}, nil
	})

	s.RegisterTool(ToolSchema{
		Name:        "gem2_skill_get",
		Description: "Get a skill by ID",
		InputSchema: schema(props(
			"skill_id", str("Skill ID"),
			"user_id", str("User ID"),
		), []string{"skill_id"}),
	}, func(args json.RawMessage) (any, *RPCError) {
		var req struct {
			SkillID string `json:"skill_id"`
			UserID  string `json:"user_id"`
		}
		if e := json.Unmarshal(args, &req); e != nil {
			return nil, NewInvalidParams(e.Error())
		}
		sk, err := s.db.GetSkill(req.SkillID, req.UserID)
		if err != nil {
			return nil, NewInternalError(err.Error())
		}
		return map[string]any{"skill": sk}, nil
	})

	s.RegisterTool(ToolSchema{
		Name:        "gem2_skill_search",
		Description: "Search skills by user, query, type, or tags",
		InputSchema: schema(props(
			"user_id", str("Filter by user"),
			"query", str("Full-text search query"),
			"skill_type", str("Filter by skill type"),
			"tags", strArr("Filter by tags"),
			"page_size", num("Max results"),
		), nil),
	}, func(args json.RawMessage) (any, *RPCError) {
		var req struct {
			UserID    string   `json:"user_id"`
			Query     string   `json:"query"`
			SkillType string   `json:"skill_type"`
			Tags      []string `json:"tags"`
			PageSize  int      `json:"page_size"`
		}
		if e := json.Unmarshal(args, &req); e != nil {
			return nil, NewInvalidParams(e.Error())
		}
		skills, err := s.db.SearchSkills(req.UserID, req.Query, req.SkillType, req.Tags, req.PageSize)
		if err != nil {
			return nil, NewInternalError(err.Error())
		}
		if skills == nil {
			skills = []store.Skill{}
		}
		return map[string]any{"results": skills}, nil
	})

	s.RegisterTool(ToolSchema{
		Name:        "gem2_skill_delete",
		Description: "Delete a skill by ID",
		InputSchema: schema(props(
			"skill_id", str("Skill ID"),
			"user_id", str("User ID"),
		), []string{"skill_id"}),
	}, func(args json.RawMessage) (any, *RPCError) {
		var req struct {
			SkillID string `json:"skill_id"`
			UserID  string `json:"user_id"`
		}
		if e := json.Unmarshal(args, &req); e != nil {
			return nil, NewInvalidParams(e.Error())
		}
		if err := s.db.DeleteSkill(req.SkillID, req.UserID); err != nil {
			return nil, NewInternalError(err.Error())
		}
		return map[string]any{"deleted": true}, nil
	})
}

// --- Session + Status (2) ---

func (s *Server) registerSessionTools() {
	s.RegisterTool(ToolSchema{
		Name:        "gem2_session_context",
		Description: "Get session context: active/pending tasks, recent messages, and active decisions for a project",
		InputSchema: schema(props(
			"role", str("Filter by role"),
			"project_slug", str("Project slug"),
			"msg_limit", num("Max messages (default 10)"),
			"task_limit", num("Max tasks (default 20)"),
			"decision_limit", num("Max decisions (default 10)"),
		), []string{"project_slug"}),
	}, func(args json.RawMessage) (any, *RPCError) {
		var req struct {
			Role          string `json:"role"`
			ProjectSlug   string `json:"project_slug"`
			MsgLimit      int    `json:"msg_limit"`
			TaskLimit     int    `json:"task_limit"`
			DecisionLimit int    `json:"decision_limit"`
		}
		if e := json.Unmarshal(args, &req); e != nil {
			return nil, NewInvalidParams(e.Error())
		}
		if req.MsgLimit <= 0 {
			req.MsgLimit = 10
		}
		if req.TaskLimit <= 0 {
			req.TaskLimit = 20
		}
		if req.DecisionLimit <= 0 {
			req.DecisionLimit = 10
		}
		inProgress, _ := s.db.SearchTasks(req.Role, "IN_PROGRESS", req.ProjectSlug, "", "", nil, req.TaskLimit)
		pending, _ := s.db.SearchTasks(req.Role, "PENDING", req.ProjectSlug, "", "", nil, req.TaskLimit)
		tasks := append(inProgress, pending...)
		messages, _ := s.db.SearchMessages("", "", req.ProjectSlug, "", "", req.MsgLimit)
		decisions, _ := s.db.SearchDecisions("", "", "ACTIVE", req.ProjectSlug, "", nil, req.DecisionLimit)
		if tasks == nil {
			tasks = []store.Task{}
		}
		if messages == nil {
			messages = []store.Message{}
		}
		if decisions == nil {
			decisions = []store.Decision{}
		}
		return map[string]any{"messages": messages, "tasks": tasks, "decisions": decisions}, nil
	})

	s.RegisterTool(ToolSchema{
		Name:        "gem2_status",
		Description: "Get project status: task counters, active tasks, and blockers",
		InputSchema: schema(props(
			"project_slug", str("Project slug"),
			"role", str("Filter by role"),
		), []string{"project_slug"}),
	}, func(args json.RawMessage) (any, *RPCError) {
		var req struct {
			ProjectSlug string `json:"project_slug"`
			Role        string `json:"role"`
		}
		if e := json.Unmarshal(args, &req); e != nil {
			return nil, NewInvalidParams(e.Error())
		}
		counters := map[string]int{}
		for _, status := range []string{"PENDING", "IN_PROGRESS", "COMPLETED", "ABORTED", "BLOCKED"} {
			tasks, err := s.db.SearchTasks(req.Role, status, req.ProjectSlug, "", "", nil, 0)
			if err != nil {
				continue
			}
			counters[status] = len(tasks)
		}
		active, _ := s.db.SearchTasks(req.Role, "IN_PROGRESS", req.ProjectSlug, "", "", nil, 20)
		blocked, _ := s.db.SearchTasks(req.Role, "BLOCKED", req.ProjectSlug, "", "", nil, 20)
		if active == nil {
			active = []store.Task{}
		}
		if blocked == nil {
			blocked = []store.Task{}
		}
		return map[string]any{
			"project_slug": req.ProjectSlug, "task_counters": counters,
			"active_tasks": active, "blockers": blocked,
		}, nil
	})
}

// --- Projects (3) ---

func (s *Server) registerProjectTools() {
	s.RegisterTool(ToolSchema{
		Name:        "gem2_project_create",
		Description: "Create or register a project",
		InputSchema: schema(props(
			"project_slug", str("Project slug (unique identifier)"),
			"name", str("Human-readable project name"),
			"root_path", str("Project root directory path"),
			"metadata_json", str("Additional metadata"),
		), []string{"project_slug", "name"}),
	}, func(args json.RawMessage) (any, *RPCError) {
		var req struct {
			ProjectSlug  string `json:"project_slug"`
			Name         string `json:"name"`
			RootPath     string `json:"root_path"`
			MetadataJSON string `json:"metadata_json"`
		}
		if e := json.Unmarshal(args, &req); e != nil {
			return nil, NewInvalidParams(e.Error())
		}
		p := &store.Project{
			ProjectSlug: req.ProjectSlug, Name: req.Name,
			RootPath: req.RootPath, MetadataJSON: req.MetadataJSON,
		}
		if err := s.db.CreateProject(p); err != nil {
			return nil, NewInternalError(err.Error())
		}
		return map[string]any{"project": p}, nil
	})

	s.RegisterTool(ToolSchema{
		Name:        "gem2_project_get",
		Description: "Get a project by slug",
		InputSchema: schema(props("project_slug", str("Project slug")), []string{"project_slug"}),
	}, func(args json.RawMessage) (any, *RPCError) {
		var req struct {
			ProjectSlug string `json:"project_slug"`
		}
		if e := json.Unmarshal(args, &req); e != nil {
			return nil, NewInvalidParams(e.Error())
		}
		p, err := s.db.GetProject(req.ProjectSlug)
		if err != nil {
			return nil, NewInternalError(err.Error())
		}
		return map[string]any{"project": p}, nil
	})

	s.RegisterTool(ToolSchema{
		Name:        "gem2_project_list",
		Description: "List all registered projects",
		InputSchema: schema(props(), nil),
	}, func(args json.RawMessage) (any, *RPCError) {
		projects, err := s.db.ListProjects()
		if err != nil {
			return nil, NewInternalError(err.Error())
		}
		if projects == nil {
			projects = []store.Project{}
		}
		return map[string]any{"results": projects}, nil
	})
}

// asyncEmbed generates and stores an embedding (non-blocking).
func (s *Server) asyncEmbed(knowledgeID, contentNL string) {
	if contentNL == "" {
		return
	}
	vec, err := s.embedSvc.Embed(contentNL)
	if err != nil {
		log.Printf("mcp: embed %s: %v", knowledgeID, err)
		return
	}
	encoded := embedding.EncodeEmbedding(vec)
	if err := s.db.StoreEmbedding(knowledgeID, encoded, s.embedSvc.Model(), s.embedSvc.Dimensions()); err != nil {
		log.Printf("mcp: store embedding %s: %v", knowledgeID, err)
	}
}
