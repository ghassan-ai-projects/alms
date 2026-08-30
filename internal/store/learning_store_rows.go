package store

import (
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/ghassan/alms/internal/models"
)

// scanLearnings scans rows into LearningRecord slices.
func scanLearnings(rows pgx.Rows) ([]models.LearningRecord, error) {
	var records []models.LearningRecord
	for rows.Next() {
		record, err := scanLearningRow(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan learnings: %w", err)
	}
	return records, nil
}

func scanLearningRow(rows pgx.Rows) (models.LearningRecord, error) {
	var record models.LearningRecord
	var srcAgentID, supersededByID *string
	var enrichmentData []byte
	if err := rows.Scan(
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
		&record.Resolution,
		&supersededByID,
		&record.TTLDays,
		&record.CreatedAt,
		&enrichmentData,
	); err != nil {
		return models.LearningRecord{}, fmt.Errorf("scan learning row: %w", err)
	}
	populateLearningOptionalFields(&record, srcAgentID, supersededByID, enrichmentData)
	return record, nil
}

func populateLearningOptionalFields(record *models.LearningRecord, srcAgentID, supersededByID *string, enrichmentData []byte) {
	if srcAgentID != nil {
		record.SrcAgentID = *srcAgentID
	}
	if supersededByID != nil {
		record.SupersededBy = *supersededByID
	}
	if len(enrichmentData) > 0 {
		record.EnrichmentMetadata = enrichmentData
	}
}
