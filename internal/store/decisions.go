package store

import "fmt"

func (s *DB) CreateDecision(d *Decision) error {
	d.DecisionID = newID()
	_, err := s.db.Exec(`
		INSERT INTO decisions (decision_id, role, category, title, content, status, superseded_by, project_slug, tags, metadata_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		d.DecisionID, d.Role, d.Category, d.Title, d.Content, defaultStr(d.Status, "ACTIVE"),
		d.SupersededBy, d.ProjectSlug, d.Tags.String(), d.MetadataJSON)
	if err != nil {
		return fmt.Errorf("create decision: %w", err)
	}
	return s.getDecisionByID(d.DecisionID, d)
}

func (s *DB) SearchDecisions(role, category, status, projectSlug, query string, tags []string, limit int) ([]Decision, error) {
	q := `SELECT decision_id, role, category, title, content, status, superseded_by, project_slug, tags, metadata_json, decided_at, created_at, updated_at FROM decisions WHERE 1=1`
	args := []any{}

	if role != "" {
		q += ` AND role = ?`
		args = append(args, role)
	}
	if category != "" {
		q += ` AND category = ?`
		args = append(args, category)
	}
	if status != "" {
		q += ` AND status = ?`
		args = append(args, status)
	}
	if projectSlug != "" {
		q += ` AND project_slug = ?`
		args = append(args, projectSlug)
	}
	if query != "" {
		frag, fargs := s.ftsSearchIDs("decisions", "decisions_fts", query, []string{"title", "content"})
		q += ` AND ` + frag
		args = append(args, fargs...)
	}
	q += ` ORDER BY created_at DESC`
	if limit > 0 {
		q += fmt.Sprintf(` LIMIT %d`, limit)
	} else {
		q += ` LIMIT 50`
	}

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("search decisions: %w", err)
	}
	defer rows.Close()

	var decs []Decision
	for rows.Next() {
		var d Decision
		var tagsStr string
		if err := rows.Scan(&d.DecisionID, &d.Role, &d.Category, &d.Title, &d.Content, &d.Status,
			&d.SupersededBy, &d.ProjectSlug, &tagsStr, &d.MetadataJSON, &d.DecidedAt, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan decision: %w", err)
		}
		d.Tags = ParseTags(tagsStr)
		decs = append(decs, d)
	}
	return decs, rows.Err()
}

func (s *DB) UpdateDecision(decisionID, status, supersededBy string, tags []string, content string) (*Decision, error) {
	sets := []string{"updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')"}
	args := []any{}

	if status != "" {
		sets = append(sets, "status = ?")
		args = append(args, status)
	}
	if supersededBy != "" {
		sets = append(sets, "superseded_by = ?")
		args = append(args, supersededBy)
	}
	if tags != nil {
		sets = append(sets, "tags = ?")
		args = append(args, Tags(tags).String())
	}
	if content != "" {
		sets = append(sets, "content = ?")
		args = append(args, content)
	}

	args = append(args, decisionID)
	q := fmt.Sprintf("UPDATE decisions SET %s WHERE decision_id = ?", joinStr(sets, ", "))
	res, err := s.db.Exec(q, args...)
	if err != nil {
		return nil, fmt.Errorf("update decision: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, fmt.Errorf("decision %s not found", decisionID)
	}

	var d Decision
	if err := s.getDecisionByID(decisionID, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *DB) getDecisionByID(decisionID string, d *Decision) error {
	var tagsStr string
	err := s.db.QueryRow(`
		SELECT decision_id, role, category, title, content, status, superseded_by, project_slug, tags, metadata_json, decided_at, created_at, updated_at
		FROM decisions WHERE decision_id = ?`, decisionID).Scan(
		&d.DecisionID, &d.Role, &d.Category, &d.Title, &d.Content, &d.Status,
		&d.SupersededBy, &d.ProjectSlug, &tagsStr, &d.MetadataJSON, &d.DecidedAt, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return fmt.Errorf("get decision %s: %w", decisionID, err)
	}
	d.Tags = ParseTags(tagsStr)
	return nil
}
