package metadata

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MaxAttempts is the per-job retry cap before marking failed.
const MaxAttempts = 5

// Job is a row in metadata_enrichment_job, returned by ClaimNext.
type Job struct {
	AudiobookID string
	Attempts    int
}

// Queue wraps metadata_enrichment_job persistence.
type Queue struct {
	pool *pgxpool.Pool
}

// NewQueue constructs a Queue against the given pool.
func NewQueue(pool *pgxpool.Pool) *Queue { return &Queue{pool: pool} }

// ErrQueueEmpty is returned when no pending jobs are ready to run.
var ErrQueueEmpty = errors.New("metadata enrichment queue empty")

// Enqueue inserts a pending row for the audiobook. If a row already exists
// (any status), it is reset to pending so a re-enrichment runs.
func (q *Queue) Enqueue(ctx context.Context, audiobookID string) error {
	_, err := q.pool.Exec(ctx, `
		INSERT INTO metadata_enrichment_job (audiobook_id)
		VALUES ($1)
		ON CONFLICT (audiobook_id) DO UPDATE
		  SET status = 'pending',
		      attempts = 0,
		      run_after = now(),
		      last_error = '',
		      finished_at = NULL
	`, audiobookID)
	return err
}

// ClaimNext atomically picks one ready pending job, increments its attempt
// count, and returns it. The job remains in 'pending' status until
// MarkCompleted or MarkFailed is called by the worker. Uses FOR UPDATE
// SKIP LOCKED so concurrent workers don't double-claim.
func (q *Queue) ClaimNext(ctx context.Context) (Job, error) {
	tx, err := q.pool.Begin(ctx)
	if err != nil {
		return Job{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var j Job
	err = tx.QueryRow(ctx, `
		SELECT audiobook_id, attempts FROM metadata_enrichment_job
		WHERE status = 'pending' AND run_after <= now()
		ORDER BY run_after
		FOR UPDATE SKIP LOCKED
		LIMIT 1
	`).Scan(&j.AudiobookID, &j.Attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return Job{}, ErrQueueEmpty
	}
	if err != nil {
		return Job{}, err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE metadata_enrichment_job SET attempts = attempts + 1
		WHERE audiobook_id = $1
	`, j.AudiobookID); err != nil {
		return Job{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Job{}, err
	}

	j.Attempts++ // reflect the just-written value; SELECT returned pre-increment
	return j, nil
}

// MarkCompleted records success and finalizes the job.
func (q *Queue) MarkCompleted(ctx context.Context, audiobookID string) error {
	_, err := q.pool.Exec(ctx, `
		UPDATE metadata_enrichment_job
		SET status = 'completed', finished_at = now(), last_error = ''
		WHERE audiobook_id = $1
	`, audiobookID)
	return err
}

// MarkFailed records a failure. If attempts < MaxAttempts, reschedules with
// exponential backoff (2^attempts minutes). At MaxAttempts, marks failed.
// Uses make_interval() because Go's Duration.String() format is not valid
// Postgres interval syntax (same constraint as Cache.EvictExpired).
func (q *Queue) MarkFailed(ctx context.Context, audiobookID string, attempts int, errText string) error {
	if attempts >= MaxAttempts {
		_, err := q.pool.Exec(ctx, `
			UPDATE metadata_enrichment_job
			SET status = 'failed', finished_at = now(), last_error = $2
			WHERE audiobook_id = $1
		`, audiobookID, errText)
		return err
	}
	backoff := time.Duration(1<<uint(attempts)) * time.Minute
	_, err := q.pool.Exec(ctx, `
		UPDATE metadata_enrichment_job
		SET run_after = now() + make_interval(secs => $2), last_error = $3
		WHERE audiobook_id = $1
	`, audiobookID, backoff.Seconds(), errText)
	return err
}
