package api

import (
	"net/http"

	"github.com/gem-squared/gem2-lfs/internal/store"
)

func (s *Server) handleSkillUpsert(w http.ResponseWriter, r *http.Request) {
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
	if err := readBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	sk := &store.Skill{
		SkillID: req.SkillID, UserID: req.UserID, SkillType: req.SkillType,
		Title: req.Title, Content: req.Content, Tags: req.Tags,
		SourceTool: req.SourceTool, Metadata: req.Metadata,
	}
	if err := s.db.UpsertSkill(sk); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"skill": sk})
}

func (s *Server) handleSkillGet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SkillID string `json:"skill_id"`
		UserID  string `json:"user_id"`
	}
	if err := readBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	sk, err := s.db.GetSkill(req.SkillID, req.UserID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"skill": sk})
}

func (s *Server) handleSkillSearch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID    string   `json:"user_id"`
		Query     string   `json:"query"`
		SkillType string   `json:"skill_type"`
		Tags      []string `json:"tags"`
		Pagination struct {
			PageSize int `json:"page_size"`
		} `json:"pagination"`
	}
	if err := readBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	skills, err := s.db.SearchSkills(req.UserID, req.Query, req.SkillType, req.Tags, req.Pagination.PageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if skills == nil {
		skills = []store.Skill{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": skills})
}

func (s *Server) handleSkillDelete(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SkillID string `json:"skill_id"`
		UserID  string `json:"user_id"`
	}
	if err := readBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.db.DeleteSkill(req.SkillID, req.UserID); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
}
