package store

import "fmt"

func (s *DB) CreateProject(p *Project) error {
	_, err := s.db.Exec(`
		INSERT INTO projects (project_slug, name, root_path, status, metadata_json)
		VALUES (?, ?, ?, ?, ?)`,
		p.ProjectSlug, p.Name, p.RootPath, defaultStr(p.Status, "active"), p.MetadataJSON)
	if err != nil {
		return fmt.Errorf("create project: %w", err)
	}
	return s.db.QueryRow(`SELECT project_slug, name, root_path, status, metadata_json, created_at FROM projects WHERE project_slug = ?`, p.ProjectSlug).Scan(
		&p.ProjectSlug, &p.Name, &p.RootPath, &p.Status, &p.MetadataJSON, &p.CreatedAt)
}

func (s *DB) GetProject(slug string) (*Project, error) {
	var p Project
	err := s.db.QueryRow(`SELECT project_slug, name, root_path, status, metadata_json, created_at FROM projects WHERE project_slug = ?`, slug).Scan(
		&p.ProjectSlug, &p.Name, &p.RootPath, &p.Status, &p.MetadataJSON, &p.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get project %s: %w", slug, err)
	}
	return &p, nil
}

func (s *DB) ListProjects() ([]Project, error) {
	rows, err := s.db.Query(`SELECT project_slug, name, root_path, status, metadata_json, created_at FROM projects ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ProjectSlug, &p.Name, &p.RootPath, &p.Status, &p.MetadataJSON, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (s *DB) EnsureProject(slug, name, rootPath string) (*Project, error) {
	p, err := s.GetProject(slug)
	if err == nil {
		return p, nil
	}
	p = &Project{ProjectSlug: slug, Name: name, RootPath: rootPath}
	if err := s.CreateProject(p); err != nil {
		return nil, err
	}
	return p, nil
}
