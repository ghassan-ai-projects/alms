package server

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/server"

	"github.com/ghassan/alms/internal/models"
	"github.com/ghassan/alms/internal/service"
	"github.com/ghassan/alms/internal/service/storemock"
)

// helperServerWithResources creates a test server with all resources registered
// using the real resource registration code.
func helperServerWithResources(t *testing.T) (*server.MCPServer, *service.Registry, *service.Learning) {
	t.Helper()

	mcpSrv := server.NewMCPServer("ALMS-Test-Resources", "0.0.0")
	aStore := storemock.NewAgentStore()
	lStore := storemock.NewLearningStore()
	pStore := storemock.NewProtocolStore()

	registry := service.NewRegistry(aStore)
	syncer := service.NewSyncer(lStore, aStore, pStore)
	learningSvc := service.NewLearning(lStore, pStore)

	// Register all tools (needed for some resources)
	registerAgentTools(mcpSrv, registry)
	registerLearningTools(mcpSrv, syncer)
	registerLearningStoreTools(mcpSrv, learningSvc)
	registerProtocolTools(mcpSrv, syncer)
	registerProtocolStoreTools(mcpSrv, learningSvc)
	registerHealthTools(mcpSrv, registry)

	// Register resources using the real registration code
	s := &Server{mcp: mcpSrv}
	s.registerResources(registry, learningSvc)

	return mcpSrv, registry, learningSvc
}

// readResource sends a resources/read request to the MCP server and returns
// the raw response text (or fails the test).
func readResource(t *testing.T, srv *server.MCPServer, uri string) string {
	t.Helper()

	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "resources/read",
		"params": map[string]any{
			"uri": uri,
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	resp := srv.HandleMessage(context.Background(), raw)

	// The response is a JSONRPCResponse — marshal it back to JSON and extract contents
	respRaw, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	var respJSON map[string]any
	if err := json.Unmarshal(respRaw, &respJSON); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Check for error
	if errDetails, hasError := respJSON["error"]; hasError {
		t.Fatalf("resource read returned error: %v", errDetails)
	}

	resultRaw, ok := respJSON["result"]
	if !ok {
		t.Fatal("response has no result")
	}

	resultJSON, err := json.Marshal(resultRaw)
	if err != nil {
		t.Fatalf("failed to marshal result: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	contentsRaw, ok := result["contents"]
	if !ok {
		t.Fatal("result has no contents")
	}

	contentsJSON, err := json.Marshal(contentsRaw)
	if err != nil {
		t.Fatalf("failed to marshal contents: %v", err)
	}

	var contents []map[string]any
	if err := json.Unmarshal(contentsJSON, &contents); err != nil {
		t.Fatalf("failed to unmarshal contents: %v", err)
	}

	if len(contents) == 0 {
		t.Fatal("no contents in result")
	}

	text, ok := contents[0]["text"].(string)
	if !ok {
		t.Fatal("contents[0].text is not a string")
	}

	return text
}

func TestResources(t *testing.T) {
	t.Parallel()

	t.Run("alms://learnings returns json", func(t *testing.T) {
		srv, _, learningSvc := helperServerWithResources(t)
		ctx := context.Background()

		_, _ = learningSvc.Store(ctx, models.LearningRecord{
			Title: "Test",
			Type:  models.LearningTypeConfig,
		}, "")

		text := readResource(t, srv, "alms://learnings")

		var records []models.LearningRecord
		if err := json.Unmarshal([]byte(text), &records); err != nil {
			t.Fatalf("failed to unmarshal learnings: %v", err)
		}
		if len(records) == 0 {
			t.Error("expected at least 1 learning")
		}
	})

	t.Run("alms://tools returns json", func(t *testing.T) {
		srv, _, _ := helperServerWithResources(t)

		text := readResource(t, srv, "alms://tools")

		var tools []map[string]any
		if err := json.Unmarshal([]byte(text), &tools); err != nil {
			t.Fatalf("failed to unmarshal tools: %v", err)
		}
		if len(tools) == 0 {
			t.Error("expected at least 1 tool")
		}
		// Verify it has all 16 Phase 2 tools (includes learning.get)
		if len(tools) != 16 {
			t.Errorf("expected 16 tools, got %d", len(tools))
		}
	})

	t.Run("alms://protocols returns json", func(t *testing.T) {
		srv, _, learningSvc := helperServerWithResources(t)
		ctx := context.Background()

		_, _ = learningSvc.ProtocolPush(ctx, models.ProtocolRecord{
			Title: "Test Protocol",
		})

		text := readResource(t, srv, "alms://protocols")

		var protocols []models.ProtocolRecord
		if err := json.Unmarshal([]byte(text), &protocols); err != nil {
			t.Fatalf("failed to unmarshal protocols: %v", err)
		}
		if len(protocols) == 0 {
			t.Error("expected at least 1 protocol")
		}
	})

	t.Run("alms://agents returns json", func(t *testing.T) {
		srv, _, _ := helperServerWithResources(t)

		text := readResource(t, srv, "alms://agents")

		var agents []models.AgentSpec
		if err := json.Unmarshal([]byte(text), &agents); err != nil {
			t.Fatalf("failed to unmarshal agents: %v", err)
		}
	})

	t.Run("alms://health returns json", func(t *testing.T) {
		srv, _, _ := helperServerWithResources(t)

		text := readResource(t, srv, "alms://health")

		var health map[string]any
		if err := json.Unmarshal([]byte(text), &health); err != nil {
			t.Fatalf("failed to unmarshal health: %v", err)
		}
		if health["status"] != "ok" {
			t.Errorf("status = %q, want %q", health["status"], "ok")
		}
	})
}
