package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestGenerateBarChartJSON(t *testing.T) {
	tests := []struct {
		name     string
		args     BarChartArgs
		validate func(t *testing.T, result map[string]any)
	}{
		{
			name: "basic bar chart",
			args: BarChartArgs{
				Title:        "Q1 Sales",
				DatasetLabel: "Revenue",
				Points: []BarPoint{
					{Label: "Jan", Value: 100},
					{Label: "Feb", Value: 150},
					{Label: "Mar", Value: 200},
				},
			},
			validate: func(t *testing.T, result map[string]any) {
				if result["type"] != "bar" {
					t.Errorf("expected type=bar, got %v", result["type"])
				}

				data := result["data"].(map[string]any)
				labels := data["labels"].([]string)
				if len(labels) != 3 {
					t.Errorf("expected 3 labels, got %d", len(labels))
				}
				if labels[0] != "Jan" || labels[1] != "Feb" || labels[2] != "Mar" {
					t.Errorf("unexpected labels: %v", labels)
				}

				datasets := data["datasets"].([]map[string]any)
				if len(datasets) != 1 {
					t.Errorf("expected 1 dataset, got %d", len(datasets))
				}
				if datasets[0]["label"] != "Revenue" {
					t.Errorf("expected label=Revenue, got %v", datasets[0]["label"])
				}

				dataPoints := datasets[0]["data"].([]float64)
				if len(dataPoints) != 3 {
					t.Errorf("expected 3 data points, got %d", len(dataPoints))
				}
				if dataPoints[0] != 100 || dataPoints[1] != 150 || dataPoints[2] != 200 {
					t.Errorf("unexpected data: %v", dataPoints)
				}
			},
		},
		{
			name: "chart without title",
			args: BarChartArgs{
				DatasetLabel: "Sales",
				Points: []BarPoint{
					{Label: "A", Value: 10},
					{Label: "B", Value: 20},
				},
			},
			validate: func(t *testing.T, result map[string]any) {
				options := result["options"].(map[string]any)
				plugins := options["plugins"].(map[string]any)
				title := plugins["title"].(map[string]any)
				if title["display"] != false {
					t.Errorf("expected title display=false, got %v", title["display"])
				}
			},
		},
		{
			name: "chart without dataset label",
			args: BarChartArgs{
				Title: "Test",
				Points: []BarPoint{
					{Label: "X", Value: 5},
				},
			},
			validate: func(t *testing.T, result map[string]any) {
				data := result["data"].(map[string]any)
				datasets := data["datasets"].([]map[string]any)
				if datasets[0]["label"] != "Series 1" {
					t.Errorf("expected default label='Series 1', got %v", datasets[0]["label"])
				}
			},
		},
		{
			name: "single data point",
			args: BarChartArgs{
				Points: []BarPoint{
					{Label: "Only", Value: 42},
				},
			},
			validate: func(t *testing.T, result map[string]any) {
				data := result["data"].(map[string]any)
				labels := data["labels"].([]string)
				if len(labels) != 1 || labels[0] != "Only" {
					t.Errorf("unexpected labels: %v", labels)
				}
			},
		},
		{
			name: "negative values",
			args: BarChartArgs{
				Title: "Profit/Loss",
				Points: []BarPoint{
					{Label: "Jan", Value: -50},
					{Label: "Feb", Value: 100},
					{Label: "Mar", Value: -25},
				},
			},
			validate: func(t *testing.T, result map[string]any) {
				data := result["data"].(map[string]any)
				datasets := data["datasets"].([]map[string]any)
				dataPoints := datasets[0]["data"].([]float64)
				if dataPoints[0] != -50 || dataPoints[2] != -25 {
					t.Errorf("negative values not preserved: %v", dataPoints)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateBarChartJSON(tt.args)
			tt.validate(t, result)

			// ensure result is valid JSON
			_, err := json.Marshal(result)
			if err != nil {
				t.Errorf("result is not valid JSON: %v", err)
			}
		})
	}
}

func TestGenerateLineChartJSON(t *testing.T) {
	tests := []struct {
		name     string
		args     LineChartArgs
		validate func(t *testing.T, result map[string]any)
	}{
		{
			name: "basic line chart",
			args: LineChartArgs{
				Title:        "Temperature",
				DatasetLabel: "Celsius",
				Points: []LinePoint{
					{X: "Mon", Y: 20},
					{X: "Tue", Y: 22},
					{X: "Wed", Y: 19},
				},
			},
			validate: func(t *testing.T, result map[string]any) {
				if result["type"] != "line" {
					t.Errorf("expected type=line, got %v", result["type"])
				}

				data := result["data"].(map[string]any)
				labels := data["labels"].([]string)
				if len(labels) != 3 {
					t.Errorf("expected 3 labels, got %d", len(labels))
				}

				datasets := data["datasets"].([]map[string]any)
				if datasets[0]["label"] != "Celsius" {
					t.Errorf("expected label=Celsius, got %v", datasets[0]["label"])
				}

				dataPoints := datasets[0]["data"].([]float64)
				if dataPoints[0] != 20 || dataPoints[1] != 22 || dataPoints[2] != 19 {
					t.Errorf("unexpected data: %v", dataPoints)
				}
			},
		},
		{
			name: "line chart without dataset label",
			args: LineChartArgs{
				Points: []LinePoint{
					{X: "A", Y: 1},
					{X: "B", Y: 2},
				},
			},
			validate: func(t *testing.T, result map[string]any) {
				data := result["data"].(map[string]any)
				datasets := data["datasets"].([]map[string]any)
				if datasets[0]["label"] != "Series 1" {
					t.Errorf("expected default label='Series 1', got %v", datasets[0]["label"])
				}
			},
		},
		{
			name: "time series data",
			args: LineChartArgs{
				Title:        "Stock Price",
				DatasetLabel: "AAPL",
				Points: []LinePoint{
					{X: "2025-01-01", Y: 150.5},
					{X: "2025-01-02", Y: 152.3},
					{X: "2025-01-03", Y: 151.8},
				},
			},
			validate: func(t *testing.T, result map[string]any) {
				data := result["data"].(map[string]any)
				labels := data["labels"].([]string)
				if labels[0] != "2025-01-01" {
					t.Errorf("date labels not preserved: %v", labels)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateLineChartJSON(tt.args)
			tt.validate(t, result)

			// ensure result is valid JSON
			_, err := json.Marshal(result)
			if err != nil {
				t.Errorf("result is not valid JSON: %v", err)
			}
		})
	}
}

func TestChartJSONStructure(t *testing.T) {
	tests := []struct {
		name      string
		generator func() map[string]any
		chartType string
	}{
		{
			name: "bar chart structure",
			generator: func() map[string]any {
				return generateBarChartJSON(BarChartArgs{
					Title:        "Test Chart",
					DatasetLabel: "Test Data",
					Points: []BarPoint{
						{Label: "A", Value: 100},
					},
				})
			},
			chartType: "bar",
		},
		{
			name: "line chart structure",
			generator: func() map[string]any {
				return generateLineChartJSON(LineChartArgs{
					Title:        "Test Chart",
					DatasetLabel: "Test Data",
					Points: []LinePoint{
						{X: "A", Y: 100},
					},
				})
			},
			chartType: "line",
		},
		{
			name: "area chart structure",
			generator: func() map[string]any {
				return generateAreaChartJSON(AreaChartArgs{
					Title:        "Test Chart",
					DatasetLabel: "Test Data",
					Points: []AreaPoint{
						{X: "A", Y: 100},
					},
				})
			},
			chartType: "line",
		},
		{
			name: "doughnut chart structure",
			generator: func() map[string]any {
				return generateDoughnutChartJSON(DoughnutChartArgs{
					Title:        "Test Chart",
					DatasetLabel: "Test Data",
					Points: []DoughnutPoint{
						{Label: "A", Value: 100},
					},
				})
			},
			chartType: "doughnut",
		},
		{
			name: "pie chart structure",
			generator: func() map[string]any {
				return generatePieChartJSON(PieChartArgs{
					Title:        "Test Chart",
					DatasetLabel: "Test Data",
					Points: []PiePoint{
						{Label: "A", Value: 100},
					},
				})
			},
			chartType: "pie",
		},
		{
			name: "polar area chart structure",
			generator: func() map[string]any {
				return generatePolarAreaChartJSON(PolarAreaChartArgs{
					Title:        "Test Chart",
					DatasetLabel: "Test Data",
					Points: []PolarAreaPoint{
						{Label: "A", Value: 100},
					},
				})
			},
			chartType: "polarArea",
		},
		{
			name: "radar chart structure",
			generator: func() map[string]any {
				return generateRadarChartJSON(RadarChartArgs{
					Title:        "Test Chart",
					DatasetLabel: "Test Data",
					Points: []RadarPoint{
						{Label: "A", Value: 100},
					},
				})
			},
			chartType: "radar",
		},
		{
			name: "scatter chart structure",
			generator: func() map[string]any {
				return generateScatterChartJSON(ScatterChartArgs{
					Title:        "Test Chart",
					DatasetLabel: "Test Data",
					Points: []ScatterPoint{
						{X: 10, Y: 100},
					},
				})
			},
			chartType: "scatter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.generator()

			// check type matches
			if result["type"] != tt.chartType {
				t.Errorf("expected type=%s, got %v", tt.chartType, result["type"])
			}

			// check required top-level fields
			requiredFields := []string{"type", "data", "options"}
			for _, field := range requiredFields {
				if _, ok := result[field]; !ok {
					t.Errorf("missing required field: %s", field)
				}
			}

			// check data structure
			data := result["data"].(map[string]any)
			// scatter charts don't have labels, they use direct x,y coordinates
			if tt.chartType != "scatter" {
				if _, ok := data["labels"]; !ok {
					t.Error("missing data.labels")
				}
			}
			if _, ok := data["datasets"]; !ok {
				t.Error("missing data.datasets")
			}

			datasets := data["datasets"].([]map[string]any)
			if len(datasets) == 0 {
				t.Error("datasets array is empty")
			}
			if _, ok := datasets[0]["label"]; !ok {
				t.Error("missing dataset label")
			}
			if _, ok := datasets[0]["data"]; !ok {
				t.Error("missing dataset data")
			}

			// check options structure
			options := result["options"].(map[string]any)
			if responsive, ok := options["responsive"]; !ok || responsive != true {
				t.Error("missing or invalid options.responsive")
			}
			if _, ok := options["plugins"]; !ok {
				t.Error("missing options.plugins")
			}

			// verify scales based on chart type
			if tt.chartType == "bar" || tt.chartType == "line" {
				if _, ok := options["scales"]; !ok {
					t.Error("missing options.scales")
				}
				scales := options["scales"].(map[string]any)
				y := scales["y"].(map[string]any)
				if beginAtZero, ok := y["beginAtZero"]; !ok || beginAtZero != true {
					t.Error("missing or invalid scales.y.beginAtZero")
				}
			} else if tt.chartType == "radar" {
				if _, ok := options["scales"]; !ok {
					t.Error("missing options.scales for radar chart")
				}
				scales := options["scales"].(map[string]any)
				r := scales["r"].(map[string]any)
				if beginAtZero, ok := r["beginAtZero"]; !ok || beginAtZero != true {
					t.Error("missing or invalid scales.r.beginAtZero")
				}
			} else if tt.chartType == "scatter" {
				if _, ok := options["scales"]; !ok {
					t.Error("missing options.scales for scatter chart")
				}
				scales := options["scales"].(map[string]any)
				x := scales["x"].(map[string]any)
				y := scales["y"].(map[string]any)
				if x["type"] != "linear" {
					t.Error("missing or invalid scales.x.type for scatter chart")
				}
				if beginAtZero, ok := y["beginAtZero"]; !ok || beginAtZero != true {
					t.Error("missing or invalid scales.y.beginAtZero for scatter chart")
				}
			}
		})
	}
}

func TestGenerateAreaChartJSON(t *testing.T) {
	tests := []struct {
		name     string
		args     AreaChartArgs
		validate func(t *testing.T, result map[string]any)
	}{
		{
			name: "basic area chart",
			args: AreaChartArgs{
				Title:        "Temperature Over Time",
				DatasetLabel: "Celsius",
				Points: []AreaPoint{
					{X: "Mon", Y: 20},
					{X: "Tue", Y: 22},
					{X: "Wed", Y: 19},
				},
			},
			validate: func(t *testing.T, result map[string]any) {
				if result["type"] != "line" {
					t.Errorf("expected type=line, got %v", result["type"])
				}

				data := result["data"].(map[string]any)
				datasets := data["datasets"].([]map[string]any)
				if datasets[0]["fill"] != true {
					t.Errorf("expected fill=true for area chart, got %v", datasets[0]["fill"])
				}

				if datasets[0]["label"] != "Celsius" {
					t.Errorf("expected label=Celsius, got %v", datasets[0]["label"])
				}
			},
		},
		{
			name: "area chart without title",
			args: AreaChartArgs{
				DatasetLabel: "Data",
				Points: []AreaPoint{
					{X: "A", Y: 10},
					{X: "B", Y: 20},
				},
			},
			validate: func(t *testing.T, result map[string]any) {
				options := result["options"].(map[string]any)
				plugins := options["plugins"].(map[string]any)
				title := plugins["title"].(map[string]any)
				if title["display"] != false {
					t.Errorf("expected title display=false, got %v", title["display"])
				}
			},
		},
		{
			name: "area chart without dataset label",
			args: AreaChartArgs{
				Points: []AreaPoint{
					{X: "X", Y: 5},
				},
			},
			validate: func(t *testing.T, result map[string]any) {
				data := result["data"].(map[string]any)
				datasets := data["datasets"].([]map[string]any)
				if datasets[0]["label"] != "Series 1" {
					t.Errorf("expected default label='Series 1', got %v", datasets[0]["label"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateAreaChartJSON(tt.args)
			tt.validate(t, result)

			// ensure result is valid JSON
			_, err := json.Marshal(result)
			if err != nil {
				t.Errorf("result is not valid JSON: %v", err)
			}
		})
	}
}

func TestGenerateDoughnutChartJSON(t *testing.T) {
	tests := []struct {
		name     string
		args     DoughnutChartArgs
		validate func(t *testing.T, result map[string]any)
	}{
		{
			name: "basic doughnut chart",
			args: DoughnutChartArgs{
				Title:        "Market Share",
				DatasetLabel: "Products",
				Points: []DoughnutPoint{
					{Label: "Product A", Value: 300},
					{Label: "Product B", Value: 50},
					{Label: "Product C", Value: 100},
				},
			},
			validate: func(t *testing.T, result map[string]any) {
				if result["type"] != "doughnut" {
					t.Errorf("expected type=doughnut, got %v", result["type"])
				}

				data := result["data"].(map[string]any)
				labels := data["labels"].([]string)
				if len(labels) != 3 {
					t.Errorf("expected 3 labels, got %d", len(labels))
				}

				datasets := data["datasets"].([]map[string]any)
				dataPoints := datasets[0]["data"].([]float64)
				if dataPoints[0] != 300 || dataPoints[1] != 50 || dataPoints[2] != 100 {
					t.Errorf("unexpected data: %v", dataPoints)
				}
			},
		},
		{
			name: "doughnut chart without dataset label",
			args: DoughnutChartArgs{
				Points: []DoughnutPoint{
					{Label: "A", Value: 10},
				},
			},
			validate: func(t *testing.T, result map[string]any) {
				data := result["data"].(map[string]any)
				datasets := data["datasets"].([]map[string]any)
				if datasets[0]["label"] != "Dataset" {
					t.Errorf("expected default label='Dataset', got %v", datasets[0]["label"])
				}
			},
		},
		{
			name: "single segment",
			args: DoughnutChartArgs{
				Points: []DoughnutPoint{
					{Label: "Only", Value: 100},
				},
			},
			validate: func(t *testing.T, result map[string]any) {
				data := result["data"].(map[string]any)
				labels := data["labels"].([]string)
				if len(labels) != 1 || labels[0] != "Only" {
					t.Errorf("unexpected labels: %v", labels)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateDoughnutChartJSON(tt.args)
			tt.validate(t, result)

			_, err := json.Marshal(result)
			if err != nil {
				t.Errorf("result is not valid JSON: %v", err)
			}
		})
	}
}

func TestGeneratePieChartJSON(t *testing.T) {
	tests := []struct {
		name     string
		args     PieChartArgs
		validate func(t *testing.T, result map[string]any)
	}{
		{
			name: "basic pie chart",
			args: PieChartArgs{
				Title:        "Sales Distribution",
				DatasetLabel: "Revenue",
				Points: []PiePoint{
					{Label: "Q1", Value: 1000},
					{Label: "Q2", Value: 1500},
					{Label: "Q3", Value: 1200},
				},
			},
			validate: func(t *testing.T, result map[string]any) {
				if result["type"] != "pie" {
					t.Errorf("expected type=pie, got %v", result["type"])
				}

				data := result["data"].(map[string]any)
				labels := data["labels"].([]string)
				if len(labels) != 3 {
					t.Errorf("expected 3 labels, got %d", len(labels))
				}

				datasets := data["datasets"].([]map[string]any)
				if datasets[0]["label"] != "Revenue" {
					t.Errorf("expected label=Revenue, got %v", datasets[0]["label"])
				}
			},
		},
		{
			name: "pie chart without title",
			args: PieChartArgs{
				Points: []PiePoint{
					{Label: "A", Value: 25},
					{Label: "B", Value: 75},
				},
			},
			validate: func(t *testing.T, result map[string]any) {
				options := result["options"].(map[string]any)
				plugins := options["plugins"].(map[string]any)
				title := plugins["title"].(map[string]any)
				if title["display"] != false {
					t.Errorf("expected title display=false, got %v", title["display"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generatePieChartJSON(tt.args)
			tt.validate(t, result)

			_, err := json.Marshal(result)
			if err != nil {
				t.Errorf("result is not valid JSON: %v", err)
			}
		})
	}
}

func TestGeneratePolarAreaChartJSON(t *testing.T) {
	tests := []struct {
		name     string
		args     PolarAreaChartArgs
		validate func(t *testing.T, result map[string]any)
	}{
		{
			name: "basic polar area chart",
			args: PolarAreaChartArgs{
				Title:        "Metrics",
				DatasetLabel: "Performance",
				Points: []PolarAreaPoint{
					{Label: "Speed", Value: 11},
					{Label: "Efficiency", Value: 16},
					{Label: "Quality", Value: 7},
				},
			},
			validate: func(t *testing.T, result map[string]any) {
				if result["type"] != "polarArea" {
					t.Errorf("expected type=polarArea, got %v", result["type"])
				}

				data := result["data"].(map[string]any)
				labels := data["labels"].([]string)
				if len(labels) != 3 {
					t.Errorf("expected 3 labels, got %d", len(labels))
				}

				datasets := data["datasets"].([]map[string]any)
				dataPoints := datasets[0]["data"].([]float64)
				if dataPoints[0] != 11 || dataPoints[1] != 16 || dataPoints[2] != 7 {
					t.Errorf("unexpected data: %v", dataPoints)
				}
			},
		},
		{
			name: "polar area chart without dataset label",
			args: PolarAreaChartArgs{
				Points: []PolarAreaPoint{
					{Label: "A", Value: 5},
				},
			},
			validate: func(t *testing.T, result map[string]any) {
				data := result["data"].(map[string]any)
				datasets := data["datasets"].([]map[string]any)
				if datasets[0]["label"] != "Dataset" {
					t.Errorf("expected default label='Dataset', got %v", datasets[0]["label"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generatePolarAreaChartJSON(tt.args)
			tt.validate(t, result)

			_, err := json.Marshal(result)
			if err != nil {
				t.Errorf("result is not valid JSON: %v", err)
			}
		})
	}
}

func TestGenerateRadarChartJSON(t *testing.T) {
	tests := []struct {
		name     string
		args     RadarChartArgs
		validate func(t *testing.T, result map[string]any)
	}{
		{
			name: "basic radar chart",
			args: RadarChartArgs{
				Title:        "Skills Assessment",
				DatasetLabel: "Employee A",
				Points: []RadarPoint{
					{Label: "Communication", Value: 65},
					{Label: "Teamwork", Value: 59},
					{Label: "Technical", Value: 90},
				},
			},
			validate: func(t *testing.T, result map[string]any) {
				if result["type"] != "radar" {
					t.Errorf("expected type=radar, got %v", result["type"])
				}

				data := result["data"].(map[string]any)
				labels := data["labels"].([]string)
				if len(labels) != 3 {
					t.Errorf("expected 3 labels, got %d", len(labels))
				}

				datasets := data["datasets"].([]map[string]any)
				if datasets[0]["label"] != "Employee A" {
					t.Errorf("expected label=Employee A, got %v", datasets[0]["label"])
				}

				// check radar-specific scale
				options := result["options"].(map[string]any)
				scales := options["scales"].(map[string]any)
				r := scales["r"].(map[string]any)
				if r["beginAtZero"] != true {
					t.Error("expected r.beginAtZero=true for radar chart")
				}
			},
		},
		{
			name: "radar chart without dataset label",
			args: RadarChartArgs{
				Points: []RadarPoint{
					{Label: "A", Value: 10},
				},
			},
			validate: func(t *testing.T, result map[string]any) {
				data := result["data"].(map[string]any)
				datasets := data["datasets"].([]map[string]any)
				if datasets[0]["label"] != "Series 1" {
					t.Errorf("expected default label='Series 1', got %v", datasets[0]["label"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateRadarChartJSON(tt.args)
			tt.validate(t, result)

			_, err := json.Marshal(result)
			if err != nil {
				t.Errorf("result is not valid JSON: %v", err)
			}
		})
	}
}

func TestGenerateScatterChartJSON(t *testing.T) {
	tests := []struct {
		name     string
		args     ScatterChartArgs
		validate func(t *testing.T, result map[string]any)
	}{
		{
			name: "basic scatter chart",
			args: ScatterChartArgs{
				Title:        "Correlation Analysis",
				DatasetLabel: "Data Points",
				Points: []ScatterPoint{
					{X: -10, Y: 0},
					{X: 0, Y: 10},
					{X: 10, Y: 5},
					{X: 0.5, Y: 5.5},
				},
			},
			validate: func(t *testing.T, result map[string]any) {
				if result["type"] != "scatter" {
					t.Errorf("expected type=scatter, got %v", result["type"])
				}

				data := result["data"].(map[string]any)
				datasets := data["datasets"].([]map[string]any)
				dataPoints := datasets[0]["data"].([]map[string]float64)

				if len(dataPoints) != 4 {
					t.Errorf("expected 4 data points, got %d", len(dataPoints))
				}

				if dataPoints[0]["x"] != -10 || dataPoints[0]["y"] != 0 {
					t.Errorf("unexpected first point: %v", dataPoints[0])
				}

				// check scatter-specific scales
				options := result["options"].(map[string]any)
				scales := options["scales"].(map[string]any)
				x := scales["x"].(map[string]any)
				if x["type"] != "linear" {
					t.Errorf("expected x.type=linear, got %v", x["type"])
				}
				if x["position"] != "bottom" {
					t.Errorf("expected x.position=bottom, got %v", x["position"])
				}
			},
		},
		{
			name: "scatter chart without dataset label",
			args: ScatterChartArgs{
				Points: []ScatterPoint{
					{X: 1, Y: 2},
				},
			},
			validate: func(t *testing.T, result map[string]any) {
				data := result["data"].(map[string]any)
				datasets := data["datasets"].([]map[string]any)
				if datasets[0]["label"] != "Series 1" {
					t.Errorf("expected default label='Series 1', got %v", datasets[0]["label"])
				}
			},
		},
		{
			name: "negative coordinates",
			args: ScatterChartArgs{
				Points: []ScatterPoint{
					{X: -5, Y: -10},
					{X: -2.5, Y: 3.7},
				},
			},
			validate: func(t *testing.T, result map[string]any) {
				data := result["data"].(map[string]any)
				datasets := data["datasets"].([]map[string]any)
				dataPoints := datasets[0]["data"].([]map[string]float64)
				if dataPoints[0]["x"] != -5 || dataPoints[0]["y"] != -10 {
					t.Errorf("negative values not preserved: %v", dataPoints[0])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateScatterChartJSON(tt.args)
			tt.validate(t, result)

			_, err := json.Marshal(result)
			if err != nil {
				t.Errorf("result is not valid JSON: %v", err)
			}
		})
	}
}

func TestDataConsistency(t *testing.T) {
	t.Run("bar chart labels and data length match", func(t *testing.T) {
		args := BarChartArgs{
			Points: []BarPoint{
				{Label: "A", Value: 1},
				{Label: "B", Value: 2},
				{Label: "C", Value: 3},
			},
		}
		result := generateBarChartJSON(args)
		data := result["data"].(map[string]any)
		labels := data["labels"].([]string)
		datasets := data["datasets"].([]map[string]any)
		dataPoints := datasets[0]["data"].([]float64)

		if len(labels) != len(dataPoints) {
			t.Errorf("labels length (%d) != data length (%d)", len(labels), len(dataPoints))
		}
	})

	t.Run("line chart labels and data length match", func(t *testing.T) {
		args := LineChartArgs{
			Points: []LinePoint{
				{X: "A", Y: 1},
				{X: "B", Y: 2},
			},
		}
		result := generateLineChartJSON(args)
		data := result["data"].(map[string]any)
		labels := data["labels"].([]string)
		datasets := data["datasets"].([]map[string]any)
		dataPoints := datasets[0]["data"].([]float64)

		if len(labels) != len(dataPoints) {
			t.Errorf("labels length (%d) != data length (%d)", len(labels), len(dataPoints))
		}
	})

	t.Run("area chart labels and data length match", func(t *testing.T) {
		args := AreaChartArgs{
			Points: []AreaPoint{
				{X: "A", Y: 1},
				{X: "B", Y: 2},
			},
		}
		result := generateAreaChartJSON(args)
		data := result["data"].(map[string]any)
		labels := data["labels"].([]string)
		datasets := data["datasets"].([]map[string]any)
		dataPoints := datasets[0]["data"].([]float64)

		if len(labels) != len(dataPoints) {
			t.Errorf("labels length (%d) != data length (%d)", len(labels), len(dataPoints))
		}
	})

	t.Run("doughnut chart labels and data length match", func(t *testing.T) {
		args := DoughnutChartArgs{
			Points: []DoughnutPoint{
				{Label: "A", Value: 10},
				{Label: "B", Value: 20},
			},
		}
		result := generateDoughnutChartJSON(args)
		data := result["data"].(map[string]any)
		labels := data["labels"].([]string)
		datasets := data["datasets"].([]map[string]any)
		dataPoints := datasets[0]["data"].([]float64)

		if len(labels) != len(dataPoints) {
			t.Errorf("labels length (%d) != data length (%d)", len(labels), len(dataPoints))
		}
	})
}

func TestGenerateSequenceDiagramDSL(t *testing.T) {
	tests := []struct {
		name     string
		args     SequenceDiagramArgs
		validate func(t *testing.T, result string)
	}{
		{
			name: "basic sequence diagram with messages",
			args: SequenceDiagramArgs{
				Messages: []Message{
					{From: "Alice", To: "Bob", Text: "Hello Bob"},
					{From: "Bob", To: "Alice", Text: "Hi Alice"},
				},
			},
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "sequenceDiagram") {
					t.Error("missing sequenceDiagram keyword")
				}
				if !strings.Contains(result, "Alice->>Bob: Hello Bob") {
					t.Error("missing first message")
				}
				if !strings.Contains(result, "Bob->>Alice: Hi Alice") {
					t.Error("missing second message")
				}
			},
		},
		{
			name: "diagram with title and autonumber",
			args: SequenceDiagramArgs{
				Title:      "Login Flow",
				AutoNumber: true,
				Messages: []Message{
					{From: "User", To: "Server", Text: "Login request"},
					{From: "Server", To: "Database", Text: "Verify credentials"},
				},
			},
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "title: Login Flow") {
					t.Error("missing title")
				}
				if !strings.Contains(result, "autonumber") {
					t.Error("missing autonumber")
				}
			},
		},
		{
			name: "diagram with explicit participants",
			args: SequenceDiagramArgs{
				Participants: []Participant{
					{ID: "A", Label: "Alice", Type: "actor"},
					{ID: "B", Label: "Bob", Type: "participant"},
				},
				Messages: []Message{
					{From: "A", To: "B", Text: "Test"},
				},
			},
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "actor A as Alice") {
					t.Error("missing actor participant")
				}
				if !strings.Contains(result, "participant B as Bob") {
					t.Error("missing participant")
				}
			},
		},
		{
			name: "diagram with different arrow types",
			args: SequenceDiagramArgs{
				Messages: []Message{
					{From: "A", To: "B", Text: "Sync call", ArrowType: "->>"},
					{From: "B", To: "A", Text: "Response", ArrowType: "-->>"},
					{From: "A", To: "C", Text: "Async", ArrowType: "-)"},
					{From: "A", To: "D", Text: "Lost", ArrowType: "-x"},
				},
			},
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "A->>B: Sync call") {
					t.Error("missing sync arrow")
				}
				if !strings.Contains(result, "B-->>A: Response") {
					t.Error("missing dotted arrow")
				}
				if !strings.Contains(result, "A-)C: Async") {
					t.Error("missing async arrow")
				}
				if !strings.Contains(result, "A-xD: Lost") {
					t.Error("missing cross arrow")
				}
			},
		},
		{
			name: "diagram with activations",
			args: SequenceDiagramArgs{
				Messages: []Message{
					{From: "A", To: "B", Text: "Request", Activate: true},
					{From: "B", To: "A", Text: "Response", Deactivate: true},
				},
			},
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "A->>+B: Request") {
					t.Error("missing activation")
				}
				if !strings.Contains(result, "B->>-A: Response") {
					t.Error("missing deactivation")
				}
			},
		},
		{
			name: "diagram with notes",
			args: SequenceDiagramArgs{
				Messages: []Message{
					{From: "A", To: "B", Text: "Hello"},
				},
				Notes: []Note{
					{Position: "right of", Participants: []string{"A"}, Text: "Note on A"},
					{Position: "over", Participants: []string{"A", "B"}, Text: "Spanning note"},
				},
			},
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "Note right of A: Note on A") {
					t.Error("missing right of note")
				}
				if !strings.Contains(result, "Note over A,B: Spanning note") {
					t.Error("missing over note")
				}
			},
		},
		{
			name: "diagram with boxes",
			args: SequenceDiagramArgs{
				Participants: []Participant{
					{ID: "A"},
					{ID: "B"},
				},
				Boxes: []Box{
					{Label: "Backend", Color: "lightblue", Participants: []string{"A", "B"}},
				},
				Messages: []Message{
					{From: "A", To: "B", Text: "Internal call"},
				},
			},
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "box lightblue Backend") {
					t.Error("missing box definition")
				}
				if !strings.Contains(result, "end") {
					t.Error("missing box end")
				}
			},
		},
		{
			name: "diagram with loops",
			args: SequenceDiagramArgs{
				Messages: []Message{
					{From: "A", To: "B", Text: "Start"},
				},
				Loops: []Loop{
					{
						Text: "Every minute",
						Messages: []Message{
							{From: "A", To: "B", Text: "Ping"},
							{From: "B", To: "A", Text: "Pong"},
						},
					},
				},
			},
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "loop Every minute") {
					t.Error("missing loop definition")
				}
				if !strings.Contains(result, "A->>B: Ping") {
					t.Error("missing message in loop")
				}
			},
		},
		{
			name: "diagram with alt/else",
			args: SequenceDiagramArgs{
				Messages: []Message{
					{From: "User", To: "Server", Text: "Login"},
				},
				Alts: []Alt{
					{
						IfText: "Valid credentials",
						IfMessages: []Message{
							{From: "Server", To: "User", Text: "Success"},
						},
						ElseText: "Invalid credentials",
						ElseMessages: []Message{
							{From: "Server", To: "User", Text: "Error"},
						},
					},
				},
			},
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "alt Valid credentials") {
					t.Error("missing alt definition")
				}
				if !strings.Contains(result, "else Invalid credentials") {
					t.Error("missing else definition")
				}
			},
		},
		{
			name: "diagram with opt (no else)",
			args: SequenceDiagramArgs{
				Messages: []Message{
					{From: "A", To: "B", Text: "Request"},
				},
				Alts: []Alt{
					{
						IfText: "Cache exists",
						IfMessages: []Message{
							{From: "B", To: "Cache", Text: "Get from cache"},
						},
					},
				},
			},
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "alt Cache exists") {
					t.Error("missing alt definition")
				}
				if strings.Contains(result, "else") {
					t.Error("should not have else block")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateSequenceDiagramDSL(tt.args)
			fmt.Println("Generated DSL:\n", result)
			tt.validate(t, result)
		})
	}
}

func TestValidateSequenceDiagramArgs(t *testing.T) {
	t.Run("valid basic diagram", func(t *testing.T) {
		args := SequenceDiagramArgs{
			Messages: []Message{
				{From: "A", To: "B", Text: "Hello"},
			},
		}
		if err := validateSequenceDiagramArgs(args); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("empty messages", func(t *testing.T) {
		args := SequenceDiagramArgs{
			Messages: []Message{},
		}
		if err := validateSequenceDiagramArgs(args); err == nil {
			t.Error("expected error for empty messages")
		}
	})

	t.Run("message with empty from", func(t *testing.T) {
		args := SequenceDiagramArgs{
			Messages: []Message{
				{From: "", To: "B", Text: "Hello"},
			},
		}
		if err := validateSequenceDiagramArgs(args); err == nil {
			t.Error("expected error for empty from field")
		}
	})

	t.Run("message with empty to", func(t *testing.T) {
		args := SequenceDiagramArgs{
			Messages: []Message{
				{From: "A", To: "", Text: "Hello"},
			},
		}
		if err := validateSequenceDiagramArgs(args); err == nil {
			t.Error("expected error for empty to field")
		}
	})

	t.Run("participant with empty id", func(t *testing.T) {
		args := SequenceDiagramArgs{
			Participants: []Participant{
				{ID: "", Label: "Alice"},
			},
			Messages: []Message{
				{From: "A", To: "B", Text: "Test"},
			},
		}
		if err := validateSequenceDiagramArgs(args); err == nil {
			t.Error("expected error for empty participant ID")
		}
	})

	t.Run("note with unknown participant", func(t *testing.T) {
		args := SequenceDiagramArgs{
			Messages: []Message{
				{From: "A", To: "B", Text: "Test"},
			},
			Notes: []Note{
				{Position: "right of", Participants: []string{"C"}, Text: "Note"},
			},
		}
		if err := validateSequenceDiagramArgs(args); err == nil {
			t.Error("expected error for unknown participant in note")
		}
	})

	t.Run("note right of with multiple participants", func(t *testing.T) {
		args := SequenceDiagramArgs{
			Messages: []Message{
				{From: "A", To: "B", Text: "Test"},
			},
			Notes: []Note{
				{Position: "right of", Participants: []string{"A", "B"}, Text: "Note"},
			},
		}
		if err := validateSequenceDiagramArgs(args); err == nil {
			t.Error("expected error for right of with multiple participants")
		}
	})

	t.Run("note over with too many participants", func(t *testing.T) {
		args := SequenceDiagramArgs{
			Messages: []Message{
				{From: "A", To: "B", Text: "Test"},
			},
			Notes: []Note{
				{Position: "over", Participants: []string{"A", "B", "C"}, Text: "Note"},
			},
		}
		if err := validateSequenceDiagramArgs(args); err == nil {
			t.Error("expected error for over with too many participants")
		}
	})

	t.Run("box with unknown participant", func(t *testing.T) {
		args := SequenceDiagramArgs{
			Messages: []Message{
				{From: "A", To: "B", Text: "Test"},
			},
			Boxes: []Box{
				{Label: "Group", Participants: []string{"C"}},
			},
		}
		if err := validateSequenceDiagramArgs(args); err == nil {
			t.Error("expected error for unknown participant in box")
		}
	})

	t.Run("loop with empty text", func(t *testing.T) {
		args := SequenceDiagramArgs{
			Messages: []Message{
				{From: "A", To: "B", Text: "Test"},
			},
			Loops: []Loop{
				{
					Text: "",
					Messages: []Message{
						{From: "A", To: "B", Text: "Repeat"},
					},
				},
			},
		}
		if err := validateSequenceDiagramArgs(args); err == nil {
			t.Error("expected error for loop with empty text")
		}
	})

	t.Run("loop with no messages", func(t *testing.T) {
		args := SequenceDiagramArgs{
			Messages: []Message{
				{From: "A", To: "B", Text: "Test"},
			},
			Loops: []Loop{
				{
					Text:     "Loop",
					Messages: []Message{},
				},
			},
		}
		if err := validateSequenceDiagramArgs(args); err == nil {
			t.Error("expected error for loop with no messages")
		}
	})

	t.Run("alt with empty ifText", func(t *testing.T) {
		args := SequenceDiagramArgs{
			Messages: []Message{
				{From: "A", To: "B", Text: "Test"},
			},
			Alts: []Alt{
				{
					IfText: "",
					IfMessages: []Message{
						{From: "A", To: "B", Text: "Yes"},
					},
				},
			},
		}
		if err := validateSequenceDiagramArgs(args); err == nil {
			t.Error("expected error for alt with empty ifText")
		}
	})

	t.Run("alt with no ifMessages", func(t *testing.T) {
		args := SequenceDiagramArgs{
			Messages: []Message{
				{From: "A", To: "B", Text: "Test"},
			},
			Alts: []Alt{
				{
					IfText:     "Condition",
					IfMessages: []Message{},
				},
			},
		}
		if err := validateSequenceDiagramArgs(args); err == nil {
			t.Error("expected error for alt with no ifMessages")
		}
	})

	t.Run("alt with elseText but no elseMessages", func(t *testing.T) {
		args := SequenceDiagramArgs{
			Messages: []Message{
				{From: "A", To: "B", Text: "Test"},
			},
			Alts: []Alt{
				{
					IfText: "Condition",
					IfMessages: []Message{
						{From: "A", To: "B", Text: "Yes"},
					},
					ElseText:     "Otherwise",
					ElseMessages: []Message{},
				},
			},
		}
		if err := validateSequenceDiagramArgs(args); err == nil {
			t.Error("expected error for alt with elseText but no elseMessages")
		}
	})
}

func TestGenerateFlowchartDSL(t *testing.T) {
	tests := []struct {
		name     string
		args     FlowchartArgs
		validate func(t *testing.T, result string)
	}{
		{
			name: "basic flowchart with links",
			args: FlowchartArgs{
				Links: []FlowchartLink{
					{From: "A", To: "B"},
					{From: "B", To: "C"},
				},
			},
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "flowchart TB") {
					t.Error("missing flowchart keyword with default direction")
				}
				if !strings.Contains(result, "A --> B") {
					t.Error("missing first link")
				}
				if !strings.Contains(result, "B --> C") {
					t.Error("missing second link")
				}
			},
		},
		{
			name: "flowchart with custom direction",
			args: FlowchartArgs{
				Direction: "LR",
				Links: []FlowchartLink{
					{From: "Start", To: "End"},
				},
			},
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "flowchart LR") {
					t.Error("missing LR direction")
				}
			},
		},
		{
			name: "flowchart with explicit nodes and shapes",
			args: FlowchartArgs{
				Nodes: []FlowchartNode{
					{ID: "A", Label: "Start", Shape: "circle"},
					{ID: "B", Label: "Process", Shape: "rectangle"},
					{ID: "C", Label: "Decision", Shape: "diamond"},
					{ID: "D", Label: "End", Shape: "double-circle"},
				},
				Links: []FlowchartLink{
					{From: "A", To: "B"},
					{From: "B", To: "C"},
					{From: "C", To: "D"},
				},
			},
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "A((Start))") {
					t.Error("missing circle node")
				}
				if !strings.Contains(result, "B[Process]") {
					t.Error("missing rectangle node")
				}
				if !strings.Contains(result, "C{Decision}") {
					t.Error("missing diamond node")
				}
				if !strings.Contains(result, "D(((End)))") {
					t.Error("missing double-circle node")
				}
			},
		},
		{
			name: "flowchart with different arrow types",
			args: FlowchartArgs{
				Links: []FlowchartLink{
					{From: "A", To: "B", ArrowType: "-->"},
					{From: "B", To: "C", ArrowType: "---"},
					{From: "C", To: "D", ArrowType: "-.->"},
					{From: "D", To: "E", ArrowType: "==>"},
					{From: "E", To: "F", ArrowType: "--o"},
					{From: "F", To: "G", ArrowType: "--x"},
				},
			},
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "A --> B") {
					t.Error("missing solid arrow")
				}
				if !strings.Contains(result, "B --- C") {
					t.Error("missing open link")
				}
				if !strings.Contains(result, "C -.-> D") {
					t.Error("missing dotted arrow")
				}
				if !strings.Contains(result, "D ==> E") {
					t.Error("missing thick arrow")
				}
				if !strings.Contains(result, "E --o F") {
					t.Error("missing circle edge")
				}
				if !strings.Contains(result, "F --x G") {
					t.Error("missing cross edge")
				}
			},
		},
		{
			name: "flowchart with link text",
			args: FlowchartArgs{
				Links: []FlowchartLink{
					{From: "A", To: "B", Text: "yes"},
					{From: "A", To: "C", Text: "no"},
				},
			},
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "A -->|yes| B") {
					t.Error("missing link with yes text")
				}
				if !strings.Contains(result, "A -->|no| C") {
					t.Error("missing link with no text")
				}
			},
		},
		{
			name: "flowchart with extended link length",
			args: FlowchartArgs{
				Links: []FlowchartLink{
					{From: "A", To: "B", Length: 2},
					{From: "C", To: "D", ArrowType: "-.->", Length: 3},
				},
			},
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "A ----> B") {
					t.Error("missing extended solid arrow")
				}
				if !strings.Contains(result, "C -...-> D") {
					t.Error("missing extended dotted arrow")
				}
			},
		},
		{
			name: "flowchart with subgraph",
			args: FlowchartArgs{
				Nodes: []FlowchartNode{
					{ID: "A", Label: "Outside"},
					{ID: "B", Label: "Inside1"},
					{ID: "C", Label: "Inside2"},
				},
				Links: []FlowchartLink{
					{From: "A", To: "B"},
					{From: "B", To: "C"},
				},
				Subgraphs: []FlowchartSubgraph{
					{
						Title: "My Subgraph",
						Nodes: []string{"B", "C"},
					},
				},
			},
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "subgraph My Subgraph") {
					t.Error("missing subgraph definition")
				}
				if !strings.Contains(result, "end") {
					t.Error("missing subgraph end")
				}
			},
		},
		{
			name: "flowchart with subgraph ID and direction",
			args: FlowchartArgs{
				Nodes: []FlowchartNode{
					{ID: "A"},
					{ID: "B"},
				},
				Links: []FlowchartLink{
					{From: "A", To: "B"},
				},
				Subgraphs: []FlowchartSubgraph{
					{
						ID:        "sg1",
						Title:     "Group",
						Nodes:     []string{"A", "B"},
						Direction: "LR",
					},
				},
			},
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "subgraph sg1 [Group]") {
					t.Error("missing subgraph with ID")
				}
				if !strings.Contains(result, "direction LR") {
					t.Error("missing subgraph direction")
				}
			},
		},
		{
			name: "flowchart with styles",
			args: FlowchartArgs{
				Nodes: []FlowchartNode{
					{ID: "A", Label: "Styled Node"},
				},
				Links: []FlowchartLink{
					{From: "A", To: "B"},
				},
				Styles: []FlowchartStyle{
					{Target: "A", Properties: "fill:#f9f,stroke:#333"},
				},
			},
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "style A fill:#f9f,stroke:#333") {
					t.Error("missing style definition")
				}
			},
		},
		{
			name: "flowchart with class definitions",
			args: FlowchartArgs{
				Nodes: []FlowchartNode{
					{ID: "A"},
					{ID: "B"},
				},
				Links: []FlowchartLink{
					{From: "A", To: "B"},
				},
				ClassDefs: []FlowchartClassDef{
					{
						ClassName:  "highlight",
						Properties: "fill:#ff0,stroke:#f00",
						Nodes:      []string{"A", "B"},
					},
				},
			},
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "classDef highlight fill:#ff0,stroke:#f00") {
					t.Error("missing classDef definition")
				}
				if !strings.Contains(result, "class A,B highlight") {
					t.Error("missing class application")
				}
			},
		},
		{
			name: "flowchart with all node shapes",
			args: FlowchartArgs{
				Nodes: []FlowchartNode{
					{ID: "N1", Shape: "rectangle"},
					{ID: "N2", Shape: "round"},
					{ID: "N3", Shape: "stadium"},
					{ID: "N4", Shape: "subroutine"},
					{ID: "N5", Shape: "cylinder"},
					{ID: "N6", Shape: "circle"},
					{ID: "N7", Shape: "asymmetric"},
					{ID: "N8", Shape: "diamond"},
					{ID: "N9", Shape: "hexagon"},
					{ID: "N10", Shape: "parallelogram"},
					{ID: "N11", Shape: "trapezoid"},
					{ID: "N12", Shape: "double-circle"},
				},
				Links: []FlowchartLink{
					{From: "N1", To: "N2"},
				},
			},
			validate: func(t *testing.T, result string) {
				shapes := []string{
					"N1[N1]",       // rectangle
					"N2(N2)",       // round
					"N3([N3])",     // stadium
					"N4[[N4]]",     // subroutine
					"N5[(N5)]",     // cylinder
					"N6((N6))",     // circle
					"N7>N7]",       // asymmetric
					"N8{N8}",       // diamond
					"N9{{N9}}",     // hexagon
					"N10[/N10/]",   // parallelogram
					"N11[/N11\\]",  // trapezoid
					"N12(((N12)))", // double-circle
				}
				for _, shape := range shapes {
					if !strings.Contains(result, shape) {
						t.Errorf("missing shape: %s", shape)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateFlowchartDSL(tt.args)
			tt.validate(t, result)
		})
	}
}

func TestValidateFlowchartArgs(t *testing.T) {
	t.Run("valid basic flowchart", func(t *testing.T) {
		args := FlowchartArgs{
			Links: []FlowchartLink{
				{From: "A", To: "B"},
			},
		}
		if err := validateFlowchartArgs(args); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("empty links", func(t *testing.T) {
		args := FlowchartArgs{
			Links: []FlowchartLink{},
		}
		if err := validateFlowchartArgs(args); err == nil {
			t.Error("expected error for empty links")
		}
	})

	t.Run("invalid direction", func(t *testing.T) {
		args := FlowchartArgs{
			Direction: "INVALID",
			Links: []FlowchartLink{
				{From: "A", To: "B"},
			},
		}
		if err := validateFlowchartArgs(args); err == nil {
			t.Error("expected error for invalid direction")
		}
	})

	t.Run("link with empty from", func(t *testing.T) {
		args := FlowchartArgs{
			Links: []FlowchartLink{
				{From: "", To: "B"},
			},
		}
		if err := validateFlowchartArgs(args); err == nil {
			t.Error("expected error for empty from field")
		}
	})

	t.Run("link with empty to", func(t *testing.T) {
		args := FlowchartArgs{
			Links: []FlowchartLink{
				{From: "A", To: ""},
			},
		}
		if err := validateFlowchartArgs(args); err == nil {
			t.Error("expected error for empty to field")
		}
	})

	t.Run("node with empty id", func(t *testing.T) {
		args := FlowchartArgs{
			Nodes: []FlowchartNode{
				{ID: ""},
			},
			Links: []FlowchartLink{
				{From: "A", To: "B"},
			},
		}
		if err := validateFlowchartArgs(args); err == nil {
			t.Error("expected error for empty node ID")
		}
	})

	t.Run("subgraph with empty title", func(t *testing.T) {
		args := FlowchartArgs{
			Links: []FlowchartLink{
				{From: "A", To: "B"},
			},
			Subgraphs: []FlowchartSubgraph{
				{Title: "", Nodes: []string{"A"}},
			},
		}
		if err := validateFlowchartArgs(args); err == nil {
			t.Error("expected error for empty subgraph title")
		}
	})

	t.Run("subgraph with no nodes", func(t *testing.T) {
		args := FlowchartArgs{
			Links: []FlowchartLink{
				{From: "A", To: "B"},
			},
			Subgraphs: []FlowchartSubgraph{
				{Title: "Group", Nodes: []string{}},
			},
		}
		if err := validateFlowchartArgs(args); err == nil {
			t.Error("expected error for subgraph with no nodes")
		}
	})

	t.Run("subgraph with unknown node", func(t *testing.T) {
		args := FlowchartArgs{
			Links: []FlowchartLink{
				{From: "A", To: "B"},
			},
			Subgraphs: []FlowchartSubgraph{
				{Title: "Group", Nodes: []string{"C"}},
			},
		}
		if err := validateFlowchartArgs(args); err == nil {
			t.Error("expected error for subgraph with unknown node")
		}
	})

	t.Run("subgraph with invalid direction", func(t *testing.T) {
		args := FlowchartArgs{
			Links: []FlowchartLink{
				{From: "A", To: "B"},
			},
			Subgraphs: []FlowchartSubgraph{
				{Title: "Group", Nodes: []string{"A"}, Direction: "INVALID"},
			},
		}
		if err := validateFlowchartArgs(args); err == nil {
			t.Error("expected error for invalid subgraph direction")
		}
	})

	t.Run("style with empty target", func(t *testing.T) {
		args := FlowchartArgs{
			Links: []FlowchartLink{
				{From: "A", To: "B"},
			},
			Styles: []FlowchartStyle{
				{Target: "", Properties: "fill:#f9f"},
			},
		}
		if err := validateFlowchartArgs(args); err == nil {
			t.Error("expected error for style with empty target")
		}
	})

	t.Run("style with empty properties", func(t *testing.T) {
		args := FlowchartArgs{
			Links: []FlowchartLink{
				{From: "A", To: "B"},
			},
			Styles: []FlowchartStyle{
				{Target: "A", Properties: ""},
			},
		}
		if err := validateFlowchartArgs(args); err == nil {
			t.Error("expected error for style with empty properties")
		}
	})

	t.Run("classDef with empty class name", func(t *testing.T) {
		args := FlowchartArgs{
			Links: []FlowchartLink{
				{From: "A", To: "B"},
			},
			ClassDefs: []FlowchartClassDef{
				{ClassName: "", Properties: "fill:#f9f"},
			},
		}
		if err := validateFlowchartArgs(args); err == nil {
			t.Error("expected error for classDef with empty class name")
		}
	})

	t.Run("classDef with empty properties", func(t *testing.T) {
		args := FlowchartArgs{
			Links: []FlowchartLink{
				{From: "A", To: "B"},
			},
			ClassDefs: []FlowchartClassDef{
				{ClassName: "myClass", Properties: ""},
			},
		}
		if err := validateFlowchartArgs(args); err == nil {
			t.Error("expected error for classDef with empty properties")
		}
	})

	t.Run("classDef with unknown node", func(t *testing.T) {
		args := FlowchartArgs{
			Links: []FlowchartLink{
				{From: "A", To: "B"},
			},
			ClassDefs: []FlowchartClassDef{
				{ClassName: "myClass", Properties: "fill:#f9f", Nodes: []string{"C"}},
			},
		}
		if err := validateFlowchartArgs(args); err == nil {
			t.Error("expected error for classDef with unknown node")
		}
	})
}
