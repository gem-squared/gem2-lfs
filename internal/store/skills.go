package store

import "fmt"

func (s *DB) UpsertSkill(sk *Skill) error {
	// Try to find existing by skill_id or (user_id, title).
	var existingID string
	if sk.SkillID != "" {
		s.db.QueryRow("SELECT skill_id FROM skills WHERE skill_id = ?", sk.SkillID).Scan(&existingID)
	}
	if existingID == "" {
		s.db.QueryRow("SELECT skill_id FROM skills WHERE user_id = ? AND title = ?", sk.UserID, sk.Title).Scan(&existingID)
	}

	if existingID != "" {
		// Update.
		sk.SkillID = existingID
		_, err := s.db.Exec(`
			UPDATE skills SET content = ?, skill_type = ?, tags = ?, source_tool = ?, version = version + 1, metadata = ?,
			updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE skill_id = ?`,
			sk.Content, sk.SkillType, sk.Tags.String(), sk.SourceTool, sk.Metadata, existingID)
		if err != nil {
			return fmt.Errorf("update skill: %w", err)
		}
	} else {
		// Create.
		sk.SkillID = newID()
		_, err := s.db.Exec(`
			INSERT INTO skills (skill_id, user_id, skill_type, title, content, tags, source_tool, version, metadata)
			VALUES (?, ?, ?, ?, ?, ?, ?, 1, ?)`,
			sk.SkillID, sk.UserID, sk.SkillType, sk.Title, sk.Content, sk.Tags.String(), sk.SourceTool, sk.Metadata)
		if err != nil {
			return fmt.Errorf("create skill: %w", err)
		}
	}
	return s.getSkillByID(sk.SkillID, sk)
}

func (s *DB) GetSkill(skillID, userID string) (*Skill, error) {
	var sk Skill
	var tagsStr string
	err := s.db.QueryRow(`
		SELECT skill_id, user_id, skill_type, title, content, tags, source_tool, version, metadata, created_at, updated_at
		FROM skills WHERE skill_id = ? AND (user_id = ? OR ? = '')`, skillID, userID, userID).Scan(
		&sk.SkillID, &sk.UserID, &sk.SkillType, &sk.Title, &sk.Content, &tagsStr,
		&sk.SourceTool, &sk.Version, &sk.Metadata, &sk.CreatedAt, &sk.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get skill %s: %w", skillID, err)
	}
	sk.Tags = ParseTags(tagsStr)
	return &sk, nil
}

func (s *DB) SearchSkills(userID, query, skillType string, tags []string, limit int) ([]Skill, error) {
	q := `SELECT skill_id, user_id, skill_type, title, content, tags, source_tool, version, metadata, created_at, updated_at FROM skills WHERE 1=1`
	args := []any{}

	if userID != "" {
		q += ` AND user_id = ?`
		args = append(args, userID)
	}
	if skillType != "" {
		q += ` AND skill_type = ?`
		args = append(args, skillType)
	}
	if query != "" {
		frag, fargs := s.ftsSearchIDs("skills", "skills_fts", query, []string{"title", "content"})
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
		return nil, fmt.Errorf("search skills: %w", err)
	}
	defer rows.Close()

	var skills []Skill
	for rows.Next() {
		var sk Skill
		var tagsStr string
		if err := rows.Scan(&sk.SkillID, &sk.UserID, &sk.SkillType, &sk.Title, &sk.Content, &tagsStr,
			&sk.SourceTool, &sk.Version, &sk.Metadata, &sk.CreatedAt, &sk.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan skill: %w", err)
		}
		sk.Tags = ParseTags(tagsStr)
		skills = append(skills, sk)
	}
	return skills, rows.Err()
}

func (s *DB) DeleteSkill(skillID, userID string) error {
	res, err := s.db.Exec("DELETE FROM skills WHERE skill_id = ? AND (user_id = ? OR ? = '')", skillID, userID, userID)
	if err != nil {
		return fmt.Errorf("delete skill: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("skill %s not found", skillID)
	}
	return nil
}

func (s *DB) getSkillByID(skillID string, sk *Skill) error {
	var tagsStr string
	return s.db.QueryRow(`
		SELECT skill_id, user_id, skill_type, title, content, tags, source_tool, version, metadata, created_at, updated_at
		FROM skills WHERE skill_id = ?`, skillID).Scan(
		&sk.SkillID, &sk.UserID, &sk.SkillType, &sk.Title, &sk.Content, &tagsStr,
		&sk.SourceTool, &sk.Version, &sk.Metadata, &sk.CreatedAt, &sk.UpdatedAt)
}
