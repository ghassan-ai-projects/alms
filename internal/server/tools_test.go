package server

import (
	"context"
	"encoding/json"
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
	learningSvc := service.NewLearning(lStore)

	// Register tools directly on the MCP server
	registerAgentTools(mcpSrv, registry)
	registerLearningTools(mcpSrv, syncer)
	registerProtocolTools(mcpSrv, syncer)

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
