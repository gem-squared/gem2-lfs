package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/gem-squared/gem2-lfs/internal/api"
	"github.com/gem-squared/gem2-lfs/internal/embedding"
	mcpserver "github.com/gem-squared/gem2-lfs/internal/mcp"
	"github.com/gem-squared/gem2-lfs/internal/store"
)

var version = "0.1.0-dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "serve":
		cmdServe(os.Args[2:])
	case "mcp":
		cmdMCP(os.Args[2:])
	case "init":
		cmdInit(os.Args[2:])
	case "doctor":
		cmdDoctor(os.Args[2:])
	case "version":
		fmt.Printf("gem2-lfs %s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `gem2-lfs — Lightweight Local Knowledge Store (gem2-kg-light)

Usage:
  gem2-lfs serve   [flags]   Start HTTP API server
  gem2-lfs mcp     [flags]   Start MCP stdio server (L0 standalone)
  gem2-lfs init    [flags]   Initialize database and check dependencies
  gem2-lfs doctor  [flags]   Health check: SQLite, Ollama status
  gem2-lfs version           Print version
  gem2-lfs help              Show this help

`)
}

func cmdServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	port := fs.Int("port", 9090, "HTTP server port")
	dbPath := fs.String("db-path", ".gem2-lfs/data.db", "SQLite database file path")
	mode := fs.String("mode", "sqlite-only", "Operating mode: sqlite-only or sqlite-ollama")
	ollamaURL := fs.String("ollama-url", "http://localhost:11434", "Ollama API URL (used in sqlite-ollama mode)")
	fs.Parse(args)

	// Validate mode.
	if *mode != "sqlite-only" && *mode != "sqlite-ollama" {
		log.Fatalf("invalid mode %q: must be sqlite-only or sqlite-ollama", *mode)
	}

	// Open store.
	db, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	// Set up embedding service if in ollama mode.
	var embedSvc *embedding.OllamaService
	if *mode == "sqlite-ollama" {
		embedSvc = embedding.NewOllamaService(*ollamaURL)
		if err := embedSvc.HealthCheck(); err != nil {
			log.Printf("WARNING: Ollama not reachable at %s: %v (falling back to sqlite-only for semantic search)", *ollamaURL, err)
			embedSvc = nil
		} else {
			log.Printf("Ollama connected at %s", *ollamaURL)
		}
	}

	// Start server.
	srv := api.NewServer(db, embedSvc, api.Config{
		Port: *port,
		Mode: *mode,
	})
	log.Printf("gem2-lfs %s starting on :%d (mode=%s, db=%s)", version, *port, *mode, *dbPath)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func cmdMCP(args []string) {
	fs := flag.NewFlagSet("mcp", flag.ExitOnError)
	dbPath := fs.String("db-path", ".gem2-lfs/data.db", "SQLite database file path")
	mode := fs.String("mode", "sqlite-only", "Operating mode: sqlite-only or sqlite-ollama")
	ollamaURL := fs.String("ollama-url", "http://localhost:11434", "Ollama API URL")
	fs.Parse(args)

	if *mode != "sqlite-only" && *mode != "sqlite-ollama" {
		log.Fatalf("invalid mode %q: must be sqlite-only or sqlite-ollama", *mode)
	}

	db, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	var embedSvc *embedding.OllamaService
	if *mode == "sqlite-ollama" {
		embedSvc = embedding.NewOllamaService(*ollamaURL)
		if err := embedSvc.HealthCheck(); err != nil {
			log.Printf("WARNING: Ollama not reachable at %s: %v (semantic search unavailable)", *ollamaURL, err)
			embedSvc = nil
		}
	}

	srv := mcpserver.NewServer(db, embedSvc, *mode, os.Stdin, os.Stdout)
	srv.RegisterAllTools()
	log.Printf("gem2-lfs %s MCP server started (mode=%s, db=%s)", version, *mode, *dbPath)
	if err := srv.Run(context.Background()); err != nil {
		log.Fatalf("mcp server: %v", err)
	}
}

func cmdInit(args []string) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	dbPath := fs.String("db-path", ".gem2-lfs/data.db", "SQLite database file path")
	mode := fs.String("mode", "sqlite-only", "Operating mode: sqlite-only or sqlite-ollama")
	ollamaURL := fs.String("ollama-url", "http://localhost:11434", "Ollama API URL")
	seed := fs.Bool("seed", false, "Load seed data (12 L1 core TPMN skills)")
	fs.Parse(args)

	// Create directory for DB.
	if dir := dirOf(*dbPath); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("create directory: %v", err)
		}
	}

	// Open and initialize schema.
	db, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("init database: %v", err)
	}

	// Load seed data if requested.
	if *seed {
		if err := db.SeedData(); err != nil {
			log.Fatalf("seed data: %v", err)
		}
		fmt.Println("Seed data loaded (12 L1 core TPMN skills).")
	}

	db.Close()
	fmt.Printf("Database initialized at %s\n", *dbPath)

	// Check Ollama if in ollama mode.
	if *mode == "sqlite-ollama" {
		svc := embedding.NewOllamaService(*ollamaURL)
		if err := svc.HealthCheck(); err != nil {
			fmt.Printf("WARNING: Ollama not reachable at %s: %v\n", *ollamaURL, err)
			fmt.Println("Install Ollama and pull the model: ollama pull nomic-embed-text:v1.5")
		} else {
			fmt.Printf("Ollama OK at %s\n", *ollamaURL)
		}
	}

	fmt.Println("gem2-lfs init complete.")
}

func cmdDoctor(args []string) {
	fs := flag.NewFlagSet("doctor", flag.ExitOnError)
	dbPath := fs.String("db-path", ".gem2-lfs/data.db", "SQLite database file path")
	ollamaURL := fs.String("ollama-url", "http://localhost:11434", "Ollama API URL")
	fs.Parse(args)

	fmt.Println("gem2-lfs doctor")
	fmt.Println("───────────────")

	// Check SQLite.
	if _, err := os.Stat(*dbPath); err == nil {
		db, err := store.Open(*dbPath)
		if err != nil {
			fmt.Printf("SQLite: FAIL (%v)\n", err)
		} else {
			db.Close()
			fmt.Printf("SQLite: OK (%s)\n", *dbPath)
		}
	} else {
		fmt.Printf("SQLite: NOT INITIALIZED (run gem2-lfs init)\n")
	}

	// Check Ollama.
	svc := embedding.NewOllamaService(*ollamaURL)
	if err := svc.HealthCheck(); err != nil {
		fmt.Printf("Ollama: NOT AVAILABLE (%v)\n", err)
	} else {
		fmt.Printf("Ollama: OK (%s)\n", *ollamaURL)
	}
}

func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return ""
}
