package api

import (
	"log"
	"net/http"
	"sort"

	"github.com/gem-squared/gem2-lfs/internal/embedding"
	"github.com/gem-squared/gem2-lfs/internal/store"
)

func (s *Server) handleKnowledgeCreate(w http.ResponseWriter, r *http.Request) {
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
	if err := readBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	k := &store.Knowledge{
		EntityType: req.EntityType, Title: req.Title, ContentRaw: req.ContentRaw,
		ContentNL: req.ContentNL, ProjectSlug: req.ProjectSlug, Tags: req.Tags,
		FilePath: req.FilePath, LineCount: req.LineCount, Metadata: req.Metadata,
	}
	chunks, err := s.db.CreateKnowledge(k)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Async embedding generation (non-blocking).
	if s.embedSvc != nil {
		go s.embedKnowledge(k.KnowledgeID, k.ContentNL)
	}

	writeJSON(w, http.StatusOK, map[string]any{"knowledge": k, "chunks_created": chunks})
}

func (s *Server) handleKnowledgeGet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		KnowledgeID string `json:"knowledge_id"`
	}
	if err := readBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	k, chunks, err := s.db.GetKnowledge(req.KnowledgeID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"knowledge": k, "chunks": chunks})
}

func (s *Server) handleKnowledgeSearch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectSlug string   `json:"project_slug"`
		EntityType  string   `json:"entity_type"`
		Query       string   `json:"query"`
		Tags        []string `json:"tags"`
		Pagination  struct {
			PageSize int `json:"page_size"`
		} `json:"pagination"`
	}
	if err := readBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	results, err := s.db.SearchKnowledge(req.ProjectSlug, req.EntityType, req.Query, req.Tags, req.Pagination.PageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if results == nil {
		results = []store.Knowledge{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": results})
}

func (s *Server) handleKnowledgeUpsert(w http.ResponseWriter, r *http.Request) {
	var req struct {
		EntityType  string         `json:"entity_type"`
		Title       string         `json:"title"`
		ContentRaw  string         `json:"content"`
		ContentNL   string         `json:"content_nl"`
		ProjectSlug string         `json:"project_slug"`
		Tags        []string       `json:"tags"`
		FilePath    string         `json:"file_path"`
		LineCount   int            `json:"line_count"`
		UserID      string         `json:"user_id"`
		Metadata    map[string]any `json:"metadata_json"`
	}
	if err := readBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	k := &store.Knowledge{
		EntityType: req.EntityType, Title: req.Title, ContentRaw: req.ContentRaw,
		ContentNL: req.ContentNL, ProjectSlug: req.ProjectSlug, Tags: req.Tags,
		FilePath: req.FilePath, LineCount: req.LineCount, Metadata: req.Metadata,
	}
	knowledgeID, created, version, chunksCreated, err := s.db.UpsertKnowledge(k)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Async embedding generation (non-blocking).
	if s.embedSvc != nil {
		go s.embedKnowledge(knowledgeID, k.ContentNL)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"knowledge_id":   knowledgeID,
		"created":        created,
		"version":        version,
		"chunks_created": chunksCreated,
	})
}

func (s *Server) handleSemanticSearch(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Mode != "sqlite-ollama" || s.embedSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "semantic search requires sqlite-ollama mode")
		return
	}

	var req struct {
		Query       string `json:"query"`
		ProjectSlug string `json:"project_slug"`
		Model       string `json:"model"`
		Limit       int    `json:"limit"`
	}
	if err := readBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}

	// Embed the query.
	queryVec, err := s.embedSvc.Embed(req.Query)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "embedding failed: "+err.Error())
		return
	}

	// Brute-force cosine similarity against all stored embeddings.
	knowledges, embBytes, err := s.db.AllKnowledgeWithEmbeddings()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	type scored struct {
		knowledge store.Knowledge
		score     float64
	}
	var results []scored
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
		results = append(results, scored{knowledge: k, score: sim})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})
	if len(results) > req.Limit {
		results = results[:req.Limit]
	}

	var out []store.SemanticResult
	for _, r := range results {
		out = append(out, store.SemanticResult{
			Knowledge: r.knowledge,
			Score:     r.score,
		})
	}
	if out == nil {
		out = []store.SemanticResult{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": out})
}

func (s *Server) handleSimilarPatterns(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Mode != "sqlite-ollama" || s.embedSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "similar patterns requires sqlite-ollama mode")
		return
	}

	var req struct {
		KnowledgeID      string `json:"knowledge_id"`
		Limit            int    `json:"limit"`
		CrossProjectOnly bool   `json:"cross_project_only"`
	}
	if err := readBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}

	// Get the source knowledge embedding.
	sourceK, _, err := s.db.GetKnowledge(req.KnowledgeID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	// If source has no embedding, generate one.
	knowledges, embBytes, err := s.db.AllKnowledgeWithEmbeddings()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Find source embedding.
	dims := s.embedSvc.Dimensions()
	var sourceVec []float32
	for i, k := range knowledges {
		if k.KnowledgeID == req.KnowledgeID {
			sourceVec = embedding.DecodeEmbedding(embBytes[i], dims)
			break
		}
	}
	if sourceVec == nil {
		// Generate embedding for source on-the-fly.
		text := sourceK.ContentNL
		if text == "" {
			text = sourceK.ContentRaw
		}
		sourceVec, err = s.embedSvc.Embed(text)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "embedding failed: "+err.Error())
			return
		}
		// Store it for future use.
		s.db.StoreEmbedding(req.KnowledgeID, embedding.EncodeEmbedding(sourceVec), s.embedSvc.Model(), dims)
	}

	type scored struct {
		knowledge store.Knowledge
		score     float64
	}
	var results []scored
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
		results = append(results, scored{knowledge: k, score: sim})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})
	if len(results) > req.Limit {
		results = results[:req.Limit]
	}

	var out []store.SemanticResult
	for _, r := range results {
		out = append(out, store.SemanticResult{
			Knowledge: r.knowledge,
			Score:     r.score,
		})
	}
	if out == nil {
		out = []store.SemanticResult{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": out})
}

// embedKnowledge generates and stores an embedding for a knowledge document (async).
func (s *Server) embedKnowledge(knowledgeID, contentNL string) {
	if contentNL == "" {
		return
	}
	vec, err := s.embedSvc.Embed(contentNL)
	if err != nil {
		log.Printf("embed knowledge %s: %v (non-blocking)", knowledgeID, err)
		return
	}
	encoded := embedding.EncodeEmbedding(vec)
	if err := s.db.StoreEmbedding(knowledgeID, encoded, s.embedSvc.Model(), s.embedSvc.Dimensions()); err != nil {
		log.Printf("store embedding %s: %v", knowledgeID, err)
	}
}

func (s *Server) handleListModels(w http.ResponseWriter, r *http.Request) {
	models := []store.EmbeddingModel{}
	if s.cfg.Mode == "sqlite-ollama" && s.embedSvc != nil {
		models = append(models, store.EmbeddingModel{
			ModelName:  s.embedSvc.Model(),
			Provider:   "ollama",
			Dimensions: s.embedSvc.Dimensions(),
			Status:     "active",
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"models": models})
}

// --- Edge handlers ---

func (s *Server) handleEdgeCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SourceID     string   `json:"source_id"`
		TargetID     string   `json:"target_id"`
		ProjectSlug  string   `json:"project_slug"`
		Tags         []string `json:"tags"`
		MetadataJSON string   `json:"metadata_json"`
	}
	if err := readBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	e := &store.Edge{
		SourceID: req.SourceID, TargetID: req.TargetID,
		ProjectSlug: req.ProjectSlug, Tags: req.Tags, MetadataJSON: req.MetadataJSON,
	}
	if err := s.db.CreateEdge(e); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"edge": e})
}

func (s *Server) handleEdgeSearch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SourceID    string   `json:"source_id"`
		TargetID    string   `json:"target_id"`
		ProjectSlug string   `json:"project_slug"`
		Tags        []string `json:"tags"`
		Pagination  struct {
			PageSize int `json:"page_size"`
		} `json:"pagination"`
	}
	if err := readBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	edges, err := s.db.SearchEdges(req.SourceID, req.TargetID, req.ProjectSlug, req.Tags, req.Pagination.PageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if edges == nil {
		edges = []store.Edge{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": edges})
}

func (s *Server) handleEdgeDelete(w http.ResponseWriter, r *http.Request) {
	var req struct {
		EdgeID string `json:"edge_id"`
	}
	if err := readBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.db.DeleteEdge(req.EdgeID); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
}
