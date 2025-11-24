package main

import (
	"fmt"

	"github.com/mark3labs/mcp-go/server"
)

type PiePoint struct {
	Label string  `json:"label" jsonschema:"description=Label for the segment (e.g. 'Product A', 'Category 1')"`
	Value float64 `json:"value" jsonschema:"description=Numeric value for the segment"`
}

type PieChartArgs struct {
	Title            string     `json:"title,omitempty" jsonschema:"description=Chart title"`
	DatasetLabel     string     `json:"datasetLabel,omitempty" jsonschema:"description=Label for the data series"`
	Points           []PiePoint `json:"points" jsonschema:"description=Array of data points with labels and values,minItems=1"`
	BackgroundColors []string   `json:"backgroundColors,omitempty" jsonschema:"description=Optional array of background colors for pie segments (e.g. ['rgba(255, 99, 132, 0.8)', '#FF6384']). If not provided, uses default color palette."`
	BorderColors     []string   `json:"borderColors,omitempty" jsonschema:"description=Optional array of border colors for pie segments. If not provided, uses default color palette."`
}

func generatePieChartJSON(args PieChartArgs) map[string]any {
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

	backgroundColor := getColors(args.BackgroundColors, len(args.Points))
	borderColor := getBorderColors(args.BorderColors, len(args.Points))

	return map[string]any{
		"type": "pie",
		"data": map[string]any{
			"labels": labels,
			"datasets": []map[string]any{{
				"label":           datasetLabel,
				"data":            data,
				"backgroundColor": backgroundColor,
				"borderColor":     borderColor,
				"borderWidth":     1,
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

func validatePieChartArgs(args PieChartArgs) error {
	if len(args.Points) == 0 {
		return fmt.Errorf("points must contain at least one item")
	}
	return nil
}

func registerPieChartTool(srv *server.MCPServer) {
	registerChartTool(srv, chartToolConfig{
		name: "pie-chart-generator",
		description: `Generates a Chart.js pie chart configuration.
		              Pie charts are divided into segments where each segment shows the proportional value of each piece of data.
		              Use this to show relative proportions or percentages between categories.
		              Excellent for displaying part-to-whole relationships and data composition.`,
	},
		generatePieChartJSON,
		validatePieChartArgs,
	)
}
