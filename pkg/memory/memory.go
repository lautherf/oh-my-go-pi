package memory

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type Memory struct {
	Content   string
	CreatedAt time.Time
	Score     float64
}

type Store struct {
	db *sql.DB
	mu sync.Mutex
}

func New(dsn string) (*Store, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("memory open: %w", err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS memories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		content TEXT NOT NULL,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		db.Close()
		return nil, fmt.Errorf("memory init: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *Store) Remember(ctx context.Context, content string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db == nil {
		return fmt.Errorf("store closed")
	}

	_, err := s.db.ExecContext(ctx,
		"INSERT INTO memories (content, created_at) VALUES (?, ?)",
		content, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("remember: %w", err)
	}
	return nil
}

func (s *Store) Recall(ctx context.Context, query string, limit ...int) ([]Memory, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db == nil {
		return nil, fmt.Errorf("store closed")
	}

	maxResults := 5
	if len(limit) > 0 && limit[0] > 0 {
		maxResults = limit[0]
	}

	// Simple LIKE-based search (vector search can replace this later)
	like := "%" + strings.ReplaceAll(query, " ", "%") + "%"
	rows, err := s.db.QueryContext(ctx,
		"SELECT content, created_at FROM memories WHERE content LIKE ? ORDER BY id DESC LIMIT ?",
		like, maxResults)
	if err != nil {
		return nil, fmt.Errorf("recall: %w", err)
	}
	defer rows.Close()

	var results []Memory
	for rows.Next() {
		var m Memory
		var createdStr string
		if err := rows.Scan(&m.Content, &createdStr); err != nil {
			return nil, fmt.Errorf("recall scan: %w", err)
		}
		m.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		results = append(results, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("recall rows: %w", err)
	}

	return results, nil
}
