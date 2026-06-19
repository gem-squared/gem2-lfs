# gem2-lfs

Lightweight Local Knowledge Store — a SQLite-based drop-in replacement for gem2-kg.

## What It Does

gem2-lfs implements the same 28 HTTP API endpoints as gem2-kg, backed by SQLite + FTS5 instead of PostgreSQL + pgvector. It serves as:

- **L0 secondary store** — enriches the filesystem-based TPMN workflow with persistent cross-session data (gem2-studio-p)
- **L1/L2 fallback** — local safety net when gem2-kg (Docker or cloud) is unavailable (gem2-studio-x)

## Architecture

```
Claude Code
  │
  ├── L0 (standalone): → gem2-lfs mcp (stdio, direct)
  └── L1 (orchestrated): → gem2-studio (MCP proxy) → gem2-lfs serve (HTTP :9090)

gem2-studio (MCP server)
  │
  ├── L0: → gem2-lfs (SQLite, localhost:9090)
  ├── L1: → gem2-kg  (Docker, PostgreSQL) → gem2-lfs (fallback)
  └── L2: → gem2-kg  (cloud)             → gem2-lfs (fallback)
```

## Two Modes

| Mode | Storage | Search | Embeddings |
|------|---------|--------|------------|
| `sqlite-only` | SQLite + FTS5 | Full-text search | None |
| `sqlite-ollama` | SQLite + FTS5 + BLOB embeddings | Full-text + semantic search | Ollama (CPU, nomic-embed-text:latest, 768-dim) |

## Quick Start

### npm (recommended)

```bash
npm install -g @gem_squared/gem2-lfs
gem2-lfs setup    # interactive wizard: DB init, Ollama check, MCP registration
gem2-lfs serve    # start HTTP API server
```

### Local Binary

```bash
go build -o gem2-lfs ./cmd/gem2-lfs/
./gem2-lfs init --mode sqlite-only
./gem2-lfs serve --port 9090
```

### Docker

```bash
docker compose up -d
```

### Register with gem2-studio (L1)

```bash
GEM2_KG_URL=http://localhost:9090 gem2-studio serve
```

## MCP Server (L0 Standalone)

gem2-lfs includes a native MCP server over stdio (JSON-RPC 2.0). Claude Code can call gem2-lfs directly — no gem2-studio required.

### Setup

Add to your project's `.mcp.json`:

```json
{
  "mcpServers": {
    "gem2-lfs": {
      "command": "gem2-lfs",
      "args": ["mcp", "--db-path", ".gem2-lfs/data.db"]
    }
  }
}
```

Or with Ollama semantic search:

```json
{
  "mcpServers": {
    "gem2-lfs": {
      "command": "gem2-lfs",
      "args": ["mcp", "--db-path", ".gem2-lfs/data.db", "--mode", "sqlite-ollama"]
    }
  }
}
```

Or run `gem2-lfs setup` — it offers to register automatically.

### MCP Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--db-path` | `.gem2-lfs/data.db` | SQLite database file path |
| `--mode` | `sqlite-only` | `sqlite-only` or `sqlite-ollama` |
| `--ollama-url` | `http://localhost:11434` | Ollama API URL (sqlite-ollama mode) |

### MCP Tools (29)

All tools use the `gem2_` prefix. Same data surface as the HTTP API.

#### Tasks (4)

| Tool | Description |
|------|-------------|
| `gem2_task_create` | Create a task. Required: `title`, `project_slug`. Optional: `role`, `description`, `priority`, `plan_doc`, `parent_task_id`, `tags`, `metadata_json`. |
| `gem2_task_get` | Get a task by ID. Required: `task_id`. |
| `gem2_task_search` | Search tasks. Optional: `role`, `status`, `project_slug`, `priority`, `query`, `tags`, `page_size`. |
| `gem2_task_update` | Update a task. Required: `task_id`. Optional: `status`, `result_summary`, `state`, `tags`, `metadata_json`. |

#### Messages (3)

| Tool | Description |
|------|-------------|
| `gem2_msg_create` | Create a protocol message. Required: `from_role`, `to_role`, `message`. Optional: `content`, `project_slug`, `session_id`, `terminal_id`, `terminal_title`, `metadata_json`. |
| `gem2_msg_get` | Get a message by ID. Required: `msg_id`. |
| `gem2_msg_search` | Search messages. Optional: `from_role`, `to_role`, `project_slug`, `session_id`, `query`, `page_size`. |

#### Decisions (3)

| Tool | Description |
|------|-------------|
| `gem2_decision_create` | Create a decision record. Required: `title`, `content`. Optional: `role`, `category`, `status`, `project_slug`, `tags`, `metadata_json`. |
| `gem2_decision_search` | Search decisions. Optional: `role`, `category`, `status`, `project_slug`, `query`, `tags`, `page_size`. |
| `gem2_decision_update` | Update a decision. Required: `decision_id`. Optional: `status`, `superseded_by`, `tags`, `content`. |

#### Knowledge (5)

| Tool | Description |
|------|-------------|
| `gem2_knowledge_create` | Create a knowledge document. Required: `title`, `content`. Optional: `entity_type`, `content_nl`, `project_slug`, `tags`, `file_path`, `line_count`, `metadata_json`. |
| `gem2_knowledge_get` | Get a knowledge document + chunks. Required: `knowledge_id`. |
| `gem2_knowledge_search` | Search knowledge. Optional: `project_slug`, `entity_type`, `query`, `tags`, `page_size`. |
| `gem2_knowledge_upsert` | Create or update (deduplicates by title+type+project). Required: `title`, `content`. Optional: same as create + `user_id`. |
| `gem2_semantic_search` | Semantic similarity search (sqlite-ollama only). Required: `query`. Optional: `project_slug`, `model`, `limit`. |

#### Knowledge Extras (2)

| Tool | Description |
|------|-------------|
| `gem2_similar_patterns` | Find similar documents (sqlite-ollama only). Required: `knowledge_id`. Optional: `limit`, `cross_project_only`. |
| `gem2_list_models` | List available embedding models. No arguments. |

#### Edges (3)

| Tool | Description |
|------|-------------|
| `gem2_edge_create` | Create a provenance edge. Required: `source_id`, `target_id`. Optional: `project_slug`, `tags`, `metadata_json`. |
| `gem2_edge_search` | Search edges. Optional: `source_id`, `target_id`, `project_slug`, `tags`, `page_size`. |
| `gem2_edge_delete` | Delete an edge. Required: `edge_id`. |

#### Skills (4)

| Tool | Description |
|------|-------------|
| `gem2_skill_upsert` | Create or update a skill. Required: `title`, `content`. Optional: `skill_id`, `user_id`, `skill_type`, `tags`, `source_tool`, `metadata`. |
| `gem2_skill_get` | Get a skill. Required: `skill_id`. Optional: `user_id`. |
| `gem2_skill_search` | Search skills. Optional: `user_id`, `query`, `skill_type`, `tags`, `page_size`. |
| `gem2_skill_delete` | Delete a skill. Required: `skill_id`. Optional: `user_id`. |

#### Session + Status (2)

| Tool | Description |
|------|-------------|
| `gem2_session_context` | Get session context (active/pending tasks, recent messages, active decisions). Required: `project_slug`. Optional: `role`, `msg_limit`, `task_limit`, `decision_limit`. |
| `gem2_status` | Get project status (task counters, active tasks, blockers). Required: `project_slug`. Optional: `role`. |

#### Projects (3)

| Tool | Description |
|------|-------------|
| `gem2_project_create` | Register a project. Required: `project_slug`, `name`. Optional: `root_path`, `metadata_json`. |
| `gem2_project_get` | Get a project. Required: `project_slug`. |
| `gem2_project_list` | List all projects. No arguments. |

## API

Same endpoints as gem2-kg. See `/capabilities` for feature availability:

```bash
curl http://localhost:9090/capabilities
```

## Endpoints (28 + 1)

| Entity | Endpoints | Count |
|--------|-----------|-------|
| Tasks | create, get, search, update | 4 |
| Messages | create, get, search | 3 |
| Decisions | create, search, update | 3 |
| Knowledge | create, get, search, upsert, semantic-search, similar-patterns, models | 7 |
| Edges | create, search, delete | 3 |
| Skills | upsert, get, search, delete | 4 |
| Projects | create, get, list | 3 |
| Session | context | 1 |
| Status | status | 1 |
| Health | health | 1 |
| **Capabilities** | **/capabilities (GET)** | **1** |

## CLI Commands

| Command | Description |
|---------|-------------|
| `gem2-lfs setup` | Interactive first-run wizard (Node.js wrapper) |
| `gem2-lfs serve` | Start HTTP API server (default :9090) |
| `gem2-lfs mcp` | Start MCP stdio server (L0 standalone) |
| `gem2-lfs init` | Initialize database |
| `gem2-lfs doctor` | Health check (SQLite + Ollama) |
| `gem2-lfs version` | Print version |

## Requirements

- Go 1.25+ (build from source) or Node.js 16+ (npm install)
- Ollama (optional, for sqlite-ollama mode): `ollama pull nomic-embed-text:latest`

## License

MIT

---

*gem2-lfs v0.1.0 | GEM2.AI*
