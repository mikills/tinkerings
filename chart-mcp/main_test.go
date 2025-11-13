package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestChartGenerate_Success(t *testing.T) {
	tests := []struct {
		name      string
		chartType string
		N         int
		D         int
		title     string
		specJSON  string
	}{
		{
			name:      "line chart",
			chartType: "line",
			N:         8,
			D:         2,
			title:     "Throughput",
			specJSON: `{
				"title": "Throughput",
				"type": "line",
				"labels": ["t00","t01","t02","t03","t04","t05","t06","t07"],
				"datasets": [
					{"label":"Requests","data":[10,12,14,16,18,20,22,24]},
					{"label":"Errors","data":[1,1,2,2,3,3,4,4]}
				]
			}`,
		},
		{
			name:      "bar chart",
			chartType: "bar",
			N:         4,
			D:         1,
			title:     "Revenue",
			specJSON: `{
				"title": "Revenue",
				"type": "bar",
				"labels": ["Q1","Q2","Q3","Q4"],
				"datasets": [{"label":"Rev","data":[100,120,140,160]}]
			}`,
		},
		{
			name:      "bar chart with code fences",
			chartType: "bar",
			N:         4,
			D:         1,
			title:     "Revenue",
			specJSON:  "```json\n{\n  \"title\": \"Revenue\",\n  \"type\": \"bar\",\n  \"labels\": [\"Q1\",\"Q2\",\"Q3\",\"Q4\"],\n  \"datasets\": [{\"label\":\"Rev\",\"data\":[100,120,140,160]}]\n}\n```",
		},
		{
			name:      "histogram",
			chartType: "histogram",
			N:         6,
			D:         1,
			title:     "Age Distribution",
			specJSON: `{
				"title": "Age Distribution",
				"type": "histogram",
				"labels": ["0-10","11-20","21-30","31-40","41-50","51-60"],
				"datasets": [{"label":"Frequency","data":[5,12,18,15,8,3]}]
			}`,
		},
		{
			name:      "scatter",
			chartType: "scatter",
			N:         5,
			D:         1,
			title:     "Test Scatter",
			specJSON: `{
				"type": "scatter",
				"title": "Test Scatter",
				"datasets": [{
					"label": "Series 1",
					"data": [
						{"x": 1.0, "y": 2.0},
						{"x": 2.0, "y": 3.0},
						{"x": 3.0, "y": 4.0},
						{"x": 4.0, "y": 5.0},
						{"x": 5.0, "y": 6.0}
					]
				}]
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := startMockOllama(t, tt.specJSON, 200)
			defer srv.Close()
			reset := withEnv(t, "OLLAMA_BASE_URL", srv.URL)
			defer reset()

			var schema map[string]any
			var jsonSchema any
			var sys, usr string

			switch tt.chartType {
			case "line":
				schema = schemaForLine(tt.N, tt.D)
				jsonSchema = jsonSchemaForLine(tt.N, tt.D)
				sys = systemPrompt(schema)
				usr = userPrompt(tt.chartType, tt.N, tt.D, tt.title)
			case "bar":
				schema = schemaForBar(tt.N, tt.D)
				jsonSchema = jsonSchemaForBar(tt.N, tt.D)
				sys = systemPrompt(schema)
				usr = userPrompt(tt.chartType, tt.N, tt.D, tt.title)
			case "histogram":
				schema = schemaForHistogram(tt.N)
				jsonSchema = jsonSchemaForHistogram(tt.N)
				sys = systemPrompt(schema)
				usr = userPrompt(tt.chartType, tt.N, 1, tt.title)
			case "scatter":
				schema = schemaForScatter(tt.N, tt.D)
				jsonSchema = jsonSchemaForScatter(tt.N, tt.D)
				sys = systemPrompt(schema)
				usr = userPromptScatter(tt.N, tt.D, tt.title)
			}

			spec, raw, err := generateWithModelAndFormat(sys, usr, "test-model", jsonSchema, tt.chartType, tt.N, tt.D, tt.title)
			if err != nil {
				t.Fatalf("generate error: %v\nraw: %s", err, raw)
			}

			if spec.Type != tt.chartType {
				t.Errorf("expected type %q, got %q", tt.chartType, spec.Type)
			}

			if tt.chartType != "scatter" && len(spec.Labels) != tt.N {
				t.Errorf("expected %d labels, got %d", tt.N, len(spec.Labels))
			}

			if len(spec.Datasets) != tt.D {
				t.Errorf("expected %d datasets, got %d", tt.D, len(spec.Datasets))
			}
		})
	}
}

func TestChartValidation_Errors(t *testing.T) {
	tests := []struct {
		name          string
		chartType     string
		N             int
		D             int
		specJSON      string
		wantErrSubstr string
	}{
		{
			name:      "type mismatch",
			chartType: "line",
			N:         6,
			D:         1,
			specJSON: `{
				"title": "Wrong type",
				"type": "bar",
				"labels": ["t00","t01","t02","t03","t04","t05"],
				"datasets": [{"label":"A","data":[1,2,3,4,5,6]}]
			}`,
			wantErrSubstr: "type mismatch",
		},
		{
			name:      "length mismatch",
			chartType: "line",
			N:         5,
			D:         1,
			specJSON: `{
				"title": "Bad lengths",
				"type": "line",
				"labels": ["t0","t1","t2","t3","t4"],
				"datasets": [{"label":"A","data":[1,2,3]}]
			}`,
			wantErrSubstr: "expected 5 values, got 3",
		},
		{
			name:      "negative values",
			chartType: "line",
			N:         4,
			D:         1,
			specJSON: `{
				"title": "Has negatives",
				"type": "line",
				"labels": ["t0","t1","t2","t3"],
				"datasets": [{"label":"A","data":[10, -1, 5, 6]}]
			}`,
			wantErrSubstr: ">= 0",
		},
		{
			name:      "wrong dataset count",
			chartType: "bar",
			N:         3,
			D:         2,
			specJSON: `{
				"type": "bar",
				"labels": ["A","B","C"],
				"datasets": [{"label":"Only One","data":[1,2,3]}]
			}`,
			wantErrSubstr: "expected 2 datasets, got 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := startMockOllama(t, tt.specJSON, 200)
			defer srv.Close()
			reset := withEnv(t, "OLLAMA_BASE_URL", srv.URL)
			defer reset()

			var schema map[string]any
			var jsonSchema any
			var sys, usr string

			switch tt.chartType {
			case "line":
				schema = schemaForLine(tt.N, tt.D)
				jsonSchema = jsonSchemaForLine(tt.N, tt.D)
				sys = systemPrompt(schema)
				usr = userPrompt(tt.chartType, tt.N, tt.D, "Test")
			case "bar":
				schema = schemaForBar(tt.N, tt.D)
				jsonSchema = jsonSchemaForBar(tt.N, tt.D)
				sys = systemPrompt(schema)
				usr = userPrompt(tt.chartType, tt.N, tt.D, "Test")
			}

			_, raw, err := generateWithModelAndFormat(sys, usr, "test-model", jsonSchema, tt.chartType, tt.N, tt.D, "Test")
			if err == nil {
				t.Fatalf("expected validation error, got nil\nraw: %s", raw)
			}

			if !strings.Contains(err.Error(), tt.wantErrSubstr) {
				t.Errorf("expected error containing %q, got: %v", tt.wantErrSubstr, err)
			}
		})
	}
}

func TestMermaidGenerate_Success(t *testing.T) {
	tests := []struct {
		name        string
		diagramType string
		nodes       int
		edges       int
		direction   string
		description string
		mermaidCode string
	}{
		{
			name:        "sequence diagram",
			diagramType: "sequence",
			nodes:       3,
			edges:       4,
			description: "Simple chat",
			mermaidCode: `sequenceDiagram
    participant A as Alice
    participant B as Bob
    participant C as Charlie
    A->>B: Hello Bob
    B->>C: Hi Charlie
    C->>A: Hey Alice
    A->>B: How are you?`,
		},
		{
			name:        "flowchart",
			diagramType: "flowchart",
			nodes:       4,
			edges:       3,
			direction:   "TD",
			description: "Simple process",
			mermaidCode: `flowchart TD
    A[Start] --> B{Decision}
    B --> C[Process]
    C --> D[End]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := startMockOllama(t, tt.mermaidCode, 200)
			defer srv.Close()
			reset := withEnv(t, "OLLAMA_BASE_URL", srv.URL)
			defer reset()

			var schema map[string]any
			var sys, usr string
			var validator func(string) error

			if tt.diagramType == "sequence" {
				schema = schemaForSequence(tt.nodes, tt.edges)
				sys = mermaidSystemPrompt(schema)
				usr = sequencePrompt(tt.nodes, tt.edges, tt.description)
				validator = func(content string) error {
					return validateMermaidSequence(content, tt.nodes, tt.edges)
				}
			} else {
				schema = schemaForFlowchart(tt.nodes, tt.edges, tt.direction)
				sys = mermaidSystemPrompt(schema)
				usr = flowchartPrompt(tt.nodes, tt.edges, tt.direction, tt.description)
				validator = func(content string) error {
					return validateMermaidFlowchart(content, tt.nodes, tt.edges, tt.direction)
				}
			}

			mermaid, raw, err := generateMermaid(sys, usr, "test-model", validator)
			if err != nil {
				t.Fatalf("generateMermaid error: %v\nraw: %s", err, raw)
			}

			expectedPrefix := tt.diagramType
			if tt.diagramType == "flowchart" {
				expectedPrefix = "flowchart"
			} else {
				expectedPrefix = "sequenceDiagram"
			}

			if !strings.Contains(mermaid, expectedPrefix) {
				t.Errorf("expected mermaid to contain %q, got: %s", expectedPrefix, mermaid)
			}
		})
	}
}

func TestMermaidValidation_Errors(t *testing.T) {
	tests := []struct {
		name          string
		diagramType   string
		nodes         int
		edges         int
		direction     string
		mermaidCode   string
		wantErrSubstr string
	}{
		{
			name:        "sequence wrong participant count",
			diagramType: "sequence",
			nodes:       2,
			edges:       2,
			mermaidCode: `sequenceDiagram
    participant A as Alice
    A->>A: Hello
    A->>A: World`,
			wantErrSubstr: "expected 2 participants",
		},
		{
			name:        "flowchart wrong node count",
			diagramType: "flowchart",
			nodes:       3,
			edges:       2,
			direction:   "LR",
			mermaidCode: `flowchart LR
    A[Start] --> B[End]`,
			wantErrSubstr: "expected 3 nodes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := startMockOllama(t, tt.mermaidCode, 200)
			defer srv.Close()
			reset := withEnv(t, "OLLAMA_BASE_URL", srv.URL)
			defer reset()

			var schema map[string]any
			var sys, usr string
			var validator func(string) error

			if tt.diagramType == "sequence" {
				schema = schemaForSequence(tt.nodes, tt.edges)
				sys = mermaidSystemPrompt(schema)
				usr = sequencePrompt(tt.nodes, tt.edges, "Test")
				validator = func(content string) error {
					return validateMermaidSequence(content, tt.nodes, tt.edges)
				}
			} else {
				schema = schemaForFlowchart(tt.nodes, tt.edges, tt.direction)
				sys = mermaidSystemPrompt(schema)
				usr = flowchartPrompt(tt.nodes, tt.edges, tt.direction, "Test")
				validator = func(content string) error {
					return validateMermaidFlowchart(content, tt.nodes, tt.edges, tt.direction)
				}
			}

			_, raw, err := generateMermaid(sys, usr, "test-model", validator)
			if err == nil {
				t.Fatalf("expected validation error, got nil\nraw: %s", raw)
			}

			if !strings.Contains(err.Error(), tt.wantErrSubstr) {
				t.Errorf("expected error containing %q, got: %v", tt.wantErrSubstr, err)
			}
		})
	}
}

func startMockOllama(t *testing.T, content string, status int) *httptest.Server {
	t.Helper()
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(status)
		if status >= 200 && status < 300 {
			resp := map[string]any{
				"message": map[string]any{
					"content": content,
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		_, _ = w.Write([]byte(`{"error":"boom"}`))
	})
	return httptest.NewServer(h)
}

func withEnv(t *testing.T, key, val string) func() {
	t.Helper()
	old, had := os.LookupEnv(key)
	_ = os.Setenv(key, val)
	return func() {
		if had {
			_ = os.Setenv(key, old)
		} else {
			_ = os.Unsetenv(key)
		}
	}
}
