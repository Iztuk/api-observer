package database

import (
	"database/sql"
	_ "embed"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed migrations/001_init.sql
var schemaSQL string

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	// SQLite behaves best for this small test service with one writer connection.
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("create database schema: %w", err)
	}

	if err := seed(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func seed(db *sql.DB) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)

	users := []struct {
		id, email, password, name, role string
	}{
		{"11111111-1111-1111-1111-111111111111", "alice@example.com", "password123", "Alice", "standard"},
		{"22222222-2222-2222-2222-222222222222", "bob@example.com", "password123", "Bob", "standard"},
		{"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "admin@example.com", "adminpass123", "Administrator", "admin"},
	}

	for _, u := range users {
		if _, err := db.Exec(`
			INSERT OR IGNORE INTO users(id, email, password, name, role, created_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, u.id, u.email, u.password, u.name, u.role, now); err != nil {
			return fmt.Errorf("seed user %s: %w", u.email, err)
		}
	}

	projects := []struct {
		id, ownerID, name, description string
	}{
		{"10000000-0000-0000-0000-000000000001", users[0].id, "Alice Project", "Project owned by Alice"},
		{"10000000-0000-0000-0000-000000000002", users[1].id, "Bob Project", "Project owned by Bob"},
	}

	for _, p := range projects {
		if _, err := db.Exec(`
			INSERT OR IGNORE INTO projects(id, owner_id, name, description, created_at)
			VALUES (?, ?, ?, ?, ?)
		`, p.id, p.ownerID, p.name, p.description, now); err != nil {
			return fmt.Errorf("seed project %s: %w", p.name, err)
		}
	}

	if _, err := db.Exec(`
		INSERT OR IGNORE INTO tasks(id, project_id, title, description, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, "20000000-0000-0000-0000-000000000001", projects[0].id, "Alice Task", "Seed task", "open", now); err != nil {
		return fmt.Errorf("seed task: %w", err)
	}

	return nil
}
