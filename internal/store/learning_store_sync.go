package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ghassan/alms/internal/models"
)

// Sync returns learnings created after the given timestamp, optionally filtered
// by type and tags. Only non-deleted records are returned.
func (s *LearningStore) Sync(ctx context.Context, agentID string, since time.Time, ltype string, tags []string) ([]models.LearningRecord, error) {
	args := make([]any, 0, 4)
	args = append(args, since)

	query := `
		SELECT l.learning_id, l.type, l.title, l.body, l.tags, l.severity, l.author,
		       l.src_agent_id, l.ai_generated, l.score, l.is_pinned, l.resolution,
		       l.superseded_by, l.ttl_days, l.created_at, l.enrichment_metadata
		FROM learnings l
		WHERE l.created_at > $1
		  AND NOT l.is_deleted
	`

	argIdx := 2
	if ltype != "" {
		query += fmt.Sprintf(" AND l.type = $%d", argIdx)
		args = append(args, ltype)
		argIdx++
	}
	if len(tags) > 0 {
		query += fmt.Sprintf(" AND l.tags && $%d", argIdx)
		args = append(args, tags)
	}
	query += " ORDER BY l.created_at ASC"

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("sync learnings for %s: %w", agentID, err)
	}
	defer rows.Close()

	return scanLearnings(rows)
}

// SyncAck inserts acknowledgements for the given learning IDs and advances the
// agent's sync cursor. The caller MUST validate gaps before calling this.
func (s *LearningStore) SyncAck(ctx context.Context, agentID string, learningIDs []string) error {
	if len(learningIDs) == 0 {
		return nil
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	lastID := learningIDs[len(learningIDs)-1]
	if err := insertLearningAcknowledgements(ctx, tx, agentID, learningIDs); err != nil {
		return err
	}
	if err := advanceAgentSyncCursor(ctx, tx, agentID, lastID); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit ack tx: %w", err)
	}
	return nil
}

func insertLearningAcknowledgements(ctx context.Context, tx pgx.Tx, agentID string, learningIDs []string) error {
	// Insert acknowledgements (idempotent via ON CONFLICT DO NOTHING)
	for _, learningID := range learningIDs {
		_, err := tx.Exec(ctx, `
			INSERT INTO learning_acknowledgements (agent_id, learning_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, agentID, learningID)
		if err != nil {
			return fmt.Errorf("insert ack %s: %w", learningID, err)
		}
	}
	return nil
}

func advanceAgentSyncCursor(ctx context.Context, tx pgx.Tx, agentID, learningID string) error {
	// Advance the agent's sync cursor to the newest acknowledged learning's timestamp.
	query := `
		UPDATE agents a
		SET last_sync_ts = (
			SELECT l.created_at FROM learnings l
			WHERE l.learning_id = $2
		),
		last_sync_at = now(),
		updated_at = now()
		WHERE a.agent_id = $1
	`
	tag, err := tx.Exec(ctx, query, agentID, learningID)
	if err != nil {
		return fmt.Errorf("update sync cursor: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("agent %s: %w", agentID, models.ErrNotFound)
	}
	return nil
}

// ExpectedSyncIDs returns the ordered list of learning IDs the agent should
// have received since the given timestamp. Used for gap-safe ack validation.
func (s *LearningStore) ExpectedSyncIDs(ctx context.Context, agentID string, since time.Time) ([]string, error) {
	query := `
		SELECT l.learning_id FROM learnings l
		WHERE l.created_at > $1
		  AND NOT l.is_deleted
		ORDER BY l.created_at ASC
	`
	rows, err := s.pool.Query(ctx, query, since)
	if err != nil {
		return nil, fmt.Errorf("expected sync IDs for %s: %w", agentID, err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan expected ID: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("expected sync IDs: %w", err)
	}
	return ids, nil
}
