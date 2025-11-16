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
			if _, ok := data["labels"]; !ok {
				t.Error("missing data.labels")
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
			if _, ok := options["scales"]; !ok {
				t.Error("missing options.scales")
			}

			// verify scales.y.beginAtZero
			scales := options["scales"].(map[string]any)
			y := scales["y"].(map[string]any)
			if beginAtZero, ok := y["beginAtZero"]; !ok || beginAtZero != true {
				t.Error("missing or invalid scales.y.beginAtZero")
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
}
