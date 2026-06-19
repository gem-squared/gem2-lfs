package api

import (
	"net/http"

	"github.com/gem-squared/gem2-lfs/internal/store"
)

func (s *Server) handleDecisionCreate(w http.ResponseWriter, r *http.Request) {
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
	if err := readBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	d := &store.Decision{
		Role: req.Role, Category: req.Category, Title: req.Title, Content: req.Content,
		Status: req.Status, ProjectSlug: req.ProjectSlug, Tags: req.Tags, MetadataJSON: req.MetadataJSON,
	}
	if err := s.db.CreateDecision(d); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"decision": d})
}

func (s *Server) handleDecisionSearch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Role        string   `json:"role"`
		Category    string   `json:"category"`
		Status      string   `json:"status"`
		ProjectSlug string   `json:"project_slug"`
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
	decs, err := s.db.SearchDecisions(req.Role, req.Category, req.Status, req.ProjectSlug, req.Query, req.Tags, req.Pagination.PageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if decs == nil {
		decs = []store.Decision{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": decs})
}

func (s *Server) handleDecisionUpdate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DecisionID   string   `json:"decision_id"`
		Status       string   `json:"status"`
		SupersededBy string   `json:"superseded_by"`
		Tags         []string `json:"tags"`
		Content      string   `json:"content"`
	}
	if err := readBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	d, err := s.db.UpdateDecision(req.DecisionID, req.Status, req.SupersededBy, req.Tags, req.Content)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"decision": d})
}
