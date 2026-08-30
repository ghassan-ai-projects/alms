package service

import (
	"context"
	"fmt"
	"sort"
	"time"
)

const (
	okfVersion          = "0.1"
	defaultOKFStatus    = "accepted"
	defaultOKFMinScore  = 4.0
	defaultOKFLimit     = 50
	maxOKFDescription   = 180
	okfLearningBasePath = "learnings"
)

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

	files := []OKFBundleFile{{Path: "index.md"}}
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
