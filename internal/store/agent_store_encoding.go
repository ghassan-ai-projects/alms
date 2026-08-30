package store

import (
	"encoding/json"
	"fmt"

	"github.com/ghassan/alms/internal/models"
)

func marshalAgentData(spec models.AgentSpec) ([]byte, []byte, error) {
	capBytes, err := json.Marshal(spec.Capabilities)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal capabilities: %w", err)
	}
	metaBytes, err := json.Marshal(spec.Metadata)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal metadata: %w", err)
	}
	return capBytes, metaBytes, nil
}

func decodeAgentData(capBytes, metaBytes []byte, spec *models.AgentSpec) error {
	if len(capBytes) > 0 {
		if err := json.Unmarshal(capBytes, &spec.Capabilities); err != nil {
			return fmt.Errorf("unmarshal capabilities: %w", err)
		}
	}
	if len(metaBytes) > 0 {
		if err := json.Unmarshal(metaBytes, &spec.Metadata); err != nil {
			return fmt.Errorf("unmarshal metadata: %w", err)
		}
	}
	return nil
}
