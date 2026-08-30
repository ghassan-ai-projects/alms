package store

import (
	"context"
	"errors"
	"fmt"

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

	var record models.LearningRecord
	var srcAgentID, supersededByID *string
	var enrichmentData []byte

	err := s.pool.QueryRow(ctx, query, learningID).Scan(
		&record.LearningID,
		&record.Type,
		&record.Title,
		&record.Body,
		&record.Tags,
		&record.Severity,
		&record.Author,
		&srcAgentID,
		&record.AIGenerated,
		&record.Score,
		&record.IsPinned,
		&record.IsDeleted,
		&record.Resolution,
		&supersededByID,
		&record.TTLDays,
		&record.CreatedAt,
		&record.DeletedAt,
		&enrichmentData,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return record, fmt.Errorf("learning %s: %w", learningID, models.ErrNotFound)
		}
		return record, fmt.Errorf("get learning %s: %w", learningID, err)
	}

	populateLearningOptionalFields(&record, srcAgentID, supersededByID, enrichmentData)
	return record, nil
}
