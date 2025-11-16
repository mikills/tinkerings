package main

import (
	"fmt"

	"github.com/mark3labs/mcp-go/server"
)

type PolarAreaPoint struct {
	Label string  `json:"label" jsonschema:"description=Label for the segment (e.g. 'Category A', 'Metric 1')"`
	Value float64 `json:"value" jsonschema:"description=Numeric value for the segment"`
}

type PolarAreaChartArgs struct {
	Title        string           `json:"title,omitempty" jsonschema:"description=Chart title"`
	DatasetLabel string           `json:"datasetLabel,omitempty" jsonschema:"description=Label for the data series"`
	Points       []PolarAreaPoint `json:"points" jsonschema:"description=Array of data points with labels and values,minItems=1"`
}

func generatePolarAreaChartJSON(args PolarAreaChartArgs) map[string]any {
	labels := make([]string, len(args.Points))
	data := make([]float64, len(args.Points))

	for i, point := range args.Points {
		labels[i] = point.Label
		data[i] = point.Value
	}

	datasetLabel := args.DatasetLabel
	if datasetLabel == "" {
		datasetLabel = "Dataset"
	}

	return map[string]any{
		"type": "polarArea",
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
		},
	}
}

func validatePolarAreaChartArgs(args PolarAreaChartArgs) error {
	if len(args.Points) == 0 {
		return fmt.Errorf("points must contain at least one item")
	}
	return nil
}

func registerPolarAreaChartTool(srv *server.MCPServer) {
	registerChartTool(srv, chartToolConfig{
		name: "polar-area-chart-generator",
		description: `Generates a Chart.js polar area chart configuration.
		              Polar area charts are similar to pie charts but each segment has the same angle, with radius varying based on value.
		              Use this to show comparison data similar to a pie chart while also displaying a scale of values for context.
		              Useful for cyclical data or when you need to compare magnitudes across categories in a radial format.`,
	},
		generatePolarAreaChartJSON,
		validatePolarAreaChartArgs,
	)
}
