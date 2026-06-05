package models

import (
	"fmt"
	"strings"
	"time"
)

// ProtocolRecord represents a shared protocol document between agents.
type ProtocolRecord struct {
	ProtocolID string    `json:"protocol_id,omitempty"`
	Title      string    `json:"title"`
	Body       string    `json:"body,omitempty"`
	TargetTags []string  `json:"target_tags,omitempty"`
	Version    int       `json:"version,omitempty"`
	Author     string    `json:"author,omitempty"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at,omitempty"`
	UpdatedAt  *time.Time `json:"updated_at,omitempty"`
}

// Validate checks the ProtocolRecord for required fields.
func (pr ProtocolRecord) Validate() error {
	var errs []string

	if strings.TrimSpace(pr.Title) == "" {
		errs = append(errs, "title is required")
	}
	if len(errs) > 0 {
		return fmt.Errorf("%w: %s", ErrValidation, strings.Join(errs, "; "))
	}
	return nil
}
