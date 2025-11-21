package main

import (
	"encoding/json"
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
