package store

import (
	"context"
	"fmt"
	"time"
)

type ScanEvent struct {
	ID            int64
	LibraryPathID *int64
	StartedAt     time.Time
	FinishedAt    *time.Time
	BooksAdded    int
	BooksChanged  int
	BooksDeleted  int
	ErrorText     *string
}

func (s *Store) InsertScanEvent(ctx context.Context, libraryPathID *int64) (int64, error) {
	const q = `INSERT INTO scan_event (library_path_id) VALUES ($1) RETURNING id`
	var id int64
	if err := s.pool.QueryRow(ctx, q, libraryPathID).Scan(&id); err != nil {
		return 0, fmt.Errorf("store.InsertScanEvent: %w", err)
	}
	return id, nil
}

func (s *Store) FinishScanEvent(ctx context.Context, id int64, added, changed, deleted int, errText string) error {
	const q = `
UPDATE scan_event
SET finished_at = now(),
    books_added = $2, books_changed = $3, books_deleted = $4,
    error_text = NULLIF($5, '')
WHERE id = $1`
	_, err := s.pool.Exec(ctx, q, id, added, changed, deleted, errText)
	if err != nil {
		return fmt.Errorf("store.FinishScanEvent: %w", err)
	}
	return nil
}

func (s *Store) ListRecentScanEvents(ctx context.Context, limit int) ([]*ScanEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	const q = `SELECT id, library_path_id, started_at, finished_at, books_added, books_changed, books_deleted, error_text
FROM scan_event ORDER BY started_at DESC LIMIT $1`
	rows, err := s.pool.Query(ctx, q, limit)
	if err != nil {
		return nil, fmt.Errorf("store.ListRecentScanEvents: %w", err)
	}
	defer rows.Close()
	var out []*ScanEvent
	for rows.Next() {
		e := &ScanEvent{}
		if err := rows.Scan(&e.ID, &e.LibraryPathID, &e.StartedAt, &e.FinishedAt, &e.BooksAdded, &e.BooksChanged, &e.BooksDeleted, &e.ErrorText); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) HasInFlightScan(ctx context.Context, libraryPathID *int64) (int64, error) {
	q := `SELECT id FROM scan_event WHERE finished_at IS NULL`
	args := []any{}
	if libraryPathID != nil {
		q += " AND library_path_id = $1"
		args = append(args, *libraryPathID)
	}
	q += " ORDER BY started_at DESC LIMIT 1"
	var id int64
	err := s.pool.QueryRow(ctx, q, args...).Scan(&id)
	if err != nil {
		return 0, nil
	}
	return id, nil
}
