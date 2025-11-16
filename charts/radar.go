package main

import (
	"fmt"

	"github.com/mark3labs/mcp-go/server"
)

type RadarPoint struct {
	Label string  `json:"label" jsonschema:"description=Label for the axis point (e.g. 'Speed', 'Strength', 'Agility')"`
	Value float64 `json:"value" jsonschema:"description=Numeric value for the axis point"`
}

type RadarChartArgs struct {
	Title        string       `json:"title,omitempty" jsonschema:"description=Chart title"`
	DatasetLabel string       `json:"datasetLabel,omitempty" jsonschema:"description=Label for the data series"`
	Points       []RadarPoint `json:"points" jsonschema:"description=Array of data points with labels and values,minItems=1"`
}

func generateRadarChartJSON(args RadarChartArgs) map[string]any {
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
		"type": "radar",
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
				"r": map[string]any{"beginAtZero": true},
			},
		},
	}
}

func validateRadarChartArgs(args RadarChartArgs) error {
	if len(args.Points) == 0 {
		return fmt.Errorf("points must contain at least one item")
	}
	return nil
}

func registerRadarChartTool(srv *server.MCPServer) {
	registerChartTool(srv, chartToolConfig{
		name: "radar-chart-generator",
		description: `Generates a Chart.js radar chart configuration.
		              Radar charts show multiple data points and the variation between them on axes starting from the same center point.
		              Use this for comparing multiple variables or showing multivariate data in a compact visual format.
		              Ideal for displaying performance metrics, skill assessments, or multi-dimensional comparisons.`,
	},
		generateRadarChartJSON,
		validateRadarChartArgs,
	)
}
