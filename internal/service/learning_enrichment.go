package service

import (
	"encoding/json"
	"fmt"
)

// extractScoreFromEnrichment extracts "quality_score" or "score" (taking
// quality_score first) from a JSON enrichment patch. Returns an error if
// neither field is present or if the value is not a float64.
func extractScoreFromEnrichment(data json.RawMessage) (float64, error) {
	if len(data) == 0 {
		return 0, fmt.Errorf("empty enrichment data")
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return 0, fmt.Errorf("parse enrichment: %w", err)
	}

	// Try quality_score first (it's more specific)
	if v, ok := m["quality_score"]; ok {
		score, ok := v.(float64)
		if !ok {
			return 0, fmt.Errorf("quality_score is not a number")
		}
		return score, nil
	}

	// Fall back to score
	if v, ok := m["score"]; ok {
		score, ok := v.(float64)
		if !ok {
			return 0, fmt.Errorf("score is not a number")
		}
		return score, nil
	}

	return 0, fmt.Errorf("no score field in enrichment")
}
