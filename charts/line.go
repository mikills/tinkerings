package main

import (
	"fmt"

	"github.com/mark3labs/mcp-go/server"
)

type LinePoint struct {
	X string  `json:"x" jsonschema:"description=X-axis value (e.g. time period, category)"`
	Y float64 `json:"y" jsonschema:"description=Y-axis numeric value"`
}

type LineChartArgs struct {
	Title           string      `json:"title,omitempty" jsonschema:"description=Chart title"`
	DatasetLabel    string      `json:"datasetLabel,omitempty" jsonschema:"description=Label for the data series"`
	Points          []LinePoint `json:"points" jsonschema:"description=Array of data points,minItems=1"`
	BackgroundColor string      `json:"backgroundColor,omitempty" jsonschema:"description=Optional background color for points (e.g. 'rgba(255, 99, 132, 0.8)', '#FF6384'). If not provided, uses default color palette."`
	BorderColor     string      `json:"borderColor,omitempty" jsonschema:"description=Optional line color. If not provided, uses default color palette."`
}

func generateLineChartJSON(args LineChartArgs) map[string]any {
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

	backgroundColor := getSingleColor([]string{args.BackgroundColor}, 0)
	borderColor := getSingleBorderColor([]string{args.BorderColor}, 0)

	return map[string]any{
		"type": "line",
		"data": map[string]any{
			"labels": labels,
			"datasets": []map[string]any{{
				"label":           datasetLabel,
				"data":            data,
				"backgroundColor": backgroundColor,
				"borderColor":     borderColor,
				"borderWidth":     2,
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

func validateLineChartArgs(args LineChartArgs) error {
	if len(args.Points) == 0 {
		return fmt.Errorf("points must contain at least one item")
	}
	return nil
}

func registerLineChartTool(srv *server.MCPServer) {
	registerChartTool(srv, chartToolConfig{
		name: "line-chart-generator",
		description: `Generates a Chart.js line chart configuration.
		              Line charts plot data points connected by lines and are often used to show trend data or compare two data sets over time.
		              Use this for time series data, sequential data, or showing continuous relationships between variables.`,
	},
		generateLineChartJSON,
		validateLineChartArgs,
	)
}
