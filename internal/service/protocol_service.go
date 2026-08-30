package service

import (
	"context"
	"fmt"

	"github.com/ghassan/alms/internal/models"
)

// ProtocolPush creates a new protocol record.
func (l *Learning) ProtocolPush(ctx context.Context, record models.ProtocolRecord) (string, error) {
	if err := record.Validate(); err != nil {
		return "", fmt.Errorf("protocol push: %w", err)
	}
	id, err := l.pStore.Create(ctx, record)
	if err != nil {
		return "", fmt.Errorf("protocol push: %w", err)
	}
	return id, nil
}

// ProtocolList returns all protocol records.
func (l *Learning) ProtocolList(ctx context.Context) ([]models.ProtocolRecord, error) {
	protocols, err := l.pStore.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list protocols: %w", err)
	}
	return protocols, nil
}
