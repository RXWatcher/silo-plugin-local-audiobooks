package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type Cover struct {
	AudiobookID string
	ContentType string
	Bytes       []byte
	Source      string // "embedded" | "sidecar"
}

func (s *Store) UpsertCover(ctx context.Context, c Cover) error {
	const q = `
INSERT INTO cover (audiobook_id, content_type, bytes, source)
VALUES ($1,$2,$3,$4)
ON CONFLICT (audiobook_id) DO UPDATE SET
    content_type = EXCLUDED.content_type,
    bytes = EXCLUDED.bytes,
    source = EXCLUDED.source`
	_, err := s.pool.Exec(ctx, q, c.AudiobookID, c.ContentType, c.Bytes, c.Source)
	if err != nil {
		return fmt.Errorf("store.UpsertCover: %w", err)
	}
	return nil
}

func (s *Store) GetCover(ctx context.Context, audiobookID string) (*Cover, error) {
	const q = `SELECT audiobook_id, content_type, bytes, source FROM cover WHERE audiobook_id = $1`
	row := s.pool.QueryRow(ctx, q, audiobookID)
	c := &Cover{}
	err := row.Scan(&c.AudiobookID, &c.ContentType, &c.Bytes, &c.Source)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("store.GetCover: %w", err)
	}
	return c, nil
}
