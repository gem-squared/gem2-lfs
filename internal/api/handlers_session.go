package api

import (
	"net/http"

	"github.com/gem-squared/gem2-lfs/internal/store"
)

func (s *Server) handleSessionContext(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Role          string `json:"role"`
		ProjectSlug   string `json:"project_slug"`
		MsgLimit      int    `json:"msg_limit"`
		TaskLimit     int    `json:"task_limit"`
		DecisionLimit int    `json:"decision_limit"`
	}
	if err := readBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
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

	// Fetch IN_PROGRESS tasks.
	inProgress, err := s.db.SearchTasks(req.Role, "IN_PROGRESS", req.ProjectSlug, "", "", nil, req.TaskLimit)
	if err != nil {
		inProgress = []store.Task{}
	}

	// Fetch PENDING tasks.
	pending, err := s.db.SearchTasks(req.Role, "PENDING", req.ProjectSlug, "", "", nil, req.TaskLimit)
	if err != nil {
		pending = []store.Task{}
	}

	tasks := append(inProgress, pending...)

	// Fetch recent messages.
	messages, err := s.db.SearchMessages("", "", req.ProjectSlug, "", "", req.MsgLimit)
	if err != nil {
		messages = []store.Message{}
	}

	// Fetch active decisions.
	decisions, err := s.db.SearchDecisions("", "", "ACTIVE", req.ProjectSlug, "", nil, req.DecisionLimit)
	if err != nil {
		decisions = []store.Decision{}
	}

	if tasks == nil {
		tasks = []store.Task{}
	}
	if messages == nil {
		messages = []store.Message{}
	}
	if decisions == nil {
		decisions = []store.Decision{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"messages":  messages,
		"tasks":     tasks,
		"decisions": decisions,
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectSlug string `json:"project_slug"`
		Role        string `json:"role"`
	}
	if err := readBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Count tasks by status.
	counters := map[string]int{}
	for _, status := range []string{"PENDING", "IN_PROGRESS", "COMPLETED", "ABORTED", "BLOCKED"} {
		tasks, err := s.db.SearchTasks(req.Role, status, req.ProjectSlug, "", "", nil, 0)
		if err != nil {
			continue
		}
		counters[status] = len(tasks)
	}

	// Get active tasks (IN_PROGRESS).
	active, _ := s.db.SearchTasks(req.Role, "IN_PROGRESS", req.ProjectSlug, "", "", nil, 20)
	if active == nil {
		active = []store.Task{}
	}

	// Get blockers (BLOCKED).
	blocked, _ := s.db.SearchTasks(req.Role, "BLOCKED", req.ProjectSlug, "", "", nil, 20)
	if blocked == nil {
		blocked = []store.Task{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"project_slug":  req.ProjectSlug,
		"task_counters": counters,
		"active_tasks":  active,
		"blockers":      blocked,
	})
}
