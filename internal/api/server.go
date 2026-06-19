package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gem-squared/gem2-lfs/internal/embedding"
	"github.com/gem-squared/gem2-lfs/internal/store"
)

// Config holds server configuration.
type Config struct {
	Port int
	Mode string // "sqlite-only" or "sqlite-ollama"
}

// Server is the gem2-lfs HTTP API server.
type Server struct {
	db       *store.DB
	embedSvc *embedding.OllamaService
	cfg      Config
	mux      *http.ServeMux
}

// NewServer creates a new API server.
func NewServer(db *store.DB, embedSvc *embedding.OllamaService, cfg Config) *Server {
	s := &Server{
		db:       db,
		embedSvc: embedSvc,
		cfg:      cfg,
		mux:      http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
	addr := fmt.Sprintf(":%d", s.cfg.Port)
	return http.ListenAndServe(addr, s.mux)
}

func (s *Server) registerRoutes() {
	// Health + Capabilities.
	s.mux.HandleFunc("POST /api/v1/health", s.handleHealth)
	s.mux.HandleFunc("GET /capabilities", s.handleCapabilities)

	// Tasks (4).
	s.mux.HandleFunc("POST /api/v1/tasks", s.handleTaskCreate)
	s.mux.HandleFunc("POST /api/v1/tasks/get", s.handleTaskGet)
	s.mux.HandleFunc("POST /api/v1/tasks/search", s.handleTaskSearch)
	s.mux.HandleFunc("POST /api/v1/tasks/update", s.handleTaskUpdate)

	// Messages (3).
	s.mux.HandleFunc("POST /api/v1/messages", s.handleMsgCreate)
	s.mux.HandleFunc("POST /api/v1/messages/get", s.handleMsgGet)
	s.mux.HandleFunc("POST /api/v1/messages/search", s.handleMsgSearch)

	// Decisions (3).
	s.mux.HandleFunc("POST /api/v1/decisions", s.handleDecisionCreate)
	s.mux.HandleFunc("POST /api/v1/decisions/search", s.handleDecisionSearch)
	s.mux.HandleFunc("POST /api/v1/decisions/update", s.handleDecisionUpdate)

	// Knowledge (7 + 3 edges = 10).
	s.mux.HandleFunc("POST /api/v1/knowledge/create", s.handleKnowledgeCreate)
	s.mux.HandleFunc("POST /api/v1/knowledge/get", s.handleKnowledgeGet)
	s.mux.HandleFunc("POST /api/v1/knowledge/search", s.handleKnowledgeSearch)
	s.mux.HandleFunc("POST /v1/knowledge/upsert", s.handleKnowledgeUpsert)
	s.mux.HandleFunc("POST /api/v1/knowledge/semantic-search", s.handleSemanticSearch)
	s.mux.HandleFunc("POST /api/v1/knowledge/similar-patterns", s.handleSimilarPatterns)
	s.mux.HandleFunc("POST /api/v1/knowledge/models", s.handleListModels)
	s.mux.HandleFunc("POST /api/v1/knowledge/edges", s.handleEdgeCreate)
	s.mux.HandleFunc("POST /api/v1/knowledge/edges/search", s.handleEdgeSearch)
	s.mux.HandleFunc("POST /api/v1/knowledge/edges/delete", s.handleEdgeDelete)

	// Skills (4).
	s.mux.HandleFunc("POST /v1/skills/upsert", s.handleSkillUpsert)
	s.mux.HandleFunc("POST /v1/skills/get", s.handleSkillGet)
	s.mux.HandleFunc("POST /v1/skills/retrieve", s.handleSkillSearch)
	s.mux.HandleFunc("POST /v1/skills/delete", s.handleSkillDelete)

	// Session + Status (2).
	s.mux.HandleFunc("POST /api/v1/session/context", s.handleSessionContext)
	s.mux.HandleFunc("POST /api/v1/status", s.handleStatus)

	// Projects (3).
	s.mux.HandleFunc("POST /api/v1/projects", s.handleProjectCreate)
	s.mux.HandleFunc("POST /api/v1/projects/get", s.handleProjectGet)
	s.mux.HandleFunc("POST /api/v1/projects/list", s.handleProjectList)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"version": "0.1.0",
		"mode":    s.cfg.Mode,
	})
}

func (s *Server) handleCapabilities(w http.ResponseWriter, r *http.Request) {
	semanticSearch := s.cfg.Mode == "sqlite-ollama" && s.embedSvc != nil
	writeJSON(w, http.StatusOK, map[string]any{
		"mode": s.cfg.Mode,
		"features": map[string]bool{
			"semantic_search":  semanticSearch,
			"similar_patterns": semanticSearch,
			"fts5_search":      true,
			"task_crud":        true,
			"message_crud":     true,
			"decision_crud":    true,
			"knowledge_crud":   true,
			"skill_crud":       true,
			"project_crud":     true,
			"edge_crud":        true,
		},
		"version": "0.1.0",
	})
}

// --- Helpers ---

func readBody(r *http.Request, v any) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, v)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write json: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
