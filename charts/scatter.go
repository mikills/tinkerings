package main

import (
	"fmt"

	"github.com/mark3labs/mcp-go/server"
)

type ScatterPoint struct {
	X float64 `json:"x" jsonschema:"description=X-axis numeric value"`
	Y float64 `json:"y" jsonschema:"description=Y-axis numeric value"`
}

type ScatterChartArgs struct {
	Title        string         `json:"title,omitempty" jsonschema:"description=Chart title"`
	DatasetLabel string         `json:"datasetLabel,omitempty" jsonschema:"description=Label for the data series"`
	Points       []ScatterPoint `json:"points" jsonschema:"description=Array of data points with x and y numeric values,minItems=1"`
}

func generateScatterChartJSON(args ScatterChartArgs) map[string]any {
	data := make([]map[string]float64, len(args.Points))

	for i, point := range args.Points {
		data[i] = map[string]float64{
			"x": point.X,
			"y": point.Y,
		}
	}

	datasetLabel := args.DatasetLabel
	if datasetLabel == "" {
		datasetLabel = "Series 1"
	}

	return map[string]any{
		"type": "scatter",
		"data": map[string]any{
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
				"x": map[string]any{
					"type":        "linear",
					"position":    "bottom",
					"beginAtZero": true,
				},
				"y": map[string]any{"beginAtZero": true},
			},
		},
	}
}

func validateScatterChartArgs(args ScatterChartArgs) error {
	if len(args.Points) == 0 {
		return fmt.Errorf("points must contain at least one item")
	}
	return nil
}

func registerScatterChartTool(srv *server.MCPServer) {
	registerChartTool(srv, chartToolConfig{
		name: "scatter-chart-generator",
		description: `Generates a Chart.js scatter chart configuration.
		              Scatter charts plot data points on a two-dimensional graph with numeric x and y axes.
		              Use this to show relationships, correlations, or distributions between two numeric variables.
		              Perfect for statistical analysis, finding patterns, or visualizing data clustering.`,
	},
		generateScatterChartJSON,
		validateScatterChartArgs,
	)
}
