package store

import (
	"strings"
	"testing"
)

func TestNullIfEmpty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  any
	}{
		{
			name:  "empty string returns nil",
			input: "",
			want:  nil,
		},
		{
			name:  "non-empty string returns original string",
			input: "agent-1",
			want:  "agent-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nullIfEmpty(tt.input)
			if got != tt.want {
				t.Errorf("nullIfEmpty(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestLearningStoreSkipNoPG(t *testing.T) {
	t.Skip("PostgreSQL required — skipping learning store tests")
}

func TestLearningStoreCRUD(t *testing.T) {
	t.Skip("PostgreSQL required — skipping learning store CRUD tests")
}

func TestLearningStoreSyncAckValidation(t *testing.T) {
	t.Skip("PostgreSQL required — skipping learning store sync ack tests")
}

func TestLearningStoreSearch(t *testing.T) {
	t.Skip("PostgreSQL required — skipping learning store search tests")
}

func TestBuildSearchWithStatusQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		query              string
		ltype              string
		tags               []string
		status             string
		includeRejected    bool
		wantTextSearch     bool
		wantVisibilityGate bool
		wantArgs           int
	}{
		{
			name:               "empty query builds filter-only search",
			query:              "",
			ltype:              "pattern",
			tags:               []string{"deploy"},
			status:             "accepted",
			wantTextSearch:     false,
			wantVisibilityGate: true,
			wantArgs:           4,
		},
		{
			name:               "non-empty query includes text search",
			query:              "payment timeout",
			status:             "accepted",
			wantTextSearch:     true,
			wantVisibilityGate: true,
			wantArgs:           3,
		},
		{
			name:               "include rejected skips visibility gate",
			query:              "payment timeout",
			includeRejected:    true,
			wantTextSearch:     true,
			wantVisibilityGate: false,
			wantArgs:           2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args := buildSearchWithStatusQuery(tt.query, tt.ltype, tt.tags, 50, tt.status, tt.includeRejected)

			hasTextSearch := strings.Contains(sql, "plainto_tsquery")
			if hasTextSearch != tt.wantTextSearch {
				t.Errorf("text search presence = %v, want %v\nSQL:\n%s", hasTextSearch, tt.wantTextSearch, sql)
			}
			hasVisibilityGate := strings.Contains(sql, "is_visible")
			if hasVisibilityGate != tt.wantVisibilityGate {
				t.Errorf("visibility gate presence = %v, want %v\nSQL:\n%s", hasVisibilityGate, tt.wantVisibilityGate, sql)
			}
			if len(args) != tt.wantArgs {
				t.Errorf("len(args) = %d, want %d; args=%v", len(args), tt.wantArgs, args)
			}
		})
	}
}
