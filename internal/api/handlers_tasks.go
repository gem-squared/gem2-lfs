package api

import (
	"net/http"

	"github.com/gem-squared/gem2-lfs/internal/store"
)

func (s *Server) handleTaskCreate(w http.ResponseWriter, r *http.Request) {
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
	if err := readBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	t := &store.Task{
		Role: req.Role, Title: req.Title, Description: req.Description,
		Priority: req.Priority, ProjectSlug: req.ProjectSlug, PlanDoc: req.PlanDoc,
		ParentTaskID: req.ParentTaskID, Tags: req.Tags, MetadataJSON: req.MetadataJSON,
	}
	if err := s.db.CreateTask(t); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"task": t})
}

func (s *Server) handleTaskGet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TaskID string `json:"task_id"`
	}
	if err := readBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	t, err := s.db.GetTask(req.TaskID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"task": t})
}

func (s *Server) handleTaskSearch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Role        string   `json:"role"`
		Status      string   `json:"status"`
		ProjectSlug string   `json:"project_slug"`
		Priority    string   `json:"priority"`
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
	limit := req.Pagination.PageSize
	tasks, err := s.db.SearchTasks(req.Role, req.Status, req.ProjectSlug, req.Priority, req.Query, req.Tags, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if tasks == nil {
		tasks = []store.Task{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": tasks})
}

func (s *Server) handleTaskUpdate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TaskID        string   `json:"task_id"`
		Status        string   `json:"status"`
		ResultSummary string   `json:"result_summary"`
		State         string   `json:"state"`
		Tags          []string `json:"tags"`
		MetadataJSON  string   `json:"metadata_json"`
	}
	if err := readBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	t, err := s.db.UpdateTask(req.TaskID, req.Status, req.ResultSummary, req.State, req.Tags, req.MetadataJSON)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"task": t})
}
