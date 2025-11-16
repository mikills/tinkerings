package main

import (
	"fmt"

	"github.com/mark3labs/mcp-go/server"
)

type DoughnutPoint struct {
	Label string  `json:"label" jsonschema:"description=Label for the segment (e.g. 'Product A', 'Category 1')"`
	Value float64 `json:"value" jsonschema:"description=Numeric value for the segment"`
}

type DoughnutChartArgs struct {
	Title        string          `json:"title,omitempty" jsonschema:"description=Chart title"`
	DatasetLabel string          `json:"datasetLabel,omitempty" jsonschema:"description=Label for the data series"`
	Points       []DoughnutPoint `json:"points" jsonschema:"description=Array of data points with labels and values,minItems=1"`
}

func generateDoughnutChartJSON(args DoughnutChartArgs) map[string]any {
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
		"type": "doughnut",
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

func validateDoughnutChartArgs(args DoughnutChartArgs) error {
	if len(args.Points) == 0 {
		return fmt.Errorf("points must contain at least one item")
	}
	return nil
}

func registerDoughnutChartTool(srv *server.MCPServer) {
	registerChartTool(srv, chartToolConfig{
		name: "doughnut-chart-generator",
		description: `Generates a Chart.js doughnut chart configuration.
		              Doughnut charts are divided into segments with a hole in the center, where each segment shows the proportional value of each piece of data.
		              Use this to show relative proportions or percentages between categories.
		              Excellent for displaying part-to-whole relationships with a modern look.`,
	},
		generateDoughnutChartJSON,
		validateDoughnutChartArgs,
	)
}
