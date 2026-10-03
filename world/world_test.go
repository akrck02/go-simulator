package world

import (
	"testing"

	"github.com/akrck02/go-simulator/spatial"
)

func boat(identifier Identifier) *Entity {
	transform := spatial.Identity()
	return &Entity{Identifier: identifier, Kind: "boat", Transform: &transform, Properties: map[string]any{"bob": true}}
}

func TestApplyAddsModifiesAndRemoves(t *testing.T) {
	world := New()
	moved := spatial.Translation(spatial.Vector{1, 2, 3})
	err := world.Apply(ChangeSet{Time: 4, Changes: []Change{
		AddEntity(boat("boat-1")),
		ModifyEntity(&Entity{Identifier: "boat-1", Transform: &moved, Properties: map[string]any{"bob": nil, "name": "Anya"}}),
	}})
	if err != nil {
		t.Fatal(err)
	}
	entity := world.Entity("boat-1")
	if entity.Transform.Position() != (spatial.Vector{1, 2, 3}) || entity.HasProperty("bob") || entity.Properties["name"] != "Anya" {
		t.Fatalf("modification not applied: %+v", entity)
	}
	if world.Time != 4 {
		t.Fatalf("the clock should move to the set's time, got %v", world.Time)
	}
	if err := world.ApplyChange(RemoveEntity("boat-1")); err != nil || world.Entity("boat-1") != nil {
		t.Fatalf("removal failed: %v", err)
	}
}

func TestRemovingAnEntityRemovesWhatIsAttachedToIt(t *testing.T) {
	world := New()
	sail := &Entity{Identifier: "sail", Parent: "boat-1"}
	flag := &Entity{Identifier: "flag", Parent: "sail"}
	for _, change := range []Change{AddEntity(boat("boat-1")), AddEntity(sail), AddEntity(flag), AddEntity(boat("boat-2"))} {
		if err := world.ApplyChange(change); err != nil {
			t.Fatal(err)
		}
	}
	if err := world.ApplyChange(RemoveEntity("boat-1")); err != nil {
		t.Fatal(err)
	}
	if remaining := world.Entities(); len(remaining) != 1 || remaining[0].Identifier != "boat-2" {
		t.Fatalf("only boat-2 should remain, got %d entities", len(remaining))
	}
}

func TestChangesThatCannotApplyAreErrors(t *testing.T) {
	world := New()
	for _, change := range []Change{
		RemoveEntity("ghost"),
		ModifyEntity(&Entity{Identifier: "ghost"}),
		SetFieldValues(&FieldPatch{Name: "elevation", Values: []float32{1}}),
		{Operation: "teleport"},
	} {
		if err := world.ApplyChange(change); err == nil {
			t.Fatalf("%s should fail", change.Operation)
		}
	}
	_ = world.ApplyChange(AddEntity(boat("boat-1")))
	if err := world.ApplyChange(AddEntity(boat("boat-1"))); err == nil {
		t.Fatalf("adding the same identifier twice should fail")
	}
}

func TestFieldRegionsChangeOnlyTheirCells(t *testing.T) {
	world := New()
	create := &FieldPatch{Name: "elevation", Columns: 3, Rows: 2, Extent: &Extent{MaximumX: 3, MaximumZ: 2}, Values: []float32{1, 2, 3, 4, 5, 6}}
	region := &FieldPatch{Name: "elevation", Region: &Region{Column: 1, Row: 1, Columns: 2, Rows: 1}, Values: []float32{50, 60}}
	if err := world.Apply(ChangeSet{Changes: []Change{SetFieldValues(create), SetFieldValues(region)}}); err != nil {
		t.Fatal(err)
	}
	field := world.Field("elevation")
	if field.At(0, 1) != 4 || field.At(1, 1) != 50 || field.At(2, 1) != 60 || field.At(2, 0) != 3 {
		t.Fatalf("unexpected values %v", field.Values)
	}
	outside := &FieldPatch{Name: "elevation", Region: &Region{Column: 2, Row: 1, Columns: 2, Rows: 1}, Values: []float32{0, 0}}
	if err := world.ApplyChange(SetFieldValues(outside)); err == nil {
		t.Fatalf("a region outside the field should fail")
	}
}

func TestEnvironmentPatchesOnlyChangeWhatTheySet(t *testing.T) {
	world := New()
	sun, angle := spatial.Vector{0, 1, 0}, float32(1)
	_ = world.ApplyChange(ChangeEnvironment(&Environment{SunDirection: &sun, TideAngle: &angle}))
	wind := Wind{Direction: spatial.Vector{1, 0, 0}, Strength: 0.5}
	_ = world.ApplyChange(ChangeEnvironment(&Environment{Wind: &wind}))
	if *world.Environment.SunDirection != sun || *world.Environment.TideAngle != 1 || world.Environment.Wind.Strength != 0.5 {
		t.Fatalf("patch replaced more than it set: %+v", world.Environment)
	}
}

func TestInterestCoversOnlyWhatARuleReads(t *testing.T) {
	world := New()
	_ = world.ApplyChange(AddEntity(boat("boat-1")))
	_ = world.ApplyChange(AddEntity(&Entity{Identifier: "house-1", Kind: "house"}))
	boats := Interest{Properties: []string{"bob"}}
	if !boats.Covers(RemoveEntity("boat-1"), world) || boats.Covers(RemoveEntity("house-1"), world) {
		t.Fatalf("removal coverage is wrong")
	}
	becomesBoat := ModifyEntity(&Entity{Identifier: "house-1", Properties: map[string]any{"bob": 1.0}})
	if !boats.Covers(becomesBoat, world) {
		t.Fatalf("an entity that gains a read property should be covered")
	}
	if boats.Covers(ChangeEnvironment(&Environment{}), world) || !(Interest{Environment: true}).Covers(ChangeEnvironment(&Environment{}), world) {
		t.Fatalf("environment coverage is wrong")
	}
	snapshot := world.Snapshot(boats)
	if len(snapshot.Changes) != 1 || snapshot.Changes[0].Entity.Identifier != "boat-1" {
		t.Fatalf("the snapshot should only hold boat-1, got %+v", snapshot.Changes)
	}
}
