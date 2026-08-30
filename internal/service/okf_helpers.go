package service

import (
	"encoding/json"
	"regexp"
	"strings"
	"unicode"

	"github.com/ghassan/alms/internal/models"
)

var nonSlugChars = regexp.MustCompile(`[^a-z0-9]+`)

func formatOKFLearningType(learningType models.LearningType) string {
	switch learningType {
	case models.LearningTypePattern:
		return "ALMS Pattern"
	case models.LearningTypeFailure:
		return "ALMS Failure Lesson"
	case models.LearningTypeConfig:
		return "ALMS Configuration Lesson"
	case models.LearningTypeProtocol:
		return "ALMS Protocol Lesson"
	case models.LearningTypeEdgeCase:
		return "ALMS Edge Case"
	default:
		return "ALMS Lesson"
	}
}

func buildOKFDescription(record models.LearningRecord) string {
	source := strings.TrimSpace(record.Body)
	if source == "" {
		source = record.Title
	}
	words := strings.Fields(source)
	description := strings.Join(words, " ")
	if description == "" {
		return record.Title
	}
	for i, r := range description {
		if (r == '.' || r == '!' || r == '?') && i > 0 {
			description = description[:i+1]
			break
		}
	}
	if len(description) > maxOKFDescription {
		description = strings.TrimSpace(description[:maxOKFDescription-1]) + "..."
	}
	return description
}

func buildOKFLearningSlug(record models.LearningRecord) string {
	base := strings.ToLower(record.Title)
	base = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return '-'
	}, base)
	base = strings.Trim(nonSlugChars.ReplaceAllString(base, "-"), "-")
	if base == "" {
		base = "learning"
	}
	id := strings.ToLower(strings.TrimSpace(record.LearningID))
	if id == "" {
		return base
	}
	return base + "-" + id
}

func extractEnrichmentStatus(data json.RawMessage) string {
	if len(data) == 0 {
		return ""
	}
	var meta map[string]any
	if err := json.Unmarshal(data, &meta); err != nil {
		return ""
	}
	status, _ := meta["status"].(string)
	return status
}
