package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
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

		if len(tools) != 2 {
			t.Errorf("expected 2 tools, got %d", len(tools))
		}

		// verify bar chart tool exists
		foundBar := false
		foundLine := false
		for _, tool := range tools {
			toolMap := tool.(map[string]any)
			name := toolMap["name"].(string)
			if name == "bar-chart-generator" {
				foundBar = true
				t.Logf("Found tool: %s - %s", name, toolMap["description"])
			}
			if name == "line-chart-generator" {
				foundLine = true
				t.Logf("Found tool: %s - %s", name, toolMap["description"])
			}
		}

		if !foundBar {
			t.Error("bar-chart-generator tool not found")
		}
		if !foundLine {
			t.Error("line-chart-generator tool not found")
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
