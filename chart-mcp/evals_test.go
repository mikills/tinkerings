package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type evalRequest struct {
	ChartType string `json:"chart_type"`
	Points    int    `json:"points"`
	Datasets  int    `json:"datasets"`
	Title     string `json:"title"`
}

type evalResult struct {
	Request       evalRequest `json:"request"`
	Model         string      `json:"model"`
	Success       bool        `json:"success"`
	ErrorMsg      string      `json:"error_msg,omitempty"`
	GeneratedSpec *ChartSpec  `json:"generated_spec,omitempty"`
	RawResponse   string      `json:"raw_response,omitempty"`
	DurationMs    int64       `json:"duration_ms"`
}

type evalReport struct {
	Timestamp     string       `json:"timestamp"`
	Model         string       `json:"model"`
	OllamaBaseURL string       `json:"ollama_base_url"`
	TotalRequests int          `json:"total_requests"`
	SuccessCount  int          `json:"success_count"`
	FailureCount  int          `json:"failure_count"`
	AvgDurationMs int64        `json:"avg_duration_ms"`
	Results       []evalResult `json:"results"`
}

func runEvalTests(t *testing.T, testCases []evalRequest, filePrefix string) {
	model := defaultModel()
	if m := os.Getenv("OLLAMA_MODEL"); m != "" {
		model = m
	}

	results := make([]evalResult, 0, len(testCases))
	var totalDuration int64
	var successCount int

	for i, tc := range testCases {
		start := time.Now()
		var result evalResult
		result.Request = tc
		result.Model = model

		var err error
		var raw string

		switch tc.ChartType {
		case "line", "bar", "histogram", "scatter":
			var schema map[string]any
			var jsonSchema any
			var sys, usr string

			switch tc.ChartType {
			case "line":
				schema = schemaForLine(tc.Points, tc.Datasets)
				jsonSchema = jsonSchemaForLine(tc.Points, tc.Datasets)
				sys = systemPrompt(schema)
				usr = userPrompt(tc.ChartType, tc.Points, tc.Datasets, tc.Title)
			case "bar":
				schema = schemaForBar(tc.Points, tc.Datasets)
				jsonSchema = jsonSchemaForBar(tc.Points, tc.Datasets)
				sys = systemPrompt(schema)
				usr = userPrompt(tc.ChartType, tc.Points, tc.Datasets, tc.Title)
			case "histogram":
				schema = schemaForHistogram(tc.Points)
				jsonSchema = jsonSchemaForHistogram(tc.Points)
				sys = systemPrompt(schema)
				usr = userPrompt(tc.ChartType, tc.Points, 1, tc.Title)
			case "scatter":
				schema = schemaForScatter(tc.Points, tc.Datasets)
				jsonSchema = jsonSchemaForScatter(tc.Points, tc.Datasets)
				sys = systemPrompt(schema)
				usr = userPromptScatter(tc.Points, tc.Datasets, tc.Title)
			}

			var spec ChartSpec
			spec, raw, err = generateWithModelAndFormat(sys, usr, model, jsonSchema, tc.ChartType, tc.Points, tc.Datasets, tc.Title)
			if err == nil {
				result.GeneratedSpec = &spec
			}

		case "sequence", "flowchart":
			var schema map[string]any
			var sys, usr string
			var validator func(string) error

			if tc.ChartType == "sequence" {
				schema = schemaForSequence(tc.Points, tc.Datasets)
				sys = mermaidSystemPrompt(schema)
				usr = sequencePrompt(tc.Points, tc.Datasets, tc.Title)
				validator = func(content string) error {
					return validateMermaidSequence(content, tc.Points, tc.Datasets)
				}
			} else {
				schema = schemaForFlowchart(tc.Points, tc.Datasets, "TD")
				sys = mermaidSystemPrompt(schema)
				usr = flowchartPrompt(tc.Points, tc.Datasets, "TD", tc.Title)
				validator = func(content string) error {
					return validateMermaidFlowchart(content, tc.Points, tc.Datasets, "TD")
				}
			}

			var mermaidCode string
			mermaidCode, raw, err = generateMermaid(sys, usr, model, validator)
			if err == nil {
				result.RawResponse = mermaidCode
			}
		}

		elapsed := time.Since(start)
		result.DurationMs = elapsed.Milliseconds()
		totalDuration += result.DurationMs

		if err != nil {
			result.Success = false
			result.ErrorMsg = err.Error()
			if result.RawResponse == "" {
				result.RawResponse = raw
			}
		} else {
			result.Success = true
			successCount++
		}

		results = append(results, result)
		t.Logf("[%d/%d] %s (N=%d,D=%d): %v (%dms)",
			i+1, len(testCases), tc.ChartType, tc.Points, tc.Datasets,
			map[bool]string{true: "✓", false: "✗"}[result.Success],
			result.DurationMs)
	}

	avgDuration := totalDuration / int64(len(testCases))
	report := evalReport{
		Timestamp:     time.Now().Format(time.RFC3339),
		Model:         model,
		OllamaBaseURL: ollamaBase(),
		TotalRequests: len(testCases),
		SuccessCount:  successCount,
		FailureCount:  len(testCases) - successCount,
		AvgDurationMs: avgDuration,
		Results:       results,
	}

	// ensure eval_results directory exists
	if err := os.MkdirAll("eval_results", 0755); err != nil {
		t.Fatalf("failed to create eval_results directory: %v", err)
	}

	filename := fmt.Sprintf("%s_%s.json", filePrefix, time.Now().Format("20060102_150405"))
	filepath := filepath.Join("eval_results", filename)
	b, _ := json.MarshalIndent(report, "", "  ")
	if err := os.WriteFile(filepath, b, 0644); err != nil {
		t.Fatalf("failed to write eval results: %v", err)
	}

	successRate := float64(successCount) / float64(len(testCases)) * 100
	t.Logf("\n=== Summary ===")
	t.Logf("Total: %d | Success: %d | Failure: %d | Rate: %.1f%% | Avg: %dms",
		report.TotalRequests, report.SuccessCount, report.FailureCount,
		successRate, report.AvgDurationMs)
	t.Logf("Results: %s", filepath)

	if successCount == 0 {
		t.Fatal("all eval requests failed")
	}
}

func TestEval_AllChartTypes(t *testing.T) {
	if os.Getenv("EVAL_MODE") == "" {
		t.Skip("skipping eval test; set EVAL_MODE=1 to run")
	}

	testCases := []evalRequest{
		{"line", 5, 1, "CPU Usage"},
		{"line", 8, 2, "Throughput and Errors"},
		{"bar", 4, 1, "Quarterly Revenue"},
		{"bar", 6, 2, "Sales vs Returns"},
		{"line", 10, 3, "Network Metrics"},
		{"histogram", 6, 1, "Age Distribution"},
		{"histogram", 8, 1, "Response Time Buckets"},
		{"scatter", 10, 1, "Temperature vs Pressure"},
		{"scatter", 15, 2, "Height vs Weight by Gender"},
		{"sequence", 3, 5, "User Login Flow"},
		{"sequence", 4, 8, "API Request Sequence"},
		{"flowchart", 5, 4, "Decision Process"},
		{"flowchart", 6, 7, "Data Pipeline"},
	}

	runEvalTests(t, testCases, "eval_results")
}

func TestEval_ScatterOnly(t *testing.T) {
	if os.Getenv("EVAL_MODE") == "" {
		t.Skip("skipping eval test; set EVAL_MODE=1 to run")
	}

	testCases := []evalRequest{
		{"scatter", 5, 1, "Simple Scatter"},
		{"scatter", 10, 1, "Temperature vs Pressure"},
		{"scatter", 15, 1, "Single Dataset Large"},
		{"scatter", 8, 2, "Two Datasets Medium"},
		{"scatter", 12, 2, "Two Datasets Large"},
		{"scatter", 15, 2, "Height vs Weight by Gender"},
		{"scatter", 20, 1, "Very Large Single"},
		{"scatter", 10, 3, "Three Datasets"},
	}

	runEvalTests(t, testCases, "eval_scatter_results")
}

func TestEval_MermaidOnly(t *testing.T) {
	if os.Getenv("EVAL_MODE") == "" {
		t.Skip("skipping eval test; set EVAL_MODE=1 to run")
	}

	testCases := []evalRequest{
		{"sequence", 2, 3, "Simple Login Flow"},
		{"sequence", 3, 5, "API Request Sequence"},
		{"sequence", 4, 6, "Payment Processing"},
		{"flowchart", 3, 2, "Simple Decision"},
		{"flowchart", 5, 4, "Data Pipeline"},
		{"flowchart", 6, 7, "Complex Workflow"},
	}

	runEvalTests(t, testCases, "eval_mermaid_results")
}
