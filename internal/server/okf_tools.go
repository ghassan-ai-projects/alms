package server

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/ghassan/alms/internal/service"
)

// registerOKFExportTools registers the OKF export tool.
func registerOKFExportTools(mcpSrv *server.MCPServer, learning *service.Learning) {
	mcpSrv.AddTool(mcp.NewTool("okf.export",
		mcp.WithDescription("Export accepted, high-confidence ALMS learnings as an OKF v0.1 bundle payload"),
		mcp.WithString("query", mcp.Description("Optional search query selecting candidate learnings")),
		mcp.WithString("type", mcp.Description("Optional learning type filter")),
		mcp.WithArray("tags",
			mcp.Description("Optional tag filters"),
			mcp.Items(map[string]any{"type": "string"}),
		),
		mcp.WithNumber("limit", mcp.Description("Max candidate learnings to inspect (default 50)")),
		mcp.WithString("status", mcp.Description("Enrichment status to export (default accepted; use all to disable status filtering)")),
		mcp.WithNumber("min_score", mcp.Description("Minimum learning score to export (default 4.0)")),
		mcp.WithBoolean("include_rejected", mcp.Description("Include records hidden by enrichment visibility metadata (default false)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		options := service.OKFExportOptions{
			Query:           req.GetString("query", ""),
			Type:            req.GetString("type", ""),
			Tags:            req.GetStringSlice("tags", nil),
			Limit:           req.GetInt("limit", 0),
			Status:          req.GetString("status", ""),
			MinScore:        req.GetFloat("min_score", 0),
			IncludeRejected: req.GetBool("include_rejected", false),
		}

		bundle, err := learning.ExportOKF(ctx, options)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return marshalResult(bundle)
	})
}
