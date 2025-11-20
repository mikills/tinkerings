package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Participant represents a participant in the sequence diagram
type Participant struct {
	ID    string `json:"id" jsonschema:"description=Unique identifier for the participant,required"`
	Label string `json:"label,omitempty" jsonschema:"description=Display label (if different from ID)"`
	Type  string `json:"type,omitempty" jsonschema:"description=Participant type: participant, actor, boundary, control, entity, database, collections, queue,enum=participant,enum=actor,enum=boundary,enum=control,enum=entity,enum=database,enum=collections,enum=queue"`
}

// Message represents a message between participants
type Message struct {
	From       string `json:"from" jsonschema:"description=Source participant ID,required"`
	To         string `json:"to" jsonschema:"description=Target participant ID,required"`
	Text       string `json:"text" jsonschema:"description=Message text,required"`
	ArrowType  string `json:"arrowType,omitempty" jsonschema:"description=Arrow type,enum=->,enum=-->,enum=->>,enum=-->>,enum=-x,enum=--x,enum=-),enum=--),enum=<<->>,enum=<<-->>"`
	Activate   bool   `json:"activate,omitempty" jsonschema:"description=Activate the target participant"`
	Deactivate bool   `json:"deactivate,omitempty" jsonschema:"description=Deactivate the target participant"`
}

// Note represents a note in the sequence diagram
type Note struct {
	Position     string   `json:"position" jsonschema:"description=Note position: right of, left of, or over,required,enum=right of,enum=left of,enum=over"`
	Participants []string `json:"participants" jsonschema:"description=Participant ID(s) - one for 'right of'/'left of', one or two for 'over',required,minItems=1,maxItems=2"`
	Text         string   `json:"text" jsonschema:"description=Note text,required"`
}

// Box represents a grouping box for participants
type Box struct {
	Label        string   `json:"label,omitempty" jsonschema:"description=Box label"`
	Color        string   `json:"color,omitempty" jsonschema:"description=Box color (e.g. 'Aqua', 'rgb(33,66,99)', 'rgba(0,0,0,0.5)', 'transparent')"`
	Participants []string `json:"participants" jsonschema:"description=Participant IDs to group,required,minItems=1"`
}

// Loop represents a loop block
type Loop struct {
	Text     string    `json:"text" jsonschema:"description=Loop description,required"`
	Messages []Message `json:"messages" jsonschema:"description=Messages within the loop,required,minItems=1"`
}

// Alt represents an alt/else block
type Alt struct {
	IfText       string    `json:"ifText" jsonschema:"description=If condition description,required"`
	IfMessages   []Message `json:"ifMessages" jsonschema:"description=Messages in if block,required,minItems=1"`
	ElseText     string    `json:"elseText,omitempty" jsonschema:"description=Else condition description (omit for opt)"`
	ElseMessages []Message `json:"elseMessages,omitempty" jsonschema:"description=Messages in else block"`
}

// SequenceElement represents any element that appears in sequence (messages, create/destroy, etc.)
type SequenceElement struct {
	Type     string         `json:"type" jsonschema:"description=Element type,enum=message,enum=create,enum=destroy,enum=loop,enum=alt,enum=parallel,enum=critical,enum=break,enum=rect,enum=note,required"`
	Message  *Message       `json:"message,omitempty" jsonschema:"description=Message details (when type=message)"`
	Create   *CreateDestroy `json:"create,omitempty" jsonschema:"description=Create details (when type=create)"`
	Destroy  *CreateDestroy `json:"destroy,omitempty" jsonschema:"description=Destroy details (when type=destroy)"`
	Loop     *Loop          `json:"loop,omitempty" jsonschema:"description=Loop details (when type=loop)"`
	Alt      *Alt           `json:"alt,omitempty" jsonschema:"description=Alt details (when type=alt)"`
	Parallel *Parallel      `json:"parallel,omitempty" jsonschema:"description=Parallel details (when type=parallel)"`
	Critical *Critical      `json:"critical,omitempty" jsonschema:"description=Critical details (when type=critical)"`
	Break    *Break         `json:"break,omitempty" jsonschema:"description=Break details (when type=break)"`
	Rect     *Rect          `json:"rect,omitempty" jsonschema:"description=Rect details (when type=rect)"`
	Note     *Note          `json:"note,omitempty" jsonschema:"description=Note details (when type=note)"`
}

// Updated CreateDestroy to be simpler
type CreateDestroy struct {
	ParticipantID    string `json:"participantId" jsonschema:"description=Participant ID,required"`
	ParticipantType  string `json:"participantType,omitempty" jsonschema:"description=Type when creating (participant, actor, etc.),enum=participant,enum=actor,enum=boundary,enum=control,enum=entity,enum=database,enum=collections,enum=queue"`
	ParticipantLabel string `json:"participantLabel,omitempty" jsonschema:"description=Label for the participant (for 'as' syntax)"`
}

// Updated SequenceDiagramArgs to use a sequential approach
type SequenceDiagramArgs struct {
	Title        string            `json:"title,omitempty" jsonschema:"description=Diagram title"`
	AutoNumber   bool              `json:"autoNumber,omitempty" jsonschema:"description=Enable automatic message numbering"`
	Participants []Participant     `json:"participants,omitempty" jsonschema:"description=Participants to declare at the start (not created dynamically)"`
	Boxes        []Box             `json:"boxes,omitempty" jsonschema:"description=Boxes to group participants"`
	Elements     []SequenceElement `json:"elements" jsonschema:"description=Sequence elements in order,required,minItems=1"`
	Comments     []string          `json:"comments,omitempty" jsonschema:"description=Comments to add at the top"`
}

// ParallelBranch represents a branch in a parallel block
type ParallelBranch struct {
	Text     string    `json:"text" jsonschema:"description=Branch description,required"`
	Messages []Message `json:"messages" jsonschema:"description=Messages in this parallel branch,required,minItems=1"`
}

// Parallel represents a parallel execution block
type Parallel struct {
	Branches []ParallelBranch `json:"branches" jsonschema:"description=Parallel branches (at least 2 required),required,minItems=2"`
}

// CriticalOption represents an option in a critical block
type CriticalOption struct {
	Text     string    `json:"text" jsonschema:"description=Option description,required"`
	Messages []Message `json:"messages" jsonschema:"description=Messages in this option,required,minItems=1"`
}

// Critical represents a critical region
type Critical struct {
	Text     string           `json:"text" jsonschema:"description=Critical action description,required"`
	Messages []Message        `json:"messages" jsonschema:"description=Messages in critical section,required,minItems=1"`
	Options  []CriticalOption `json:"options,omitempty" jsonschema:"description=Optional handling options"`
}

// Break represents a break/exception in the flow
type Break struct {
	Text     string    `json:"text" jsonschema:"description=Break condition description,required"`
	Messages []Message `json:"messages" jsonschema:"description=Messages before break,required,minItems=1"`
}

// Rect represents a background highlight rectangle
type Rect struct {
	Color    string    `json:"color" jsonschema:"description=Rectangle color in rgb(r,g,b) or rgba(r,g,b,a) format,required,pattern=^rgba?\\([0-9]{1\\,3}\\,[0-9]{1\\,3}\\,[0-9]{1\\,3}(\\,[0-9\\.]+)?\\)$"`
	Messages []Message `json:"messages" jsonschema:"description=Messages within the highlighted area,required,minItems=1"`
}

func formatParticipant(p Participant) string {
	pType := p.Type
	if pType == "" {
		pType = "participant"
	}

	line := fmt.Sprintf("    %s %s", pType, p.ID)
	if p.Label != "" {
		line += fmt.Sprintf(" as %s", p.Label)
	}
	return line
}

func formatMessage(msg Message) string {
	arrowType := msg.ArrowType
	if arrowType == "" {
		arrowType = "->>"
	}

	// add activation suffixes
	if msg.Activate {
		arrowType += "+"
	} else if msg.Deactivate {
		arrowType += "-"
	}

	return fmt.Sprintf("    %s%s%s: %s", msg.From, arrowType, msg.To, msg.Text)
}

func formatNote(note Note) string {
	participantPart := strings.Join(note.Participants, ",")
	return fmt.Sprintf("    Note %s %s: %s", note.Position, participantPart, note.Text)
}

var rgbRegex = regexp.MustCompile(`^rgba?\(\s*\d{1,3}\s*,\s*\d{1,3}\s*,\s*\d{1,3}\s*(,\s*[0-9.]+\s*)?\)$`)

var validColors = map[string]bool{
	"Aqua": true, "Black": true, "Blue": true, "Brown": true,
	"Cyan": true, "Gray": true, "Grey": true, "Green": true,
	"Lime": true, "Magenta": true, "Navy": true, "Orange": true,
	"Pink": true, "Purple": true, "Red": true, "Silver": true,
	"White": true, "Yellow": true, "Teal": true, "Indigo": true,
}

func isValidColor(color string) bool {
	// check for transparent
	if color == "transparent" {
		return true
	}

	// check for rgb/rgba format
	if rgbRegex.MatchString(color) {
		return true
	}

	// common CSS color names (extend as needed)
	return validColors[color]
}

func registerSequenceDiagramTool(srv *server.MCPServer) {
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args SequenceDiagramArgs
		if err := req.BindArguments(&args); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("bind arguments: %v", err)), nil
		}

		if err := validateSequenceDiagramArgs(args); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		dsl := generateSequenceDiagramDSL(args)
		return mcp.NewToolResultText(dsl), nil
	}

	tool := mcp.NewTool(
		"sequence-diagram-generator",
		mcp.WithDescription(`Generates a Mermaid sequence diagram DSL.
Sequence diagrams show how processes or objects interact with one another and in what order.
Use this for depicting API flows, communication protocols, multi-actor workflows, or any sequential interaction patterns.

The tool generates valid Mermaid DSL that can be rendered in Markdown, documentation, or Mermaid-compatible tools.

Key features:
- Multiple participant types (participant, actor, boundary, control, entity, database, collections, queue)
- Various arrow types for different message semantics (solid/dotted, sync/async, bidirectional, with/without arrows)
- Activations to show when participants are processing
- Notes positioned relative to participants
- Boxes to group related participants
- Loops for repeated interactions
- Alt/Opt for conditional flows
- Parallel blocks for concurrent execution
- Critical regions with optional circumstances
- Break blocks for exceptions
- Background highlighting with colored rectangles
- Participant creation and destruction
- Comments for documentation
- Auto-numbering of messages

Arrow types:
  ->   : solid line without arrow
  -->  : dotted line without arrow
  ->>  : solid line with arrowhead (default, synchronous call)
  -->> : dotted line with arrowhead (response/return)
  -x   : solid line with cross (lost message)
  --x  : dotted line with cross
  -)   : solid line with open arrow (asynchronous)
  --)  : dotted line with open arrow
  <<->>  : bidirectional solid (v11.0.0+)
  <<-->> : bidirectional dotted (v11.0.0+)

Activation suffixes:
  +    : activate the target
  -    : deactivate the target`),
		mcp.WithInputSchema[SequenceDiagramArgs](),
	)

	srv.AddTool(tool, handler)
}

func generateSequenceDiagramDSL(args SequenceDiagramArgs) string {
	var lines []string

	// start with sequenceDiagram keyword
	lines = append(lines, "sequenceDiagram")

	// add title if provided
	if args.Title != "" {
		lines = append(lines, fmt.Sprintf("    title %s", args.Title))
	}

	// add autonumber if enabled
	if args.AutoNumber {
		lines = append(lines, "    autonumber")
	}

	// add comments at the top if any
	for _, comment := range args.Comments {
		lines = append(lines, "    %% "+comment)
	}

	// add boxes if provided
	for _, box := range args.Boxes {
		boxLine := "    box"
		if box.Color != "" {
			boxLine += " " + box.Color
		}
		if box.Label != "" {
			boxLine += " " + box.Label
		}
		lines = append(lines, boxLine)

		for _, pid := range box.Participants {
			var p *Participant
			for i := range args.Participants {
				if args.Participants[i].ID == pid {
					p = &args.Participants[i]
					break
				}
			}
			if p != nil {
				lines = append(lines, formatParticipant(*p))
			} else {
				lines = append(lines, fmt.Sprintf("        participant %s", pid))
			}
		}
		lines = append(lines, "    end")
	}

	// add explicit participants (those not in boxes)
	boxedParticipants := make(map[string]bool)
	for _, box := range args.Boxes {
		for _, pid := range box.Participants {
			boxedParticipants[pid] = true
		}
	}

	for _, p := range args.Participants {
		if !boxedParticipants[p.ID] {
			lines = append(lines, formatParticipant(p))
		}
	}

	// process elements in sequence
	for _, elem := range args.Elements {
		switch elem.Type {
		case "message":
			if elem.Message != nil {
				lines = append(lines, formatMessage(*elem.Message))
			}
		case "create":
			if elem.Create != nil {
				createLine := "    create"
				pType := elem.Create.ParticipantType
				if pType == "" {
					pType = "participant"
				}
				createLine += " " + pType + " " + elem.Create.ParticipantID
				if elem.Create.ParticipantLabel != "" {
					createLine += " as " + elem.Create.ParticipantLabel
				}
				lines = append(lines, createLine)
			}
		case "destroy":
			if elem.Destroy != nil {
				lines = append(lines, fmt.Sprintf("    destroy %s", elem.Destroy.ParticipantID))
			}
		case "loop":
			if elem.Loop != nil {
				lines = append(lines, fmt.Sprintf("    loop %s", elem.Loop.Text))
				for _, msg := range elem.Loop.Messages {
					lines = append(lines, "    "+formatMessage(msg))
				}
				lines = append(lines, "    end")
			}
		case "alt":
			if elem.Alt != nil {
				if elem.Alt.ElseText == "" && len(elem.Alt.ElseMessages) == 0 {
					lines = append(lines, fmt.Sprintf("    opt %s", elem.Alt.IfText))
				} else {
					lines = append(lines, fmt.Sprintf("    alt %s", elem.Alt.IfText))
				}
				for _, msg := range elem.Alt.IfMessages {
					lines = append(lines, "    "+formatMessage(msg))
				}
				if elem.Alt.ElseText != "" {
					lines = append(lines, fmt.Sprintf("    else %s", elem.Alt.ElseText))
					for _, msg := range elem.Alt.ElseMessages {
						lines = append(lines, "    "+formatMessage(msg))
					}
				}
				lines = append(lines, "    end")
			}
		case "parallel":
			if elem.Parallel != nil && len(elem.Parallel.Branches) > 0 {
				lines = append(lines, fmt.Sprintf("    par %s", elem.Parallel.Branches[0].Text))
				for _, msg := range elem.Parallel.Branches[0].Messages {
					lines = append(lines, "    "+formatMessage(msg))
				}
				for i := 1; i < len(elem.Parallel.Branches); i++ {
					lines = append(lines, fmt.Sprintf("    and %s", elem.Parallel.Branches[i].Text))
					for _, msg := range elem.Parallel.Branches[i].Messages {
						lines = append(lines, "    "+formatMessage(msg))
					}
				}
				lines = append(lines, "    end")
			}
		case "critical":
			if elem.Critical != nil {
				lines = append(lines, fmt.Sprintf("    critical %s", elem.Critical.Text))
				for _, msg := range elem.Critical.Messages {
					lines = append(lines, "    "+formatMessage(msg))
				}
				for _, opt := range elem.Critical.Options {
					lines = append(lines, fmt.Sprintf("    option %s", opt.Text))
					for _, msg := range opt.Messages {
						lines = append(lines, "    "+formatMessage(msg))
					}
				}
				lines = append(lines, "    end")
			}
		case "break":
			if elem.Break != nil {
				lines = append(lines, fmt.Sprintf("    break %s", elem.Break.Text))
				for _, msg := range elem.Break.Messages {
					lines = append(lines, "    "+formatMessage(msg))
				}
				lines = append(lines, "    end")
			}
		case "rect":
			if elem.Rect != nil {
				lines = append(lines, fmt.Sprintf("    rect %s", elem.Rect.Color))
				for _, msg := range elem.Rect.Messages {
					lines = append(lines, "    "+formatMessage(msg))
				}
				lines = append(lines, "    end")
			}
		case "note":
			if elem.Note != nil {
				lines = append(lines, formatNote(*elem.Note))
			}
		}
	}

	return strings.Join(lines, "\n")
}

func validateSequenceDiagramArgs(args SequenceDiagramArgs) error {
	if len(args.Elements) == 0 {
		return fmt.Errorf("elements must contain at least one item")
	}

	// track active participants (created but not destroyed)
	activeParticipants := make(map[string]bool)

	// add pre-declared participants
	for _, p := range args.Participants {
		if p.ID == "" {
			return fmt.Errorf("participant ID cannot be empty")
		}
		activeParticipants[p.ID] = true
	}

	// add participants from boxes
	for _, box := range args.Boxes {
		for _, pid := range box.Participants {
			activeParticipants[pid] = true
		}
	}

	// validate elements in sequence
	for i, elem := range args.Elements {
		switch elem.Type {
		case "message":
			if elem.Message == nil {
				return fmt.Errorf("element %d: message element must have message details", i)
			}
			msg := elem.Message
			if msg.From == "" || msg.To == "" || msg.Text == "" {
				return fmt.Errorf("element %d: message must have from, to, and text fields", i)
			}

			// auto-add participants if not explicitly declared
			if !activeParticipants[msg.From] {
				activeParticipants[msg.From] = true
			}
			if !activeParticipants[msg.To] {
				activeParticipants[msg.To] = true
			}

			if msg.Activate && msg.Deactivate {
				return fmt.Errorf("element %d: message cannot both activate and deactivate", i)
			}

			// validate arrow type
			if msg.ArrowType != "" {
				validArrowTypes := map[string]bool{
					"->": true, "-->": true, "->>": true, "-->>": true,
					"-x": true, "--x": true, "-)": true, "--)": true,
					"<<->>": true, "<<-->>": true,
				}
				if !validArrowTypes[msg.ArrowType] {
					return fmt.Errorf("element %d: invalid arrow type: %s", i, msg.ArrowType)
				}
			}

		case "create":
			if elem.Create == nil {
				return fmt.Errorf("element %d: create element must have create details", i)
			}
			if elem.Create.ParticipantID == "" {
				return fmt.Errorf("element %d: create participant ID cannot be empty", i)
			}
			if activeParticipants[elem.Create.ParticipantID] {
				return fmt.Errorf("element %d: cannot create participant %s - already exists", i, elem.Create.ParticipantID)
			}
			activeParticipants[elem.Create.ParticipantID] = true

		case "destroy":
			if elem.Destroy == nil {
				return fmt.Errorf("element %d: destroy element must have destroy details", i)
			}
			if elem.Destroy.ParticipantID == "" {
				return fmt.Errorf("element %d: destroy participant ID cannot be empty", i)
			}
			if !activeParticipants[elem.Destroy.ParticipantID] {
				return fmt.Errorf("element %d: cannot destroy participant %s - does not exist", i, elem.Destroy.ParticipantID)
			}
			delete(activeParticipants, elem.Destroy.ParticipantID)

		case "loop":
			if elem.Loop == nil {
				return fmt.Errorf("element %d: loop element must have loop details", i)
			}
			if elem.Loop.Text == "" {
				return fmt.Errorf("element %d: loop text cannot be empty", i)
			}
			if len(elem.Loop.Messages) == 0 {
				return fmt.Errorf("element %d: loop must contain at least one message", i)
			}

		case "alt":
			if elem.Alt == nil {
				return fmt.Errorf("element %d: alt element must have alt details", i)
			}
			if elem.Alt.IfText == "" {
				return fmt.Errorf("element %d: alt ifText cannot be empty", i)
			}
			if len(elem.Alt.IfMessages) == 0 {
				return fmt.Errorf("element %d: alt must have at least one if message", i)
			}

		case "note":
			if elem.Note == nil {
				return fmt.Errorf("element %d: note element must have note details", i)
			}
			if elem.Note.Text == "" {
				return fmt.Errorf("element %d: note text cannot be empty", i)
			}
			// validate note participants exist
			for _, pid := range elem.Note.Participants {
				if !activeParticipants[pid] {
					return fmt.Errorf("element %d: note references unknown participant: %s", i, pid)
				}
			}

		}
	}

	// validate boxes reference valid participants
	for _, box := range args.Boxes {
		if len(box.Participants) == 0 {
			return fmt.Errorf("box must contain at least one participant")
		}
		if box.Color != "" && !isValidColor(box.Color) {
			return fmt.Errorf("invalid box color: %s", box.Color)
		}
	}

	return nil
}
