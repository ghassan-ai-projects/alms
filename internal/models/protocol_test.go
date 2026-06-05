package models

import (
	"errors"
	"testing"
)

func TestProtocolRecordValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		pr      ProtocolRecord
		wantErr bool
		errType error
	}{
		{
			name: "valid protocol",
			pr: ProtocolRecord{
				Title: "Handshake Protocol v2",
			},
			wantErr: false,
		},
		{
			name: "valid protocol with all fields",
			pr: ProtocolRecord{
				Title:      "Data Exchange Format",
				Body:       "JSON-based exchange...",
				TargetTags: []string{"network", "data"},
				Version:    3,
				Author:     "agent-1",
				IsActive:   true,
			},
			wantErr: false,
		},
		{
			name:    "missing title",
			pr:      ProtocolRecord{},
			wantErr: true,
			errType: ErrValidation,
		},
		{
			name: "empty title whitespace",
			pr: ProtocolRecord{
				Title: "  ",
			},
			wantErr: true,
			errType: ErrValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.pr.Validate()
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
