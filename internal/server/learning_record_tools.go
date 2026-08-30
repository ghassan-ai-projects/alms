package server

import (
	"github.com/mark3labs/mcp-go/server"

	"github.com/ghassan/alms/internal/service"
)

// registerLearningRecordTools registers CRUD and search tools for learning records.
func registerLearningRecordTools(mcpSrv *server.MCPServer, learning *service.Learning) {
	registerLearningStoreTool(mcpSrv, learning)
	registerLearningSearchTool(mcpSrv, learning)
	registerLearningDeleteTool(mcpSrv, learning)
	registerLearningGetTool(mcpSrv, learning)
}
