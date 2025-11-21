package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type FlowchartNode struct {
	ID            string            `json:"id" jsonschema:"description=Unique identifier for the node,required"`
	Label         string            `json:"label,omitempty" jsonschema:"description=Display label (if different from ID)"`
	Shape         string            `json:"shape,omitempty" jsonschema:"description=Node shape"`
	IsMarkdown    bool              `json:"isMarkdown,omitempty" jsonschema:"description=Whether the label contains markdown"`
	Click         string            `json:"click,omitempty" jsonschema:"description=Click action (URL or callback)"`
	Tooltip       string            `json:"tooltip,omitempty" jsonschema:"description=Tooltip text on hover"`
	OpenLinkInNew bool              `json:"openLinkInNew,omitempty" jsonschema:"description=Open click URL in new tab/window"`
	UseNewSyntax  bool              `json:"useNewSyntax,omitempty" jsonschema:"description=Force use of @{} syntax"`
	Attributes    map[string]string `json:"attributes,omitempty" jsonschema:"description=Additional attributes for @{} syntax"`
}

type FlowchartLink struct {
	From        string            `json:"from" jsonschema:"description=Source node ID,required"`
	To          string            `json:"to" jsonschema:"description=Target node ID,required"`
	Text        string            `json:"text,omitempty" jsonschema:"description=Text label on the link"`
	ArrowType   string            `json:"arrowType,omitempty" jsonschema:"description=Arrow style,enum=-->,enum=---,enum=-.->,enum=-.-,enum==>,enum=~~~,enum=--o,enum=--x,enum=<-->,enum=o--o,enum=x--x"`
	Length      int               `json:"length,omitempty" jsonschema:"description=Extra length (number of additional segments),minimum=0,maximum=5"`
	Style       map[string]string `json:"style,omitempty" jsonschema:"description=Link styling properties"`
	Interpolate string            `json:"interpolate,omitempty" jsonschema:"description=Curve interpolation"`
}

type FlowchartSubgraph struct {
	ID        string   `json:"id,omitempty" jsonschema:"description=Subgraph identifier"`
	Title     string   `json:"title" jsonschema:"description=Subgraph title,required"`
	Nodes     []string `json:"nodes" jsonschema:"description=Node IDs in this subgraph,required,minItems=1"`
	Direction string   `json:"direction,omitempty" jsonschema:"description=Subgraph direction,enum=TB,enum=TD,enum=BT,enum=LR,enum=RL"`
}

type FlowchartStyle struct {
	Target     string `json:"target" jsonschema:"description=Node ID or class name to style,required"`
	Properties string `json:"properties" jsonschema:"description=CSS properties,required"`
}

type FlowchartClassDef struct {
	ClassName  string   `json:"className" jsonschema:"description=Class name,required"`
	Properties string   `json:"properties" jsonschema:"description=CSS properties,required"`
	Nodes      []string `json:"nodes,omitempty" jsonschema:"description=Node IDs to apply this class to"`
}

type LinkStyle struct {
	LinkNumbers []int  `json:"linkNumbers" jsonschema:"description=Link indices to style (0-based),required,minItems=1"`
	Properties  string `json:"properties" jsonschema:"description=CSS properties for the links,required"`
}

type Comment struct {
	Text     string `json:"text" jsonschema:"description=Comment text,required"`
	Position int    `json:"position,omitempty" jsonschema:"description=Line position for the comment"`
}

type FlowchartArgs struct {
	Direction  string              `json:"direction,omitempty" jsonschema:"description=Flow direction,enum=TB,enum=TD,enum=BT,enum=LR,enum=RL"`
	Nodes      []FlowchartNode     `json:"nodes,omitempty" jsonschema:"description=Explicit node definitions"`
	Links      []FlowchartLink     `json:"links" jsonschema:"description=Array of links between nodes,required,minItems=1"`
	Subgraphs  []FlowchartSubgraph `json:"subgraphs,omitempty" jsonschema:"description=Array of subgraphs to group nodes"`
	Styles     []FlowchartStyle    `json:"styles,omitempty" jsonschema:"description=Array of style definitions for nodes"`
	ClassDefs  []FlowchartClassDef `json:"classDefs,omitempty" jsonschema:"description=Array of class definitions"`
	LinkStyles []LinkStyle         `json:"linkStyles,omitempty" jsonschema:"description=Array of link style definitions"`
	Comments   []Comment           `json:"comments,omitempty" jsonschema:"description=Comments to include in the diagram"`
}

// Classic shapes with bracket notation
var classicShapes = map[string]string{
	"rectangle":         "[%s]",
	"round":             "(%s)",
	"stadium":           "([%s])",
	"circle":            "((%s))",
	"diamond":           "{%s}",
	"hexagon":           "{{%s}}",
	"cylinder":          "[(%s)]",
	"subroutine":        "[[%s]]",
	"parallelogram":     "[/%s/]",
	"parallelogram-alt": "[\\%s\\]",
	"trapezoid":         "[/%s\\]",
	"trapezoid-alt":     "[\\%s/]",
	"double-circle":     "(((%s)))",
	"asymmetric":        ">%s]",
}

// New Mermaid shapes requiring @{} syntax
var newShapes = map[string]bool{
	"rect": true, "rounded": true, "fr-rect": true, "cyl": true, "odd": true,
	"diam": true, "hex": true, "lean-r": true, "lean-l": true, "trap-b": true,
	"trap-t": true, "dbl-circ": true, "text": true, "notch-rect": true,
	"lin-rect": true, "sm-circ": true, "fr-circ": true, "fork": true,
	"hourglass": true, "brace": true, "brace-r": true, "braces": true,
	"bolt": true, "doc": true, "delay": true, "h-cyl": true, "lin-cyl": true,
	"curv-trap": true, "div-rect": true, "tri": true, "win-pane": true,
	"f-circ": true, "lin-doc": true, "notch-pent": true, "flip-tri": true,
	"sl-rect": true, "docs": true, "st-rect": true, "flag": true,
	"bow-rect": true, "cross-circ": true, "tag-doc": true, "tag-rect": true,
	"st-doc": true, "manual-file": true, "manual-input": true, "procs": true,
	"paper-tape": true, "database": true, "subprocess": true, "document": true,
	"processes": true, "stacked-document": true, "flipped-triangle": true,
	"sloped-rectangle": true, "proc": true, "process": true, "event": true,
	"terminal": true, "pill": true, "subproc": true, "framed-rectangle": true,
	"db": true, "circ": true, "decision": true, "prepare": true,
	"lean-right": true, "in-out": true, "lean-left": true, "out-in": true,
	"priority": true, "trapezoid-bottom": true, "manual": true,
	"trapezoid-top": true, "start": true,
	"small-circle": true, "stop": true, "framed-circle": true, "junction": true,
	"filled-circle": true, "summary": true, "crossed-circle": true, "card": true,
	"notched-rectangle": true, "lined-document": true, "documents": true,
	"tagged-document": true, "lined-rectangle": true, "lined-proc": true,
	"lin-proc": true, "shaded-process": true, "divided-rectangle": true,
	"divided-process": true, "div-proc": true, "stacked-rect": true,
	"tagged-rectangle": true, "tag-proc": true, "tagged-process": true,
	"half-rounded-rectangle": true, "das": true, "horizontal-cylinder": true,
	"disk": true, "lined-cylinder": true, "internal-storage": true,
	"window-pane": true, "stored-data": true, "bow-tie-rectangle": true,
	"join": true, "comment": true, "brace-l": true, "com-link": true,
	"lightning-bolt": true, "curved-trapezoid": true, "display": true,
	"extract": true, "triangle": true, "loop-limit": true,
	"notched-pentagon": true,
}

func generateFlowchartDSL(args FlowchartArgs) string {
	var lines []string

	for _, comment := range args.Comments {
		if comment.Position == 0 {
			lines = append(lines, fmt.Sprintf("%%%% %s", comment.Text))
		}
	}

	direction := args.Direction
	if direction == "" {
		direction = "TB"
	}
	lines = append(lines, fmt.Sprintf("flowchart %s", direction))

	allNodeIDs := make(map[string]bool)
	for _, link := range args.Links {
		allNodeIDs[link.From] = true
		allNodeIDs[link.To] = true
	}

	nodeMap := make(map[string]FlowchartNode)
	for _, node := range args.Nodes {
		nodeMap[node.ID] = node
		allNodeIDs[node.ID] = true
	}

	subgraphNodes := make(map[string]bool)
	for _, sg := range args.Subgraphs {
		for _, nid := range sg.Nodes {
			subgraphNodes[nid] = true
		}
	}

	for _, node := range args.Nodes {
		if !subgraphNodes[node.ID] {
			lines = append(lines, formatFlowchartNode(node))
		}
	}

	for _, sg := range args.Subgraphs {
		sgLines := formatSubgraph(sg, nodeMap)
		lines = append(lines, sgLines...)
	}

	linkIndex := 0
	for _, link := range args.Links {
		lines = append(lines, formatFlowchartLink(link))

		if link.Style != nil && len(link.Style) > 0 {
			styleProps := []string{}
			for key, value := range link.Style {
				styleProps = append(styleProps, fmt.Sprintf("%s:%s", key, value))
			}
			lines = append(lines, fmt.Sprintf("    linkStyle %d %s", linkIndex, strings.Join(styleProps, ",")))
		}

		if link.Interpolate != "" {
			lines = append(lines, fmt.Sprintf("    linkStyle %d interpolate %s", linkIndex, link.Interpolate))
		}

		linkIndex++
	}

	for _, ls := range args.LinkStyles {
		linkNums := []string{}
		for _, num := range ls.LinkNumbers {
			linkNums = append(linkNums, fmt.Sprintf("%d", num))
		}
		lines = append(lines, fmt.Sprintf("    linkStyle %s %s", strings.Join(linkNums, ","), ls.Properties))
	}

	for _, classDef := range args.ClassDefs {
		lines = append(lines, fmt.Sprintf("    classDef %s %s", classDef.ClassName, classDef.Properties))
		if len(classDef.Nodes) > 0 {
			nodeList := strings.Join(classDef.Nodes, ",")
			lines = append(lines, fmt.Sprintf("    class %s %s", nodeList, classDef.ClassName))
		}
	}

	for _, style := range args.Styles {
		lines = append(lines, fmt.Sprintf("    style %s %s", style.Target, style.Properties))
	}

	for _, node := range args.Nodes {
		if node.Click != "" {
			// URLs need to be quoted
			clickLine := fmt.Sprintf("    click %s \"%s\"", sanitiseNodeID(node.ID), escapeString(node.Click))
			if node.Tooltip != "" {
				clickLine += fmt.Sprintf(" \"%s\"", escapeString(node.Tooltip))
			}
			if node.OpenLinkInNew {
				clickLine += " _blank"
			}
			lines = append(lines, clickLine)
		}
	}

	return strings.Join(lines, "\n")
}

func formatFlowchartNode(node FlowchartNode) string {
	label := node.Label
	if label == "" {
		label = node.ID
	}

	if node.IsMarkdown {
		label = fmt.Sprintf("`%s`", label)
	} else {
		label = escapeNodeLabel(label)
	}

	// Use @{} syntax if forced or if it's a new shape
	if node.UseNewSyntax || newShapes[node.Shape] {
		attrs := []string{}
		attrs = append(attrs, fmt.Sprintf("shape: %s", node.Shape))
		if node.Label != "" && node.Label != node.ID {
			attrs = append(attrs, fmt.Sprintf("label: \"%s\"", node.Label))
		}
		for key, value := range node.Attributes {
			attrs = append(attrs, fmt.Sprintf("%s: %s", key, value))
		}
		return fmt.Sprintf("    %s@{ %s}", sanitiseNodeID(node.ID), strings.Join(attrs, ", "))
	}

	// Use classic bracket notation for old shapes
	if format, ok := classicShapes[node.Shape]; ok {
		return fmt.Sprintf("    %s"+format, sanitiseNodeID(node.ID), label)
	}

	// Default to rectangle
	shape := node.Shape
	if shape == "" {
		shape = "rectangle"
	}

	if format, ok := classicShapes[shape]; ok {
		return fmt.Sprintf("    %s"+format, sanitiseNodeID(node.ID), label)
	}

	return fmt.Sprintf("    %s[%s]", sanitiseNodeID(node.ID), label)
}

func formatFlowchartLink(link FlowchartLink) string {
	arrowType := link.ArrowType
	if arrowType == "" {
		arrowType = "-->"
	}

	if link.Length > 0 {
		arrowType = extendArrow(arrowType, link.Length)
	}

	fromID := sanitiseNodeID(link.From)
	toID := sanitiseNodeID(link.To)

	var linkStr string
	if link.Text != "" {
		if strings.Contains(arrowType, "-.") {
			// dotted lines use different text format
			if strings.HasSuffix(arrowType, ">") {
				linkStr = fmt.Sprintf("    %s -. %s .-> %s", fromID, escapeNodeLabel(link.Text), toID)
			} else {
				linkStr = fmt.Sprintf("    %s -. %s .- %s", fromID, escapeNodeLabel(link.Text), toID)
			}
		} else if strings.Contains(arrowType, "==") {
			// thick lines
			if strings.HasSuffix(arrowType, ">") {
				linkStr = fmt.Sprintf("    %s == %s ==> %s", fromID, escapeNodeLabel(link.Text), toID)
			} else {
				linkStr = fmt.Sprintf("    %s == %s === %s", fromID, escapeNodeLabel(link.Text), toID)
			}
		} else {
			// standard format with pipes
			linkStr = fmt.Sprintf("    %s %s|%s| %s", fromID, arrowType, escapeNodeLabel(link.Text), toID)
		}
	} else {
		linkStr = fmt.Sprintf("    %s %s %s", fromID, arrowType, toID)
	}

	return linkStr
}

func extendArrow(arrowType string, extraLength int) string {
	switch arrowType {
	case "-->":
		return strings.Repeat("-", extraLength+2) + ">"
	case "---":
		return strings.Repeat("-", extraLength+3)
	case "-.->":
		// Mermaid uses dots for extended dotted lines: -..-> not -.-.-
		if extraLength == 0 {
			return "-.->"
		}
		return "-" + strings.Repeat(".", extraLength+1) + "->"
	case "-.-":
		// Dotted line without arrow
		if extraLength == 0 {
			return "-.-"
		}
		return "-" + strings.Repeat(".", extraLength+1) + "-"
	case "==>":
		return strings.Repeat("=", extraLength+2) + ">"
	case "===":
		return strings.Repeat("=", extraLength+3)
	case "~~~":
		return strings.Repeat("~", extraLength+3)
	case "<-->":
		return "<" + strings.Repeat("-", extraLength+2) + ">"
	case "--o":
		return strings.Repeat("-", extraLength+2) + "o"
	case "--x":
		return strings.Repeat("-", extraLength+2) + "x"
	case "o--o":
		return "o" + strings.Repeat("-", extraLength+2) + "o"
	case "x--x":
		return "x" + strings.Repeat("-", extraLength+2) + "x"
	default:
		return arrowType
	}
}

func formatSubgraph(sg FlowchartSubgraph, nodeMap map[string]FlowchartNode) []string {
	var lines []string

	if sg.ID != "" {
		lines = append(lines, fmt.Sprintf("    subgraph %s [%s]", sanitiseNodeID(sg.ID), escapeNodeLabel(sg.Title)))
	} else {
		lines = append(lines, fmt.Sprintf("    subgraph %s", escapeNodeLabel(sg.Title)))
	}

	if sg.Direction != "" {
		lines = append(lines, fmt.Sprintf("        direction %s", sg.Direction))
	}

	for _, nodeID := range sg.Nodes {
		if node, exists := nodeMap[nodeID]; exists {
			nodeLine := formatFlowchartNode(node)
			// subgraph nodes need extra indentation (8 spaces total)
			lines = append(lines, "    "+nodeLine)
		} else {
			lines = append(lines, fmt.Sprintf("        %s", sanitiseNodeID(nodeID)))
		}
	}

	lines = append(lines, "    end")
	return lines
}

func sanitiseNodeID(id string) string {
	// Replace spaces and special characters with underscores for valid Mermaid IDs
	sanitized := id

	// Replace spaces with underscores
	sanitized = strings.ReplaceAll(sanitized, " ", "_")

	// Replace other problematic characters
	sanitized = strings.ReplaceAll(sanitized, "-", "_")
	sanitized = strings.ReplaceAll(sanitized, ">", "_")
	sanitized = strings.ReplaceAll(sanitized, "<", "_")
	sanitized = strings.ReplaceAll(sanitized, "[", "_")
	sanitized = strings.ReplaceAll(sanitized, "]", "_")
	sanitized = strings.ReplaceAll(sanitized, "{", "_")
	sanitized = strings.ReplaceAll(sanitized, "}", "_")
	sanitized = strings.ReplaceAll(sanitized, "(", "_")
	sanitized = strings.ReplaceAll(sanitized, ")", "_")
	sanitized = strings.ReplaceAll(sanitized, "#", "_")
	sanitized = strings.ReplaceAll(sanitized, ".", "_")
	sanitized = strings.ReplaceAll(sanitized, ":", "_")
	sanitized = strings.ReplaceAll(sanitized, "|", "_")

	// If ID starts with a number, prepend "node_"
	if len(sanitized) > 0 && sanitized[0] >= '0' && sanitized[0] <= '9' {
		sanitized = "node_" + sanitized
	}

	// Check for reserved keywords
	reserved := []string{"graph", "flowchart", "subgraph", "end", "style", "classDef", "class", "click", "direction"}
	for _, r := range reserved {
		if strings.EqualFold(sanitized, r) {
			return sanitized + "_node"
		}
	}

	return sanitized
}

func escapeNodeLabel(label string) string {
	label = strings.ReplaceAll(label, "&", "&amp;")
	label = strings.ReplaceAll(label, "\"", "&quot;")
	label = strings.ReplaceAll(label, "'", "&apos;")
	label = strings.ReplaceAll(label, "<", "&lt;")
	label = strings.ReplaceAll(label, ">", "&gt;")

	if strings.ContainsAny(label, "|[]{}()") {
		return fmt.Sprintf("\"%s\"", label)
	}
	return label
}

func escapeString(s string) string {
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "'", "\\'")
	return s
}

func validateFlowchartArgs(args FlowchartArgs) error {
	if len(args.Links) == 0 {
		return fmt.Errorf("links must contain at least one item")
	}

	if args.Direction != "" {
		validDirections := map[string]bool{"TB": true, "TD": true, "BT": true, "LR": true, "RL": true}
		if !validDirections[args.Direction] {
			return fmt.Errorf("invalid direction: %s (must be TB, TD, BT, LR, or RL)", args.Direction)
		}
	}

	nodeIDs := make(map[string]bool)
	for _, node := range args.Nodes {
		if node.ID == "" {
			return fmt.Errorf("node ID cannot be empty")
		}
		nodeIDs[node.ID] = true
	}

	for _, link := range args.Links {
		if link.From == "" {
			return fmt.Errorf("link 'from' field cannot be empty")
		}
		if link.To == "" {
			return fmt.Errorf("link 'to' field cannot be empty")
		}
		nodeIDs[link.From] = true
		nodeIDs[link.To] = true

		if link.ArrowType != "" {
			validArrows := map[string]bool{
				"-->": true, "---": true, "-.->": true, "-.-": true,
				"==>": true, "===": true, "~~~": true,
				"--o": true, "--x": true, "<-->": true,
				"o--o": true, "x--x": true,
			}
			if !validArrows[link.ArrowType] {
				return fmt.Errorf("invalid arrow type: %s", link.ArrowType)
			}
		}

		if link.Interpolate != "" {
			validInterpolations := map[string]bool{
				"basis": true, "linear": true, "monotoneX": true,
				"monotoneY": true, "natural": true, "step": true,
				"stepAfter": true, "stepBefore": true,
			}
			if !validInterpolations[link.Interpolate] {
				return fmt.Errorf("invalid interpolation type: %s", link.Interpolate)
			}
		}
	}

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

	for _, style := range args.Styles {
		if style.Target == "" {
			return fmt.Errorf("style target cannot be empty")
		}
		if style.Properties == "" {
			return fmt.Errorf("style properties cannot be empty")
		}
	}

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

	for _, ls := range args.LinkStyles {
		if len(ls.LinkNumbers) == 0 {
			return fmt.Errorf("link style must specify at least one link number")
		}
		if ls.Properties == "" {
			return fmt.Errorf("link style properties cannot be empty")
		}
		for _, num := range ls.LinkNumbers {
			if num < 0 || num >= len(args.Links) {
				return fmt.Errorf("link style references invalid link index: %d (must be 0-%d)", num, len(args.Links)-1)
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
		mcp.WithDescription(`
Generates a Mermaid flowchart diagram DSL.
Flowcharts visualize processes, algorithms, workflows, and decision trees using nodes (shapes) and directed edges (arrows).
Use this for process flows, decision logic, system architectures, algorithm visualization, or any directed graph representation.

The tool generates valid Mermaid DSL that can be rendered in Markdown, documentation, or Mermaid-compatible tools.

Key features:
- Flow direction control (top-down, left-right, etc.)
- Multiple node shapes (rectangle, circle, diamond, hexagon, cylinder, stadium, etc.)
- Various arrow types (solid, dotted, thick, bidirectional, circle/cross edges)
- Link text labels with proper formatting
- Variable link lengths for layout control
- Subgraphs for grouping related nodes
- Node and link styling with CSS properties
- Class definitions for reusable styles
- Click events and tooltips for interactive diagrams
- Markdown support in node labels
- Link interpolation for curved edges
- Comments for documentation
- Support for 30+ new Mermaid shapes with @{} syntax

Classic Node shapes:
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

New shapes (use @{} syntax):
  manual-file  : manual file operation
  manual-input : manual input step
  docs         : multiple documents
  procs        : multiple processes
  paper-tape   : paper tape
  database     : database storage
  And 30+ more shapes introduced in Mermaid 2024

Arrow types:
  -->  : solid arrow (default)
  ---  : solid line without arrow
  -.-> : dotted arrow
  -.-  : dotted line
  ==>  : thick arrow
  ===  : thick line
  ~~~  : invisible link (for layout)
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
