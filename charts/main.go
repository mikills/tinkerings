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

type BarPoint struct {
	Label string  `json:"label" jsonschema:"description=Category/label for the bar (x-axis, e.g. 'January', 'Product A')"`
	Value float64 `json:"value" jsonschema:"description=Numeric value for the bar (y-axis, e.g. sales amount, quantity)"`
}

type BarChartArgs struct {
	Title        string     `json:"title,omitempty" jsonschema:"description=Chart title (e.g. 'Q1 Sales Report')"`
	DatasetLabel string     `json:"datasetLabel,omitempty" jsonschema:"description=Label for the data series (e.g. 'Revenue', 'Units Sold')"`
	Points       []BarPoint `json:"points" jsonschema:"description=Array of data points with category labels and numeric values,minItems=1"`
}

type LinePoint struct {
	X string  `json:"x" jsonschema:"description=X-axis value (e.g. time period, category)"`
	Y float64 `json:"y" jsonschema:"description=Y-axis numeric value"`
}

type LineChartArgs struct {
	Title        string      `json:"title,omitempty" jsonschema:"description=Chart title"`
	DatasetLabel string      `json:"datasetLabel,omitempty" jsonschema:"description=Label for the data series"`
	Points       []LinePoint `json:"points" jsonschema:"description=Array of data points,minItems=1"`
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

	return map[string]any{
		"type": "line",
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

func validateLineChartArgs(args LineChartArgs) error {
	if len(args.Points) == 0 {
		return fmt.Errorf("points must contain at least one item")
	}
	return nil
}

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

	registerChartTool(srv, chartToolConfig{
		name: "line-chart-generator",
		description: `Generates a Chart.js line chart configuration.
		              Line charts plot data points connected by lines and are often used to show trend data or compare two data sets over time.
		              Use this for time series data, sequential data, or showing continuous relationships between variables.`,
	},
		generateLineChartJSON,
		validateLineChartArgs,
	)

	if err := server.ServeStdio(srv); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
