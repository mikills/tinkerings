package main

import "github.com/mark3labs/mcp-go/server"

func New() *server.MCPServer {
	srv := server.NewMCPServer("chartjs-generator", "2.2.0")

	registerAreaChartTool(srv)
	registerBarChartTool(srv)
	registerDoughnutChartTool(srv)
	registerFlowchartTool(srv)
	registerLineChartTool(srv)
	registerPieChartTool(srv)
	registerPolarAreaChartTool(srv)
	registerRadarChartTool(srv)
	registerScatterChartTool(srv)
	registerSequenceDiagramTool(srv)

	return srv
}
