package store

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (s *DB) CreateKnowledge(k *Knowledge) (int, error) {
	k.KnowledgeID = newID()
	metaJSON, _ := json.Marshal(k.Metadata)
	_, err := s.db.Exec(`
		INSERT INTO knowledge (knowledge_id, entity_type, title, content_raw, content_nl, project_slug, tags, file_path, line_count, metadata_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		k.KnowledgeID, k.EntityType, k.Title, k.ContentRaw, k.ContentNL, k.ProjectSlug,
		k.Tags.String(), k.FilePath, k.LineCount, string(metaJSON))
	if err != nil {
		return 0, fmt.Errorf("create knowledge: %w", err)
	}

	// Simple heading-based chunking.
	chunks := chunkContent(k.KnowledgeID, k.ContentRaw)
	for _, c := range chunks {
		_, err := s.db.Exec(`
			INSERT INTO knowledge_chunks (chunk_id, knowledge_id, chunk_index, content_nl, heading, text)
			VALUES (?, ?, ?, ?, ?, ?)`,
			c.ChunkID, c.KnowledgeID, c.ChunkIndex, c.ContentNL, c.Heading, c.Text)
		if err != nil {
			return 0, fmt.Errorf("create chunk: %w", err)
		}
	}
	return len(chunks), nil
}

func (s *DB) GetKnowledge(knowledgeID string) (*Knowledge, []KnowledgeChunk, error) {
	var k Knowledge
	var tagsStr, metaStr string
	err := s.db.QueryRow(`
		SELECT knowledge_id, entity_type, title, content_raw, content_nl, project_slug, tags, file_path, line_count, metadata_json, embedding_model, embedding_dim, created_at, updated_at
		FROM knowledge WHERE knowledge_id = ?`, knowledgeID).Scan(
		&k.KnowledgeID, &k.EntityType, &k.Title, &k.ContentRaw, &k.ContentNL, &k.ProjectSlug,
		&tagsStr, &k.FilePath, &k.LineCount, &metaStr, &k.EmbeddingModel, &k.EmbeddingDim, &k.CreatedAt, &k.UpdatedAt)
	if err != nil {
		return nil, nil, fmt.Errorf("get knowledge %s: %w", knowledgeID, err)
	}
	k.Tags = ParseTags(tagsStr)
	json.Unmarshal([]byte(metaStr), &k.Metadata)

	rows, err := s.db.Query(`
		SELECT chunk_id, knowledge_id, chunk_index, content_nl, tags, heading, text
		FROM knowledge_chunks WHERE knowledge_id = ? ORDER BY chunk_index`, knowledgeID)
	if err != nil {
		return &k, nil, nil
	}
	defer rows.Close()

	var chunks []KnowledgeChunk
	for rows.Next() {
		var c KnowledgeChunk
		var cTagsStr string
		if err := rows.Scan(&c.ChunkID, &c.KnowledgeID, &c.ChunkIndex, &c.ContentNL, &cTagsStr, &c.Heading, &c.Text); err != nil {
			continue
		}
		c.Tags = ParseTags(cTagsStr)
		chunks = append(chunks, c)
	}
	return &k, chunks, nil
}

func (s *DB) UpsertKnowledge(k *Knowledge) (knowledgeID string, created bool, version int, chunksCreated int, err error) {
	// Try to find existing by (project_slug, entity_type, title).
	var existingID string
	var existingVersion int
	findErr := s.db.QueryRow(`
		SELECT knowledge_id, COALESCE(json_extract(metadata_json, '$.version'), 1)
		FROM knowledge WHERE project_slug = ? AND entity_type = ? AND title = ?`,
		k.ProjectSlug, k.EntityType, k.Title).Scan(&existingID, &existingVersion)

	metaJSON, _ := json.Marshal(k.Metadata)

	if findErr != nil {
		// Create new.
		k.KnowledgeID = newID()
		_, err = s.db.Exec(`
			INSERT INTO knowledge (knowledge_id, entity_type, title, content_raw, content_nl, project_slug, tags, file_path, line_count, metadata_json)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			k.KnowledgeID, k.EntityType, k.Title, k.ContentRaw, k.ContentNL, k.ProjectSlug,
			k.Tags.String(), k.FilePath, k.LineCount, string(metaJSON))
		if err != nil {
			return "", false, 0, 0, fmt.Errorf("upsert create: %w", err)
		}
		created = true
		version = 1
	} else {
		// Update existing.
		k.KnowledgeID = existingID
		version = existingVersion + 1
		_, err = s.db.Exec(`
			UPDATE knowledge SET content_raw = ?, content_nl = ?, tags = ?, file_path = ?, line_count = ?, metadata_json = ?,
			updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE knowledge_id = ?`,
			k.ContentRaw, k.ContentNL, k.Tags.String(), k.FilePath, k.LineCount, string(metaJSON), existingID)
		if err != nil {
			return "", false, 0, 0, fmt.Errorf("upsert update: %w", err)
		}
		// Delete old chunks.
		s.db.Exec("DELETE FROM knowledge_chunks WHERE knowledge_id = ?", existingID)
	}

	// Re-chunk.
	chunks := chunkContent(k.KnowledgeID, k.ContentRaw)
	for _, c := range chunks {
		s.db.Exec(`INSERT INTO knowledge_chunks (chunk_id, knowledge_id, chunk_index, content_nl, heading, text) VALUES (?, ?, ?, ?, ?, ?)`,
			c.ChunkID, c.KnowledgeID, c.ChunkIndex, c.ContentNL, c.Heading, c.Text)
	}

	return k.KnowledgeID, created, version, len(chunks), nil
}

func (s *DB) SearchKnowledge(projectSlug, entityType, query string, tags []string, limit int) ([]Knowledge, error) {
	q := `SELECT knowledge_id, entity_type, title, content_raw, content_nl, project_slug, tags, file_path, line_count, metadata_json, created_at, updated_at FROM knowledge WHERE 1=1`
	args := []any{}

	if projectSlug != "" {
		q += ` AND project_slug = ?`
		args = append(args, projectSlug)
	}
	if entityType != "" {
		q += ` AND entity_type = ?`
		args = append(args, entityType)
	}
	if query != "" {
		frag, fargs := s.ftsSearchIDs("knowledge", "knowledge_fts", query, []string{"title", "content_raw", "content_nl"})
		q += ` AND ` + frag
		args = append(args, fargs...)
	}
	q += ` ORDER BY updated_at DESC`
	if limit > 0 {
		q += fmt.Sprintf(` LIMIT %d`, limit)
	} else {
		q += ` LIMIT 50`
	}

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("search knowledge: %w", err)
	}
	defer rows.Close()

	var results []Knowledge
	for rows.Next() {
		var k Knowledge
		var tagsStr, metaStr string
		if err := rows.Scan(&k.KnowledgeID, &k.EntityType, &k.Title, &k.ContentRaw, &k.ContentNL,
			&k.ProjectSlug, &tagsStr, &k.FilePath, &k.LineCount, &metaStr, &k.CreatedAt, &k.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan knowledge: %w", err)
		}
		k.Tags = ParseTags(tagsStr)
		json.Unmarshal([]byte(metaStr), &k.Metadata)
		results = append(results, k)
	}
	return results, rows.Err()
}

// StoreEmbedding saves an embedding for a knowledge document.
func (s *DB) StoreEmbedding(knowledgeID string, embedding []byte, model string, dim int) error {
	_, err := s.db.Exec(`UPDATE knowledge SET embedding = ?, embedding_model = ?, embedding_dim = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE knowledge_id = ?`,
		embedding, model, dim, knowledgeID)
	return err
}

// StoreChunkEmbedding saves an embedding for a knowledge chunk.
func (s *DB) StoreChunkEmbedding(chunkID string, embedding []byte) error {
	_, err := s.db.Exec(`UPDATE knowledge_chunks SET embedding = ? WHERE chunk_id = ?`, embedding, chunkID)
	return err
}

// AllKnowledgeWithEmbeddings returns all knowledge docs that have embeddings.
func (s *DB) AllKnowledgeWithEmbeddings() ([]Knowledge, [][]byte, error) {
	rows, err := s.db.Query(`SELECT knowledge_id, entity_type, title, content_raw, content_nl, project_slug, tags, file_path, line_count, metadata_json, embedding_model, embedding_dim, created_at, updated_at, embedding FROM knowledge WHERE embedding IS NOT NULL`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var knowledges []Knowledge
	var embeddings [][]byte
	for rows.Next() {
		var k Knowledge
		var tagsStr, metaStr string
		var emb []byte
		if err := rows.Scan(&k.KnowledgeID, &k.EntityType, &k.Title, &k.ContentRaw, &k.ContentNL, &k.ProjectSlug,
			&tagsStr, &k.FilePath, &k.LineCount, &metaStr, &k.EmbeddingModel, &k.EmbeddingDim, &k.CreatedAt, &k.UpdatedAt, &emb); err != nil {
			continue
		}
		k.Tags = ParseTags(tagsStr)
		knowledges = append(knowledges, k)
		embeddings = append(embeddings, emb)
	}
	return knowledges, embeddings, rows.Err()
}

// AllChunksWithEmbeddings returns all chunks that have embeddings, for a given knowledge_id or all.
func (s *DB) AllChunksWithEmbeddings(knowledgeID string) ([]KnowledgeChunk, [][]byte, error) {
	q := `SELECT chunk_id, knowledge_id, chunk_index, content_nl, tags, heading, text, embedding FROM knowledge_chunks WHERE embedding IS NOT NULL`
	args := []any{}
	if knowledgeID != "" {
		q += ` AND knowledge_id = ?`
		args = append(args, knowledgeID)
	}
	q += ` ORDER BY chunk_index`

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var chunks []KnowledgeChunk
	var embeddings [][]byte
	for rows.Next() {
		var c KnowledgeChunk
		var tagsStr string
		var emb []byte
		if err := rows.Scan(&c.ChunkID, &c.KnowledgeID, &c.ChunkIndex, &c.ContentNL, &tagsStr, &c.Heading, &c.Text, &emb); err != nil {
			continue
		}
		c.Tags = ParseTags(tagsStr)
		chunks = append(chunks, c)
		embeddings = append(embeddings, emb)
	}
	return chunks, embeddings, rows.Err()
}

// chunkContent splits content by markdown headings.
func chunkContent(knowledgeID, content string) []KnowledgeChunk {
	if content == "" {
		return nil
	}

	lines := strings.Split(content, "\n")
	var chunks []KnowledgeChunk
	var currentHeading string
	var currentLines []string
	idx := 0

	flush := func() {
		if len(currentLines) == 0 {
			return
		}
		text := strings.TrimSpace(strings.Join(currentLines, "\n"))
		if text == "" {
			return
		}
		chunks = append(chunks, KnowledgeChunk{
			ChunkID:     newID(),
			KnowledgeID: knowledgeID,
			ChunkIndex:  idx,
			Heading:     currentHeading,
			Text:        text,
			ContentNL:   text,
		})
		idx++
		currentLines = nil
	}

	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			flush()
			currentHeading = strings.TrimSpace(strings.TrimLeft(line, "#"))
		}
		currentLines = append(currentLines, line)
	}
	flush()

	return chunks
}
