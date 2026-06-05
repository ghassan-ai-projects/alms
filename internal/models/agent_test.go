package models

import (
	"errors"
	"strings"
	"testing"
)

func TestAgentSpecValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		spec    AgentSpec
		wantErr bool
		errType error
	}{
		{
			name: "valid systemd agent",
			spec: AgentSpec{
				AgentID:   "agent-1",
				AgentType: AgentTypeSystemd,
			},
			wantErr: false,
		},
		{
			name: "valid mcp_client agent",
			spec: AgentSpec{
				AgentID:   "agent-2",
				AgentType: AgentTypeMCPClient,
			},
			wantErr: false,
		},
		{
			name: "missing agent_id",
			spec: AgentSpec{
				AgentType: AgentTypeSystemd,
			},
			wantErr: true,
			errType: ErrValidation,
		},
		{
			name: "empty agent_id",
			spec: AgentSpec{
				AgentID:   "  ",
				AgentType: AgentTypeSystemd,
			},
			wantErr: true,
			errType: ErrValidation,
		},
		{
			name: "agent_id too long",
			spec: AgentSpec{
				AgentID:   strings.Repeat("a", 65),
				AgentType: AgentTypeSystemd,
			},
			wantErr: true,
			errType: ErrValidation,
		},
		{
			name: "agent_id at max length (64)",
			spec: AgentSpec{
				AgentID:   strings.Repeat("a", 64),
				AgentType: AgentTypeSystemd,
			},
			wantErr: false,
		},
		{
			name: "invalid agent type",
			spec: AgentSpec{
				AgentID:   "agent-3",
				AgentType: "invalid",
			},
			wantErr: true,
			errType: ErrValidation,
		},
		{
			name: "empty agent type",
			spec: AgentSpec{
				AgentID:   "agent-4",
				AgentType: AgentType(""),
			},
			wantErr: true,
			errType: ErrValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate()
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

func TestAgentSpecValidateFuzz(t *testing.T) {
	// Fuzz-like edge cases for Validate()
	t.Parallel()

	edgeCases := []AgentSpec{
		{AgentID: "a", AgentType: AgentTypeSystemd},
		{AgentID: "a", AgentType: AgentTypeMCPClient},
		{AgentID: "", AgentType: AgentType("")},
		{AgentID: "0", AgentType: "0"},
		{AgentID: "\x00", AgentType: AgentTypeSystemd},
		{AgentID: "a\nb", AgentType: AgentTypeSystemd},
	}

	for _, tc := range edgeCases {
		t.Run("edge_case/validate_nopanic", func(t *testing.T) {
			_ = tc.Validate() // Should not panic
		})
	}
}
