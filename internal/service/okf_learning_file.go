package service

import (
	"fmt"
	"path"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/ghassan/alms/internal/models"
)

func buildOKFLearningFile(record models.LearningRecord) (OKFBundleFile, error) {
	frontmatter, status := buildOKFFrontmatter(record)
	yamlData, err := yaml.Marshal(frontmatter)
	if err != nil {
		return OKFBundleFile{}, fmt.Errorf("marshal okf frontmatter: %w", err)
	}

	return OKFBundleFile{
		Path:    path.Join(okfLearningBasePath, string(record.Type), buildOKFLearningSlug(record)+".md"),
		Content: buildOKFLearningDocument(yamlData, record, status),
	}, nil
}

func buildOKFFrontmatter(record models.LearningRecord) (map[string]any, string) {
	frontmatter := map[string]any{
		"type":               formatOKFLearningType(record.Type),
		"title":              record.Title,
		"description":        buildOKFDescription(record),
		"resource":           "alms://learnings/" + record.LearningID,
		"tags":               record.Tags,
		"timestamp":          record.CreatedAt.UTC().Format(time.RFC3339),
		"alms_learning_id":   record.LearningID,
		"alms_learning_type": string(record.Type),
		"alms_score":         record.Score,
		"alms_resolution":    string(record.Resolution),
		"alms_severity":      string(record.Severity),
		"alms_author":        record.Author,
		"alms_src_agent_id":  record.SrcAgentID,
		"ai_generated":       record.AIGenerated,
	}
	if len(record.Tags) == 0 {
		delete(frontmatter, "tags")
	}
	if record.CreatedAt.IsZero() {
		delete(frontmatter, "timestamp")
	}
	if record.Severity == "" {
		delete(frontmatter, "alms_severity")
	}
	if record.Author == "" {
		delete(frontmatter, "alms_author")
	}
	if record.SrcAgentID == "" {
		delete(frontmatter, "alms_src_agent_id")
	}
	status := extractEnrichmentStatus(record.EnrichmentMetadata)
	if status != "" {
		frontmatter["alms_status"] = status
	}
	return frontmatter, status
}

func buildOKFLearningDocument(yamlData []byte, record models.LearningRecord, status string) string {
	var body strings.Builder
	body.WriteString("---\n")
	body.Write(yamlData)
	body.WriteString("---\n\n")
	body.WriteString("# Lesson\n\n")
	if strings.TrimSpace(record.Body) != "" {
		body.WriteString(strings.TrimSpace(record.Body))
		body.WriteString("\n\n")
	} else {
		body.WriteString(record.Title)
		body.WriteString("\n\n")
	}
	body.WriteString("# ALMS Provenance\n\n")
	fmt.Fprintf(&body, "- Learning ID: `%s`\n", record.LearningID)
	fmt.Fprintf(&body, "- Learning type: `%s`\n", record.Type)
	fmt.Fprintf(&body, "- Score: `%.2f`\n", record.Score)
	fmt.Fprintf(&body, "- Resolution: `%s`\n", record.Resolution)
	if record.Author != "" {
		fmt.Fprintf(&body, "- Author: `%s`\n", record.Author)
	}
	if status != "" {
		fmt.Fprintf(&body, "- Enrichment status: `%s`\n", status)
	}
	return body.String()
}
