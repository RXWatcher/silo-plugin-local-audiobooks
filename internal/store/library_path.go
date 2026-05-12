package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// LibraryPath is a configured filesystem root the scanner walks.
type LibraryPath struct {
	ID            int64
	Path          string
	Enabled       bool
	LastScannedAt *time.Time
	CreatedAt     time.Time
}

// ErrNotFound signals a missing row.
var ErrNotFound = errors.New("not found")

// UpsertLibraryPath inserts the path or no-ops if it already exists.
// Returns the row.
func (s *Store) UpsertLibraryPath(ctx context.Context, path string) (*LibraryPath, error) {
	const q = `
INSERT INTO library_path (path) VALUES ($1)
ON CONFLICT (path) DO UPDATE SET path = EXCLUDED.path
RETURNING id, path, enabled, last_scanned_at, created_at`
	row := s.pool.QueryRow(ctx, q, path)
	out := &LibraryPath{}
	if err := row.Scan(&out.ID, &out.Path, &out.Enabled, &out.LastScannedAt, &out.CreatedAt); err != nil {
		return nil, fmt.Errorf("store.UpsertLibraryPath: %w", err)
	}
	return out, nil
}

// ListLibraryPaths returns every configured root, enabled or not.
func (s *Store) ListLibraryPaths(ctx context.Context) ([]*LibraryPath, error) {
	const q = `SELECT id, path, enabled, last_scanned_at, created_at
FROM library_path ORDER BY id`
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("store.ListLibraryPaths: %w", err)
	}
	defer rows.Close()
	var out []*LibraryPath
	for rows.Next() {
		lp := &LibraryPath{}
		if err := rows.Scan(&lp.ID, &lp.Path, &lp.Enabled, &lp.LastScannedAt, &lp.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, lp)
	}
	return out, rows.Err()
}

// DeleteLibraryPath removes a root and (via ON DELETE CASCADE) all its
// audiobook rows.
func (s *Store) DeleteLibraryPath(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM library_path WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("store.DeleteLibraryPath: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// MarkLibraryScanned updates last_scanned_at = now() for the given id.
func (s *Store) MarkLibraryScanned(ctx context.Context, id int64) error {
	const q = `UPDATE library_path SET last_scanned_at = now() WHERE id = $1`
	_, err := s.pool.Exec(ctx, q, id)
	return err
}

// Keep pgx import meaningful (sentinel; not yet used outside this file).
var _ = pgx.ErrNoRows
