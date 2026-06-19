package api

import (
	"net/http"

	"github.com/gem-squared/gem2-lfs/internal/store"
)

func (s *Server) handleMsgCreate(w http.ResponseWriter, r *http.Request) {
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
	if err := readBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	m := &store.Message{
		FromRole: req.FromRole, ToRole: req.ToRole, MessageText: req.Message,
		Content: req.Content, ProjectSlug: req.ProjectSlug, SessionID: req.SessionID,
		TerminalID: req.TerminalID, TerminalTitle: req.TerminalTitle, MetadataJSON: req.MetadataJSON,
	}
	if err := s.db.CreateMessage(m); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": m})
}

func (s *Server) handleMsgGet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MsgID string `json:"msg_id"`
	}
	if err := readBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	m, err := s.db.GetMessage(req.MsgID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": m})
}

func (s *Server) handleMsgSearch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FromRole    string `json:"from_role"`
		ToRole      string `json:"to_role"`
		ProjectSlug string `json:"project_slug"`
		SessionID   string `json:"session_id"`
		Query       string `json:"query"`
		Pagination  struct {
			PageSize int `json:"page_size"`
		} `json:"pagination"`
	}
	if err := readBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	msgs, err := s.db.SearchMessages(req.FromRole, req.ToRole, req.ProjectSlug, req.SessionID, req.Query, req.Pagination.PageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if msgs == nil {
		msgs = []store.Message{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": msgs})
}
