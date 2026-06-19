package store

import "fmt"

func (s *DB) CreateMessage(m *Message) error {
	m.MsgID = newID()
	_, err := s.db.Exec(`
		INSERT INTO messages (msg_id, msg_type, from_role, to_role, message, content, project_slug, g_service_id, session_id, terminal_id, terminal_title, metadata_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.MsgID, m.MsgType, m.FromRole, m.ToRole, m.MessageText, m.Content,
		m.ProjectSlug, m.GServiceID, m.SessionID, m.TerminalID, m.TerminalTitle, m.MetadataJSON)
	if err != nil {
		return fmt.Errorf("create message: %w", err)
	}
	return s.getMessageByID(m.MsgID, m)
}

func (s *DB) GetMessage(msgID string) (*Message, error) {
	var m Message
	if err := s.getMessageByID(msgID, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *DB) SearchMessages(fromRole, toRole, projectSlug, sessionID, query string, limit int) ([]Message, error) {
	q := `SELECT msg_id, msg_type, from_role, to_role, message, content, project_slug, g_service_id, session_id, terminal_id, terminal_title, metadata_json, created_at, updated_at FROM messages WHERE 1=1`
	args := []any{}

	if fromRole != "" {
		q += ` AND from_role = ?`
		args = append(args, fromRole)
	}
	if toRole != "" {
		q += ` AND to_role = ?`
		args = append(args, toRole)
	}
	if projectSlug != "" {
		q += ` AND project_slug = ?`
		args = append(args, projectSlug)
	}
	if sessionID != "" {
		q += ` AND session_id = ?`
		args = append(args, sessionID)
	}
	if query != "" {
		frag, fargs := s.ftsSearchIDs("messages", "messages_fts", query, []string{"message", "content"})
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
		return nil, fmt.Errorf("search messages: %w", err)
	}
	defer rows.Close()

	var msgs []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.MsgID, &m.MsgType, &m.FromRole, &m.ToRole, &m.MessageText, &m.Content,
			&m.ProjectSlug, &m.GServiceID, &m.SessionID, &m.TerminalID, &m.TerminalTitle, &m.MetadataJSON,
			&m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func (s *DB) getMessageByID(msgID string, m *Message) error {
	return s.db.QueryRow(`
		SELECT msg_id, msg_type, from_role, to_role, message, content, project_slug, g_service_id, session_id, terminal_id, terminal_title, metadata_json, created_at, updated_at
		FROM messages WHERE msg_id = ?`, msgID).Scan(
		&m.MsgID, &m.MsgType, &m.FromRole, &m.ToRole, &m.MessageText, &m.Content,
		&m.ProjectSlug, &m.GServiceID, &m.SessionID, &m.TerminalID, &m.TerminalTitle, &m.MetadataJSON,
		&m.CreatedAt, &m.UpdatedAt)
}
