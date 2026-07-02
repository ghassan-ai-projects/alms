package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ghassan/alms/internal/models"
)

// LearningStore provides CRUD and sync operations for learnings in PostgreSQL.
type LearningStore struct {
	pool *pgxpool.Pool
}

// NewLearningStore creates a new LearningStore backed by the given pool.
func NewLearningStore(pool *pgxpool.Pool) *LearningStore {
	return &LearningStore{pool: pool}
}

// Create inserts a new learning record and returns the generated UUID.
func (s *LearningStore) Create(ctx context.Context, record models.LearningRecord) (string, error) {
	enrichmentJSON := models.NormalizeEnrichmentMetadata(record.EnrichmentMetadata)

	query := `
		INSERT INTO learnings (type, title, body, tags, severity, author, src_agent_id, ai_generated, score, is_pinned, resolution, ttl_days, enrichment_metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING learning_id
	`

	var id string
	err := s.pool.QueryRow(ctx, query,
		string(record.Type),
		record.Title,
		record.Body,
		record.Tags,
		string(record.Severity),
		record.Author,
		nullIfEmpty(record.SrcAgentID),
		record.AIGenerated,
		record.Score,
		record.IsPinned,
		nullIfEmpty(string(record.Resolution)),
		record.TTLDays,
		enrichmentJSON,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("create learning: %w", err)
	}
	return id, nil
}

// Get retrieves a single learning record by ID.
func (s *LearningStore) Get(ctx context.Context, learningID string) (models.LearningRecord, error) {
	query := `
		SELECT learning_id, type, title, body, tags, severity, author, src_agent_id,
		       ai_generated, score, is_pinned, is_deleted, resolution, superseded_by,
		       ttl_days, created_at, deleted_at, enrichment_metadata
		FROM learnings
		WHERE learning_id = $1
	`

	var rec models.LearningRecord
	var srcAgentID, supBy *string
	var enrichmentData []byte

	err := s.pool.QueryRow(ctx, query, learningID).Scan(
		&rec.LearningID,
		&rec.Type,
		&rec.Title,
		&rec.Body,
		&rec.Tags,
		&rec.Severity,
		&rec.Author,
		&srcAgentID,
		&rec.AIGenerated,
		&rec.Score,
		&rec.IsPinned,
		&rec.IsDeleted,
		&rec.Resolution,
		&supBy,
		&rec.TTLDays,
		&rec.CreatedAt,
		&rec.DeletedAt,
		&enrichmentData,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return rec, fmt.Errorf("learning %s: %w", learningID, models.ErrNotFound)
		}
		return rec, fmt.Errorf("get learning %s: %w", learningID, err)
	}

	if srcAgentID != nil {
		rec.SrcAgentID = *srcAgentID
	}
	if supBy != nil {
		rec.SupersededBy = *supBy
	}
	if len(enrichmentData) > 0 {
		rec.EnrichmentMetadata = enrichmentData
	}

	return rec, nil
}

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

	// Insert acknowledgements (idempotent via ON CONFLICT DO NOTHING)
	for _, lid := range learningIDs {
		_, err := tx.Exec(ctx, `
			INSERT INTO learning_acknowledgements (agent_id, learning_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, agentID, lid)
		if err != nil {
			return fmt.Errorf("insert ack %s: %w", lid, err)
		}
	}

	// Advance the agent's sync cursor to the newest acknowledged learning's timestamp
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
	lastID := learningIDs[len(learningIDs)-1]
	tag, err := tx.Exec(ctx, query, agentID, lastID)
	if err != nil {
		return fmt.Errorf("update sync cursor: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("agent %s: %w", agentID, models.ErrNotFound)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit ack tx: %w", err)
	}
	return nil
}

// Search performs full-text search on learnings using PostgreSQL tsquery.
func (s *LearningStore) Search(ctx context.Context, query, ltype string, tags []string, limit int) ([]models.LearningRecord, error) {
	if limit <= 0 {
		limit = 20
	}

	args := make([]any, 0, 5)
	args = append(args, query)

	q := `
		SELECT l.learning_id, l.type, l.title, l.body, l.tags, l.severity, l.author,
		       l.src_agent_id, l.ai_generated, l.score, l.is_pinned, l.resolution,
		       l.superseded_by, l.ttl_days, l.created_at, l.enrichment_metadata
		FROM learnings l
		WHERE l.search_vector @@ plainto_tsquery('english', $1)
		  AND NOT l.is_deleted
	`
	argIdx := 2
	if ltype != "" {
		q += fmt.Sprintf(" AND l.type = $%d", argIdx)
		args = append(args, ltype)
		argIdx++
	}
	if len(tags) > 0 {
		q += fmt.Sprintf(" AND l.tags && $%d", argIdx)
		args = append(args, tags)
		argIdx++
	}
	q += fmt.Sprintf(" ORDER BY l.score DESC LIMIT $%d", argIdx)
	args = append(args, limit)

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("search learnings: %w", err)
	}
	defer rows.Close()

	return scanLearnings(rows)
}

// SoftDelete marks a learning as deleted (soft delete).
func (s *LearningStore) SoftDelete(ctx context.Context, learningID string) error {
	query := `UPDATE learnings SET is_deleted = true, deleted_at = now() WHERE learning_id = $1`
	tag, err := s.pool.Exec(ctx, query, learningID)
	if err != nil {
		return fmt.Errorf("soft delete learning %s: %w", learningID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("learning %s: %w", learningID, models.ErrNotFound)
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

// Supersede marks a learning as superseded by another.
func (s *LearningStore) Supersede(ctx context.Context, oldID, newID string) error {
	query := `UPDATE learnings SET resolution = 'superseded', superseded_by = $2 WHERE learning_id = $1`
	tag, err := s.pool.Exec(ctx, query, oldID, newID)
	if err != nil {
		return fmt.Errorf("supersede learning %s -> %s: %w", oldID, newID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("learning %s: %w", oldID, models.ErrNotFound)
	}
	return nil
}

// UpdateScore updates the score of a learning record.
func (s *LearningStore) UpdateScore(ctx context.Context, learningID string, score float64) error {
	query := `UPDATE learnings SET score = $2 WHERE learning_id = $1`
	tag, err := s.pool.Exec(ctx, query, learningID, score)
	if err != nil {
		return fmt.Errorf("update score %s: %w", learningID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("learning %s: %w", learningID, models.ErrNotFound)
	}
	return nil
}

// UpdateEnrichment merges a JSON patch into enrichment_metadata for a learning.
// Uses JSONB || operator for shallow merge. Must NOT re-trigger "pending" status.
func (s *LearningStore) UpdateEnrichment(ctx context.Context, learningID string, enrichmentJSON json.RawMessage) error {
	query := `UPDATE learnings SET enrichment_metadata = enrichment_metadata || $2::jsonb WHERE learning_id = $1`
	tag, err := s.pool.Exec(ctx, query, learningID, enrichmentJSON)
	if err != nil {
		return fmt.Errorf("update enrichment %s: %w", learningID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("learning %s: %w", learningID, models.ErrNotFound)
	}
	return nil
}

// SearchWithStatus performs the same search as Search() with additional filters.
// If status is non-empty, filters by enrichment_metadata->>'status' = status.
// If includeRejected is false (default), filters out rejected learnings.
func (s *LearningStore) SearchWithStatus(ctx context.Context, query, ltype string, tags []string, limit int, status string, includeRejected bool) ([]models.LearningRecord, error) {
	if limit <= 0 {
		limit = 20
	}

	q, args := buildSearchWithStatusQuery(query, ltype, tags, limit, status, includeRejected)

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("search learnings: %w", err)
	}
	defer rows.Close()

	return scanLearnings(rows)
}

func buildSearchWithStatusQuery(query, ltype string, tags []string, limit int, status string, includeRejected bool) (string, []any) {
	args := make([]any, 0, 6)

	q := `
		SELECT l.learning_id, l.type, l.title, l.body, l.tags, l.severity, l.author,
		       l.src_agent_id, l.ai_generated, l.score, l.is_pinned, l.resolution,
		       l.superseded_by, l.ttl_days, l.created_at, l.enrichment_metadata
		FROM learnings l
		WHERE NOT l.is_deleted
	`
	argIdx := 1

	if strings.TrimSpace(query) != "" {
		q += fmt.Sprintf(" AND l.search_vector @@ plainto_tsquery('english', $%d)", argIdx)
		args = append(args, query)
		argIdx++
	}

	if ltype != "" {
		q += fmt.Sprintf(" AND l.type = $%d", argIdx)
		args = append(args, ltype)
		argIdx++
	}
	if len(tags) > 0 {
		q += fmt.Sprintf(" AND l.tags && $%d", argIdx)
		args = append(args, tags)
		argIdx++
	}
	if status != "" {
		q += fmt.Sprintf(" AND l.enrichment_metadata->>'status' = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	if !includeRejected {
		q += ` AND (l.enrichment_metadata->>'is_visible' IS NULL OR l.enrichment_metadata->>'is_visible' != 'false')`
	}

	q += fmt.Sprintf(" ORDER BY l.score DESC LIMIT $%d", argIdx)
	args = append(args, limit)

	return q, args
}

// scanLearnings scans rows into LearningRecord slices.
func scanLearnings(rows pgx.Rows) ([]models.LearningRecord, error) {
	var records []models.LearningRecord
	for rows.Next() {
		var rec models.LearningRecord
		var srcAgentID, supBy *string
		var enrichmentData []byte
		if err := rows.Scan(
			&rec.LearningID,
			&rec.Type,
			&rec.Title,
			&rec.Body,
			&rec.Tags,
			&rec.Severity,
			&rec.Author,
			&srcAgentID,
			&rec.AIGenerated,
			&rec.Score,
			&rec.IsPinned,
			&rec.Resolution,
			&supBy,
			&rec.TTLDays,
			&rec.CreatedAt,
			&enrichmentData,
		); err != nil {
			return nil, fmt.Errorf("scan learning row: %w", err)
		}
		if srcAgentID != nil {
			rec.SrcAgentID = *srcAgentID
		}
		if supBy != nil {
			rec.SupersededBy = *supBy
		}
		if len(enrichmentData) > 0 {
			rec.EnrichmentMetadata = enrichmentData
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan learnings: %w", err)
	}
	return records, nil
}

// nullIfEmpty returns nil for empty strings (used for nullable PG columns).
func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// Ensure *pgx.Rows implements the scanLearnings scanner interface.
// keep imports
