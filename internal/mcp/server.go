package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"sage/internal/graph"
)

// Server wraps the MCP server instance and BadgerDB graph engine.
type Server struct {
	Engine *graph.Engine
}

// NewServer creates a new Sage MCP server.
func NewServer(engine *graph.Engine) *Server {
	return &Server{Engine: engine}
}

// StartStdio starts the MCP server over stdio for Cursor, Claude Code, and Antigravity.
func (s *Server) StartStdio() error {
	sServer := server.NewMCPServer("sage", "1.0.0")

	// 1. Tool: sage_who_owns
	whoTool := mcp.NewTool("sage_who_owns",
		mcp.WithDescription("Find the formal owner and top recent expert contributors for a file or directory"),
		mcp.WithString("path", mcp.Required(), mcp.Description("File or directory path in the repository")),
	)
	sServer.AddTool(whoTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		path := req.GetString("path", "")
		owner, experts, err := s.Engine.QueryOwner(path)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		res := fmt.Sprintf("Formal Owner: %s\nRecent Experts:\n- %s", owner, strings.Join(experts, "\n- "))
		return mcp.NewToolResultText(res), nil
	})

	// 2. Tool: sage_why
	whyTool := mcp.NewTool("sage_why",
		mcp.WithDescription("Trace why a file, component, or commit exists, including PR debates and decision rationale"),
		mcp.WithString("target", mcp.Required(), mcp.Description("File path or commit SHA to trace")),
	)
	sServer.AddTool(whyTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		target := req.GetString("target", "")
		lineage, err := s.Engine.QueryWhy(target)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		res := strings.Join(lineage, "\n\n")
		return mcp.NewToolResultText(res), nil
	})

	// 3. Tool: sage_overview
	overviewTool := mcp.NewTool("sage_overview",
		mcp.WithDescription("Get a compact overview of repository graph statistics"),
	)
	sServer.AddTool(overviewTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		stats, err := s.Engine.QueryOverview()
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		var sb strings.Builder
		sb.WriteString("Sage Knowledge Graph Overview:\n")
		for k, v := range stats {
			sb.WriteString(fmt.Sprintf("- %s: %d\n", k, v))
		}
		return mcp.NewToolResultText(sb.String()), nil
	})

	// 4. Tool: sage_query
	queryTool := mcp.NewTool("sage_query",
		mcp.WithDescription("Execute prefix query to inspect records in BadgerDB"),
		mcp.WithString("query", mcp.Required(), mcp.Description("Key prefix to scan in BadgerDB")),
	)
	sServer.AddTool(queryTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q := req.GetString("query", "")
		if q == "" {
			q = req.GetString("cypher", "")
		}
		rows, err := s.Engine.ExecuteRawCypher(q)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		var sb strings.Builder
		for _, row := range rows {
			sb.WriteString(strings.Join(row, " | ") + "\n")
		}
		return mcp.NewToolResultText(sb.String()), nil
	})

	return server.ServeStdio(sServer)
}
