package server

import (
	"github.com/mark3labs/mcp-go/server"

	"github.com/ghassan/alms/internal/service"
)

func registerAgentTools(mcpSrv *server.MCPServer, registry *service.Registry) {
	registerAgentRegisterTool(mcpSrv, registry)
	registerAgentUnregisterTool(mcpSrv, registry)
	registerAgentUpdateTool(mcpSrv, registry)
	registerAgentListTool(mcpSrv, registry)
	registerAgentHeartbeatTool(mcpSrv, registry)
}
