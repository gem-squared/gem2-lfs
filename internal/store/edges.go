package store

import "fmt"

func (s *DB) CreateEdge(e *Edge) error {
	e.EdgeID = newID()
	_, err := s.db.Exec(`
		INSERT INTO edges (edge_id, source_id, target_id, project_slug, tags, metadata_json)
		VALUES (?, ?, ?, ?, ?, ?)`,
		e.EdgeID, e.SourceID, e.TargetID, e.ProjectSlug, e.Tags.String(), e.MetadataJSON)
	if err != nil {
		return fmt.Errorf("create edge: %w", err)
	}
	return s.db.QueryRow(`SELECT edge_id, source_id, target_id, project_slug, tags, metadata_json, created_at FROM edges WHERE edge_id = ?`, e.EdgeID).Scan(
		&e.EdgeID, &e.SourceID, &e.TargetID, &e.ProjectSlug, new(string), &e.MetadataJSON, &e.CreatedAt)
}

func (s *DB) SearchEdges(sourceID, targetID, projectSlug string, tags []string, limit int) ([]Edge, error) {
	q := `SELECT edge_id, source_id, target_id, project_slug, tags, metadata_json, created_at FROM edges WHERE 1=1`
	args := []any{}

	if sourceID != "" {
		q += ` AND source_id = ?`
		args = append(args, sourceID)
	}
	if targetID != "" {
		q += ` AND target_id = ?`
		args = append(args, targetID)
	}
	if projectSlug != "" {
		q += ` AND project_slug = ?`
		args = append(args, projectSlug)
	}
	q += ` ORDER BY created_at DESC`
	if limit > 0 {
		q += fmt.Sprintf(` LIMIT %d`, limit)
	} else {
		q += ` LIMIT 50`
	}

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("search edges: %w", err)
	}
	defer rows.Close()

	var edges []Edge
	for rows.Next() {
		var e Edge
		var tagsStr string
		if err := rows.Scan(&e.EdgeID, &e.SourceID, &e.TargetID, &e.ProjectSlug, &tagsStr, &e.MetadataJSON, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan edge: %w", err)
		}
		e.Tags = ParseTags(tagsStr)
		edges = append(edges, e)
	}
	return edges, rows.Err()
}

func (s *DB) DeleteEdge(edgeID string) error {
	res, err := s.db.Exec("DELETE FROM edges WHERE edge_id = ?", edgeID)
	if err != nil {
		return fmt.Errorf("delete edge: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("edge %s not found", edgeID)
	}
	return nil
}
