package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestMCPServerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	buildCmd := exec.Command("go", "build", "-o", "mcp-server-test", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("failed to build server: %v", err)
	}
	defer os.Remove("mcp-server-test")

	cmd := exec.Command("./mcp-server-test")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("failed to get stdin pipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("failed to get stdout pipe: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("failed to get stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer func() {
		stdin.Close()
		cmd.Process.Kill()
		cmd.Wait()
	}()

	// capture stderr for debugging
	go io.Copy(os.Stderr, stderr)

	// create reader for responses
	reader := bufio.NewReader(stdout)

	t.Run("initialize server", func(t *testing.T) {
		req := map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "initialize",
			"params": map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]any{},
				"clientInfo": map[string]any{
					"name":    "test-client",
					"version": "1.0.0",
				},
			},
		}

		if err := sendRequest(stdin, req); err != nil {
			t.Fatalf("failed to send initialize: %v", err)
		}

		resp, err := readResponse(reader)
		if err != nil {
			t.Fatalf("failed to read initialize response: %v", err)
		}

		if resp["error"] != nil {
			t.Fatalf("initialize returned error: %v", resp["error"])
		}

		result := resp["result"].(map[string]any)
		serverInfo := result["serverInfo"].(map[string]any)
		if serverInfo["name"] != "chartjs-generator" {
			t.Errorf("unexpected server name: %v", serverInfo["name"])
		}

		t.Logf("Server initialized: %s v%s", serverInfo["name"], serverInfo["version"])
	})

	t.Run("list tools", func(t *testing.T) {
		req := map[string]any{
			"jsonrpc": "2.0",
			"id":      2,
			"method":  "tools/list",
		}

		if err := sendRequest(stdin, req); err != nil {
			t.Fatalf("failed to send tools/list: %v", err)
		}

		resp, err := readResponse(reader)
		if err != nil {
			t.Fatalf("failed to read tools/list response: %v", err)
		}

		if resp["error"] != nil {
			t.Fatalf("tools/list returned error: %v", resp["error"])
		}

		result := resp["result"].(map[string]any)
		tools := result["tools"].([]any)

		if len(tools) != 10 {
			t.Errorf("expected 10 tools, got %d", len(tools))
		}

		// verify all chart tools exist
		expectedTools := []string{
			"area-chart-generator",
			"bar-chart-generator",
			"doughnut-chart-generator",
			"flowchart-generator",
			"line-chart-generator",
			"pie-chart-generator",
			"polar-area-chart-generator",
			"radar-chart-generator",
			"scatter-chart-generator",
			"sequence-diagram-generator",
		}

		foundTools := make(map[string]bool)
		for _, tool := range tools {
			toolMap := tool.(map[string]any)
			name := toolMap["name"].(string)
			foundTools[name] = true
			t.Logf("Found tool: %s - %s", name, toolMap["description"])
		}

		for _, expected := range expectedTools {
			if !foundTools[expected] {
				t.Errorf("%s tool not found", expected)
			}
		}
	})

	t.Run("call bar chart tool", func(t *testing.T) {
		req := map[string]any{
			"jsonrpc": "2.0",
			"id":      3,
			"method":  "tools/call",
			"params": map[string]any{
				"name": "bar-chart-generator",
				"arguments": map[string]any{
					"title":        "Q1 Sales",
					"datasetLabel": "Revenue",
					"points": []map[string]any{
						{"label": "Jan", "value": 100},
						{"label": "Feb", "value": 150},
						{"label": "Mar", "value": 200},
					},
				},
			},
		}

		if err := sendRequest(stdin, req); err != nil {
			t.Fatalf("failed to send tools/call: %v", err)
		}

		resp, err := readResponse(reader)
		if err != nil {
			t.Fatalf("failed to read tools/call response: %v", err)
		}

		if resp["error"] != nil {
			t.Fatalf("tools/call returned error: %v", resp["error"])
		}

		result := resp["result"].(map[string]any)
		content := result["content"].([]any)
		if len(content) == 0 {
			t.Fatal("no content in response")
		}

		contentItem := content[0].(map[string]any)
		if contentItem["type"] != "text" {
			t.Errorf("expected text content, got %v", contentItem["type"])
		}

		// parse the chart JSON from response
		textContent := contentItem["text"].(string)
		var chartJSON map[string]any
		if err := json.Unmarshal([]byte(textContent), &chartJSON); err != nil {
			t.Fatalf("failed to parse chart JSON: %v", err)
		}

		// validate chart structure
		if chartJSON["type"] != "bar" {
			t.Errorf("expected type=bar, got %v", chartJSON["type"])
		}

		data := chartJSON["data"].(map[string]any)
		labels := data["labels"].([]any)
		if len(labels) != 3 {
			t.Errorf("expected 3 labels, got %d", len(labels))
		}

		t.Logf("Successfully generated bar chart with %d data points", len(labels))
	})

	t.Run("call line chart tool", func(t *testing.T) {
		req := map[string]any{
			"jsonrpc": "2.0",
			"id":      4,
			"method":  "tools/call",
			"params": map[string]any{
				"name": "line-chart-generator",
				"arguments": map[string]any{
					"title":        "Temperature",
					"datasetLabel": "Celsius",
					"points": []map[string]any{
						{"x": "Mon", "y": 20},
						{"x": "Tue", "y": 22},
					},
				},
			},
		}

		if err := sendRequest(stdin, req); err != nil {
			t.Fatalf("failed to send tools/call: %v", err)
		}

		resp, err := readResponse(reader)
		if err != nil {
			t.Fatalf("failed to read tools/call response: %v", err)
		}

		if resp["error"] != nil {
			t.Fatalf("tools/call returned error: %v", resp["error"])
		}

		result := resp["result"].(map[string]any)
		content := result["content"].([]any)
		contentItem := content[0].(map[string]any)
		textContent := contentItem["text"].(string)

		var chartJSON map[string]any
		if err := json.Unmarshal([]byte(textContent), &chartJSON); err != nil {
			t.Fatalf("failed to parse chart JSON: %v", err)
		}

		if chartJSON["type"] != "line" {
			t.Errorf("expected type=line, got %v", chartJSON["type"])
		}

		t.Logf("Successfully generated line chart")
	})

	t.Run("call with invalid arguments", func(t *testing.T) {
		req := map[string]any{
			"jsonrpc": "2.0",
			"id":      5,
			"method":  "tools/call",
			"params": map[string]any{
				"name": "bar-chart-generator",
				"arguments": map[string]any{
					"points": []map[string]any{},
				},
			},
		}

		if err := sendRequest(stdin, req); err != nil {
			t.Fatalf("failed to send tools/call: %v", err)
		}

		resp, err := readResponse(reader)
		if err != nil {
			t.Fatalf("failed to read tools/call response: %v", err)
		}

		result := resp["result"].(map[string]any)

		isError, hasIsError := result["isError"]
		if !hasIsError {
			t.Logf("Response structure: %+v", result)
			t.Fatal("expected isError field in result")
		}

		if isErrorBool, ok := isError.(bool); !ok || !isErrorBool {
			t.Error("expected error result for empty points")
		}

		// check that content explains the error
		content := result["content"].([]any)
		if len(content) > 0 {
			contentItem := content[0].(map[string]any)
			errorText := contentItem["text"].(string)
			t.Logf("Error message: %s", errorText)

			if errorText != "points must contain at least one item" {
				t.Errorf("unexpected error message: %s", errorText)
			}
		}

		t.Logf("Correctly rejected invalid arguments")
	})

	t.Run("call area chart tool", func(t *testing.T) {
		req := map[string]any{
			"jsonrpc": "2.0",
			"id":      6,
			"method":  "tools/call",
			"params": map[string]any{
				"name": "area-chart-generator",
				"arguments": map[string]any{
					"title":        "Temperature",
					"datasetLabel": "Celsius",
					"points": []map[string]any{
						{"x": "Mon", "y": 20},
						{"x": "Tue", "y": 22},
					},
				},
			},
		}

		if err := sendRequest(stdin, req); err != nil {
			t.Fatalf("failed to send tools/call: %v", err)
		}

		resp, err := readResponse(reader)
		if err != nil {
			t.Fatalf("failed to read tools/call response: %v", err)
		}

		if resp["error"] != nil {
			t.Fatalf("tools/call returned error: %v", resp["error"])
		}

		result := resp["result"].(map[string]any)
		content := result["content"].([]any)
		contentItem := content[0].(map[string]any)
		textContent := contentItem["text"].(string)

		var chartJSON map[string]any
		if err := json.Unmarshal([]byte(textContent), &chartJSON); err != nil {
			t.Fatalf("failed to parse chart JSON: %v", err)
		}

		if chartJSON["type"] != "line" {
			t.Errorf("expected type=line, got %v", chartJSON["type"])
		}

		// verify fill is enabled for area chart
		data := chartJSON["data"].(map[string]any)
		datasets := data["datasets"].([]any)
		dataset := datasets[0].(map[string]any)
		if dataset["fill"] != true {
			t.Error("expected fill=true for area chart")
		}

		t.Logf("Successfully generated area chart")
	})

	t.Run("call doughnut chart tool", func(t *testing.T) {
		req := map[string]any{
			"jsonrpc": "2.0",
			"id":      7,
			"method":  "tools/call",
			"params": map[string]any{
				"name": "doughnut-chart-generator",
				"arguments": map[string]any{
					"title":        "Market Share",
					"datasetLabel": "Products",
					"points": []map[string]any{
						{"label": "Product A", "value": 300},
						{"label": "Product B", "value": 50},
						{"label": "Product C", "value": 100},
					},
				},
			},
		}

		if err := sendRequest(stdin, req); err != nil {
			t.Fatalf("failed to send tools/call: %v", err)
		}

		resp, err := readResponse(reader)
		if err != nil {
			t.Fatalf("failed to read tools/call response: %v", err)
		}

		if resp["error"] != nil {
			t.Fatalf("tools/call returned error: %v", resp["error"])
		}

		result := resp["result"].(map[string]any)
		content := result["content"].([]any)
		contentItem := content[0].(map[string]any)
		textContent := contentItem["text"].(string)

		var chartJSON map[string]any
		if err := json.Unmarshal([]byte(textContent), &chartJSON); err != nil {
			t.Fatalf("failed to parse chart JSON: %v", err)
		}

		if chartJSON["type"] != "doughnut" {
			t.Errorf("expected type=doughnut, got %v", chartJSON["type"])
		}

		t.Logf("Successfully generated doughnut chart")
	})

	t.Run("call pie chart tool", func(t *testing.T) {
		req := map[string]any{
			"jsonrpc": "2.0",
			"id":      8,
			"method":  "tools/call",
			"params": map[string]any{
				"name": "pie-chart-generator",
				"arguments": map[string]any{
					"title":        "Sales Distribution",
					"datasetLabel": "Revenue",
					"points": []map[string]any{
						{"label": "Q1", "value": 1000},
						{"label": "Q2", "value": 1500},
					},
				},
			},
		}

		if err := sendRequest(stdin, req); err != nil {
			t.Fatalf("failed to send tools/call: %v", err)
		}

		resp, err := readResponse(reader)
		if err != nil {
			t.Fatalf("failed to read tools/call response: %v", err)
		}

		if resp["error"] != nil {
			t.Fatalf("tools/call returned error: %v", resp["error"])
		}

		result := resp["result"].(map[string]any)
		content := result["content"].([]any)
		contentItem := content[0].(map[string]any)
		textContent := contentItem["text"].(string)

		var chartJSON map[string]any
		if err := json.Unmarshal([]byte(textContent), &chartJSON); err != nil {
			t.Fatalf("failed to parse chart JSON: %v", err)
		}

		if chartJSON["type"] != "pie" {
			t.Errorf("expected type=pie, got %v", chartJSON["type"])
		}

		t.Logf("Successfully generated pie chart")
	})

	t.Run("call polar area chart tool", func(t *testing.T) {
		req := map[string]any{
			"jsonrpc": "2.0",
			"id":      9,
			"method":  "tools/call",
			"params": map[string]any{
				"name": "polar-area-chart-generator",
				"arguments": map[string]any{
					"title":        "Performance Metrics",
					"datasetLabel": "Scores",
					"points": []map[string]any{
						{"label": "Speed", "value": 11},
						{"label": "Efficiency", "value": 16},
						{"label": "Quality", "value": 7},
					},
				},
			},
		}

		if err := sendRequest(stdin, req); err != nil {
			t.Fatalf("failed to send tools/call: %v", err)
		}

		resp, err := readResponse(reader)
		if err != nil {
			t.Fatalf("failed to read tools/call response: %v", err)
		}

		if resp["error"] != nil {
			t.Fatalf("tools/call returned error: %v", resp["error"])
		}

		result := resp["result"].(map[string]any)
		content := result["content"].([]any)
		contentItem := content[0].(map[string]any)
		textContent := contentItem["text"].(string)

		var chartJSON map[string]any
		if err := json.Unmarshal([]byte(textContent), &chartJSON); err != nil {
			t.Fatalf("failed to parse chart JSON: %v", err)
		}

		if chartJSON["type"] != "polarArea" {
			t.Errorf("expected type=polarArea, got %v", chartJSON["type"])
		}

		t.Logf("Successfully generated polar area chart")
	})

	t.Run("call radar chart tool", func(t *testing.T) {
		req := map[string]any{
			"jsonrpc": "2.0",
			"id":      10,
			"method":  "tools/call",
			"params": map[string]any{
				"name": "radar-chart-generator",
				"arguments": map[string]any{
					"title":        "Skills Assessment",
					"datasetLabel": "Employee A",
					"points": []map[string]any{
						{"label": "Communication", "value": 65},
						{"label": "Teamwork", "value": 59},
						{"label": "Technical", "value": 90},
					},
				},
			},
		}

		if err := sendRequest(stdin, req); err != nil {
			t.Fatalf("failed to send tools/call: %v", err)
		}

		resp, err := readResponse(reader)
		if err != nil {
			t.Fatalf("failed to read tools/call response: %v", err)
		}

		if resp["error"] != nil {
			t.Fatalf("tools/call returned error: %v", resp["error"])
		}

		result := resp["result"].(map[string]any)
		content := result["content"].([]any)
		contentItem := content[0].(map[string]any)
		textContent := contentItem["text"].(string)

		var chartJSON map[string]any
		if err := json.Unmarshal([]byte(textContent), &chartJSON); err != nil {
			t.Fatalf("failed to parse chart JSON: %v", err)
		}

		if chartJSON["type"] != "radar" {
			t.Errorf("expected type=radar, got %v", chartJSON["type"])
		}

		t.Logf("Successfully generated radar chart")
	})

	t.Run("call scatter chart tool", func(t *testing.T) {
		req := map[string]any{
			"jsonrpc": "2.0",
			"id":      11,
			"method":  "tools/call",
			"params": map[string]any{
				"name": "scatter-chart-generator",
				"arguments": map[string]any{
					"title":        "Correlation Analysis",
					"datasetLabel": "Data Points",
					"points": []map[string]any{
						{"x": -10, "y": 0},
						{"x": 0, "y": 10},
						{"x": 10, "y": 5},
					},
				},
			},
		}

		if err := sendRequest(stdin, req); err != nil {
			t.Fatalf("failed to send tools/call: %v", err)
		}

		resp, err := readResponse(reader)
		if err != nil {
			t.Fatalf("failed to read tools/call response: %v", err)
		}

		if resp["error"] != nil {
			t.Fatalf("tools/call returned error: %v", resp["error"])
		}

		result := resp["result"].(map[string]any)
		content := result["content"].([]any)
		contentItem := content[0].(map[string]any)
		textContent := contentItem["text"].(string)

		var chartJSON map[string]any
		if err := json.Unmarshal([]byte(textContent), &chartJSON); err != nil {
			t.Fatalf("failed to parse chart JSON: %v", err)
		}

		if chartJSON["type"] != "scatter" {
			t.Errorf("expected type=scatter, got %v", chartJSON["type"])
		}

		// verify scatter-specific data structure
		data := chartJSON["data"].(map[string]any)
		datasets := data["datasets"].([]any)
		dataset := datasets[0].(map[string]any)
		dataPoints := dataset["data"].([]any)
		firstPoint := dataPoints[0].(map[string]any)
		if firstPoint["x"] != float64(-10) || firstPoint["y"] != float64(0) {
			t.Errorf("unexpected scatter point structure: %v", firstPoint)
		}

		t.Logf("Successfully generated scatter chart")
	})

	t.Run("call sequence diagram tool", func(t *testing.T) {
		req := map[string]any{
			"jsonrpc": "2.0",
			"id":      12,
			"method":  "tools/call",
			"params": map[string]any{
				"name": "sequence-diagram-generator",
				"arguments": map[string]any{
					"title":      "API Authentication Flow",
					"autoNumber": true,
					"participants": []map[string]any{
						{"id": "Client", "type": "actor"},
						{"id": "API", "type": "participant"},
						{"id": "DB", "type": "database"},
					},
					"messages": []map[string]any{
						{"from": "Client", "to": "API", "text": "POST /login", "activate": true},
						{"from": "API", "to": "DB", "text": "Query user", "arrowType": "->>"},
						{"from": "DB", "to": "API", "text": "User data", "arrowType": "-->>"},
						{"from": "API", "to": "Client", "text": "JWT token", "deactivate": true},
					},
					"notes": []map[string]any{
						{"position": "right of", "participants": []string{"Client"}, "text": "User enters credentials"},
						{"position": "over", "participants": []string{"API", "DB"}, "text": "Authentication logic"},
					},
				},
			},
		}

		if err := sendRequest(stdin, req); err != nil {
			t.Fatalf("failed to send tools/call: %v", err)
		}

		resp, err := readResponse(reader)
		if err != nil {
			t.Fatalf("failed to read tools/call response: %v", err)
		}

		if resp["error"] != nil {
			t.Fatalf("tools/call returned error: %v", resp["error"])
		}

		result := resp["result"].(map[string]any)
		content := result["content"].([]any)
		contentItem := content[0].(map[string]any)

		if contentItem["type"] != "text" {
			t.Errorf("expected content type=text, got %v", contentItem["type"])
		}

		dslText := contentItem["text"].(string)

		// verify mermaid DSL structure
		if len(dslText) == 0 {
			t.Fatal("DSL text is empty")
		}

		// check for essential mermaid elements
		requiredElements := []string{
			"sequenceDiagram",
			"title: API Authentication Flow",
			"autonumber",
			"actor Client",
			"participant API",
			"database DB",
			"Client->>+API: POST /login",
			"API->>DB: Query user",
			"DB-->>API: User data",
			"API->>-Client: JWT token",
			"Note right of Client: User enters credentials",
			"Note over API,DB: Authentication logic",
		}

		for _, elem := range requiredElements {
			found := false
			for i := 0; i <= len(dslText)-len(elem); i++ {
				if dslText[i:i+len(elem)] == elem {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("DSL missing required element: %s\nGenerated DSL:\n%s", elem, dslText)
			}
		}

		t.Logf("Successfully generated sequence diagram DSL:\n%s", dslText)
	})

	t.Run("call flowchart tool", func(t *testing.T) {
		req := map[string]any{
			"jsonrpc": "2.0",
			"id":      13,
			"method":  "tools/call",
			"params": map[string]any{
				"name": "flowchart-generator",
				"arguments": map[string]any{
					"direction": "LR",
					"nodes": []map[string]any{
						{"id": "Start", "label": "Begin", "shape": "circle"},
						{"id": "Process", "label": "Execute Task", "shape": "rectangle"},
						{"id": "Decision", "label": "Success?", "shape": "diamond"},
						{"id": "End", "label": "Finish", "shape": "double-circle"},
					},
					"links": []map[string]any{
						{"from": "Start", "to": "Process"},
						{"from": "Process", "to": "Decision"},
						{"from": "Decision", "to": "End", "text": "yes", "arrowType": "-->"},
						{"from": "Decision", "to": "Process", "text": "no", "arrowType": "-.->"},
					},
					"styles": []map[string]any{
						{"target": "Start", "properties": "fill:#90EE90"},
						{"target": "End", "properties": "fill:#FFB6C1"},
					},
				},
			},
		}

		if err := sendRequest(stdin, req); err != nil {
			t.Fatalf("failed to send tools/call: %v", err)
		}

		resp, err := readResponse(reader)
		if err != nil {
			t.Fatalf("failed to read tools/call response: %v", err)
		}

		if resp["error"] != nil {
			t.Fatalf("tools/call returned error: %v", resp["error"])
		}

		result := resp["result"].(map[string]any)
		content := result["content"].([]any)
		contentItem := content[0].(map[string]any)

		if contentItem["type"] != "text" {
			t.Errorf("expected content type=text, got %v", contentItem["type"])
		}

		dslText := contentItem["text"].(string)

		// verify mermaid DSL structure
		if len(dslText) == 0 {
			t.Fatal("DSL text is empty")
		}

		// check for essential mermaid flowchart elements
		requiredElements := []string{
			"flowchart LR",
			"Start((Begin))",
			"Process[Execute Task]",
			"Decision{Success?}",
			"End(((Finish)))",
			"Start --> Process",
			"Process --> Decision",
			"Decision -->|yes| End",
			"Decision -.->|no| Process",
			"style Start fill:#90EE90",
			"style End fill:#FFB6C1",
		}

		for _, elem := range requiredElements {
			if !strings.Contains(dslText, elem) {
				t.Errorf("DSL missing required element: %s\nGenerated DSL:\n%s", elem, dslText)
			}
		}

		t.Logf("Successfully generated flowchart DSL:\n%s", dslText)
	})
}

func sendRequest(w io.Writer, req map[string]any) error {
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "%s\n", data)
	return err
}

func readResponse(r *bufio.Reader) (map[string]any, error) {
	// set timeout
	type result struct {
		data map[string]any
		err  error
	}
	resultChan := make(chan result, 1)

	go func() {
		line, err := r.ReadBytes('\n')
		if err != nil {
			resultChan <- result{nil, err}
			return
		}

		var resp map[string]any
		if err := json.Unmarshal(line, &resp); err != nil {
			resultChan <- result{nil, fmt.Errorf("failed to unmarshal response: %w\n%s", err, string(line))}
			return
		}

		resultChan <- result{resp, nil}
	}()

	select {
	case res := <-resultChan:
		return res.data, res.err
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("timeout waiting for response")
	}
}
