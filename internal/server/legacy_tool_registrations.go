package server

import (
	"github.com/mark3labs/mcp-go/server"

	"github.com/ghassan/alms/internal/service"
)

// These aliases preserve the package-private registration helpers used by the
// existing same-package tests while the production registrations are organized
// by domain.

func registerLearningTools(mcpSrv *server.MCPServer, syncer *service.Syncer) {
	registerLearningSyncTools(mcpSrv, syncer)
}

func registerLearningStoreTools(mcpSrv *server.MCPServer, learning *service.Learning) {
	registerLearningRecordTools(mcpSrv, learning)
}

func registerProtocolTools(mcpSrv *server.MCPServer, syncer *service.Syncer) {
	registerProtocolSyncTools(mcpSrv, syncer)
}

func registerProtocolStoreTools(mcpSrv *server.MCPServer, learning *service.Learning) {
	registerProtocolManagementTools(mcpSrv, learning)
}

func registerPhase2Tools(mcpSrv *server.MCPServer, learning *service.Learning) {
	registerEnrichmentTools(mcpSrv, learning)
}

func registerOKFTools(mcpSrv *server.MCPServer, learning *service.Learning) {
	registerOKFExportTools(mcpSrv, learning)
}
