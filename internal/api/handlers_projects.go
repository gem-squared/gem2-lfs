package api

import (
	"net/http"

	"github.com/gem-squared/gem2-lfs/internal/store"
)

func (s *Server) handleProjectCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectSlug  string `json:"project_slug"`
		Name         string `json:"name"`
		RootPath     string `json:"root_path"`
		MetadataJSON string `json:"metadata_json"`
	}
	if err := readBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	p := &store.Project{
		ProjectSlug: req.ProjectSlug, Name: req.Name,
		RootPath: req.RootPath, MetadataJSON: req.MetadataJSON,
	}
	if err := s.db.CreateProject(p); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"project": p})
}

func (s *Server) handleProjectGet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectSlug string `json:"project_slug"`
	}
	if err := readBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	p, err := s.db.GetProject(req.ProjectSlug)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"project": p})
}

func (s *Server) handleProjectList(w http.ResponseWriter, r *http.Request) {
	projects, err := s.db.ListProjects()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if projects == nil {
		projects = []store.Project{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": projects})
}
