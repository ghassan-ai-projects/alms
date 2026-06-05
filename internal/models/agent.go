// Package models defines the core data structures used across ALMS.
package models

import (
	"fmt"
	"strings"
	"time"
)

// AgentType constants represent the allowed agent types.
type AgentType string

const (
	// AgentTypeSystemd represents a systemd-managed agent.
	AgentTypeSystemd AgentType = "systemd"
	// AgentTypeMCPClient represents an MCP client agent.
	AgentTypeMCPClient AgentType = "mcp_client"
)

// ValidAgentTypes is the set of valid agent type values.
var ValidAgentTypes = map[AgentType]bool{
	AgentTypeSystemd:   true,
	AgentTypeMCPClient: true,
}

// AgentCapabilities represents the tools and skills an agent supports.
type AgentCapabilities struct {
	Tools  []string `json:"tools,omitempty"`
	Skills []string `json:"skills,omitempty"`
}

// AgentMetadata holds optional metadata about an agent.
type AgentMetadata struct {
	Owner string   `json:"owner,omitempty"`
	Tags  []string `json:"tags,omitempty"`
}

// LearningsSync tracks the agent's sync cursor.
type LearningsSync struct {
	LastSyncTimestamp time.Time `json:"last_sync_timestamp,omitempty"`
	LastSyncAt        time.Time `json:"last_sync_at,omitempty"`
}

// AgentHealth represents the current health state of an agent.
type AgentHealth struct {
	HealthScore   float64   `json:"health_score"`
	LastHeartbeat time.Time `json:"last_heartbeat,omitempty"`
}

// AgentSpec is the core agent data structure.
type AgentSpec struct {
	AgentID      string            `json:"agent_id"`
	DisplayName  string            `json:"display_name,omitempty"`
	AgentType    AgentType         `json:"agent_type"`
	Capabilities AgentCapabilities `json:"capabilities,omitempty"`
	Metadata     AgentMetadata     `json:"metadata,omitempty"`
	LearningsSync
	AgentHealth
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// Validate checks the AgentSpec for required fields and valid values.
func (a AgentSpec) Validate() error {
	var errs []string

	if strings.TrimSpace(a.AgentID) == "" {
		errs = append(errs, "agent_id is required")
	}
	if len(a.AgentID) > 64 {
		errs = append(errs, "agent_id too long: max 64 characters")
	}
	if !ValidAgentTypes[a.AgentType] {
		errs = append(errs, fmt.Sprintf("invalid agent_type: %q (must be systemd or mcp_client)", a.AgentType))
	}

	if len(errs) > 0 {
		return fmt.Errorf("%w: %s", ErrValidation, strings.Join(errs, "; "))
	}
	return nil
}
