package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// DB wraps a SQLite database connection with gem2-lfs schema.
type DB struct {
	db *sql.DB
}

// Open opens (or creates) a SQLite database at the given path
// and initializes the schema.
func Open(dbPath string) (*DB, error) {
	// Ensure parent directory exists.
	dir := filepath.Dir(dbPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create db directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(wal)&_pragma=synchronous(normal)&_pragma=foreign_keys(on)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// Verify connection.
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	s := &DB{db: db}
	if err := s.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}

	return s, nil
}

// Close closes the database connection.
func (s *DB) Close() error {
	return s.db.Close()
}

// SqlDB returns the underlying *sql.DB for direct access.
func (s *DB) SqlDB() *sql.DB {
	return s.db
}

func (s *DB) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS _meta (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);
	INSERT OR IGNORE INTO _meta (key, value) VALUES ('version', '0.1.0');

	-- Tasks
	CREATE TABLE IF NOT EXISTS tasks (
		task_id        TEXT PRIMARY KEY,
		role           TEXT NOT NULL DEFAULT '',
		title          TEXT NOT NULL DEFAULT '',
		description    TEXT NOT NULL DEFAULT '',
		status         TEXT NOT NULL DEFAULT 'PENDING',
		priority       TEXT NOT NULL DEFAULT 'MEDIUM',
		project_slug   TEXT NOT NULL DEFAULT '',
		plan_doc       TEXT NOT NULL DEFAULT '',
		result_summary TEXT NOT NULL DEFAULT '',
		parent_task_id TEXT NOT NULL DEFAULT '',
		tags           TEXT NOT NULL DEFAULT '[]',
		metadata_json  TEXT NOT NULL DEFAULT '{}',
		state          TEXT NOT NULL DEFAULT '',
		started_at     TEXT,
		completed_at   TEXT,
		created_at     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
		updated_at     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
	);
	CREATE INDEX IF NOT EXISTS idx_tasks_project ON tasks(project_slug);
	CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
	CREATE INDEX IF NOT EXISTS idx_tasks_role ON tasks(role);

	-- Messages
	CREATE TABLE IF NOT EXISTS messages (
		msg_id         TEXT PRIMARY KEY,
		msg_type       TEXT NOT NULL DEFAULT '',
		from_role      TEXT NOT NULL DEFAULT '',
		to_role        TEXT NOT NULL DEFAULT '',
		message        TEXT NOT NULL DEFAULT '',
		content        TEXT NOT NULL DEFAULT '',
		project_slug   TEXT NOT NULL DEFAULT '',
		g_service_id   TEXT NOT NULL DEFAULT '',
		session_id     TEXT NOT NULL DEFAULT '',
		terminal_id    TEXT NOT NULL DEFAULT '',
		terminal_title TEXT NOT NULL DEFAULT '',
		metadata_json  TEXT NOT NULL DEFAULT '{}',
		created_at     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
		updated_at     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
	);
	CREATE INDEX IF NOT EXISTS idx_messages_project ON messages(project_slug);
	CREATE INDEX IF NOT EXISTS idx_messages_from ON messages(from_role);
	CREATE INDEX IF NOT EXISTS idx_messages_to ON messages(to_role);

	-- Decisions
	CREATE TABLE IF NOT EXISTS decisions (
		decision_id   TEXT PRIMARY KEY,
		role          TEXT NOT NULL DEFAULT '',
		category      TEXT NOT NULL DEFAULT '',
		title         TEXT NOT NULL DEFAULT '',
		content       TEXT NOT NULL DEFAULT '',
		status        TEXT NOT NULL DEFAULT 'ACTIVE',
		superseded_by TEXT NOT NULL DEFAULT '',
		project_slug  TEXT NOT NULL DEFAULT '',
		tags          TEXT NOT NULL DEFAULT '[]',
		metadata_json TEXT NOT NULL DEFAULT '{}',
		decided_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
		created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
		updated_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
	);
	CREATE INDEX IF NOT EXISTS idx_decisions_project ON decisions(project_slug);
	CREATE INDEX IF NOT EXISTS idx_decisions_status ON decisions(status);

	-- Knowledge
	CREATE TABLE IF NOT EXISTS knowledge (
		knowledge_id  TEXT PRIMARY KEY,
		entity_type   TEXT NOT NULL DEFAULT '',
		title         TEXT NOT NULL DEFAULT '',
		content_raw   TEXT NOT NULL DEFAULT '',
		content_nl    TEXT NOT NULL DEFAULT '',
		project_slug  TEXT NOT NULL DEFAULT '',
		tags          TEXT NOT NULL DEFAULT '[]',
		file_path     TEXT NOT NULL DEFAULT '',
		line_count    INTEGER NOT NULL DEFAULT 0,
		metadata_json TEXT NOT NULL DEFAULT '{}',
		embedding     BLOB,
		embedding_model TEXT NOT NULL DEFAULT '',
		embedding_dim   INTEGER NOT NULL DEFAULT 0,
		created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
		updated_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
	);
	CREATE INDEX IF NOT EXISTS idx_knowledge_project ON knowledge(project_slug);
	CREATE INDEX IF NOT EXISTS idx_knowledge_entity ON knowledge(entity_type);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_knowledge_upsert ON knowledge(project_slug, entity_type, title);

	-- Knowledge Chunks
	CREATE TABLE IF NOT EXISTS knowledge_chunks (
		chunk_id      TEXT PRIMARY KEY,
		knowledge_id  TEXT NOT NULL REFERENCES knowledge(knowledge_id) ON DELETE CASCADE,
		chunk_index   INTEGER NOT NULL DEFAULT 0,
		content_nl    TEXT NOT NULL DEFAULT '',
		tags          TEXT NOT NULL DEFAULT '[]',
		heading       TEXT NOT NULL DEFAULT '',
		text          TEXT NOT NULL DEFAULT '',
		embedding     BLOB,
		created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
	);
	CREATE INDEX IF NOT EXISTS idx_chunks_knowledge ON knowledge_chunks(knowledge_id);

	-- Edges (provenance links)
	CREATE TABLE IF NOT EXISTS edges (
		edge_id       TEXT PRIMARY KEY,
		source_id     TEXT NOT NULL,
		target_id     TEXT NOT NULL,
		project_slug  TEXT NOT NULL DEFAULT '',
		tags          TEXT NOT NULL DEFAULT '[]',
		metadata_json TEXT NOT NULL DEFAULT '{}',
		created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
		UNIQUE(source_id, target_id)
	);

	-- Skills
	CREATE TABLE IF NOT EXISTS skills (
		skill_id    TEXT PRIMARY KEY,
		user_id     TEXT NOT NULL DEFAULT '',
		skill_type  TEXT NOT NULL DEFAULT '',
		title       TEXT NOT NULL DEFAULT '',
		content     TEXT NOT NULL DEFAULT '',
		tags        TEXT NOT NULL DEFAULT '[]',
		source_tool TEXT NOT NULL DEFAULT '',
		version     INTEGER NOT NULL DEFAULT 1,
		metadata    TEXT NOT NULL DEFAULT '{}',
		created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
		updated_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
	);
	CREATE INDEX IF NOT EXISTS idx_skills_user ON skills(user_id);

	-- Projects
	CREATE TABLE IF NOT EXISTS projects (
		project_slug TEXT PRIMARY KEY,
		name         TEXT NOT NULL DEFAULT '',
		root_path    TEXT NOT NULL DEFAULT '',
		status       TEXT NOT NULL DEFAULT 'active',
		metadata_json TEXT NOT NULL DEFAULT '{}',
		created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
	);

	-- FTS5 virtual tables
	CREATE VIRTUAL TABLE IF NOT EXISTS tasks_fts USING fts5(
		title, description, content=tasks, content_rowid=rowid,
		tokenize='unicode61'
	);

	CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(
		message, content, content=messages, content_rowid=rowid,
		tokenize='unicode61'
	);

	CREATE VIRTUAL TABLE IF NOT EXISTS decisions_fts USING fts5(
		title, content, content=decisions, content_rowid=rowid,
		tokenize='unicode61'
	);

	CREATE VIRTUAL TABLE IF NOT EXISTS knowledge_fts USING fts5(
		title, content_raw, content_nl, content=knowledge, content_rowid=rowid,
		tokenize='unicode61'
	);

	CREATE VIRTUAL TABLE IF NOT EXISTS skills_fts USING fts5(
		title, content, content=skills, content_rowid=rowid,
		tokenize='unicode61'
	);

	-- FTS triggers for tasks
	CREATE TRIGGER IF NOT EXISTS tasks_ai AFTER INSERT ON tasks BEGIN
		INSERT INTO tasks_fts(rowid, title, description) VALUES (new.rowid, new.title, new.description);
	END;
	CREATE TRIGGER IF NOT EXISTS tasks_ad AFTER DELETE ON tasks BEGIN
		INSERT INTO tasks_fts(tasks_fts, rowid, title, description) VALUES('delete', old.rowid, old.title, old.description);
	END;
	CREATE TRIGGER IF NOT EXISTS tasks_au AFTER UPDATE ON tasks BEGIN
		INSERT INTO tasks_fts(tasks_fts, rowid, title, description) VALUES('delete', old.rowid, old.title, old.description);
		INSERT INTO tasks_fts(rowid, title, description) VALUES (new.rowid, new.title, new.description);
	END;

	-- FTS triggers for messages
	CREATE TRIGGER IF NOT EXISTS messages_ai AFTER INSERT ON messages BEGIN
		INSERT INTO messages_fts(rowid, message, content) VALUES (new.rowid, new.message, new.content);
	END;
	CREATE TRIGGER IF NOT EXISTS messages_ad AFTER DELETE ON messages BEGIN
		INSERT INTO messages_fts(messages_fts, rowid, message, content) VALUES('delete', old.rowid, old.message, old.content);
	END;
	CREATE TRIGGER IF NOT EXISTS messages_au AFTER UPDATE ON messages BEGIN
		INSERT INTO messages_fts(messages_fts, rowid, message, content) VALUES('delete', old.rowid, old.message, old.content);
		INSERT INTO messages_fts(rowid, message, content) VALUES (new.rowid, new.message, new.content);
	END;

	-- FTS triggers for decisions
	CREATE TRIGGER IF NOT EXISTS decisions_ai AFTER INSERT ON decisions BEGIN
		INSERT INTO decisions_fts(rowid, title, content) VALUES (new.rowid, new.title, new.content);
	END;
	CREATE TRIGGER IF NOT EXISTS decisions_ad AFTER DELETE ON decisions BEGIN
		INSERT INTO decisions_fts(decisions_fts, rowid, title, content) VALUES('delete', old.rowid, old.title, old.content);
	END;
	CREATE TRIGGER IF NOT EXISTS decisions_au AFTER UPDATE ON decisions BEGIN
		INSERT INTO decisions_fts(decisions_fts, rowid, title, content) VALUES('delete', old.rowid, old.title, old.content);
		INSERT INTO decisions_fts(rowid, title, content) VALUES (new.rowid, new.title, new.content);
	END;

	-- FTS triggers for knowledge
	CREATE TRIGGER IF NOT EXISTS knowledge_ai AFTER INSERT ON knowledge BEGIN
		INSERT INTO knowledge_fts(rowid, title, content_raw, content_nl) VALUES (new.rowid, new.title, new.content_raw, new.content_nl);
	END;
	CREATE TRIGGER IF NOT EXISTS knowledge_ad AFTER DELETE ON knowledge BEGIN
		INSERT INTO knowledge_fts(knowledge_fts, rowid, title, content_raw, content_nl) VALUES('delete', old.rowid, old.title, old.content_raw, old.content_nl);
	END;
	CREATE TRIGGER IF NOT EXISTS knowledge_au AFTER UPDATE ON knowledge BEGIN
		INSERT INTO knowledge_fts(knowledge_fts, rowid, title, content_raw, content_nl) VALUES('delete', old.rowid, old.title, old.content_raw, old.content_nl);
		INSERT INTO knowledge_fts(rowid, title, content_raw, content_nl) VALUES (new.rowid, new.title, new.content_raw, new.content_nl);
	END;

	-- FTS triggers for skills
	CREATE TRIGGER IF NOT EXISTS skills_ai AFTER INSERT ON skills BEGIN
		INSERT INTO skills_fts(rowid, title, content) VALUES (new.rowid, new.title, new.content);
	END;
	CREATE TRIGGER IF NOT EXISTS skills_ad AFTER DELETE ON skills BEGIN
		INSERT INTO skills_fts(skills_fts, rowid, title, content) VALUES('delete', old.rowid, old.title, old.content);
	END;
	CREATE TRIGGER IF NOT EXISTS skills_au AFTER UPDATE ON skills BEGIN
		INSERT INTO skills_fts(skills_fts, rowid, title, content) VALUES('delete', old.rowid, old.title, old.content);
		INSERT INTO skills_fts(rowid, title, content) VALUES (new.rowid, new.title, new.content);
	END;
	`

	_, err := s.db.Exec(schema)
	return err
}
