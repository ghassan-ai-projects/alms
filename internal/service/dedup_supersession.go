package service

import (
	"context"
	"fmt"

	"github.com/ghassan/alms/internal/models"
)

// HandleSupersession processes the supersedes parameter during learning storage.
// It validates the supersedes relationship and updates the superseded learning.
func (d *DedupEngine) HandleSupersession(ctx context.Context, newID, supersedesID string) error {
	if supersedesID == "" {
		return nil
	}

	// Validate the superseded learning exists
	existing, err := d.store.Get(ctx, supersedesID)
	if err != nil {
		return fmt.Errorf("supersession check: %w", err)
	}
	if existing.IsDeleted {
		return fmt.Errorf("%w: superseded learning %s is deleted", models.ErrValidation, supersedesID)
	}

	// Update the superseded learning's resolution
	if err := d.store.Supersede(ctx, supersedesID, newID); err != nil {
		return fmt.Errorf("supersession update: %w", err)
	}

	return nil
}
