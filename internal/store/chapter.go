package store

import (
	"context"
	"fmt"
)

type Chapter struct {
	AudiobookID string
	Idx         int
	Title       string
	StartMs     int64
	EndMs       int64
}

// ReplaceChapters deletes existing chapters for the book and inserts the
// supplied slice in a single transaction.
func (s *Store) ReplaceChapters(ctx context.Context, audiobookID string, chapters []Chapter) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM chapter WHERE audiobook_id = $1`, audiobookID); err != nil {
		return fmt.Errorf("delete chapters: %w", err)
	}
	for _, c := range chapters {
		if _, err := tx.Exec(ctx,
			`INSERT INTO chapter (audiobook_id, idx, title, start_ms, end_ms) VALUES ($1,$2,$3,$4,$5)`,
			audiobookID, c.Idx, c.Title, c.StartMs, c.EndMs); err != nil {
			return fmt.Errorf("insert chapter %d: %w", c.Idx, err)
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) ListChapters(ctx context.Context, audiobookID string) ([]Chapter, error) {
	const q = `SELECT audiobook_id, idx, title, start_ms, end_ms FROM chapter
WHERE audiobook_id = $1 ORDER BY idx`
	rows, err := s.pool.Query(ctx, q, audiobookID)
	if err != nil {
		return nil, fmt.Errorf("store.ListChapters: %w", err)
	}
	defer rows.Close()
	var out []Chapter
	for rows.Next() {
		var c Chapter
		if err := rows.Scan(&c.AudiobookID, &c.Idx, &c.Title, &c.StartMs, &c.EndMs); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
