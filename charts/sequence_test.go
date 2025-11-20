package main

import (
	"strings"
	"testing"
)

func TestBoxesWithParticipants(t *testing.T) {
	args := SequenceDiagramArgs{
		Boxes: []Box{
			{
				Color:        "Purple",
				Label:        "Alice & John",
				Participants: []string{"A", "J"},
			},
			{
				Label:        "Another Group",
				Participants: []string{"B", "C"},
			},
		},
		Elements: []SequenceElement{
			{Type: "message", Message: &Message{From: "A", To: "J", Text: "Hello John, how are you?", ArrowType: "->>"}},
			{Type: "message", Message: &Message{From: "J", To: "A", Text: "Great!", ArrowType: "->>"}},
			{Type: "message", Message: &Message{From: "A", To: "B", Text: "Hello Bob, how is Charley?", ArrowType: "->>"}},
			{Type: "message", Message: &Message{From: "B", To: "C", Text: "Hello Charley, how are you?", ArrowType: "->>"}},
		},
	}

	want := `sequenceDiagram
    box Purple Alice & John
        participant A
        participant J
    end
    box Another Group
        participant B
        participant C
    end
    A->>J: Hello John, how are you?
    J->>A: Great!
    A->>B: Hello Bob, how is Charley?
    B->>C: Hello Charley, how are you?`

	err := validateSequenceDiagramArgs(args)
	if err != nil {
		t.Errorf("validateSequenceDiagramArgs() unexpected error = %v", err)
		return
	}

	got := generateSequenceDiagramDSL(args)

	// Compare line by line for better error messages
	gotLines := strings.Split(strings.TrimSpace(got), "\n")
	wantLines := strings.Split(strings.TrimSpace(want), "\n")

	if len(gotLines) != len(wantLines) {
		t.Errorf("generateSequenceDiagramDSL() line count = %v, want %v\nGot:\n%v\nWant:\n%v",
			len(gotLines), len(wantLines), got, want)
		return
	}

	for i := range gotLines {
		if strings.TrimSpace(gotLines[i]) != strings.TrimSpace(wantLines[i]) {
			t.Errorf("generateSequenceDiagramDSL() line %d:\ngot:  %v\nwant: %v",
				i+1, gotLines[i], wantLines[i])
		}
	}
}

func TestActorsWithMessages(t *testing.T) {
	args := SequenceDiagramArgs{
		Participants: []Participant{
			{ID: "Alice", Type: "actor"},
			{ID: "Bob", Type: "actor"},
		},
		Elements: []SequenceElement{
			{Type: "message", Message: &Message{From: "Alice", To: "Bob", Text: "Hi Bob", ArrowType: "->>"}},
			{Type: "message", Message: &Message{From: "Bob", To: "Alice", Text: "Hi Alice", ArrowType: "->>"}},
		},
	}

	want := `sequenceDiagram
    actor Alice
    actor Bob
    Alice->>Bob: Hi Bob
    Bob->>Alice: Hi Alice`

	err := validateSequenceDiagramArgs(args)
	if err != nil {
		t.Errorf("validateSequenceDiagramArgs() unexpected error = %v", err)
		return
	}

	got := generateSequenceDiagramDSL(args)

	// Compare line by line for better error messages
	gotLines := strings.Split(strings.TrimSpace(got), "\n")
	wantLines := strings.Split(strings.TrimSpace(want), "\n")

	if len(gotLines) != len(wantLines) {
		t.Errorf("generateSequenceDiagramDSL() line count = %v, want %v\nGot:\n%v\nWant:\n%v",
			len(gotLines), len(wantLines), got, want)
		return
	}

	for i := range gotLines {
		if strings.TrimSpace(gotLines[i]) != strings.TrimSpace(wantLines[i]) {
			t.Errorf("generateSequenceDiagramDSL() line %d:\ngot:  %v\nwant: %v",
				i+1, gotLines[i], wantLines[i])
		}
	}
}

func TestGenerateSequenceDiagramDSL(t *testing.T) {
	tests := []struct {
		name    string
		args    SequenceDiagramArgs
		want    string
		wantErr bool
	}{{
		name: "basic messages between participants",
		args: SequenceDiagramArgs{
			Elements: []SequenceElement{
				{Type: "message", Message: &Message{From: "Alice", To: "Bob", Text: "Hello Bob, how are you ?", ArrowType: "->>"}},
				{Type: "message", Message: &Message{From: "Bob", To: "Alice", Text: "Fine, thank you. And you?", ArrowType: "->>"}},
				{Type: "create", Create: &CreateDestroy{ParticipantID: "Carl", ParticipantType: "participant"}},
				{Type: "message", Message: &Message{From: "Alice", To: "Carl", Text: "Hi Carl!", ArrowType: "->>"}},
				{Type: "create", Create: &CreateDestroy{ParticipantID: "D", ParticipantType: "actor", ParticipantLabel: "Donald"}},
				{Type: "message", Message: &Message{From: "Carl", To: "D", Text: "Hi!", ArrowType: "->>"}},
				{Type: "destroy", Destroy: &CreateDestroy{ParticipantID: "Carl"}},
				{Type: "message", Message: &Message{From: "Alice", To: "Carl", Text: "We are too many", ArrowType: "-x"}},
				{Type: "destroy", Destroy: &CreateDestroy{ParticipantID: "Bob"}},
				{Type: "message", Message: &Message{From: "Bob", To: "Alice", Text: "I agree", ArrowType: "->>"}},
			},
		},
		want: `sequenceDiagram
    Alice->>Bob: Hello Bob, how are you ?
    Bob->>Alice: Fine, thank you. And you?
    create participant Carl
    Alice->>Carl: Hi Carl!
    create actor D as Donald
    Carl->>D: Hi!
    destroy Carl
    Alice-xCarl: We are too many
    destroy Bob
    Bob->>Alice: I agree`,
	},

		{
			name: "boxes with colored box and labels",
			args: SequenceDiagramArgs{
				Boxes: []Box{
					{
						Color:        "Purple",
						Label:        "Alice & John",
						Participants: []string{"A", "J"},
					},
					{
						Label:        "Another Group",
						Participants: []string{"B", "C"},
					},
				},
				Elements: []SequenceElement{
					{Type: "message", Message: &Message{From: "A", To: "J", Text: "Hello John, how are you?", ArrowType: "->>"}},
					{Type: "message", Message: &Message{From: "J", To: "A", Text: "Great!", ArrowType: "->>"}},
					{Type: "message", Message: &Message{From: "A", To: "B", Text: "Hello Bob, how is Charley?", ArrowType: "->>"}},
					{Type: "message", Message: &Message{From: "B", To: "C", Text: "Hello Charley, how are you?", ArrowType: "->>"}},
				},
			},
			want: `sequenceDiagram
    box Purple Alice & John
        participant A
        participant J
    end
    box Another Group
        participant B
        participant C
    end
    A->>J: Hello John, how are you?
    J->>A: Great!
    A->>B: Hello Bob, how is Charley?
    B->>C: Hello Charley, how are you?`,
		},
		{
			name: "actors instead of participants",
			args: SequenceDiagramArgs{
				Participants: []Participant{
					{ID: "Alice", Type: "actor"},
					{ID: "Bob", Type: "actor"},
				},
				Elements: []SequenceElement{
					{Type: "message", Message: &Message{From: "Alice", To: "Bob", Text: "Hi Bob", ArrowType: "->>"}},
					{Type: "message", Message: &Message{From: "Bob", To: "Alice", Text: "Hi Alice", ArrowType: "->>"}},
				},
			},
			want: `sequenceDiagram
    actor Alice
    actor Bob
    Alice->>Bob: Hi Bob
    Bob->>Alice: Hi Alice`,
		},
		{
			name: "box with transparent color",
			args: SequenceDiagramArgs{
				Boxes: []Box{
					{
						Color:        "transparent",
						Label:        "Transparent Group",
						Participants: []string{"X", "Y"},
					},
				},
				Elements: []SequenceElement{
					{Type: "message", Message: &Message{From: "X", To: "Y", Text: "Test", ArrowType: "->>"}},
				},
			},
			want: `sequenceDiagram
    box transparent Transparent Group
        participant X
        participant Y
    end
    X->>Y: Test`,
		},
		{
			name: "box with rgb color",
			args: SequenceDiagramArgs{
				Boxes: []Box{
					{
						Color:        "rgb(128,128,128)",
						Label:        "Gray Box",
						Participants: []string{"P1"},
					},
				},
				Elements: []SequenceElement{
					{Type: "message", Message: &Message{From: "P1", To: "P1", Text: "Self message", ArrowType: "->>"}},
				},
			},
			want: `sequenceDiagram
    box rgb(128,128,128) Gray Box
        participant P1
    end
    P1->>P1: Self message`,
		},
		{
			name: "box without label",
			args: SequenceDiagramArgs{
				Boxes: []Box{
					{
						Color:        "Aqua",
						Participants: []string{"A", "B"},
					},
				},
				Elements: []SequenceElement{
					{Type: "message", Message: &Message{From: "A", To: "B", Text: "Message", ArrowType: "->>"}},
				},
			},
			want: `sequenceDiagram
    box Aqua
        participant A
        participant B
    end
    A->>B: Message`,
		},
		{
			name: "box without color",
			args: SequenceDiagramArgs{
				Boxes: []Box{
					{
						Label:        "No Color Box",
						Participants: []string{"X"},
					},
				},
				Elements: []SequenceElement{
					{Type: "message", Message: &Message{From: "X", To: "X", Text: "Loop", ArrowType: "->>"}},
				},
			},
			want: `sequenceDiagram
    box No Color Box
        participant X
    end
    X->>X: Loop`,
		},
		{
			name: "mixed actors and participants in boxes",
			args: SequenceDiagramArgs{
				Participants: []Participant{
					{ID: "A", Type: "actor"},
					{ID: "B", Type: "participant", Label: "System B"},
				},
				Boxes: []Box{
					{
						Label:        "Mixed Types",
						Participants: []string{"A", "B"},
					},
				},
				Elements: []SequenceElement{
					{Type: "message", Message: &Message{From: "A", To: "B", Text: "Request", ArrowType: "->>"}},
				},
			},
			want: `sequenceDiagram
    box Mixed Types
        actor A
        participant B as System B
    end
    A->>B: Request`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSequenceDiagramArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSequenceDiagramArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			got := generateSequenceDiagramDSL(tt.args)
			gotLines := strings.Split(strings.TrimSpace(got), "\n")
			wantLines := strings.Split(strings.TrimSpace(tt.want), "\n")

			if len(gotLines) != len(wantLines) {
				t.Errorf("generateSequenceDiagramDSL() line count = %v, want %v\nGot:\n%v\nWant:\n%v",
					len(gotLines), len(wantLines), got, tt.want)
				return
			}

			for i := range gotLines {
				if strings.TrimSpace(gotLines[i]) != strings.TrimSpace(wantLines[i]) {
					t.Errorf("generateSequenceDiagramDSL() line %d:\ngot:  %v\nwant: %v",
						i+1, gotLines[i], wantLines[i])
				}
			}
		})
	}
}
