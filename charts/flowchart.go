package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Node represents a node in the flowchart
type FlowchartNode struct {
	ID    string `json:"id" jsonschema:"description=Unique identifier for the node,required"`
	Label string `json:"label,omitempty" jsonschema:"description=Display label (if different from ID)"`
	Shape string `json:"shape,omitempty" jsonschema:"description=Node shape,enum=rectangle,enum=round,enum=stadium,enum=circle,enum=diamond,enum=hexagon,enum=cylinder,enum=subroutine,enum=parallelogram,enum=trapezoid,enum=double-circle,enum=asymmetric"`
}

// Link represents a connection between nodes
type FlowchartLink struct {
	From      string `json:"from" jsonschema:"description=Source node ID,required"`
	To        string `json:"to" jsonschema:"description=Target node ID,required"`
	Text      string `json:"text,omitempty" jsonschema:"description=Text label on the link"`
	ArrowType string `json:"arrowType,omitempty" jsonschema:"description=Arrow style,enum=-->,enum=---,enum=-.->,enum=-.-,enum===>,enum====,enum=--o,enum=--x,enum=<-->,enum=o--o,enum=x--x"`
	Length    int    `json:"length,omitempty" jsonschema:"description=Extra length (number of additional dashes),minimum=0,maximum=5"`
}

// Subgraph represents a grouped section of nodes
type FlowchartSubgraph struct {
	ID        string   `json:"id,omitempty" jsonschema:"description=Subgraph identifier"`
	Title     string   `json:"title" jsonschema:"description=Subgraph title,required"`
	Nodes     []string `json:"nodes" jsonschema:"description=Node IDs in this subgraph,required,minItems=1"`
	Direction string   `json:"direction,omitempty" jsonschema:"description=Subgraph direction,enum=TB,enum=TD,enum=BT,enum=LR,enum=RL"`
}

// StyleDef represents a style definition
type FlowchartStyle struct {
	Target     string `json:"target" jsonschema:"description=Node ID or class name to style,required"`
	Properties string `json:"properties" jsonschema:"description=CSS properties (e.g. fill:#f9f,stroke:#333),required"`
}

// ClassDef represents a class definition
type FlowchartClassDef struct {
	ClassName  string   `json:"className" jsonschema:"description=Class name,required"`
	Properties string   `json:"properties" jsonschema:"description=CSS properties,required"`
	Nodes      []string `json:"nodes,omitempty" jsonschema:"description=Node IDs to apply this class to"`
}

// FlowchartArgs represents the arguments for generating a flowchart
type FlowchartArgs struct {
	Direction string              `json:"direction,omitempty" jsonschema:"description=Flow direction,enum=TB,enum=TD,enum=BT,enum=LR,enum=RL"`
	Nodes     []FlowchartNode     `json:"nodes,omitempty" jsonschema:"description=Explicit node definitions (optional - nodes are auto-detected from links)"`
	Links     []FlowchartLink     `json:"links" jsonschema:"description=Array of links between nodes,required,minItems=1"`
	Subgraphs []FlowchartSubgraph `json:"subgraphs,omitempty" jsonschema:"description=Array of subgraphs to group nodes"`
	Styles    []FlowchartStyle    `json:"styles,omitempty" jsonschema:"description=Array of style definitions for nodes"`
	ClassDefs []FlowchartClassDef `json:"classDefs,omitempty" jsonschema:"description=Array of class definitions"`
}

func generateFlowchartDSL(args FlowchartArgs) string {
	var lines []string

	// start with flowchart keyword and direction
	direction := args.Direction
	if direction == "" {
		direction = "TB"
	}
	lines = append(lines, fmt.Sprintf("flowchart %s", direction))

	// collect all node IDs from links for auto-detection
	allNodeIDs := make(map[string]bool)
	for _, link := range args.Links {
		allNodeIDs[link.From] = true
		allNodeIDs[link.To] = true
	}

	// create a map of explicit node definitions
	nodeMap := make(map[string]FlowchartNode)
	for _, node := range args.Nodes {
		nodeMap[node.ID] = node
		allNodeIDs[node.ID] = true
	}

	// collect nodes in subgraphs
	subgraphNodes := make(map[string]bool)
	for _, sg := range args.Subgraphs {
		for _, nid := range sg.Nodes {
			subgraphNodes[nid] = true
		}
	}

	// define nodes that are not in subgraphs (explicit definitions only)
	for _, node := range args.Nodes {
		if !subgraphNodes[node.ID] {
			lines = append(lines, formatFlowchartNode(node))
		}
	}

	// define subgraphs
	for _, sg := range args.Subgraphs {
		sgLines := formatSubgraph(sg, nodeMap)
		lines = append(lines, sgLines...)
	}

	// define links
	for _, link := range args.Links {
		lines = append(lines, formatFlowchartLink(link, nodeMap))
	}

	// define class definitions
	for _, classDef := range args.ClassDefs {
		lines = append(lines, fmt.Sprintf("    classDef %s %s", classDef.ClassName, classDef.Properties))
		if len(classDef.Nodes) > 0 {
			nodeList := strings.Join(classDef.Nodes, ",")
			lines = append(lines, fmt.Sprintf("    class %s %s", nodeList, classDef.ClassName))
		}
	}

	// apply styles
	for _, style := range args.Styles {
		lines = append(lines, fmt.Sprintf("    style %s %s", style.Target, style.Properties))
	}

	return strings.Join(lines, "\n")
}

func formatFlowchartNode(node FlowchartNode) string {
	label := node.Label
	if label == "" {
		label = node.ID
	}

	shape := node.Shape
	if shape == "" {
		shape = "rectangle"
	}

	var nodeStr string
	switch shape {
	case "rectangle":
		nodeStr = fmt.Sprintf("    %s[%s]", node.ID, label)
	case "round":
		nodeStr = fmt.Sprintf("    %s(%s)", node.ID, label)
	case "stadium":
		nodeStr = fmt.Sprintf("    %s([%s])", node.ID, label)
	case "subroutine":
		nodeStr = fmt.Sprintf("    %s[[%s]]", node.ID, label)
	case "cylinder":
		nodeStr = fmt.Sprintf("    %s[(%s)]", node.ID, label)
	case "circle":
		nodeStr = fmt.Sprintf("    %s((%s))", node.ID, label)
	case "asymmetric":
		nodeStr = fmt.Sprintf("    %s>%s]", node.ID, label)
	case "diamond":
		nodeStr = fmt.Sprintf("    %s{%s}", node.ID, label)
	case "hexagon":
		nodeStr = fmt.Sprintf("    %s{{%s}}", node.ID, label)
	case "parallelogram":
		nodeStr = fmt.Sprintf("    %s[/%s/]", node.ID, label)
	case "trapezoid":
		nodeStr = fmt.Sprintf("    %s[/%s\\]", node.ID, label)
	case "double-circle":
		nodeStr = fmt.Sprintf("    %s(((%s)))", node.ID, label)
	default:
		nodeStr = fmt.Sprintf("    %s[%s]", node.ID, label)
	}

	return nodeStr
}

func formatFlowchartLink(link FlowchartLink, nodeMap map[string]FlowchartNode) string {
	arrowType := link.ArrowType
	if arrowType == "" {
		arrowType = "-->"
	}

	// handle link length by adding extra dashes/dots/equals
	arrow := arrowType
	if link.Length > 0 {
		arrow = extendArrow(arrowType, link.Length)
	}

	// format the link with optional text
	var linkStr string
	if link.Text != "" {
		// put text in the middle of the arrow
		linkStr = fmt.Sprintf("    %s %s|%s| %s", link.From, arrow, link.Text, link.To)
	} else {
		linkStr = fmt.Sprintf("    %s %s %s", link.From, arrow, link.To)
	}

	return linkStr
}

func extendArrow(arrowType string, extraLength int) string {
	// extend the arrow by adding more dashes, dots, or equals
	// extraLength is the number of ADDITIONAL segments beyond the base
	switch arrowType {
	case "-->":
		return "--" + strings.Repeat("-", extraLength) + ">"
	case "---":
		return "--" + strings.Repeat("-", extraLength) + "-"
	case "-.->":
		// base is -.-> (1 dot), extraLength specifies total dots desired
		return "-" + strings.Repeat(".", extraLength) + "->"
	case "-.-":
		return "-" + strings.Repeat(".", extraLength) + "-"
	case "==>":
		return "==" + strings.Repeat("=", extraLength) + ">"
	case "===":
		return "==" + strings.Repeat("=", extraLength) + "="
	case "<-->":
		return "<-" + strings.Repeat("-", extraLength) + "->"
	default:
		return arrowType
	}
}

func formatSubgraph(sg FlowchartSubgraph, nodeMap map[string]FlowchartNode) []string {
	var lines []string

	// subgraph declaration
	if sg.ID != "" {
		lines = append(lines, fmt.Sprintf("    subgraph %s [%s]", sg.ID, sg.Title))
	} else {
		lines = append(lines, fmt.Sprintf("    subgraph %s", sg.Title))
	}

	// optional direction
	if sg.Direction != "" {
		lines = append(lines, fmt.Sprintf("        direction %s", sg.Direction))
	}

	// nodes in subgraph
	for _, nodeID := range sg.Nodes {
		if node, exists := nodeMap[nodeID]; exists {
			// indent the node definition
			nodeLine := formatFlowchartNode(node)
			lines = append(lines, "    "+nodeLine)
		} else {
			// node not explicitly defined, just reference it
			lines = append(lines, fmt.Sprintf("        %s", nodeID))
		}
	}

	lines = append(lines, "    end")

	return lines
}

func validateFlowchartArgs(args FlowchartArgs) error {
	if len(args.Links) == 0 {
		return fmt.Errorf("links must contain at least one item")
	}

	// validate direction
	if args.Direction != "" {
		validDirections := map[string]bool{"TB": true, "TD": true, "BT": true, "LR": true, "RL": true}
		if !validDirections[args.Direction] {
			return fmt.Errorf("invalid direction: %s (must be TB, TD, BT, LR, or RL)", args.Direction)
		}
	}

	// collect all valid node IDs
	nodeIDs := make(map[string]bool)
	for _, node := range args.Nodes {
		if node.ID == "" {
			return fmt.Errorf("node ID cannot be empty")
		}
		nodeIDs[node.ID] = true
	}

	// auto-detect nodes from links
	for _, link := range args.Links {
		if link.From == "" {
			return fmt.Errorf("link 'from' field cannot be empty")
		}
		if link.To == "" {
			return fmt.Errorf("link 'to' field cannot be empty")
		}
		nodeIDs[link.From] = true
		nodeIDs[link.To] = true
	}

	// validate subgraph nodes reference valid nodes
	for _, sg := range args.Subgraphs {
		if sg.Title == "" {
			return fmt.Errorf("subgraph title cannot be empty")
		}
		if len(sg.Nodes) == 0 {
			return fmt.Errorf("subgraph must contain at least one node")
		}
		if sg.Direction != "" {
			validDirections := map[string]bool{"TB": true, "TD": true, "BT": true, "LR": true, "RL": true}
			if !validDirections[sg.Direction] {
				return fmt.Errorf("invalid subgraph direction: %s", sg.Direction)
			}
		}
		for _, nid := range sg.Nodes {
			if !nodeIDs[nid] {
				return fmt.Errorf("subgraph references unknown node: %s", nid)
			}
		}
	}

	// validate styles reference valid nodes
	for _, style := range args.Styles {
		if style.Target == "" {
			return fmt.Errorf("style target cannot be empty")
		}
		if style.Properties == "" {
			return fmt.Errorf("style properties cannot be empty")
		}
	}

	// validate class definitions
	for _, classDef := range args.ClassDefs {
		if classDef.ClassName == "" {
			return fmt.Errorf("class name cannot be empty")
		}
		if classDef.Properties == "" {
			return fmt.Errorf("class properties cannot be empty")
		}
		for _, nid := range classDef.Nodes {
			if !nodeIDs[nid] {
				return fmt.Errorf("class references unknown node: %s", nid)
			}
		}
	}

	return nil
}

func registerFlowchartTool(srv *server.MCPServer) {
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args FlowchartArgs
		if err := req.BindArguments(&args); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("bind arguments: %v", err)), nil
		}

		if err := validateFlowchartArgs(args); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		dsl := generateFlowchartDSL(args)
		return mcp.NewToolResultText(dsl), nil
	}

	tool := mcp.NewTool(
		"flowchart-generator",
		mcp.WithDescription(`Generates a Mermaid flowchart diagram DSL.
Flowcharts visualize processes, algorithms, workflows, and decision trees using nodes (shapes) and directed edges (arrows).
Use this for process flows, decision logic, system architectures, algorithm visualization, or any directed graph representation.

The tool generates valid Mermaid DSL that can be rendered in Markdown, documentation, or Mermaid-compatible tools.

Key features:
- Flow direction control (top-down, left-right, etc.)
- Multiple node shapes (rectangle, circle, diamond, hexagon, cylinder, stadium, etc.)
- Various arrow types (solid, dotted, thick, bidirectional, circle/cross edges)
- Link text labels
- Variable link lengths for layout control
- Subgraphs for grouping related nodes
- Styling with CSS properties
- Class definitions for reusable styles

Node shapes:
  rectangle    : standard box (default)
  round        : rounded corners
  stadium      : pill shape
  circle       : circular node
  diamond      : decision diamond
  hexagon      : hexagonal shape
  cylinder     : database/storage shape
  subroutine   : box with side bars
  parallelogram: slanted rectangle
  trapezoid    : trapezoid shape
  double-circle: double circle outline
  asymmetric   : flag shape

Arrow types:
  -->  : solid arrow (default)
  ---  : solid line without arrow
  -.-> : dotted arrow
  -.-  : dotted line
  ==>  : thick arrow
  ===  : thick line
  --o  : circle edge
  --x  : cross edge
  <--> : bidirectional arrow
  o--o : circles on both ends
  x--x : crosses on both ends

Direction options:
  TB/TD : top to bottom (default)
  BT    : bottom to top
  LR    : left to right
  RL    : right to left`),
		mcp.WithInputSchema[FlowchartArgs](),
	)

	srv.AddTool(tool, handler)
}
