package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDashboardHandler(t *testing.T) {
	t.Parallel()

	t.Run("GET /dashboard returns 200", func(t *testing.T) {
		handler := DashboardHandler()

		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/dashboard", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		contentType := rec.Header().Get("Content-Type")
		if contentType != "text/html; charset=utf-8" {
			t.Errorf("expected Content-Type text/html, got %q", contentType)
		}
		body := rec.Body.String()
		if len(body) == 0 {
			t.Error("expected non-empty body")
		}
		if !containsStr(body, "ALMS Dashboard") {
			t.Error("expected body to contain 'ALMS Dashboard'")
		}
		if !containsStr(body, "refresh()") {
			t.Error("expected body to contain refresh button function")
		}
	})

	t.Run("GET /other returns 404", func(t *testing.T) {
		handler := DashboardHandler()

		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/other", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rec.Code)
		}
	})

	t.Run("POST /dashboard returns 200", func(t *testing.T) {
		handler := DashboardHandler()

		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/dashboard", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
	})
}

// containsStr reports whether s contains substr (case-sensitive).
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
