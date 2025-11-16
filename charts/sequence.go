package main

import (
	"context"
	"fmt"
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
	ArrowType  string `json:"arrowType,omitempty" jsonschema:"description=Arrow type,enum=->,enum=-->,enum=->>,enum=-->,enum=<<->>,enum=<<-->>,enum=-x,enum=--x,enum=-),enum=--)"`
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
	Color        string   `json:"color,omitempty" jsonschema:"description=Box color (e.g. 'Aqua', 'rgb(33,66,99)', 'transparent')"`
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

// SequenceDiagramArgs represents the arguments for generating a sequence diagram
type SequenceDiagramArgs struct {
	Title        string        `json:"title,omitempty" jsonschema:"description=Diagram title"`
	AutoNumber   bool          `json:"autoNumber,omitempty" jsonschema:"description=Enable automatic message numbering"`
	Participants []Participant `json:"participants,omitempty" jsonschema:"description=Explicit participant definitions (optional - participants are auto-detected from messages)"`
	Messages     []Message     `json:"messages" jsonschema:"description=Array of messages between participants,required,minItems=1"`
	Notes        []Note        `json:"notes,omitempty" jsonschema:"description=Array of notes to add"`
	Boxes        []Box         `json:"boxes,omitempty" jsonschema:"description=Array of boxes to group participants"`
	Loops        []Loop        `json:"loops,omitempty" jsonschema:"description=Array of loop blocks"`
	Alts         []Alt         `json:"alts,omitempty" jsonschema:"description=Array of alt/opt blocks"`
}

func generateSequenceDiagramDSL(args SequenceDiagramArgs) string {
	var lines []string

	// start with sequenceDiagram keyword
	lines = append(lines, "sequenceDiagram")

	// add title if provided
	if args.Title != "" {
		lines = append(lines, fmt.Sprintf("    title: %s", args.Title))
	}

	// add autonumber if enabled
	if args.AutoNumber {
		lines = append(lines, "    autonumber")
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

		// boxes need participant definitions
		for _, pid := range box.Participants {
			// find matching participant definition
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

	// add messages
	for _, msg := range args.Messages {
		lines = append(lines, formatMessage(msg))
	}

	// add loops
	for _, loop := range args.Loops {
		lines = append(lines, fmt.Sprintf("    loop %s", loop.Text))
		for _, msg := range loop.Messages {
			lines = append(lines, "    "+formatMessage(msg))
		}
		lines = append(lines, "    end")
	}

	// add alts
	for _, alt := range args.Alts {
		lines = append(lines, fmt.Sprintf("    alt %s", alt.IfText))
		for _, msg := range alt.IfMessages {
			lines = append(lines, "    "+formatMessage(msg))
		}
		if alt.ElseText != "" {
			lines = append(lines, fmt.Sprintf("    else %s", alt.ElseText))
			for _, msg := range alt.ElseMessages {
				lines = append(lines, "    "+formatMessage(msg))
			}
		}
		lines = append(lines, "    end")
	}

	// add notes
	for _, note := range args.Notes {
		lines = append(lines, formatNote(note))
	}

	return strings.Join(lines, "\n")
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

func validateSequenceDiagramArgs(args SequenceDiagramArgs) error {
	if len(args.Messages) == 0 {
		return fmt.Errorf("messages must contain at least one item")
	}

	// validate participants referenced in messages exist
	participantIDs := make(map[string]bool)
	for _, p := range args.Participants {
		if p.ID == "" {
			return fmt.Errorf("participant ID cannot be empty")
		}
		participantIDs[p.ID] = true
	}

	// collect all participant IDs from messages (auto-participants)
	for _, msg := range args.Messages {
		if msg.From == "" {
			return fmt.Errorf("message 'from' field cannot be empty")
		}
		if msg.To == "" {
			return fmt.Errorf("message 'to' field cannot be empty")
		}
		participantIDs[msg.From] = true
		participantIDs[msg.To] = true
	}

	// validate notes reference valid participants
	for _, note := range args.Notes {
		if len(note.Participants) == 0 {
			return fmt.Errorf("note must reference at least one participant")
		}
		if note.Position == "over" && len(note.Participants) > 2 {
			return fmt.Errorf("note 'over' can reference at most two participants")
		}
		if (note.Position == "right of" || note.Position == "left of") && len(note.Participants) != 1 {
			return fmt.Errorf("note '%s' must reference exactly one participant", note.Position)
		}
		for _, pid := range note.Participants {
			if !participantIDs[pid] {
				return fmt.Errorf("note references unknown participant: %s", pid)
			}
		}
	}

	// validate boxes reference valid participants
	for _, box := range args.Boxes {
		if len(box.Participants) == 0 {
			return fmt.Errorf("box must contain at least one participant")
		}
		for _, pid := range box.Participants {
			if !participantIDs[pid] {
				return fmt.Errorf("box references unknown participant: %s", pid)
			}
		}
	}

	// validate loops
	for _, loop := range args.Loops {
		if loop.Text == "" {
			return fmt.Errorf("loop text cannot be empty")
		}
		if len(loop.Messages) == 0 {
			return fmt.Errorf("loop must contain at least one message")
		}
	}

	// validate alts
	for _, alt := range args.Alts {
		if alt.IfText == "" {
			return fmt.Errorf("alt ifText cannot be empty")
		}
		if len(alt.IfMessages) == 0 {
			return fmt.Errorf("alt must contain at least one message in if block")
		}
		if alt.ElseText != "" && len(alt.ElseMessages) == 0 {
			return fmt.Errorf("alt with elseText must have elseMessages")
		}
	}

	return nil
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
- Auto-numbering of messages

Arrow types:
  ->>  : solid line with arrowhead (default, synchronous call)
  -->> : dotted line with arrowhead (response/return)
  ->   : solid line without arrow
  -->  : dotted line without arrow
  -x   : solid line with cross (lost message)
  --x  : dotted line with cross
  -)   : solid line with open arrow (asynchronous)
  --)  : dotted line with open arrow
  <<->>  : bidirectional solid
  <<-->> : bidirectional dotted`),
		mcp.WithInputSchema[SequenceDiagramArgs](),
	)

	srv.AddTool(tool, handler)
}
