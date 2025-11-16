package main

import (
	"fmt"

	"github.com/mark3labs/mcp-go/server"
)

type AreaPoint struct {
	X string  `json:"x" jsonschema:"description=X-axis value (e.g. time period, category)"`
	Y float64 `json:"y" jsonschema:"description=Y-axis numeric value"`
}

type AreaChartArgs struct {
	Title        string      `json:"title,omitempty" jsonschema:"description=Chart title"`
	DatasetLabel string      `json:"datasetLabel,omitempty" jsonschema:"description=Label for the data series"`
	Points       []AreaPoint `json:"points" jsonschema:"description=Array of data points,minItems=1"`
}

func generateAreaChartJSON(args AreaChartArgs) map[string]any {
	labels := make([]string, len(args.Points))
	data := make([]float64, len(args.Points))

	for i, point := range args.Points {
		labels[i] = point.X
		data[i] = point.Y
	}

	datasetLabel := args.DatasetLabel
	if datasetLabel == "" {
		datasetLabel = "Series 1"
	}

	return map[string]any{
		"type": "line",
		"data": map[string]any{
			"labels": labels,
			"datasets": []map[string]any{{
				"label": datasetLabel,
				"data":  data,
				"fill":  true,
			}},
		},
		"options": map[string]any{
			"responsive": true,
			"plugins": map[string]any{
				"title": map[string]any{
					"display": args.Title != "",
					"text":    args.Title,
				},
				"legend": map[string]any{"display": true},
			},
			"scales": map[string]any{
				"y": map[string]any{"beginAtZero": true},
			},
		},
	}
}

func validateAreaChartArgs(args AreaChartArgs) error {
	if len(args.Points) == 0 {
		return fmt.Errorf("points must contain at least one item")
	}
	return nil
}

func registerAreaChartTool(srv *server.MCPServer) {
	registerChartTool(srv, chartToolConfig{
		name: "area-chart-generator",
		description: `Generates a Chart.js area chart configuration.
		              Area charts are line charts with the area below the line filled.
		              Use this to show quantitative data over time or categories with visual emphasis on volume or magnitude.
		              Perfect for showing trends with cumulative values or emphasizing the magnitude of change.`,
	},
		generateAreaChartJSON,
		validateAreaChartArgs,
	)
}
