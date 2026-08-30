package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ghassan/alms/internal/models"
)

// ProtocolStore provides CRUD and pull operations for protocols in PostgreSQL.
type ProtocolStore struct {
	pool *pgxpool.Pool
}

// NewProtocolStore creates a new ProtocolStore backed by the given pool.
func NewProtocolStore(pool *pgxpool.Pool) *ProtocolStore {
	return &ProtocolStore{pool: pool}
}

// Create inserts a new protocol record and returns the generated UUID.
func (s *ProtocolStore) Create(ctx context.Context, record models.ProtocolRecord) (string, error) {
	query := `
		INSERT INTO protocols (title, body, target_tags, version, author, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING protocol_id
	`

	var id string
	err := s.pool.QueryRow(ctx, query,
		record.Title,
		record.Body,
		record.TargetTags,
		record.Version,
		record.Author,
		record.IsActive,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("create protocol: %w", err)
	}
	return id, nil
}

// Get retrieves a single protocol record by ID.
func (s *ProtocolStore) Get(ctx context.Context, protocolID string) (models.ProtocolRecord, error) {
	query := `
		SELECT protocol_id, title, body, target_tags, version, author, is_active, created_at, updated_at
		FROM protocols
		WHERE protocol_id = $1
	`

	var rec models.ProtocolRecord
	err := s.pool.QueryRow(ctx, query, protocolID).Scan(
		&rec.ProtocolID,
		&rec.Title,
		&rec.Body,
		&rec.TargetTags,
		&rec.Version,
		&rec.Author,
		&rec.IsActive,
		&rec.CreatedAt,
		&rec.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return rec, fmt.Errorf("protocol %s: %w", protocolID, models.ErrNotFound)
		}
		return rec, fmt.Errorf("get protocol %s: %w", protocolID, err)
	}
	return rec, nil
}

// Pull returns active protocols matching the given agent tags. Returns
// protocols whose target_tags overlap with the agent's tags, or protocols with
// empty target_tags (global).
func (s *ProtocolStore) Pull(ctx context.Context, agentTags []string) ([]models.ProtocolRecord, error) {
	query := `
		SELECT protocol_id, title, body, target_tags, version, author, is_active, created_at, updated_at
		FROM protocols
		WHERE is_active
		  AND (target_tags && $1 OR target_tags = '{}')
		ORDER BY created_at DESC
	`
	return s.loadProtocols(ctx, query, agentTags)
}

// PullSince returns protocols matching the given agent tags that were created
// after the given protocol ID. If sinceID is empty, PullSince behaves like
// Pull.
func (s *ProtocolStore) PullSince(ctx context.Context, agentTags []string, sinceID string) ([]models.ProtocolRecord, error) {
	if sinceID == "" {
		return s.Pull(ctx, agentTags)
	}

	// Get the timestamp of the sinceID protocol
	var since time.Time
	err := s.pool.QueryRow(ctx, `SELECT created_at FROM protocols WHERE protocol_id = $1`, sinceID).Scan(&since)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("since protocol %s: %w", sinceID, models.ErrNotFound)
		}
		return nil, fmt.Errorf("get since protocol %s: %w", sinceID, err)
	}

	query := `
		SELECT protocol_id, title, body, target_tags, version, author, is_active, created_at, updated_at
		FROM protocols
		WHERE is_active
		  AND (target_tags && $1 OR target_tags = '{}')
		  AND created_at > $2
		ORDER BY created_at DESC
	`
	return s.loadProtocols(ctx, query, agentTags, since)
}

// List returns all protocol records ordered by creation time descending.
func (s *ProtocolStore) List(ctx context.Context) ([]models.ProtocolRecord, error) {
	query := `
		SELECT protocol_id, title, body, target_tags, version, author, is_active, created_at, updated_at
		FROM protocols
		ORDER BY created_at DESC
	`
	return s.loadProtocols(ctx, query)
}

func (s *ProtocolStore) loadProtocols(ctx context.Context, query string, args ...any) ([]models.ProtocolRecord, error) {
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query protocols: %w", err)
	}
	defer rows.Close()
	return scanProtocols(rows)
}
func scanProtocols(rows pgx.Rows) ([]models.ProtocolRecord, error) {
	var records []models.ProtocolRecord
	for rows.Next() {
		record, err := scanProtocolRow(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan protocols: %w", err)
	}
	return records, nil
}

func scanProtocolRow(rows pgx.Rows) (models.ProtocolRecord, error) {
	var record models.ProtocolRecord
	if err := rows.Scan(
		&record.ProtocolID,
		&record.Title,
		&record.Body,
		&record.TargetTags,
		&record.Version,
		&record.Author,
		&record.IsActive,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		return models.ProtocolRecord{}, fmt.Errorf("scan protocol row: %w", err)
	}
	return record, nil
}
