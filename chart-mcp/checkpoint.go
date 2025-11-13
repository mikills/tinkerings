// go run . -model gemma3:4b -type line -points 24 -datasets 2 -title "Throughput (req/s)"
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"
)

var (
	endpoint = flag.String("endpoint", "http://localhost:11434", "Ollama base URL")
	model    = flag.String("model", "gemma3:4b", "Ollama model name")
	ctype    = flag.String("type", "line", "chart type (line|bar|pie|doughnut|radar|polarArea|scatter|bubble)")
	points   = flag.Int("points", 24, "number of points (N)")
	datasets = flag.Int("datasets", 2, "number of datasets")
	title    = flag.String("title", "", "optional chart title")
	timeout  = flag.Duration("timeout", 45*time.Second, "HTTP timeout")
)

func checkpoint() {
	flag.Parse()
	if *ctype == "pie" || *ctype == "doughnut" {
		*datasets = 1
	}

	spec, raw, err := getFromModel()
	if err != nil {
		fail("ollama error: %v", err)
	}

	if isEmpty(spec) {
		fail("model returned empty or malformed data.\n--- raw ---\n%s", raw)
	}

	out, _ := json.MarshalIndent(spec, "", "  ")
	fmt.Println(string(out))
	if err := os.WriteFile("chart_spec.json", out, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write chart_spec.json: %v\n", err)
	}
}

func getFromModel() (ChartSpec, string, error) {
	req := chatRequest{
		Model:  *model,
		Stream: false,
		Messages: []chatMessage{
			{
				Role: "system",
				Content: `You are a strict JSON generator for chart data.
Return ONLY a valid JSON object matching:
{
  "title": string (optional),
  "type": "line"|"bar"|"pie"|"doughnut"|"radar"|"polarArea"|"scatter"|"bubble",
  "labels": string[] (omit for scatter/bubble),
  "datasets": [ {"label": string, "data": number[]} ]
}
No markdown, no commentary.`,
			},
			{
				Role: "user",
				Content: fmt.Sprintf(`Generate a ChartSpec for:
- type: %q
- points: %d
- datasets: %d
- title: %q
Ensure JSON is valid, complete, and not empty.`,
					*ctype, *points, *datasets, *title),
			},
		},
	}

	jsonStr, raw, err := callOllama(req)
	if err != nil {
		return ChartSpec{}, raw, err
	}

	var spec ChartSpec
	if json.Unmarshal([]byte(jsonStr), &spec) != nil {
		if alt, ok := extractFirstJSON(raw); ok {
			jsonStr = alt
		}
	}
	if err := json.Unmarshal([]byte(jsonStr), &spec); err != nil {
		return ChartSpec{}, raw, fmt.Errorf("decode JSON: %w\n--- raw ---\n%s", err, raw)
	}

	// Normalise essentials
	if spec.Type == "" {
		spec.Type = *ctype
	}
	if spec.Title == "" && *title != "" {
		spec.Title = *title
	}
	return spec, raw, nil
}

func isEmpty(spec ChartSpec) bool {
	if len(spec.Datasets) == 0 {
		return true
	}
	for _, ds := range spec.Datasets {
		b, _ := json.Marshal(ds.Data)
		var arr []any
		if json.Unmarshal(b, &arr) != nil || len(arr) == 0 {
			return true
		}
	}
	if spec.Type == "" {
		return true
	}
	return false
}

func fail(msg string, a ...any) {
	fmt.Fprintf(os.Stderr, msg+"\n", a...)
	os.Exit(1)
}
