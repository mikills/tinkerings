package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type chartToolConfig struct {
	name        string
	description string
}

func registerChartTool[T any](
	srv *server.MCPServer,
	cfg chartToolConfig,
	generator func(T) map[string]any,
	validator func(T) error,
) {
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args T
		if err := req.BindArguments(&args); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("bind arguments: %v", err)), nil
		}

		// validate if validator provided
		if validator != nil {
			if err := validator(args); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
		}

		chartJSON := generator(args)
		return mcp.NewToolResultJSON(chartJSON)
	}

	tool := mcp.NewTool(
		cfg.name,
		mcp.WithDescription(cfg.description),
		mcp.WithInputSchema[T](),
	)

	srv.AddTool(tool, handler)
}

func main() {
	_, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	srv := server.NewMCPServer("chartjs-generator", "1.2.0")

	registerBarChartTool(srv)
	registerLineChartTool(srv)

	if err := server.ServeStdio(srv); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
