package main

import (
	"fmt"

	"github.com/mark3labs/mcp-go/server"
)

type BarPoint struct {
	Label string  `json:"label" jsonschema:"description=Category/label for the bar (x-axis, e.g. 'January', 'Product A')"`
	Value float64 `json:"value" jsonschema:"description=Numeric value for the bar (y-axis, e.g. sales amount, quantity)"`
}

type BarChartArgs struct {
	Title        string     `json:"title,omitempty" jsonschema:"description=Chart title (e.g. 'Q1 Sales Report')"`
	DatasetLabel string     `json:"datasetLabel,omitempty" jsonschema:"description=Label for the data series (e.g. 'Revenue', 'Units Sold')"`
	Points       []BarPoint `json:"points" jsonschema:"description=Array of data points with category labels and numeric values,minItems=1"`
}

func generateBarChartJSON(args BarChartArgs) map[string]any {
	labels := make([]string, len(args.Points))
	data := make([]float64, len(args.Points))

	for i, point := range args.Points {
		labels[i] = point.Label
		data[i] = point.Value
	}

	datasetLabel := args.DatasetLabel
	if datasetLabel == "" {
		datasetLabel = "Series 1"
	}

	return map[string]any{
		"type": "bar",
		"data": map[string]any{
			"labels": labels,
			"datasets": []map[string]any{{
				"label": datasetLabel,
				"data":  data,
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

func validateBarChartArgs(args BarChartArgs) error {
	if len(args.Points) == 0 {
		return fmt.Errorf("points must contain at least one item")
	}
	return nil
}

func registerBarChartTool(srv *server.MCPServer) {
	registerChartTool(srv, chartToolConfig{
		name: "bar-chart-generator",
		description: `Generates a Chart.js bar chart configuration.
					  Bar charts provide a way of showing data values represented as vertical bars.
					  Use this to show trend data and compare multiple data sets side by side (e.g., monthly sales, regional comparisons, categorical counts).
					  Supports vertical bars (default) and horizontal bars via indexAxis.`,
	},
		generateBarChartJSON,
		validateBarChartArgs,
	)
}
