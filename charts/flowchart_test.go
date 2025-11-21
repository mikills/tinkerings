package main

import (
	"strings"
	"testing"
)

func TestGenerateFlowchartDSL(t *testing.T) {
	tests := []struct {
		name     string
		args     FlowchartArgs
		expected string
	}{
		{
			name: "Classic shapes",
			args: FlowchartArgs{
				Direction: "TD",
				Nodes: []FlowchartNode{
					{ID: "A", Shape: "rectangle", Label: "Box"},
					{ID: "B", Shape: "round", Label: "Round"},
					{ID: "C", Shape: "stadium", Label: "Stadium"},
					{ID: "D", Shape: "circle", Label: "Circle"},
					{ID: "E", Shape: "diamond", Label: "Diamond"},
					{ID: "F", Shape: "hexagon", Label: "Hexagon"},
					{ID: "G", Shape: "cylinder", Label: "Database"},
					{ID: "H", Shape: "subroutine", Label: "Sub"},
					{ID: "I", Shape: "parallelogram", Label: "Para"},
					{ID: "J", Shape: "parallelogram-alt", Label: "Para Alt"},
					{ID: "K", Shape: "trapezoid", Label: "Trap"},
					{ID: "L", Shape: "trapezoid-alt", Label: "Trap Alt"},
					{ID: "M", Shape: "double-circle", Label: "Double"},
					{ID: "N", Shape: "asymmetric", Label: "Flag"},
				},
				Links: []FlowchartLink{
					{From: "A", To: "B"},
				},
			},
			expected: `flowchart TD
    A[Box]
    B(Round)
    C([Stadium])
    D((Circle))
    E{Diamond}
    F{{Hexagon}}
    G[(Database)]
    H[[Sub]]
    I[/Para/]
    J[\Para Alt\]
    K[/Trap\]
    L[\Trap Alt/]
    M(((Double)))
    N>Flag]
    A --> B`,
		},
		{
			name: "New shapes with @{} syntax",
			args: FlowchartArgs{
				Direction: "RL",
				Nodes: []FlowchartNode{
					{ID: "A", Shape: "manual-file", Label: "File Handling"},
					{ID: "B", Shape: "manual-input", Label: "User Input"},
					{ID: "C", Shape: "docs", Label: "Multiple Documents"},
					{ID: "D", Shape: "procs", Label: "Process Automation"},
					{ID: "E", Shape: "paper-tape", Label: "Paper Records"},
				},
				Links: []FlowchartLink{
					{From: "A", To: "B"},
					{From: "B", To: "C"},
					{From: "C", To: "D"},
					{From: "D", To: "E"},
				},
			},
			expected: `flowchart RL
    A@{ shape: manual-file, label: "File Handling"}
    B@{ shape: manual-input, label: "User Input"}
    C@{ shape: docs, label: "Multiple Documents"}
    D@{ shape: procs, label: "Process Automation"}
    E@{ shape: paper-tape, label: "Paper Records"}
    A --> B
    B --> C
    C --> D
    D --> E`,
		},
		{
			name: "Links with text",
			args: FlowchartArgs{
				Links: []FlowchartLink{
					{From: "A", To: "B", Text: "Yes"},
					{From: "B", To: "C", Text: "No", ArrowType: "-.->"},
					{From: "C", To: "D", Text: "Maybe", ArrowType: "==>"},
				},
			},
			expected: `flowchart TB
    A -->|Yes| B
    B -. No .-> C
    C == Maybe ==> D`,
		},
		{
			name: "Extended arrows",
			args: FlowchartArgs{
				Links: []FlowchartLink{
					{From: "A", To: "B", Length: 2},
					{From: "B", To: "C", ArrowType: "-.->", Length: 1},
					{From: "C", To: "D", ArrowType: "==>", Length: 3},
				},
			},
			expected: `flowchart TB
    A ----> B
    B -..-> C
    C =====> D`,
		},
		{
			name: "Node with spaces and special characters",
			args: FlowchartArgs{
				Nodes: []FlowchartNode{
					{ID: "Node 1", Label: "First Node"},
					{ID: "Node 2", Label: "Second & Node"},
					{ID: "3Node", Label: "Third <Node>"},
				},
				Links: []FlowchartLink{
					{From: "Node 1", To: "Node 2"},
					{From: "Node 2", To: "3Node"},
				},
			},
			expected: `flowchart TB
    Node_1[First Node]
    Node_2[Second &amp; Node]
    node_3Node[Third &lt;Node&gt;]
    Node_1 --> Node_2
    Node_2 --> node_3Node`,
		},
		{
			name: "Subgraph",
			args: FlowchartArgs{
				Direction: "LR",
				Nodes: []FlowchartNode{
					{ID: "A", Label: "Start"},
					{ID: "B", Label: "Process 1"},
					{ID: "C", Label: "Process 2"},
					{ID: "D", Label: "End"},
				},
				Links: []FlowchartLink{
					{From: "A", To: "B"},
					{From: "B", To: "C"},
					{From: "C", To: "D"},
				},
				Subgraphs: []FlowchartSubgraph{
					{
						ID:    "sub1",
						Title: "Processing",
						Nodes: []string{"B", "C"},
					},
				},
			},
			expected: `flowchart LR
    A[Start]
    D[End]
    subgraph sub1 [Processing]
        B[Process 1]
        C[Process 2]
    end
    A --> B
    B --> C
    C --> D`,
		},
		{
			name: "Styles and classes",
			args: FlowchartArgs{
				Nodes: []FlowchartNode{
					{ID: "A", Label: "Node A"},
					{ID: "B", Label: "Node B"},
				},
				Links: []FlowchartLink{
					{From: "A", To: "B"},
				},
				ClassDefs: []FlowchartClassDef{
					{
						ClassName:  "important",
						Properties: "fill:#f9f,stroke:#333,stroke-width:4px",
						Nodes:      []string{"A"},
					},
				},
				Styles: []FlowchartStyle{
					{
						Target:     "B",
						Properties: "fill:#bbf,stroke:#f66,stroke-width:2px",
					},
				},
			},
			expected: `flowchart TB
    A[Node A]
    B[Node B]
    A --> B
    classDef important fill:#f9f,stroke:#333,stroke-width:4px
    class A important
    style B fill:#bbf,stroke:#f66,stroke-width:2px`,
		},
		{
			name: "Click events",
			args: FlowchartArgs{
				Nodes: []FlowchartNode{
					{
						ID:            "A",
						Label:         "Clickable",
						Click:         "https://example.com",
						Tooltip:       "Click me!",
						OpenLinkInNew: true,
					},
				},
				Links: []FlowchartLink{
					{From: "A", To: "B"},
				},
			},
			expected: `flowchart TB
    A[Clickable]
    A --> B
    click A "https://example.com" "Click me!" _blank`,
		},
		{
			name: "Markdown labels",
			args: FlowchartArgs{
				Nodes: []FlowchartNode{
					{ID: "A", Label: "**Bold Text**", IsMarkdown: true},
				},
				Links: []FlowchartLink{
					{From: "A", To: "B"},
				},
			},
			expected: "flowchart TB\n    A[`**Bold Text**`]\n    A --> B",
		},
		{
			name: "Mixed classic and new shapes",
			args: FlowchartArgs{
				Nodes: []FlowchartNode{
					{ID: "A", Shape: "rectangle", Label: "Classic"},
					{ID: "B", Shape: "manual-input", Label: "New Shape"},
					{ID: "C", Shape: "diamond", Label: "Classic Diamond"},
					{ID: "D", Shape: "database", Label: "New DB"},
				},
				Links: []FlowchartLink{
					{From: "A", To: "B"},
					{From: "B", To: "C"},
					{From: "C", To: "D"},
				},
			},
			expected: `flowchart TB
    A[Classic]
    B@{ shape: manual-input, label: "New Shape"}
    C{Classic Diamond}
    D@{ shape: database, label: "New DB"}
    A --> B
    B --> C
    C --> D`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateFlowchartDSL(tt.args)
			if got != tt.expected {
				t.Errorf("generateFlowchartDSL() mismatch\nGot:\n%s\n\nExpected:\n%s", got, tt.expected)

				// Show differences line by line
				gotLines := strings.Split(got, "\n")
				expectedLines := strings.Split(tt.expected, "\n")

				maxLines := len(gotLines)
				if len(expectedLines) > maxLines {
					maxLines = len(expectedLines)
				}

				for i := 0; i < maxLines; i++ {
					gotLine := ""
					expectedLine := ""

					if i < len(gotLines) {
						gotLine = gotLines[i]
					}
					if i < len(expectedLines) {
						expectedLine = expectedLines[i]
					}

					if gotLine != expectedLine {
						t.Errorf("Line %d differs:\n  Got:      '%s'\n  Expected: '%s'", i+1, gotLine, expectedLine)
					}
				}
			}
		})
	}
}

func TestValidateFlowchartArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    FlowchartArgs
		wantErr bool
	}{
		{
			name:    "Empty links",
			args:    FlowchartArgs{},
			wantErr: true,
		},
		{
			name: "Invalid direction",
			args: FlowchartArgs{
				Direction: "INVALID",
				Links:     []FlowchartLink{{From: "A", To: "B"}},
			},
			wantErr: true,
		},
		{
			name: "Valid flowchart",
			args: FlowchartArgs{
				Direction: "LR",
				Nodes: []FlowchartNode{
					{ID: "A", Label: "Start"},
					{ID: "B", Label: "End"},
				},
				Links: []FlowchartLink{
					{From: "A", To: "B"},
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid arrow type",
			args: FlowchartArgs{
				Links: []FlowchartLink{
					{From: "A", To: "B", ArrowType: "invalid"},
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid link style index",
			args: FlowchartArgs{
				Links: []FlowchartLink{
					{From: "A", To: "B"},
				},
				LinkStyles: []LinkStyle{
					{LinkNumbers: []int{5}, Properties: "stroke:#333"},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFlowchartArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateFlowchartArgs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSanitiseNodeID(t *testing.T) {
	tests := []struct {
		id       string
		expected string
	}{
		{"simple", "simple"},
		{"Node 1", "Node_1"},
		{"3Node", "node_3Node"},
		{"end", "end_node"},
		{"A-B", "A_B"},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			got := sanitiseNodeID(tt.id)
			if got != tt.expected {
				t.Errorf("sanitiseNodeID(%s) = %s, want %s", tt.id, got, tt.expected)
			}
		})
	}
}
