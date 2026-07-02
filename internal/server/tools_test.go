package server

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/ghassan/alms/internal/models"
	"github.com/ghassan/alms/internal/service"
	"github.com/ghassan/alms/internal/service/storemock"
)

// helperServer creates a minimal test server with mock stores.
func helperServer(t *testing.T) (*server.MCPServer, *service.Registry, *service.Syncer, *service.Learning) {
	t.Helper()

	mcpSrv := server.NewMCPServer("ALMS-Test", "0.0.0")
	aStore := storemock.NewAgentStore()
	lStore := storemock.NewLearningStore()
	pStore := storemock.NewProtocolStore()

	registry := service.NewRegistry(aStore)
	syncer := service.NewSyncer(lStore, aStore, pStore)
	learningSvc := service.NewLearning(lStore, pStore)

	// Register all tools directly on the MCP server
	registerAgentTools(mcpSrv, registry)
	registerLearningTools(mcpSrv, syncer)
	registerLearningStoreTools(mcpSrv, learningSvc)
	registerProtocolTools(mcpSrv, syncer)
	registerProtocolStoreTools(mcpSrv, learningSvc)
	registerHealthTools(mcpSrv, registry)
	registerPhase2Tools(mcpSrv, learningSvc)
	registerOKFTools(mcpSrv, learningSvc)

	return mcpSrv, registry, syncer, learningSvc
}

// callTool sends a tools/call JSON-RPC request to the MCP server and returns
// the response.
func callTool(t *testing.T, srv *server.MCPServer, toolName string, args map[string]any) mcp.JSONRPCMessage {
	t.Helper()

	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      toolName,
			"arguments": args,
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}
	return srv.HandleMessage(context.Background(), raw)
}

// requireToolResultError checks that the response contains an error in the
// CallToolResult (IsError = true).
func requireToolResultError(t *testing.T, resp mcp.JSONRPCMessage) {
	t.Helper()

	jr, ok := resp.(mcp.JSONRPCResponse)
	if !ok {
		t.Fatalf("expected JSONRPCResponse, got %T", resp)
	}
	result, ok := jr.Result.(*mcp.CallToolResult)
	if !ok {
		t.Fatalf("expected *CallToolResult, got %T", jr.Result)
	}
	if !result.IsError {
		t.Fatalf("expected IsError=true, got IsError=false")
	}
}

// getToolResultText extracts the text from a successful CallToolResult.
func getToolResultText(t *testing.T, resp mcp.JSONRPCMessage) string {
	t.Helper()

	jr, ok := resp.(mcp.JSONRPCResponse)
	if !ok {
		t.Fatalf("expected JSONRPCResponse, got %T", resp)
	}
	result, ok := jr.Result.(*mcp.CallToolResult)
	if !ok {
		t.Fatalf("expected *CallToolResult, got %T", jr.Result)
	}
	if result.IsError {
		t.Fatalf("expected IsError=false, got IsError=true")
	}
	if len(result.Content) == 0 {
		return ""
	}
	tc, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	return tc.Text
}

func TestAgentRegisterTool(t *testing.T) {
	t.Parallel()

	t.Run("register systemd agent succeeds", func(t *testing.T) {
		srv, _, _, _ := helperServer(t)

		resp := callTool(t, srv, "agent.register", map[string]any{
			"agent_id":   "test-agent-1",
			"agent_type": "systemd",
		})

		text := getToolResultText(t, resp)
		if text == "" {
			t.Fatal("expected non-empty result text")
		}

		var result models.AgentSpec
		if err := json.Unmarshal([]byte(text), &result); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}
		if result.AgentID != "test-agent-1" {
			t.Errorf("AgentID = %q, want %q", result.AgentID, "test-agent-1")
		}
		if result.AgentType != models.AgentTypeSystemd {
			t.Errorf("AgentType = %q, want %q", result.AgentType, models.AgentTypeSystemd)
		}
	})

	t.Run("register mcp_client agent succeeds", func(t *testing.T) {
		srv, _, _, _ := helperServer(t)

		resp := callTool(t, srv, "agent.register", map[string]any{
			"agent_id":   "test-agent-2",
			"agent_type": "mcp_client",
		})

		text := getToolResultText(t, resp)

		var result models.AgentSpec
		if err := json.Unmarshal([]byte(text), &result); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}
		if result.AgentType != models.AgentTypeMCPClient {
			t.Errorf("AgentType = %q, want %q", result.AgentType, models.AgentTypeMCPClient)
		}
	})

	t.Run("register with display_name", func(t *testing.T) {
		srv, _, _, _ := helperServer(t)

		resp := callTool(t, srv, "agent.register", map[string]any{
			"agent_id":     "agent-named",
			"agent_type":   "systemd",
			"display_name": "My Agent",
		})

		text := getToolResultText(t, resp)

		var result models.AgentSpec
		if err := json.Unmarshal([]byte(text), &result); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}
		if result.DisplayName != "My Agent" {
			t.Errorf("DisplayName = %q, want %q", result.DisplayName, "My Agent")
		}
	})

	t.Run("register duplicate returns error", func(t *testing.T) {
		srv, _, _, _ := helperServer(t)

		// First registration succeeds
		callTool(t, srv, "agent.register", map[string]any{
			"agent_id":   "dup-agent",
			"agent_type": "systemd",
		})

		// Second registration should fail
		resp := callTool(t, srv, "agent.register", map[string]any{
			"agent_id":   "dup-agent",
			"agent_type": "systemd",
		})

		requireToolResultError(t, resp)
	})

	t.Run("register with invalid type returns error", func(t *testing.T) {
		srv, _, _, _ := helperServer(t)

		resp := callTool(t, srv, "agent.register", map[string]any{
			"agent_id":   "bad-agent",
			"agent_type": "invalid",
		})

		requireToolResultError(t, resp)
	})

	t.Run("register with empty agent_id returns error", func(t *testing.T) {
		srv, _, _, _ := helperServer(t)

		resp := callTool(t, srv, "agent.register", map[string]any{
			"agent_id":   "",
			"agent_type": "systemd",
		})

		requireToolResultError(t, resp)
	})
}

func TestLearningStoreTool(t *testing.T) {
	t.Parallel()

	t.Run("store learning succeeds", func(t *testing.T) {
		srv, _, _, _ := helperServer(t)

		resp := callTool(t, srv, "learning.store", map[string]any{
			"agent_id": "agent-1",
			"title":    "Test Learning",
			"body":     "Test body",
			"type":     "config",
			"tags":     []any{"database"},
		})

		text := getToolResultText(t, resp)
		var result map[string]any
		if err := json.Unmarshal([]byte(text), &result); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}
		if result["status"] != "created" {
			t.Errorf("status = %q, want %q", result["status"], "created")
		}
		if id, ok := result["learning_id"].(string); !ok || id == "" {
			t.Error("expected non-empty learning_id")
		}
		enrichment, ok := result["enrichment"].(map[string]any)
		if !ok {
			t.Fatal("expected enrichment object in result")
		}
		if enrichment["status"] != "pending" {
			t.Errorf("enrichment.status = %v, want pending", enrichment["status"])
		}
	})

	t.Run("store without agent_id returns error", func(t *testing.T) {
		srv, _, _, _ := helperServer(t)

		resp := callTool(t, srv, "learning.store", map[string]any{
			"title": "Test",
			"type":  "config",
		})

		requireToolResultError(t, resp)
	})

	t.Run("store with invalid type returns error", func(t *testing.T) {
		srv, _, _, _ := helperServer(t)

		resp := callTool(t, srv, "learning.store", map[string]any{
			"agent_id": "agent-1",
			"title":    "Test",
			"type":     "invalid_type",
		})

		requireToolResultError(t, resp)
	})
}

func TestLearningSearchTool(t *testing.T) {
	t.Parallel()

	t.Run("search returns results", func(t *testing.T) {
		srv, _, _, learningSvc := helperServer(t)
		ctx := context.Background()

		// Pre-populate a learning via the service
		_, _ = learningSvc.Store(ctx, models.LearningRecord{
			Title: "Search Test",
			Body:  "This should be searchable",
			Type:  models.LearningTypeConfig,
		}, "")

		resp := callTool(t, srv, "learning.search", map[string]any{
			"query": "Search",
		})

		text := getToolResultText(t, resp)
		var results []models.LearningRecord
		if err := json.Unmarshal([]byte(text), &results); err != nil {
			t.Fatalf("failed to unmarshal results: %v", err)
		}
		if len(results) == 0 {
			t.Error("expected at least 1 result")
		}
	})

	t.Run("search without query returns error", func(t *testing.T) {
		srv, _, _, _ := helperServer(t)

		resp := callTool(t, srv, "learning.search", map[string]any{
			"query": "",
		})

		requireToolResultError(t, resp)
	})
}

func TestLearningDeleteTool(t *testing.T) {
	t.Parallel()

	t.Run("delete existing learning", func(t *testing.T) {
		srv, _, _, learningSvc := helperServer(t)
		ctx := context.Background()

		id, _ := learningSvc.Store(ctx, models.LearningRecord{
			Title: "To Delete",
			Type:  models.LearningTypeConfig,
		}, "")

		resp := callTool(t, srv, "learning.delete", map[string]any{
			"learning_id": id,
		})

		text := getToolResultText(t, resp)
		var result map[string]any
		if err := json.Unmarshal([]byte(text), &result); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}
		if result["status"] != "soft_deleted" {
			t.Errorf("status = %q, want %q", result["status"], "soft_deleted")
		}
	})

	t.Run("delete without id returns error", func(t *testing.T) {
		srv, _, _, _ := helperServer(t)

		resp := callTool(t, srv, "learning.delete", map[string]any{
			"learning_id": "",
		})

		requireToolResultError(t, resp)
	})
}

func TestLearningGetTool(t *testing.T) {
	t.Parallel()

	t.Run("get existing learning", func(t *testing.T) {
		srv, _, _, learningSvc := helperServer(t)
		ctx := context.Background()

		id, _ := learningSvc.Store(ctx, models.LearningRecord{
			Title: "To Get",
			Type:  models.LearningTypeConfig,
		}, "")

		resp := callTool(t, srv, "learning.get", map[string]any{
			"learning_id": id,
		})

		text := getToolResultText(t, resp)
		var result models.LearningRecord
		if err := json.Unmarshal([]byte(text), &result); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}
		if result.Title != "To Get" {
			t.Errorf("Title = %q, want %q", result.Title, "To Get")
		}
	})

	t.Run("get without id returns error", func(t *testing.T) {
		srv, _, _, _ := helperServer(t)

		resp := callTool(t, srv, "learning.get", map[string]any{
			"learning_id": "",
		})

		requireToolResultError(t, resp)
	})
}

func TestProtocolPushTool(t *testing.T) {
	t.Parallel()

	t.Run("push protocol succeeds", func(t *testing.T) {
		srv, _, _, _ := helperServer(t)

		resp := callTool(t, srv, "protocol.push", map[string]any{
			"title":       "New Protocol",
			"body":        "Protocol body",
			"target_tags": []any{"agent-1"},
		})

		text := getToolResultText(t, resp)
		var result map[string]any
		if err := json.Unmarshal([]byte(text), &result); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}
		if result["status"] != "created" {
			t.Errorf("status = %q, want %q", result["status"], "created")
		}
	})

	t.Run("push without title returns error", func(t *testing.T) {
		srv, _, _, _ := helperServer(t)

		resp := callTool(t, srv, "protocol.push", map[string]any{
			"title": "",
		})

		requireToolResultError(t, resp)
	})
}

func TestProtocolListTool(t *testing.T) {
	t.Parallel()

	t.Run("list protocols", func(t *testing.T) {
		srv, _, _, learningSvc := helperServer(t)
		ctx := context.Background()

		_, _ = learningSvc.ProtocolPush(ctx, models.ProtocolRecord{Title: "Proto 1"})
		_, _ = learningSvc.ProtocolPush(ctx, models.ProtocolRecord{Title: "Proto 2"})

		resp := callTool(t, srv, "protocol.list", map[string]any{})

		text := getToolResultText(t, resp)
		var results []models.ProtocolRecord
		if err := json.Unmarshal([]byte(text), &results); err != nil {
			t.Fatalf("failed to unmarshal results: %v", err)
		}
		if len(results) != 2 {
			t.Errorf("got %d protocols, want 2", len(results))
		}
	})
}

func TestLearningSearchWithStatusTool(t *testing.T) {
	t.Parallel()

	t.Run("search defaults to pending status only", func(t *testing.T) {
		srv, _, _, learningSvc := helperServer(t)
		ctx := context.Background()

		_, _ = learningSvc.Store(ctx, models.LearningRecord{
			Title: "Queue Candidate",
			Body:  "Shared query text",
			Type:  models.LearningTypeConfig,
		}, "")
		acceptedID, _ := learningSvc.Store(ctx, models.LearningRecord{
			Title: "Already Scored",
			Body:  "Shared query text",
			Type:  models.LearningTypeConfig,
		}, "")
		_ = learningSvc.UpdateEnrichment(ctx, acceptedID, json.RawMessage(`{"status":"accepted"}`))

		resp := callTool(t, srv, "learning.search", map[string]any{
			"query": "Shared",
		})

		text := getToolResultText(t, resp)
		var results []models.LearningRecord
		if err := json.Unmarshal([]byte(text), &results); err != nil {
			t.Fatalf("failed to unmarshal results: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 pending result, got %d", len(results))
		}
		if results[0].Title != "Queue Candidate" {
			t.Errorf("Title = %q, want %q", results[0].Title, "Queue Candidate")
		}
	})

	t.Run("search with include_enrichment flag", func(t *testing.T) {
		srv, _, _, learningSvc := helperServer(t)
		ctx := context.Background()

		id, _ := learningSvc.Store(ctx, models.LearningRecord{
			Title: "Enrich Test",
			Body:  "Check enrichment metadata",
			Type:  models.LearningTypeConfig,
		}, "")

		// Set enrichment via the UpdateEnrichment service
		patch := json.RawMessage(`{"status":"accepted"}`)
		_ = learningSvc.UpdateEnrichment(ctx, id, patch)

		// Search with include_enrichment=true
		resp := callTool(t, srv, "learning.search", map[string]any{
			"query":              "Enrich",
			"include_enrichment": true,
			"status":             "accepted",
		})

		text := getToolResultText(t, resp)
		var results []models.LearningRecord
		if err := json.Unmarshal([]byte(text), &results); err != nil {
			t.Fatalf("failed to unmarshal results: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("expected at least 1 result")
		}
		// With include_enrichment=true, enrichment metadata should be present
		if results[0].EnrichmentMetadata == nil {
			t.Error("expected enrichment metadata in result with include_enrichment=true")
		}
	})

	t.Run("search with include_enrichment=false hides enrichment", func(t *testing.T) {
		srv, _, _, learningSvc := helperServer(t)
		ctx := context.Background()

		_, _ = learningSvc.Store(ctx, models.LearningRecord{
			Title: "Hidden Enrich",
			Body:  "Enrichment hidden",
			Type:  models.LearningTypeConfig,
		}, "")

		resp := callTool(t, srv, "learning.search", map[string]any{
			"query":              "Hidden",
			"include_enrichment": false,
		})

		text := getToolResultText(t, resp)
		var results []models.LearningRecord
		if err := json.Unmarshal([]byte(text), &results); err != nil {
			t.Fatalf("failed to unmarshal results: %v", err)
		}
		if len(results) > 0 && results[0].EnrichmentMetadata != nil {
			t.Error("expected enrichment metadata to be nil when include_enrichment=false")
		}
	})

	t.Run("search with explicit accepted status returns accepted learnings", func(t *testing.T) {
		srv, _, _, learningSvc := helperServer(t)
		ctx := context.Background()

		pendingID, _ := learningSvc.Store(ctx, models.LearningRecord{
			Title: "Pending Status Filter Test",
			Body:  "Status filter",
			Type:  models.LearningTypeConfig,
		}, "")
		acceptedID, _ := learningSvc.Store(ctx, models.LearningRecord{
			Title: "Accepted Status Filter Test",
			Body:  "Status filter",
			Type:  models.LearningTypeConfig,
		}, "")
		_ = pendingID
		_ = learningSvc.UpdateEnrichment(ctx, acceptedID, json.RawMessage(`{"status":"accepted"}`))

		resp := callTool(t, srv, "learning.search", map[string]any{
			"query":  "Status",
			"status": "accepted",
		})

		text := getToolResultText(t, resp)
		var results []models.LearningRecord
		if err := json.Unmarshal([]byte(text), &results); err != nil {
			t.Fatalf("failed to unmarshal results: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 accepted result, got %d", len(results))
		}
		if results[0].Title != "Accepted Status Filter Test" {
			t.Errorf("Title = %q, want %q", results[0].Title, "Accepted Status Filter Test")
		}
	})

	t.Run("search with status all disables default pending filter", func(t *testing.T) {
		srv, _, _, learningSvc := helperServer(t)
		ctx := context.Background()

		_, _ = learningSvc.Store(ctx, models.LearningRecord{
			Title: "Pending All Filter Test",
			Body:  "Status all filter",
			Type:  models.LearningTypeConfig,
		}, "")
		acceptedID, _ := learningSvc.Store(ctx, models.LearningRecord{
			Title: "Accepted All Filter Test",
			Body:  "Status all filter",
			Type:  models.LearningTypeConfig,
		}, "")
		_ = learningSvc.UpdateEnrichment(ctx, acceptedID, json.RawMessage(`{"status":"accepted"}`))

		resp := callTool(t, srv, "learning.search", map[string]any{
			"query":  "Status all",
			"status": "all",
		})

		text := getToolResultText(t, resp)
		var results []models.LearningRecord
		if err := json.Unmarshal([]byte(text), &results); err != nil {
			t.Fatalf("failed to unmarshal results: %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 results with status=all, got %d", len(results))
		}
	})
}

func TestUpdateEnrichmentTool(t *testing.T) {
	t.Parallel()

	t.Run("update enrichment succeeds", func(t *testing.T) {
		srv, _, _, learningSvc := helperServer(t)
		ctx := context.Background()

		id, _ := learningSvc.Store(ctx, models.LearningRecord{
			Title: "Enrich This",
			Type:  models.LearningTypeConfig,
		}, "")

		resp := callTool(t, srv, "learning.update_enrichment", map[string]any{
			"learning_id":      id,
			"enrichment_patch": map[string]any{"status": "accepted", "score": 4.5},
		})

		text := getToolResultText(t, resp)
		var result map[string]any
		if err := json.Unmarshal([]byte(text), &result); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}
		if result["status"] != "updated" {
			t.Errorf("status = %q, want %q", result["status"], "updated")
		}
		if result["learning_id"] != id {
			t.Errorf("learning_id = %q, want %q", result["learning_id"], id)
		}

		// Verify score also synced from enrichment patch
		rec, _ := learningSvc.Get(ctx, id)
		if rec.Score != 4.5 {
			t.Errorf("Score should sync from enrichment_patch: got %f, want 4.5", rec.Score)
		}
	})

	t.Run("update enrichment with quality_score syncs score", func(t *testing.T) {
		srv, _, _, learningSvc := helperServer(t)
		ctx := context.Background()

		id, _ := learningSvc.Store(ctx, models.LearningRecord{
			Title: "Quality Score Sync",
			Type:  models.LearningTypeConfig,
			Score: 0.5,
		}, "")

		resp := callTool(t, srv, "learning.update_enrichment", map[string]any{
			"learning_id":      id,
			"enrichment_patch": map[string]any{"quality_score": 4.8, "status": "accepted"},
		})

		getToolResultText(t, resp)

		// Verify quality_score synced to score column
		rec, _ := learningSvc.Get(ctx, id)
		if rec.Score != 4.8 {
			t.Errorf("Score should sync from quality_score: got %f, want 4.8", rec.Score)
		}
	})

	t.Run("update enrichment without score does not affect score", func(t *testing.T) {
		srv, _, _, learningSvc := helperServer(t)
		ctx := context.Background()

		id, _ := learningSvc.Store(ctx, models.LearningRecord{
			Title: "No Score",
			Type:  models.LearningTypeConfig,
			Score: 0.3,
		}, "")

		resp := callTool(t, srv, "learning.update_enrichment", map[string]any{
			"learning_id":      id,
			"enrichment_patch": map[string]any{"status": "accepted"},
		})

		getToolResultText(t, resp)

		// Score should remain unchanged
		rec, _ := learningSvc.Get(ctx, id)
		if rec.Score != 0.3 {
			t.Errorf("Score should remain 0.3, got %f", rec.Score)
		}
	})

	t.Run("update enrichment without learning_id returns error", func(t *testing.T) {
		srv, _, _, _ := helperServer(t)

		resp := callTool(t, srv, "learning.update_enrichment", map[string]any{
			"learning_id":      "",
			"enrichment_patch": map[string]any{"status": "accepted"},
		})

		requireToolResultError(t, resp)
	})

	t.Run("update enrichment without patch returns error", func(t *testing.T) {
		srv, _, _, learningSvc := helperServer(t)
		ctx := context.Background()

		id, _ := learningSvc.Store(ctx, models.LearningRecord{
			Title: "No Patch",
			Type:  models.LearningTypeConfig,
		}, "")

		resp := callTool(t, srv, "learning.update_enrichment", map[string]any{
			"learning_id": id,
		})

		requireToolResultError(t, resp)
	})

	t.Run("update enrichment non-existent returns error", func(t *testing.T) {
		srv, _, _, _ := helperServer(t)

		resp := callTool(t, srv, "learning.update_enrichment", map[string]any{
			"learning_id":      "non-existent",
			"enrichment_patch": map[string]any{"status": "accepted"},
		})

		requireToolResultError(t, resp)
	})
}

func TestOKFExportTool(t *testing.T) {
	t.Parallel()

	t.Run("exports accepted high-confidence learnings", func(t *testing.T) {
		srv, _, _, learningSvc := helperServer(t)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		id, err := learningSvc.Store(ctx, models.LearningRecord{
			Title: "Retry payment API timeouts",
			Body:  "Retry twice with exponential backoff when the payment API times out.",
			Type:  models.LearningTypePattern,
			Tags:  []string{"payment"},
			Score: 4.7,
		}, "")
		if err != nil {
			t.Fatalf("Store() unexpected error: %v", err)
		}
		if err := learningSvc.UpdateEnrichment(ctx, id, json.RawMessage(`{"status":"accepted"}`)); err != nil {
			t.Fatalf("UpdateEnrichment() unexpected error: %v", err)
		}

		resp := callTool(t, srv, "okf.export", map[string]any{
			"query":     "payment",
			"min_score": 4.0,
		})

		text := getToolResultText(t, resp)
		var bundle service.OKFBundle
		if err := json.Unmarshal([]byte(text), &bundle); err != nil {
			t.Fatalf("failed to unmarshal bundle: %v", err)
		}
		if bundle.OKFVersion != "0.1" {
			t.Errorf("OKFVersion = %q, want 0.1", bundle.OKFVersion)
		}
		if bundle.Summary.Exported != 1 {
			t.Errorf("Exported = %d, want 1", bundle.Summary.Exported)
		}
		if len(bundle.Files) != 2 {
			t.Fatalf("Files length = %d, want 2", len(bundle.Files))
		}
		if !strings.Contains(bundle.Files[1].Content, "type: ALMS Pattern") {
			t.Errorf("concept content missing OKF type:\n%s", bundle.Files[1].Content)
		}
	})

	t.Run("exports with tags and no query", func(t *testing.T) {
		srv, _, _, learningSvc := helperServer(t)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		id, err := learningSvc.Store(ctx, models.LearningRecord{
			Title: "Deployment rollback pattern",
			Body:  "Rollback before retrying a failed deployment.",
			Type:  models.LearningTypePattern,
			Tags:  []string{"deploy"},
			Score: 4.5,
		}, "")
		if err != nil {
			t.Fatalf("Store() unexpected error: %v", err)
		}
		if err := learningSvc.UpdateEnrichment(ctx, id, json.RawMessage(`{"status":"accepted"}`)); err != nil {
			t.Fatalf("UpdateEnrichment() unexpected error: %v", err)
		}

		resp := callTool(t, srv, "okf.export", map[string]any{
			"tags": []any{"deploy"},
		})

		text := getToolResultText(t, resp)
		var bundle service.OKFBundle
		if err := json.Unmarshal([]byte(text), &bundle); err != nil {
			t.Fatalf("failed to unmarshal bundle: %v", err)
		}
		if bundle.Summary.Exported != 1 {
			t.Errorf("Exported = %d, want 1", bundle.Summary.Exported)
		}
	})
}

func TestHealthCheckTool(t *testing.T) {
	t.Parallel()

	t.Run("health check returns ok with agent count", func(t *testing.T) {
		srv, registry, _, _ := helperServer(t)
		ctx := context.Background()

		// Register an agent so we have a non-zero count
		_ = registry.Register(ctx, models.AgentSpec{
			AgentID:   "health-check-agent",
			AgentType: models.AgentTypeSystemd,
		})

		resp := callTool(t, srv, "health.check", map[string]any{})

		text := getToolResultText(t, resp)
		var result map[string]any
		if err := json.Unmarshal([]byte(text), &result); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}
		if result["status"] != "ok" {
			t.Errorf("status = %q, want %q", result["status"], "ok")
		}
		// Verify agent_count is present and numeric
		count, ok := result["agent_count"].(float64)
		if !ok {
			t.Fatal("agent_count field missing or not numeric")
		}
		if count < 1 {
			t.Errorf("agent_count = %v, want >= 1", count)
		}
		if _, ok := result["version"]; !ok {
			t.Fatal("version field missing")
		}
	})

	t.Run("health check returns zero agents when none registered", func(t *testing.T) {
		srv, _, _, _ := helperServer(t)

		resp := callTool(t, srv, "health.check", map[string]any{})

		text := getToolResultText(t, resp)
		var result map[string]any
		if err := json.Unmarshal([]byte(text), &result); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}
		count, ok := result["agent_count"].(float64)
		if !ok {
			t.Fatal("agent_count field missing or not numeric")
		}
		// With no agents registered, agent_count should be 0
		if count != 0 {
			t.Errorf("agent_count = %v, want 0", count)
		}
	})
}
