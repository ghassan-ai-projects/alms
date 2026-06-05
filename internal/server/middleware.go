package server

import (
	"encoding/json"
	"net/http"
)

// AuthMiddleware returns an HTTP middleware that checks the X-ALMS-TOKEN header
// against the configured token. If the token is empty (dev mode), all requests
// pass through. Otherwise, requests without a matching token receive an MCP
// JSON-RPC error response.
func AuthMiddleware(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token == "" {
				// Dev mode: no auth required
				next.ServeHTTP(w, r)
				return
			}

			got := r.Header.Get("X-ALMS-TOKEN")
			if got != token {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK) // MCP responses are always 200
				resp := map[string]any{
					"jsonrpc": "2.0",
					"error": map[string]any{
						"code":    -32001,
						"message": "unauthorized",
					},
				}
				_ = json.NewEncoder(w).Encode(resp)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
