package server

import (
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/ghassan/alms/internal/service"
)

// registerResources registers all MCP resource handlers on the server.
func (s *Server) registerResources(registry *service.Registry, learning *service.Learning) {
	registerAgentsResource(s.mcp, registry)
	registerHealthResource(s.mcp, registry)
	registerLearningsResource(s.mcp, learning)
	registerToolsResource(s.mcp)
	registerProtocolsResource(s.mcp, learning)
}

func newJSONResourceContents(uri string, value any, marshalContext string) ([]mcp.ResourceContents, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", marshalContext, err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      uri,
			MIMEType: "application/json",
			Text:     string(data),
		},
	}, nil
}
