package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

/* ==============================
   CORE TYPES
   ============================== */

type ChartSpec struct {
	Title    string         `json:"title,omitempty"`
	Type     string         `json:"type"`
	Labels   []string       `json:"labels,omitempty"`
	Datasets []Dataset      `json:"datasets"`
	Options  map[string]any `json:"options,omitempty"`
	Plugins  map[string]any `json:"plugins,omitempty"`
}

type Dataset struct {
	Label string `json:"label"`
	Data  any    `json:"data"`
}

type ChartResult struct {
	Schema map[string]any `json:"schema"`
	Spec   ChartSpec      `json:"spec"`
}

type MermaidResult struct {
	Schema      map[string]any `json:"schema"`
	DiagramType string         `json:"diagram_type"`
	Mermaid     string         `json:"mermaid"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
	Format   any           `json:"format,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
}

/* ==============================
   CONFIGURATION & HTTP
   ============================== */

func ollamaBase() string {
	if v := os.Getenv("OLLAMA_BASE_URL"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "http://localhost:11434"
}

func defaultModel() string {
	if v := os.Getenv("OLLAMA_MODEL"); v != "" {
		return v
	}
	return "gemma3:4b"
}

var httpTimeout = 45 * time.Second

func callOllama(payload chatRequest) (content, raw string, err error) {
	url := ollamaBase() + "/api/chat"
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: httpTimeout}

	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var r chatResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return "", string(body), err
	}
	return stripCodeFences(r.Message.Content), r.Message.Content, nil
}

/* ==============================
   SHARED UTILITIES
   ============================== */

func stripCodeFences(s string) string {
	if strings.Contains(s, "```") {
		re := regexp.MustCompile("(?s)```[a-zA-Z]*\\n(.*?)```")
		if m := re.FindStringSubmatch(s); len(m) == 2 {
			return strings.TrimSpace(m[1])
		}
	}
	return strings.TrimSpace(s)
}

func extractFirstJSON(s string) (string, bool) {
	start := strings.IndexByte(s, '{')
	if start < 0 {
		return "", false
	}
	depth := 0
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1], true
			}
		}
	}
	return "", false
}

func toAnySlice(v any) ([]any, bool) {
	switch t := v.(type) {
	case []any:
		return t, true
	default:
		b, _ := json.Marshal(v)
		var a []any
		if json.Unmarshal(b, &a) != nil {
			return nil, false
		}
		return a, true
	}
}

func finiteNonNeg(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		if t < 0 {
			return 0, false
		}
		return t, !math.IsNaN(t) && !math.IsInf(t, 0)
	case int, int32, int64:
		n := float64(any(t).(int64))
		if n < 0 {
			return 0, false
		}
		return n, true
	default:
		b, _ := json.Marshal(v)
		var f float64
		if json.Unmarshal(b, &f) == nil && f >= 0 && !math.IsNaN(f) && !math.IsInf(f, 0) {
			return f, true
		}
		return 0, false
	}
}

func finiteNumber(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, !math.IsNaN(t) && !math.IsInf(t, 0)
	case int, int32, int64:
		return float64(any(t).(int64)), true
	default:
		b, _ := json.Marshal(v)
		var f float64
		if json.Unmarshal(b, &f) == nil && !math.IsNaN(f) && !math.IsInf(f, 0) {
			return f, true
		}
		return 0, false
	}
}

/* ==============================
   CHART DOMAIN
   ============================== */

// --- chart schemas (human-readable) ---

func schemaForLine(N, D int) map[string]any {
	return map[string]any{
		"name":        "line",
		"description": "A value series over an ordered domain. Requires labels (length N) and D datasets of numeric length N.",
		"fields": []map[string]any{
			{"name": "labels", "type": "string[]", "length": N, "required": true, "desc": "x-axis categories/time ticks"},
			{"name": "datasets[].label", "type": "string", "required": true},
			{"name": "datasets[].data", "type": "number[]", "length": N, "required": true},
		},
		"constraints": []string{
			fmt.Sprintf("labels length == %d", N),
			fmt.Sprintf("datasets length == %d", D),
			fmt.Sprintf("each datasets[i].data length == %d", N),
			"all numbers finite and >= 0",
		},
		"example": map[string]any{
			"title": "Throughput (req/s)",
			"type":  "line",
			"labels": func() []string {
				out := make([]string, N)
				for i := range out {
					out[i] = fmt.Sprintf("t%02d", i)
				}
				return out
			}(),
			"datasets": []map[string]any{
				{"label": "Requests", "data": make([]int, N)},
				{"label": "Errors", "data": make([]int, N)},
			},
		},
	}
}

func schemaForBar(N, D int) map[string]any {
	return map[string]any{
		"name":        "bar",
		"description": "Categorical comparison. Requires labels (length N) and D datasets of numeric length N.",
		"fields": []map[string]any{
			{"name": "labels", "type": "string[]", "length": N, "required": true, "desc": "categories"},
			{"name": "datasets[].label", "type": "string", "required": true},
			{"name": "datasets[].data", "type": "number[]", "length": N, "required": true},
		},
		"constraints": []string{
			fmt.Sprintf("labels length == %d", N),
			fmt.Sprintf("datasets length == %d", D),
			fmt.Sprintf("each datasets[i].data length == %d", N),
			"all numbers finite and >= 0",
		},
		"example": map[string]any{
			"title":    "Revenue by Quarter",
			"type":     "bar",
			"labels":   []string{"Q1", "Q2", "Q3", "Q4"},
			"datasets": []map[string]any{{"label": "Revenue", "data": []int{100, 120, 140, 160}}},
		},
	}
}

func schemaForHistogram(N int) map[string]any {
	return map[string]any{
		"name":        "histogram",
		"description": "Frequency distribution. Requires labels (bins, length N) and 1 dataset with frequencies (length N).",
		"fields": []map[string]any{
			{"name": "labels", "type": "string[]", "length": N, "required": true, "desc": "bin ranges or categories"},
			{"name": "datasets[0].label", "type": "string", "required": true, "desc": "dataset label (typically 'Frequency')"},
			{"name": "datasets[0].data", "type": "number[]", "length": N, "required": true, "desc": "frequency counts"},
		},
		"constraints": []string{
			fmt.Sprintf("labels length == %d", N),
			"datasets length == 1",
			fmt.Sprintf("datasets[0].data length == %d", N),
			"all numbers finite and >= 0",
		},
		"example": map[string]any{
			"title":  "Age Distribution",
			"type":   "histogram",
			"labels": []string{"0-10", "11-20", "21-30", "31-40"},
			"datasets": []map[string]any{
				{"label": "Frequency", "data": []int{5, 12, 8, 3}},
			},
		},
	}
}

func schemaForScatter(N, D int) map[string]any {
	return map[string]any{
		"name":        "scatter",
		"description": "Point cloud visualization. Requires D datasets, each with N points as {x, y} objects.",
		"fields": []map[string]any{
			{"name": "datasets[].label", "type": "string", "required": true},
			{"name": "datasets[].data", "type": "object[]", "length": N, "required": true, "desc": "array of {x: number, y: number}"},
			{"name": "datasets[].data[].x", "type": "number", "required": true},
			{"name": "datasets[].data[].y", "type": "number", "required": true},
		},
		"constraints": []string{
			fmt.Sprintf("datasets length == %d", D),
			fmt.Sprintf("each datasets[i].data length == %d", N),
			"each point must have x and y properties",
			"all x and y values must be finite",
		},
		"example": map[string]any{
			"title": "Temperature vs Pressure",
			"type":  "scatter",
			"datasets": []map[string]any{
				{
					"label": "Experiment 1",
					"data": []map[string]any{
						{"x": 10, "y": 20},
						{"x": 15, "y": 25},
						{"x": 20, "y": 30},
					},
				},
			},
		},
	}
}

// --- chart json schemas (for ollama format field) ---

func jsonSchemaForLine(N, D int) map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"title": map[string]any{"type": "string"},
			"type":  map[string]any{"type": "string", "enum": []string{"line"}},
			"labels": map[string]any{
				"type":     "array",
				"items":    map[string]any{"type": "string"},
				"minItems": N,
				"maxItems": N,
			},
			"datasets": map[string]any{
				"type":     "array",
				"minItems": D,
				"maxItems": D,
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"label": map[string]any{"type": "string"},
						"data": map[string]any{
							"type":     "array",
							"items":    map[string]any{"type": "number", "minimum": 0},
							"minItems": N,
							"maxItems": N,
						},
					},
					"required": []string{"label", "data"},
				},
			},
		},
		"required": []string{"type", "labels", "datasets"},
	}
}

func jsonSchemaForBar(N, D int) map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"title": map[string]any{"type": "string"},
			"type":  map[string]any{"type": "string", "enum": []string{"bar"}},
			"labels": map[string]any{
				"type":     "array",
				"items":    map[string]any{"type": "string"},
				"minItems": N,
				"maxItems": N,
			},
			"datasets": map[string]any{
				"type":     "array",
				"minItems": D,
				"maxItems": D,
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"label": map[string]any{"type": "string"},
						"data": map[string]any{
							"type":     "array",
							"items":    map[string]any{"type": "number", "minimum": 0},
							"minItems": N,
							"maxItems": N,
						},
					},
					"required": []string{"label", "data"},
				},
			},
		},
		"required": []string{"type", "labels", "datasets"},
	}
}

func jsonSchemaForHistogram(N int) map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"title": map[string]any{"type": "string"},
			"type":  map[string]any{"type": "string", "enum": []string{"histogram"}},
			"labels": map[string]any{
				"type":     "array",
				"items":    map[string]any{"type": "string"},
				"minItems": N,
				"maxItems": N,
			},
			"datasets": map[string]any{
				"type":     "array",
				"minItems": 1,
				"maxItems": 1,
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"label": map[string]any{"type": "string"},
						"data": map[string]any{
							"type":     "array",
							"items":    map[string]any{"type": "number", "minimum": 0},
							"minItems": N,
							"maxItems": N,
						},
					},
					"required": []string{"label", "data"},
				},
			},
		},
		"required": []string{"type", "labels", "datasets"},
	}
}

func jsonSchemaForScatter(N, D int) map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"title": map[string]any{"type": "string"},
			"type":  map[string]any{"type": "string", "enum": []string{"scatter"}},
			"datasets": map[string]any{
				"type":     "array",
				"minItems": D,
				"maxItems": D,
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"label": map[string]any{"type": "string"},
						"data": map[string]any{
							"type":     "array",
							"minItems": N,
							"maxItems": N,
							"items": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"x": map[string]any{"type": "number"},
									"y": map[string]any{"type": "number"},
								},
								"required": []string{"x", "y"},
							},
						},
					},
					"required": []string{"label", "data"},
				},
			},
		},
		"required": []string{"type", "datasets"},
	}
}

// --- chart prompts ---

func systemPrompt(schema map[string]any) string {
	b, _ := json.MarshalIndent(schema, "", "  ")
	return "You output ONLY a single valid JSON object for chart data.\n\n" +
		"Contract (ChartSpec) and constraints:\n" + string(b) + `

Return JSON with the exact fields described above:
{
  "title"?: string,
  "type": "line"|"bar",
  "labels": string[],
  "datasets": [ { "label": string, "data": number[] } ]
}
Rules:
- JSON only (no markdown, no commentary).
- Use finite numbers only (no NaN/Infinity) and values >= 0.
- Obey lengths and dataset counts exactly.`
}

func userPrompt(chartType string, N, D int, title string) string {
	return fmt.Sprintf(
		`Generate ChartSpec with:
- "type": %q
- number of points N: %d
- number of datasets D: %d
- title (optional): %q

HARD CONSTRAINTS:
- Provide "labels" with EXACTLY N strings.
- Provide EXACTLY D datasets.
- Each dataset "data" MUST have EXACTLY N numbers.
- Values must be finite and >= 0.

Return ONLY the JSON object.`, chartType, N, D, title)
}

func userPromptScatter(N, D int, title string) string {
	return fmt.Sprintf(
		`Generate ChartSpec for scatter plot with:
- "type": "scatter"
- number of points per dataset N: %d
- number of datasets D: %d
- title (optional): %q

HARD CONSTRAINTS:
- DO NOT provide "labels" field (scatter doesn't use it).
- Provide EXACTLY D datasets.
- Each dataset "data" MUST be an array of EXACTLY N objects.
- Each object MUST have "x" and "y" properties (numbers).
- All x and y values must be finite (no NaN/Infinity).

Example structure:
{
  "type": "scatter",
  "title": "...",
  "datasets": [
    {
      "label": "Series 1",
      "data": [{"x": 1.5, "y": 2.3}, {"x": 2.1, "y": 3.7}, ...]
    }
  ]
}

Return ONLY the JSON object.`, N, D, title)
}

// --- chart validation ---

func strictValidate(spec ChartSpec, ctype string, N, D int) error {
	if spec.Type == "" {
		return errors.New(`missing "type"`)
	}
	if spec.Type != ctype {
		return fmt.Errorf(`type mismatch: expected %q, got %q`, ctype, spec.Type)
	}

	if ctype != "scatter" {
		if len(spec.Labels) != N {
			return fmt.Errorf("labels length must be %d, got %d", N, len(spec.Labels))
		}
	}

	if len(spec.Datasets) != D {
		return fmt.Errorf("expected %d datasets, got %d", D, len(spec.Datasets))
	}

	if ctype == "scatter" {
		return validateScatter(spec, N)
	}

	for i, ds := range spec.Datasets {
		vals, ok := toAnySlice(ds.Data)
		if !ok {
			return fmt.Errorf("dataset %d: data must be an array", i)
		}
		if len(vals) != N {
			return fmt.Errorf("dataset %d: expected %d values, got %d", i, N, len(vals))
		}
		for j, v := range vals {
			if n, ok := finiteNonNeg(v); !ok || math.IsNaN(n) || math.IsInf(n, 0) {
				return fmt.Errorf("dataset %d value %d must be finite and >= 0", i, j)
			}
		}
	}
	return nil
}

func validateScatter(spec ChartSpec, N int) error {
	for i, ds := range spec.Datasets {
		vals, ok := toAnySlice(ds.Data)
		if !ok {
			return fmt.Errorf("dataset %d: data must be an array", i)
		}
		if len(vals) != N {
			return fmt.Errorf("dataset %d: expected %d points, got %d", i, N, len(vals))
		}
		for j, v := range vals {
			point, ok := v.(map[string]any)
			if !ok {
				return fmt.Errorf("dataset %d point %d must be an object with x and y", i, j)
			}
			x, xok := point["x"]
			y, yok := point["y"]
			if !xok || !yok {
				return fmt.Errorf("dataset %d point %d missing x or y", i, j)
			}
			if xf, ok := finiteNumber(x); !ok || math.IsNaN(xf) || math.IsInf(xf, 0) {
				return fmt.Errorf("dataset %d point %d: x must be finite", i, j)
			}
			if yf, ok := finiteNumber(y); !ok || math.IsNaN(yf) || math.IsInf(yf, 0) {
				return fmt.Errorf("dataset %d point %d: y must be finite", i, j)
			}
		}
	}
	return nil
}

// --- chart generation ---

func generateWithModelAndFormat(sysPrompt, userPrompt, model string, jsonSchema any, chartType string, N, D int, title string) (ChartSpec, string, error) {
	req := chatRequest{
		Model:  model,
		Stream: false,
		Format: jsonSchema,
		Messages: []chatMessage{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: userPrompt},
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
		return ChartSpec{}, raw, fmt.Errorf("decode JSON: %w", err)
	}

	if spec.Type == "" {
		spec.Type = chartType
	}
	if spec.Title == "" && title != "" {
		spec.Title = title
	}

	if err := strictValidate(spec, chartType, N, D); err != nil {
		return ChartSpec{}, raw, err
	}
	return spec, raw, nil
}

/* ==============================
   MERMAID DOMAIN
   ============================== */

// --- mermaid schemas ---

func schemaForSequence(participants int, interactions int) map[string]any {
	return map[string]any{
		"name":        "sequence",
		"description": "Mermaid sequence diagram showing interactions between participants",
		"fields": []map[string]any{
			{"name": "participants", "type": "string[]", "count": participants, "desc": "list of participant names"},
			{"name": "interactions", "type": "string[]", "count": interactions, "desc": "message/interaction lines"},
		},
		"constraints": []string{
			fmt.Sprintf("must have %d participants", participants),
			fmt.Sprintf("must have %d interactions", interactions),
			"must start with 'sequenceDiagram'",
			"use participant declarations",
			"use arrows: ->, -->>, -x, etc.",
		},
		"example": `sequenceDiagram
    participant A as Alice
    participant B as Bob
    A->>B: Hello Bob!
    B->>A: Hi Alice!`,
	}
}

func schemaForFlowchart(nodes int, edges int, direction string) map[string]any {
	if direction == "" {
		direction = "TD"
	}
	return map[string]any{
		"name":        "flowchart",
		"description": "Mermaid flowchart diagram with nodes and connecting edges",
		"fields": []map[string]any{
			{"name": "nodes", "type": "node[]", "count": nodes, "desc": "flowchart nodes with shapes"},
			{"name": "edges", "type": "edge[]", "count": edges, "desc": "connections between nodes"},
			{"name": "direction", "type": "string", "value": direction, "desc": "TD, LR, BT, or RL"},
		},
		"constraints": []string{
			fmt.Sprintf("must have %d nodes", nodes),
			fmt.Sprintf("must have %d edges", edges),
			fmt.Sprintf("direction: %s", direction),
			"must start with 'flowchart <direction>'",
			"use node shapes: [], (), {}, [()]",
			"use arrows: -->, -.->",
		},
		"example": fmt.Sprintf(`flowchart %s
    A[Start] --> B{Decision}
    B -->|Yes| C[Process]
    B -->|No| D[End]`, direction),
	}
}

// --- mermaid prompts ---

func mermaidSystemPrompt(schema map[string]any) string {
	b, _ := json.MarshalIndent(schema, "", "  ")
	return "You are a precise Mermaid diagram code generator. You MUST follow the user's requirements EXACTLY.\n\n" +
		"Contract and constraints:\n" + string(b) + `

CRITICAL INSTRUCTIONS:
1. Count VERY CAREFULLY - the user specifies exact numbers that you MUST match
2. For sequence diagrams: count each participant declaration and each arrow line
3. For flowcharts: count each unique node ID and each arrow
4. Output ONLY raw Mermaid syntax - no markdown fences, no explanation, no extra text
5. Verify your counts match the requirements BEFORE outputting

STRICT RULES:
- NO markdown code fences (no triple-backtick characters anywhere)
- NO explanatory text before, after, or within the diagram
- NO comments
- EXACT counts as specified by user
- Clean, valid Mermaid syntax only

If the user asks for N items, you MUST output EXACTLY N items. Not N-1, not N+1, exactly N.`
}

func sequencePrompt(participants, interactions int, description string) string {
	return fmt.Sprintf(
		`You are an expert in technical architecture diagrams specializing in Mermaid sequence diagrams.
Your task is to generate a VALID Mermaid sequence diagram strictly following the requirements below.

SCENARIO: %q

STRICT REQUIREMENTS (FOLLOW EXACTLY):
1. Output ONLY raw Mermaid syntax - NO markdown fences, NO explanations, NO commentary
2. Start with exactly: sequenceDiagram
3. Define EXACTLY %d participants using format: participant X as Name
   - Use clear, meaningful names based on the scenario
   - Examples: participant A as Client, participant B as Server
4. Create EXACTLY %d interaction lines (arrows between participants)
   - Each arrow line = ONE interaction
   - Valid arrow types: ->>, -->, ->, -x, --x, -), --)
   - Format: ParticipantID->>ParticipantID: Message description
5. Maintain chronological order of interactions as described in the scenario

COUNTING INSTRUCTIONS (CRITICAL):
- You MUST generate EXACTLY %d participant declarations (count them as you write)
- You MUST generate EXACTLY %d interaction/message lines with arrows (count them as you write)
- After writing each participant, mentally count: "that's 1 of %d"
- After writing each interaction, mentally count: "that's 1 of %d"
- STOP immediately when you reach the required counts

EXAMPLE STRUCTURE (2 participants, 3 interactions):
sequenceDiagram
    participant A as Alice
    participant B as Bob
    A->>B: First message
    B->>A: Second message
    A->>B: Third message

VERIFICATION CHECKLIST BEFORE OUTPUT:
☐ Started with "sequenceDiagram"?
☐ Declared exactly %d participants?
☐ Created exactly %d interaction lines?
☐ No markdown fences or extra text?
☐ Clean, valid Mermaid syntax?

OUTPUT ONLY THE MERMAID CODE NOW:`, description, participants, interactions, participants, interactions, participants, interactions, participants, interactions)
}

func flowchartPrompt(nodes, edges int, direction, description string) string {
	if direction == "" {
		direction = "TD"
	}
	return fmt.Sprintf(
		`You are a Mermaid flowchart generator. Generate EXACTLY what is specified below.

TASK: Create a Mermaid flowchart about: %q

EXACT REQUIREMENTS (YOU MUST FOLLOW THESE PRECISELY):
1. Start with the line: flowchart %s
2. Create EXACTLY %d nodes (no more, no less)
   - Each unique node ID counts as ONE node
   - Node shapes: [Square], (Round), {Diamond}
   - Format: NodeID[Label Text]
   - Use simple IDs: A, B, C, D, E, etc.
3. Create EXACTLY %d edges/arrows (no more, no less)
   - Each arrow (-->, -.-, ==>) counts as ONE edge
   - Format: NodeID --> NodeID

STEP-BY-STEP PROCESS:
Step 1: Write "flowchart %s" on the first line
Step 2: Create %d unique nodes using IDs A, B, C, etc.
Step 3: Connect them with %d arrows
Step 4: Stop (do not add extra nodes or edges)

EXAMPLE STRUCTURE for 4 nodes and 3 edges:
flowchart TD
    A[Start] --> B{Decision}
    B --> C[Process]
    C --> D[End]

Count verification: Nodes are A, B, C, D = 4 nodes. Arrows are (A-->B), (B-->C), (C-->D) = 3 edges.

CRITICAL RULES:
- NO markdown fences (no triple-backtick characters)
- NO explanatory text
- NO comments
- Count carefully: %d nodes, %d edges
- Each node ID (A, B, C...) counts once
- Each arrow (-->) counts as one edge

DOUBLE-CHECK YOUR COUNTS BEFORE OUTPUTTING:
- Count node IDs: should equal %d
- Count arrows: should equal %d

OUTPUT ONLY THE MERMAID CODE:`, description, direction, nodes, edges, direction, nodes, edges, nodes, edges, nodes, edges)
}

// --- mermaid validation ---

func validateMermaidSequence(content string, participants, interactions int) error {
	content = strings.TrimSpace(content)

	if !strings.HasPrefix(content, "sequenceDiagram") {
		return errors.New("sequence diagram must start with 'sequenceDiagram'")
	}

	lines := strings.Split(content, "\n")
	participantCount := 0
	interactionCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "%%") {
			continue
		}

		if strings.HasPrefix(line, "participant ") || strings.HasPrefix(line, "actor ") {
			participantCount++
		}

		if strings.Contains(line, "->>") || strings.Contains(line, "-->") ||
			strings.Contains(line, "->") || strings.Contains(line, "-x") ||
			strings.Contains(line, "-)") || strings.Contains(line, "--x") ||
			strings.Contains(line, "--)") {
			interactionCount++
		}
	}

	if participantCount != participants {
		return fmt.Errorf("expected %d participants, got %d", participants, participantCount)
	}

	if interactionCount != interactions {
		return fmt.Errorf("expected %d interactions, got %d", interactions, interactionCount)
	}

	return nil
}

func validateMermaidFlowchart(content string, nodes, edges int, direction string) error {
	content = strings.TrimSpace(content)

	content = strings.ReplaceAll(content, `\u003e`, ">")
	content = strings.ReplaceAll(content, `\u003c`, "<")

	expectedStart := "flowchart " + direction
	if !strings.HasPrefix(content, expectedStart) && !strings.HasPrefix(content, "graph ") {
		return fmt.Errorf("flowchart must start with '%s'", expectedStart)
	}

	lines := strings.Split(content, "\n")
	nodeIDs := make(map[string]bool)
	edgeCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "%%") || strings.HasPrefix(line, "flowchart") || strings.HasPrefix(line, "graph") {
			continue
		}

		nodePattern := regexp.MustCompile(`([A-Za-z0-9_]+)\s*(?:[\[\(\{]|-->|-\->|==>|---|->)`)
		matches := nodePattern.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) > 1 {
				nodeIDs[match[1]] = true
			}
		}

		afterArrowPattern := regexp.MustCompile(`(?:-->|-\->|==>|---|->)\s*([A-Za-z0-9_]+)`)
		afterMatches := afterArrowPattern.FindAllStringSubmatch(line, -1)
		for _, match := range afterMatches {
			if len(match) > 1 {
				nodeIDs[match[1]] = true
			}
		}

		edgeCount += strings.Count(line, "-->")
		edgeCount += strings.Count(line, "-.->")
		edgeCount += strings.Count(line, "-.-")
		edgeCount += strings.Count(line, "==>")
		edgeCount += strings.Count(line, "---")
		edgeCount += strings.Count(line, "===")
		if strings.Contains(line, "->") && !strings.Contains(line, "-->") && !strings.Contains(line, "-.->") {
			edgeCount += strings.Count(line, "->")
		}
	}

	if len(nodeIDs) != nodes {
		return fmt.Errorf("expected %d nodes, got %d unique node IDs", nodes, len(nodeIDs))
	}

	if edgeCount != edges {
		return fmt.Errorf("expected %d edges, got %d", edges, edgeCount)
	}

	return nil
}

// --- mermaid generation ---

func generateMermaid(sysPrompt, userPrompt, model string, validator func(string) error) (string, string, error) {
	req := chatRequest{
		Model:  model,
		Stream: false,
		Messages: []chatMessage{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: userPrompt},
		},
	}

	content, raw, err := callOllama(req)
	if err != nil {
		return "", raw, err
	}

	content = stripCodeFences(content)
	content = strings.TrimSpace(content)

	if validator != nil {
		if err := validator(content); err != nil {
			return "", raw, err
		}
	}

	return content, raw, nil
}

/* ==============================
   MCP SERVER
   ============================== */

func main() {
	s := server.NewMCPServer(
		"chart-data-mcp",
		"1.1.0",
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	lineTool := mcp.NewTool(
		"line_chart_generate",
		mcp.WithDescription("Generate STRICT ChartSpec JSON for a line chart (no rendering)."),
		mcp.WithNumber("points", mcp.Required(), mcp.Description("number of points (N); labels must be length N, datasets too")),
		mcp.WithNumber("datasets", mcp.Required(), mcp.Description("number of datasets (D)")),
		mcp.WithString("title", mcp.Description("optional title")),
		mcp.WithString("model", mcp.Description("Ollama model name (default from env OLLAMA_MODEL or gemma3:4b)")),
	)
	s.AddTool(lineTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		N := mcp.ParseInt(req, "points", 0)
		D := mcp.ParseInt(req, "datasets", 0)
		if N <= 0 || D <= 0 {
			return mcp.NewToolResultError("points and datasets must be > 0"), nil
		}
		title := mcp.ParseString(req, "title", "")
		model := mcp.ParseString(req, "model", defaultModel())

		schema := schemaForLine(N, D)
		jsonSchema := jsonSchemaForLine(N, D)
		sys := systemPrompt(schema)
		usr := userPrompt("line", N, D, title)

		spec, raw, err := generateWithModelAndFormat(sys, usr, model, jsonSchema, "line", N, D, title)
		if err != nil {
			return mcp.NewToolResultErrorf("generation failed: %v\n--- raw ---\n%s", err, raw), nil
		}
		out := ChartResult{Schema: schema, Spec: spec}
		b, _ := json.MarshalIndent(out, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	barTool := mcp.NewTool(
		"bar_chart_generate",
		mcp.WithDescription("Generate STRICT ChartSpec JSON for a bar chart (no rendering). Also returns a 'schema' block describing required fields."),
		mcp.WithNumber("points", mcp.Required(), mcp.Description("number of points (N)")),
		mcp.WithNumber("datasets", mcp.Required(), mcp.Description("number of datasets (D)")),
		mcp.WithString("title", mcp.Description("optional chart title")),
		mcp.WithString("model", mcp.Description("Ollama model (default from OLLAMA_MODEL or gemma3:4b)")),
	)
	s.AddTool(barTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		N := mcp.ParseInt(req, "points", 0)
		D := mcp.ParseInt(req, "datasets", 0)
		if N <= 0 || D <= 0 {
			return mcp.NewToolResultError("points and datasets must be > 0"), nil
		}
		title := mcp.ParseString(req, "title", "")
		model := mcp.ParseString(req, "model", defaultModel())

		schema := schemaForBar(N, D)
		jsonSchema := jsonSchemaForBar(N, D)
		sys := systemPrompt(schema)
		usr := userPrompt("bar", N, D, title)

		spec, raw, err := generateWithModelAndFormat(sys, usr, model, jsonSchema, "bar", N, D, title)
		if err != nil {
			return mcp.NewToolResultErrorf("generation failed: %v\n--- raw ---\n%s", err, raw), nil
		}
		out := ChartResult{Schema: schema, Spec: spec}
		b, _ := json.MarshalIndent(out, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	histogramTool := mcp.NewTool(
		"histogram_chart_generate",
		mcp.WithDescription("Generate STRICT ChartSpec JSON for a histogram (frequency distribution). Returns schema + spec."),
		mcp.WithNumber("bins", mcp.Required(), mcp.Description("number of bins/categories (N)")),
		mcp.WithString("title", mcp.Description("optional chart title")),
		mcp.WithString("model", mcp.Description("Ollama model (default from OLLAMA_MODEL or gemma3:4b)")),
	)
	s.AddTool(histogramTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		N := mcp.ParseInt(req, "bins", 0)
		if N <= 0 {
			return mcp.NewToolResultError("bins must be > 0"), nil
		}
		title := mcp.ParseString(req, "title", "")
		model := mcp.ParseString(req, "model", defaultModel())

		schema := schemaForHistogram(N)
		jsonSchema := jsonSchemaForHistogram(N)
		sys := systemPrompt(schema)
		usr := userPrompt("histogram", N, 1, title)

		spec, raw, err := generateWithModelAndFormat(sys, usr, model, jsonSchema, "histogram", N, 1, title)
		if err != nil {
			return mcp.NewToolResultErrorf("generation failed: %v\n--- raw ---\n%s", err, raw), nil
		}
		out := ChartResult{Schema: schema, Spec: spec}
		b, _ := json.MarshalIndent(out, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	scatterTool := mcp.NewTool(
		"scatter_chart_generate",
		mcp.WithDescription("Generate STRICT ChartSpec JSON for a scatter plot. Each dataset has N points with {x, y} coordinates. Returns schema + spec."),
		mcp.WithNumber("points", mcp.Required(), mcp.Description("number of points per dataset (N)")),
		mcp.WithNumber("datasets", mcp.Required(), mcp.Description("number of datasets (D)")),
		mcp.WithString("title", mcp.Description("optional chart title")),
		mcp.WithString("model", mcp.Description("Ollama model (default from OLLAMA_MODEL or gemma3:4b)")),
	)
	s.AddTool(scatterTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		N := mcp.ParseInt(req, "points", 0)
		D := mcp.ParseInt(req, "datasets", 0)
		if N <= 0 || D <= 0 {
			return mcp.NewToolResultError("points and datasets must be > 0"), nil
		}
		title := mcp.ParseString(req, "title", "")
		model := mcp.ParseString(req, "model", defaultModel())

		schema := schemaForScatter(N, D)
		jsonSchema := jsonSchemaForScatter(N, D)
		sys := systemPrompt(schema)
		usr := userPromptScatter(N, D, title)

		spec, raw, err := generateWithModelAndFormat(sys, usr, model, jsonSchema, "scatter", N, D, title)
		if err != nil {
			return mcp.NewToolResultErrorf("generation failed: %v\n--- raw ---\n%s", err, raw), nil
		}
		out := ChartResult{Schema: schema, Spec: spec}
		b, _ := json.MarshalIndent(out, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	sequenceTool := mcp.NewTool(
		"sequence_diagram_generate",
		mcp.WithDescription("Generate Mermaid sequence diagram syntax showing interactions between participants."),
		mcp.WithNumber("participants", mcp.Required(), mcp.Description("number of participants/actors in the sequence")),
		mcp.WithNumber("interactions", mcp.Required(), mcp.Description("number of message interactions between participants")),
		mcp.WithString("description", mcp.Description("description of what the sequence diagram should show")),
		mcp.WithString("model", mcp.Description("Ollama model (default from OLLAMA_MODEL or gemma3:4b)")),
	)
	s.AddTool(sequenceTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		participants := mcp.ParseInt(req, "participants", 0)
		interactions := mcp.ParseInt(req, "interactions", 0)
		if participants <= 0 || interactions <= 0 {
			return mcp.NewToolResultError("participants and interactions must be > 0"), nil
		}
		description := mcp.ParseString(req, "description", "")
		model := mcp.ParseString(req, "model", defaultModel())

		schema := schemaForSequence(participants, interactions)
		sys := mermaidSystemPrompt(schema)
		usr := sequencePrompt(participants, interactions, description)

		validator := func(content string) error {
			return validateMermaidSequence(content, participants, interactions)
		}

		mermaidCode, raw, err := generateMermaid(sys, usr, model, validator)
		if err != nil {
			return mcp.NewToolResultErrorf("generation failed: %v\n--- raw ---\n%s", err, raw), nil
		}

		out := MermaidResult{
			Schema:      schema,
			DiagramType: "sequence",
			Mermaid:     mermaidCode,
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	flowchartTool := mcp.NewTool(
		"flowchart_generate",
		mcp.WithDescription("Generate Mermaid flowchart syntax with nodes and edges."),
		mcp.WithNumber("nodes", mcp.Required(), mcp.Description("number of nodes in the flowchart")),
		mcp.WithNumber("edges", mcp.Required(), mcp.Description("number of edges (connections) between nodes")),
		mcp.WithString("direction", mcp.Description("flowchart direction: TD (top-down), LR (left-right), BT (bottom-top), RL (right-left). Default: TD")),
		mcp.WithString("description", mcp.Description("description of what the flowchart should show")),
		mcp.WithString("model", mcp.Description("Ollama model (default from OLLAMA_MODEL or gemma3:4b)")),
	)
	s.AddTool(flowchartTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		nodes := mcp.ParseInt(req, "nodes", 0)
		edges := mcp.ParseInt(req, "edges", 0)
		if nodes <= 0 || edges <= 0 {
			return mcp.NewToolResultError("nodes and edges must be > 0"), nil
		}
		direction := mcp.ParseString(req, "direction", "TD")
		description := mcp.ParseString(req, "description", "")
		model := mcp.ParseString(req, "model", defaultModel())

		validDirections := map[string]bool{"TD": true, "TB": true, "LR": true, "BT": true, "RL": true}
		if !validDirections[direction] {
			return mcp.NewToolResultError("direction must be one of: TD, TB, LR, BT, RL"), nil
		}

		schema := schemaForFlowchart(nodes, edges, direction)
		sys := mermaidSystemPrompt(schema)
		usr := flowchartPrompt(nodes, edges, direction, description)

		validator := func(content string) error {
			return validateMermaidFlowchart(content, nodes, edges, direction)
		}

		mermaidCode, raw, err := generateMermaid(sys, usr, model, validator)
		if err != nil {
			return mcp.NewToolResultErrorf("generation failed: %v\n--- raw ---\n%s", err, raw), nil
		}

		out := MermaidResult{
			Schema:      schema,
			DiagramType: "flowchart",
			Mermaid:     mermaidCode,
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
