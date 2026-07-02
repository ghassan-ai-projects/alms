package service

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"gopkg.in/yaml.v3"

	"github.com/ghassan/alms/internal/models"
)

const (
	okfVersion          = "0.1"
	defaultOKFStatus    = "accepted"
	defaultOKFMinScore  = 4.0
	defaultOKFLimit     = 50
	maxOKFDescription   = 180
	okfLearningBasePath = "learnings"
)

var nonSlugChars = regexp.MustCompile(`[^a-z0-9]+`)

// OKFExportOptions controls which ALMS learnings are eligible for OKF export.
type OKFExportOptions struct {
	Query           string   `json:"query,omitempty"`
	Type            string   `json:"type,omitempty"`
	Tags            []string `json:"tags,omitempty"`
	Limit           int      `json:"limit,omitempty"`
	Status          string   `json:"status,omitempty"`
	MinScore        float64  `json:"min_score,omitempty"`
	IncludeRejected bool     `json:"include_rejected,omitempty"`
}

// OKFBundle is a file-oriented OKF bundle payload. Callers can write each file
// to disk unchanged, commit it to git, or hand it to an OKF consumer directly.
type OKFBundle struct {
	Format      string           `json:"format"`
	OKFVersion  string           `json:"okf_version"`
	GeneratedAt time.Time        `json:"generated_at"`
	Files       []OKFBundleFile  `json:"files"`
	Summary     OKFExportSummary `json:"summary"`
}

// OKFBundleFile is one generated file in an OKF bundle.
type OKFBundleFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// OKFExportSummary describes the filtering decisions behind a bundle.
type OKFExportSummary struct {
	Query      string   `json:"query,omitempty"`
	Type       string   `json:"type,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	Status     string   `json:"status,omitempty"`
	MinScore   float64  `json:"min_score"`
	Matched    int      `json:"matched"`
	Exported   int      `json:"exported"`
	SkippedLow int      `json:"skipped_low_score"`
}

// ExportOKF searches ALMS learnings and emits high-confidence records as an
// OKF v0.1-compatible bundle.
func (l *Learning) ExportOKF(ctx context.Context, options OKFExportOptions) (OKFBundle, error) {
	options = normalizeOKFOptions(options)

	records, err := l.SearchAdvanced(ctx, options.Query, options.Type, options.Tags, options.Limit, options.Status, options.IncludeRejected)
	if err != nil {
		return OKFBundle{}, fmt.Errorf("export okf search: %w", err)
	}

	sort.SliceStable(records, func(i, j int) bool {
		return records[i].CreatedAt.Before(records[j].CreatedAt)
	})

	files := []OKFBundleFile{
		{
			Path: "index.md",
		},
	}
	summary := OKFExportSummary{
		Query:    options.Query,
		Type:     options.Type,
		Tags:     options.Tags,
		Status:   options.Status,
		MinScore: options.MinScore,
		Matched:  len(records),
	}

	for _, record := range records {
		if record.Score < options.MinScore {
			summary.SkippedLow++
			continue
		}
		file, err := buildOKFLearningFile(record)
		if err != nil {
			return OKFBundle{}, err
		}
		files = append(files, file)
		summary.Exported++
	}

	files[0].Content = buildOKFIndexForFiles(files[1:], options, summary)

	return OKFBundle{
		Format:      "okf_bundle",
		OKFVersion:  okfVersion,
		GeneratedAt: time.Now().UTC(),
		Files:       files,
		Summary:     summary,
	}, nil
}

func normalizeOKFOptions(options OKFExportOptions) OKFExportOptions {
	options.Query = strings.TrimSpace(options.Query)
	options.Type = strings.TrimSpace(options.Type)
	options.Status = strings.TrimSpace(options.Status)
	if options.Status == "" {
		options.Status = defaultOKFStatus
	}
	if options.Status == "all" {
		options.Status = ""
	}
	if options.Limit <= 0 {
		options.Limit = defaultOKFLimit
	}
	if options.MinScore <= 0 {
		options.MinScore = defaultOKFMinScore
	}
	return options
}

func buildOKFLearningFile(record models.LearningRecord) (OKFBundleFile, error) {
	frontmatter := map[string]any{
		"type":               okfType(record.Type),
		"title":              record.Title,
		"description":        summarizeForOKF(record),
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
	status := enrichmentStatus(record.EnrichmentMetadata)
	if status != "" {
		frontmatter["alms_status"] = status
	}

	yamlData, err := yaml.Marshal(frontmatter)
	if err != nil {
		return OKFBundleFile{}, fmt.Errorf("marshal okf frontmatter: %w", err)
	}

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

	return OKFBundleFile{
		Path:    path.Join(okfLearningBasePath, string(record.Type), learningSlug(record)+".md"),
		Content: body.String(),
	}, nil
}

func buildOKFIndexForFiles(files []OKFBundleFile, options OKFExportOptions, summary OKFExportSummary) string {
	var body strings.Builder
	body.WriteString("---\n")
	body.WriteString("okf_version: \"")
	body.WriteString(okfVersion)
	body.WriteString("\"\n")
	body.WriteString("---\n\n")
	body.WriteString("# ALMS Learning Export\n\n")
	body.WriteString("High-confidence ALMS learnings exported as OKF concept documents.\n\n")
	body.WriteString("# Selection\n\n")
	if options.Query != "" {
		fmt.Fprintf(&body, "- Query: `%s`\n", options.Query)
	} else {
		body.WriteString("- Query: not applied\n")
	}
	if options.Type != "" {
		fmt.Fprintf(&body, "- Type: `%s`\n", options.Type)
	}
	if options.Status != "" {
		fmt.Fprintf(&body, "- Status: `%s`\n", options.Status)
	}
	fmt.Fprintf(&body, "- Minimum score: `%.2f`\n", options.MinScore)
	fmt.Fprintf(&body, "- Matched: `%d`\n", summary.Matched)
	fmt.Fprintf(&body, "- Exported: `%d`\n", summary.Exported)
	body.WriteString("\n# Learnings\n\n")
	if len(files) == 0 {
		body.WriteString("No learnings met the export criteria.\n")
		return body.String()
	}
	for _, file := range files {
		title := titleFromOKFContent(file.Content)
		fmt.Fprintf(&body, "* [%s](%s) - exported ALMS learning\n", title, file.Path)
	}
	return body.String()
}

func okfType(learningType models.LearningType) string {
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

func summarizeForOKF(record models.LearningRecord) string {
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

func learningSlug(record models.LearningRecord) string {
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

func enrichmentStatus(data json.RawMessage) string {
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

func titleFromOKFContent(content string) string {
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "title: ") {
			return strings.Trim(strings.TrimPrefix(line, "title: "), "\"")
		}
	}
	return "Untitled"
}
