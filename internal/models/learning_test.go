package models

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestLearningRecordValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		rec     LearningRecord
		wantErr bool
		errType error
	}{
		{
			name: "valid pattern learning",
			rec: LearningRecord{
				Title: "Use config files for setup",
				Type:  LearningTypePattern,
			},
			wantErr: false,
		},
		{
			name: "valid failure with severity",
			rec: LearningRecord{
				Title:    "Service crashes on empty input",
				Type:     LearningTypeFailure,
				Severity: SeverityHigh,
			},
			wantErr: false,
		},
		{
			name: "valid config with resolution",
			rec: LearningRecord{
				Title:      "Set max connections to 100",
				Type:       LearningTypeConfig,
				Resolution: ResolutionResolved,
			},
			wantErr: false,
		},
		{
			name: "missing title",
			rec: LearningRecord{
				Type: LearningTypePattern,
			},
			wantErr: true,
			errType: ErrValidation,
		},
		{
			name: "empty title",
			rec: LearningRecord{
				Title: "  ",
				Type:  LearningTypeEdgeCase,
			},
			wantErr: true,
			errType: ErrValidation,
		},
		{
			name: "invalid type",
			rec: LearningRecord{
				Title: "Test",
				Type:  "unknown_type",
			},
			wantErr: true,
			errType: ErrValidation,
		},
		{
			name: "invalid severity",
			rec: LearningRecord{
				Title:    "Test",
				Type:     LearningTypePattern,
				Severity: "extreme",
			},
			wantErr: true,
			errType: ErrValidation,
		},
		{
			name: "invalid resolution",
			rec: LearningRecord{
				Title:      "Test",
				Type:       LearningTypePattern,
				Resolution: "deleted",
			},
			wantErr: true,
			errType: ErrValidation,
		},
		{
			name: "valid with all fields",
			rec: LearningRecord{
				Title:        "Full record",
				Type:         LearningTypeFailure,
				Body:         "Details here",
				Tags:         []string{"critical", "network"},
				Severity:     SeverityCritical,
				Author:       "agent-1",
				SrcAgentID:   "agent-1",
				AIGenerated:  true,
				Score:        0.95,
				IsPinned:     true,
				Resolution:   ResolutionSuperseded,
				SupersededBy: "lrn-002",
				TTLDays:      30,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rec.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error, got nil")
				} else if tt.errType != nil && !errors.Is(err, tt.errType) {
					t.Errorf("Validate() error type = %v, want %v", err, tt.errType)
				}
			} else if err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
			}
		})
	}
}

func TestNormalizeEnrichmentMetadata(t *testing.T) {
	t.Parallel()

	t.Run("empty metadata gets pending default", func(t *testing.T) {
		normalized := NormalizeEnrichmentMetadata(nil)

		var meta map[string]any
		if err := json.Unmarshal(normalized, &meta); err != nil {
			t.Fatalf("Unmarshal() unexpected error: %v", err)
		}

		if meta["status"] != "pending" {
			t.Errorf("status = %v, want pending", meta["status"])
		}

		quality, ok := meta["quality"].(map[string]any)
		if !ok {
			t.Fatal("expected quality object in default enrichment metadata")
		}
		if quality["score"] != 3.0 {
			t.Errorf("quality.score = %v, want 3.0", quality["score"])
		}
	})

	t.Run("existing metadata is preserved", func(t *testing.T) {
		original := json.RawMessage(`{"status":"accepted","quality_score":4.7}`)
		normalized := NormalizeEnrichmentMetadata(original)

		if string(normalized) != string(original) {
			t.Errorf("NormalizeEnrichmentMetadata() = %s, want %s", normalized, original)
		}
	})
}
