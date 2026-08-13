// Package store persists eval run summaries in a local SQLite database.
//
// The data layer uses modernc.org/sqlite: a pure-Go, zero-CGO build of SQLite
// so the binary keeps zero external runtime dependencies.
package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// DBPath returns the local store location.
func DBPath() string {
	if p := os.Getenv("PROMPT_DIFF_DB"); p != "" {
		return p
	}
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		base = "."
	}
	return filepath.Join(base, "prompt-diff", "runs.db")
}

// Store wraps a SQLite connection.
type Store struct {
	db *sql.DB
}

// Run is a stored eval run summary.
type Run struct {
	ID         int64
	CreatedAt  string
	PromptPath string
	Suite      string
	Models     string
	Passed     int
	Failed     int
	Skipped    int
	Total      int
}

// Open opens (creating if needed) the local store.
func Open() (*Store, error) {
	path := DBPath()
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS runs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		created_at TEXT NOT NULL,
		prompt_path TEXT NOT NULL,
		suite TEXT NOT NULL,
		models TEXT NOT NULL,
		passed INTEGER NOT NULL,
		failed INTEGER NOT NULL,
		skipped INTEGER NOT NULL,
		total INTEGER NOT NULL
	)`); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

// Close closes the underlying connection.
func (s *Store) Close() error { return s.db.Close() }

// RecordRun inserts a run summary and returns its id.
func (s *Store) RecordRun(promptPath, suite, models string, total, passed, failed, skipped int) (int64, error) {
	res, err := s.db.Exec(`INSERT INTO runs (created_at, prompt_path, suite, models, passed, failed, skipped, total)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		time.Now().Format("2006-01-02 15:04:05"), promptPath, suite, models, passed, failed, skipped, total)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ListRuns returns stored runs newest first.
func (s *Store) ListRuns() ([]Run, error) {
	rows, err := s.db.Query(`SELECT id, created_at, prompt_path, suite, models, passed, failed, skipped, total
		FROM runs ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Run
	for rows.Next() {
		var r Run
		if err := rows.Scan(&r.ID, &r.CreatedAt, &r.PromptPath, &r.Suite, &r.Models,
			&r.Passed, &r.Failed, &r.Skipped, &r.Total); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

var _ = fmt.Sprintf
var _ = strings.TrimSpace