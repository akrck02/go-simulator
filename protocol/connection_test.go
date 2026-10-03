package protocol

import (
	"io"
	"testing"

	"github.com/akrck02/go-simulator/world"
)

func TestFramesCarryBodiesAndFieldValues(t *testing.T) {
	reader, writer := io.Pipe()
	sender, receiver := NewConnection(nil, writer), NewConnection(reader, nil)
	sent := world.ChangeSet{Origin: "landscape", Time: 2, Changes: []world.Change{
		world.SetFieldValues(&world.FieldPatch{Name: "elevation", Columns: 2, Rows: 1, Extent: &world.Extent{MaximumX: 2, MaximumZ: 1}, Values: []float32{1.5, -2}}),
		world.EmitEvent("eroded", map[string]any{"steps": 3.0}),
		world.SetFieldValues(&world.FieldPatch{Name: "water", Region: &world.Region{Columns: 1, Rows: 1}, Values: []float32{7}}),
	}}
	go func() { _ = sender.Send(Changes, &sent) }()
	message, err := receiver.Receive()
	if err != nil {
		t.Fatal(err)
	}
	var received world.ChangeSet
	if err := message.Expect(Changes); err != nil {
		t.Fatal(err)
	}
	if err := message.Decode(&received); err != nil {
		t.Fatal(err)
	}
	if received.Origin != "landscape" || len(received.Changes) != 3 || received.Changes[1].Event.Name != "eroded" {
		t.Fatalf("body not received: %+v", received)
	}
	if values := received.Changes[0].Field.Values; len(values) != 2 || values[0] != 1.5 || values[1] != -2 {
		t.Fatalf("first field values wrong: %v", values)
	}
	if values := received.Changes[2].Field.Values; len(values) != 1 || values[0] != 7 {
		t.Fatalf("second field values wrong: %v", values)
	}
}

func TestFailureRepliesBecomeErrors(t *testing.T) {
	reader, writer := io.Pipe()
	sender, receiver := NewConnection(nil, writer), NewConnection(reader, nil)
	go func() { _ = sender.Send(Failure, FailureReply{Message: "unknown rule \"fly\""}) }()
	message, err := receiver.Receive()
	if err != nil {
		t.Fatal(err)
	}
	if err := message.Expect(Attached); err == nil || err.Error() != "unknown rule \"fly\"" {
		t.Fatalf("expected the plugin's error, got %v", err)
	}
}
