package store

import (
	"fmt"
)

func (s *DB) CreateTask(t *Task) error {
	t.TaskID = newID()
	_, err := s.db.Exec(`
		INSERT INTO tasks (task_id, role, title, description, status, priority, project_slug, plan_doc, result_summary, parent_task_id, tags, metadata_json, state)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.TaskID, t.Role, t.Title, t.Description, defaultStr(t.Status, "PENDING"), defaultStr(t.Priority, "MEDIUM"),
		t.ProjectSlug, t.PlanDoc, t.ResultSummary, t.ParentTaskID, t.Tags.String(), t.MetadataJSON, t.State)
	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}
	return s.getTaskByID(t.TaskID, t)
}

func (s *DB) GetTask(taskID string) (*Task, error) {
	var t Task
	if err := s.getTaskByID(taskID, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *DB) SearchTasks(role, status, projectSlug, priority, query string, tags []string, limit int) ([]Task, error) {
	q := `SELECT task_id, role, title, description, status, priority, project_slug, plan_doc, result_summary, parent_task_id, tags, metadata_json, state, started_at, completed_at, created_at, updated_at FROM tasks WHERE 1=1`
	args := []any{}

	if role != "" {
		q += ` AND role = ?`
		args = append(args, role)
	}
	if status != "" {
		q += ` AND status = ?`
		args = append(args, status)
	}
	if projectSlug != "" {
		q += ` AND project_slug = ?`
		args = append(args, projectSlug)
	}
	if priority != "" {
		q += ` AND priority = ?`
		args = append(args, priority)
	}
	if query != "" {
		frag, fargs := s.ftsSearchIDs("tasks", "tasks_fts", query, []string{"title", "description"})
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
		return nil, fmt.Errorf("search tasks: %w", err)
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		var tagsStr string
		var startedAt, completedAt nullStr
		if err := rows.Scan(&t.TaskID, &t.Role, &t.Title, &t.Description, &t.Status, &t.Priority,
			&t.ProjectSlug, &t.PlanDoc, &t.ResultSummary, &t.ParentTaskID, &tagsStr, &t.MetadataJSON,
			&t.State, &startedAt, &completedAt, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		t.Tags = ParseTags(tagsStr)
		t.StartedAt = string(startedAt)
		t.CompletedAt = string(completedAt)
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (s *DB) UpdateTask(taskID, status, resultSummary, state string, tags []string, metadataJSON string) (*Task, error) {
	sets := []string{"updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')"}
	args := []any{}

	if status != "" {
		sets = append(sets, "status = ?")
		args = append(args, status)
		if status == "IN_PROGRESS" {
			sets = append(sets, "started_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')")
		}
		if status == "COMPLETED" || status == "ABORTED" {
			sets = append(sets, "completed_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')")
		}
	}
	if resultSummary != "" {
		sets = append(sets, "result_summary = ?")
		args = append(args, resultSummary)
	}
	if state != "" {
		sets = append(sets, "state = ?")
		args = append(args, state)
	}
	if tags != nil {
		sets = append(sets, "tags = ?")
		args = append(args, Tags(tags).String())
	}
	if metadataJSON != "" {
		sets = append(sets, "metadata_json = ?")
		args = append(args, metadataJSON)
	}

	args = append(args, taskID)
	q := fmt.Sprintf("UPDATE tasks SET %s WHERE task_id = ?", joinStr(sets, ", "))
	res, err := s.db.Exec(q, args...)
	if err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, fmt.Errorf("task %s not found", taskID)
	}

	var t Task
	if err := s.getTaskByID(taskID, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *DB) getTaskByID(taskID string, t *Task) error {
	var tagsStr string
	var startedAt, completedAt nullStr
	err := s.db.QueryRow(`
		SELECT task_id, role, title, description, status, priority, project_slug, plan_doc, result_summary, parent_task_id, tags, metadata_json, state, started_at, completed_at, created_at, updated_at
		FROM tasks WHERE task_id = ?`, taskID).Scan(
		&t.TaskID, &t.Role, &t.Title, &t.Description, &t.Status, &t.Priority,
		&t.ProjectSlug, &t.PlanDoc, &t.ResultSummary, &t.ParentTaskID, &tagsStr, &t.MetadataJSON,
		&t.State, &startedAt, &completedAt, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return fmt.Errorf("get task %s: %w", taskID, err)
	}
	t.Tags = ParseTags(tagsStr)
	t.StartedAt = string(startedAt)
	t.CompletedAt = string(completedAt)
	return nil
}
