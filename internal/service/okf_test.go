package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ghassan/alms/internal/models"
)

func TestLearningExportOKF(t *testing.T) {
	t.Parallel()

	t.Run("exports accepted high-score learnings as OKF files", func(t *testing.T) {
		learning, _, _ := helperLearning(t)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		exportID, err := learning.Store(ctx, models.LearningRecord{
			Title:      "Retry payment API timeouts",
			Body:       "Retry twice with exponential backoff when the payment API times out during morning batch load.",
			Type:       models.LearningTypePattern,
			Tags:       []string{"payment", "api"},
			Author:     "agent-1",
			SrcAgentID: "agent-1",
			Score:      4.8,
		}, "")
		if err != nil {
			t.Fatalf("Store() unexpected error: %v", err)
		}
		if err := learning.UpdateEnrichment(ctx, exportID, json.RawMessage(`{"status":"accepted"}`)); err != nil {
			t.Fatalf("UpdateEnrichment() unexpected error: %v", err)
		}

		lowScoreID, err := learning.Store(ctx, models.LearningRecord{
			Title:      "Weak payment hunch",
			Body:       "Payment retry might help.",
			Type:       models.LearningTypePattern,
			Score:      3.2,
			Resolution: models.ResolutionResolved,
		}, "")
		if err != nil {
			t.Fatalf("Store() low score unexpected error: %v", err)
		}
		if err := learning.UpdateEnrichment(ctx, lowScoreID, json.RawMessage(`{"status":"accepted"}`)); err != nil {
			t.Fatalf("UpdateEnrichment() low score unexpected error: %v", err)
		}

		bundle, err := learning.ExportOKF(ctx, OKFExportOptions{Query: "payment"})
		if err != nil {
			t.Fatalf("ExportOKF() unexpected error: %v", err)
		}

		if bundle.Format != "okf_bundle" {
			t.Errorf("Format = %q, want okf_bundle", bundle.Format)
		}
		if bundle.OKFVersion != "0.1" {
			t.Errorf("OKFVersion = %q, want 0.1", bundle.OKFVersion)
		}
		if len(bundle.Files) != 2 {
			t.Fatalf("Files length = %d, want index plus one concept", len(bundle.Files))
		}
		if bundle.Summary.Matched != 2 {
			t.Errorf("Matched = %d, want 2", bundle.Summary.Matched)
		}
		if bundle.Summary.Exported != 1 {
			t.Errorf("Exported = %d, want 1", bundle.Summary.Exported)
		}
		if bundle.Summary.SkippedLow != 1 {
			t.Errorf("SkippedLow = %d, want 1", bundle.Summary.SkippedLow)
		}

		index := bundle.Files[0]
		if index.Path != "index.md" {
			t.Errorf("index Path = %q, want index.md", index.Path)
		}
		if !strings.Contains(index.Content, `okf_version: "0.1"`) {
			t.Errorf("index content missing okf_version: %s", index.Content)
		}
		if !strings.Contains(index.Content, "[Retry payment API timeouts]") {
			t.Errorf("index content missing exported learning link: %s", index.Content)
		}

		concept := bundle.Files[1]
		if !strings.HasPrefix(concept.Path, "learnings/pattern/retry-payment-api-timeouts-") {
			t.Errorf("concept Path = %q, want sanitized learning path", concept.Path)
		}
		for _, want := range []string{
			"---\n",
			"type: ALMS Pattern",
			"title: Retry payment API timeouts",
			"resource: alms://learnings/" + exportID,
			"alms_status: accepted",
			"# Lesson",
			"# ALMS Provenance",
		} {
			if !strings.Contains(concept.Content, want) {
				t.Errorf("concept content missing %q:\n%s", want, concept.Content)
			}
		}
	})

	t.Run("exports with filters and no query", func(t *testing.T) {
		learning, _, _ := helperLearning(t)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		patternID, err := learning.Store(ctx, models.LearningRecord{
			Title: "Morning deploy rollback",
			Body:  "Rollback before retrying when the deploy health check fails twice.",
			Type:  models.LearningTypePattern,
			Tags:  []string{"deploy"},
			Score: 4.6,
		}, "")
		if err != nil {
			t.Fatalf("Store() pattern unexpected error: %v", err)
		}
		if err := learning.UpdateEnrichment(ctx, patternID, json.RawMessage(`{"status":"accepted"}`)); err != nil {
			t.Fatalf("UpdateEnrichment() pattern unexpected error: %v", err)
		}

		configID, err := learning.Store(ctx, models.LearningRecord{
			Title: "Deployment config note",
			Body:  "Keep max surge at one.",
			Type:  models.LearningTypeConfig,
			Tags:  []string{"deploy"},
			Score: 4.9,
		}, "")
		if err != nil {
			t.Fatalf("Store() config unexpected error: %v", err)
		}
		if err := learning.UpdateEnrichment(ctx, configID, json.RawMessage(`{"status":"accepted"}`)); err != nil {
			t.Fatalf("UpdateEnrichment() config unexpected error: %v", err)
		}

		bundle, err := learning.ExportOKF(ctx, OKFExportOptions{
			Type: string(models.LearningTypePattern),
			Tags: []string{"deploy"},
		})
		if err != nil {
			t.Fatalf("ExportOKF() unexpected error: %v", err)
		}

		if bundle.Summary.Query != "" {
			t.Errorf("Summary.Query = %q, want empty", bundle.Summary.Query)
		}
		if bundle.Summary.Exported != 1 {
			t.Errorf("Exported = %d, want 1", bundle.Summary.Exported)
		}
		if len(bundle.Files) != 2 {
			t.Fatalf("Files length = %d, want 2", len(bundle.Files))
		}
		if !strings.Contains(bundle.Files[0].Content, "Query: not applied") {
			t.Errorf("index missing filter-only selection note:\n%s", bundle.Files[0].Content)
		}
		if !strings.Contains(bundle.Files[1].Content, "Morning deploy rollback") {
			t.Errorf("concept content missing selected learning:\n%s", bundle.Files[1].Content)
		}
	})
}
