package server

import "github.com/ghassan/alms/internal/service"

// registerTools registers all MCP tool handlers on the server.
func (s *Server) registerTools(registry *service.Registry, syncer *service.Syncer, learning *service.Learning) {
	registerAgentTools(s.mcp, registry)
	registerLearningSyncTools(s.mcp, syncer)
	registerLearningRecordTools(s.mcp, learning)
	registerProtocolSyncTools(s.mcp, syncer)
	registerProtocolManagementTools(s.mcp, learning)
	registerHealthTools(s.mcp, registry)
	registerEnrichmentTools(s.mcp, learning)
	registerOKFExportTools(s.mcp, learning)
}
