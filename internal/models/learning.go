package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// LearningType constants represent the allowed learning categories.
type LearningType string

const (
	// LearningTypePattern represents a reusable pattern or approach.
	LearningTypePattern LearningType = "pattern"
	// LearningTypeFailure represents a failure or bug report.
	LearningTypeFailure LearningType = "failure"
	// LearningTypeConfig represents a configuration insight.
	LearningTypeConfig LearningType = "config"
	// LearningTypeProtocol represents a shared protocol.
	LearningTypeProtocol LearningType = "protocol"
	// LearningTypeEdgeCase represents an edge case discovered.
	LearningTypeEdgeCase LearningType = "edge_case"
)

// ValidLearningTypes is the set of valid learning type values.
var ValidLearningTypes = map[LearningType]bool{
	LearningTypePattern:  true,
	LearningTypeFailure:  true,
	LearningTypeConfig:   true,
	LearningTypeProtocol: true,
	LearningTypeEdgeCase: true,
}

// Severity constants.
type Severity string

const (
	// SeverityLow represents low-severity learnings.
	SeverityLow Severity = "low"
	// SeverityMedium represents medium-severity learnings.
	SeverityMedium Severity = "medium"
	// SeverityHigh represents high-severity learnings.
	SeverityHigh Severity = "high"
	// SeverityCritical represents critical learnings.
	SeverityCritical Severity = "critical"
)

// ValidSeverities is the set of valid severity values.
var ValidSeverities = map[Severity]bool{
	SeverityLow:      true,
	SeverityMedium:   true,
	SeverityHigh:     true,
	SeverityCritical: true,
}

// Resolution constants.
type Resolution string

const (
	// ResolutionOpen represents an open/unresolved learning.
	ResolutionOpen Resolution = "open"
	// ResolutionResolved represents a resolved learning.
	ResolutionResolved Resolution = "resolved"
	// ResolutionSuperseded represents a superseded learning.
	ResolutionSuperseded Resolution = "superseded"
)

// ValidResolutions is the set of valid resolution values.
var ValidResolutions = map[Resolution]bool{
	ResolutionOpen:       true,
	ResolutionResolved:   true,
	ResolutionSuperseded: true,
}

// LearningRecord represents a single learning entry shared between agents.
type LearningRecord struct {
	LearningID  string       `json:"learning_id,omitempty"`
	Type        LearningType `json:"type"`
	Title       string       `json:"title"`
	Body        string       `json:"body,omitempty"`
	Tags        []string     `json:"tags,omitempty"`
	Severity    Severity     `json:"severity,omitempty"`
	Author      string       `json:"author,omitempty"`
	SrcAgentID  string       `json:"src_agent_id,omitempty"`
	AIGenerated bool         `json:"ai_generated"`
	Score       float64      `json:"score,omitempty"`
	IsPinned    bool         `json:"is_pinned"`
	IsDeleted   bool         `json:"is_deleted,omitempty"`
	Resolution  Resolution   `json:"resolution,omitempty"`
	SupersededBy      string           `json:"superseded_by,omitempty"`
	TTLDays           int              `json:"ttl_days,omitempty"`
	CreatedAt         time.Time        `json:"created_at,omitempty"`
	DeletedAt         *time.Time       `json:"deleted_at,omitempty"`
	EnrichmentMetadata json.RawMessage `json:"enrichment_metadata,omitempty"`
}

// Validate checks the LearningRecord for required fields and valid values.
func (lr LearningRecord) Validate() error {
	var errs []string

	if strings.TrimSpace(lr.Title) == "" {
		errs = append(errs, "title is required")
	}
	if !ValidLearningTypes[lr.Type] {
		errs = append(errs, fmt.Sprintf("invalid type: %q", lr.Type))
	}
	if lr.Severity != "" && !ValidSeverities[lr.Severity] {
		errs = append(errs, fmt.Sprintf("invalid severity: %q", lr.Severity))
	}
	if lr.Resolution != "" && !ValidResolutions[lr.Resolution] {
		errs = append(errs, fmt.Sprintf("invalid resolution: %q", lr.Resolution))
	}

	if len(errs) > 0 {
		return fmt.Errorf("%w: %s", ErrValidation, strings.Join(errs, "; "))
	}
	return nil
}
