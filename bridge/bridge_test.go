package bridge

import (
	"testing"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-simulator/spatial"
	"github.com/akrck02/go-simulator/world"
)

func islandScene() *scene.Scene {
	drawn := scene.New()
	hull := &scene.Mesh{Name: "hull"}
	boat := &scene.Node{Name: "boat", Mesh: hull, Tags: map[string]any{"bob": map[string]any{"height": 0.5}}}
	boat.Children = []*scene.Node{{Name: "sail"}}
	drawn.Nodes = []*scene.Node{boat, {Name: "boat"}}
	drawn.Environment.Water = &scene.Water{Variation: &scene.LevelVariation{Angle: 0.5}}
	return drawn
}

func TestScenesBecomeWorlds(t *testing.T) {
	described, _ := FromScene(islandScene())
	entities := described.Entities()
	if len(entities) != 3 || entities[2].Identifier != "boat#2" {
		t.Fatalf("expected boat, sail, boat#2; got %d entities", len(entities))
	}
	boat := described.Entity("boat")
	if !boat.HasProperty("bob") || boat.Shape == nil || boat.Shape.Mesh != "hull" {
		t.Fatalf("the boat keeps its tags and mesh: %+v", boat)
	}
	if described.Entity("sail").Parent != "boat" {
		t.Fatalf("children keep their parent")
	}
	if described.Environment.TideAngle == nil || *described.Environment.TideAngle != 0.5 || described.Environment.CloudOffset == nil {
		t.Fatalf("the environment comes along: %+v", described.Environment)
	}
}

func TestChangeSetsChangeTheScene(t *testing.T) {
	drawn := islandScene()
	_, bridge := FromScene(drawn)
	moved, angle := spatial.Translation(spatial.Vector{3, 0, 0}), float32(2)
	red := [4]float32{1, 0, 0, 1}
	err := bridge.Apply(world.ChangeSet{Changes: []world.Change{
		world.ModifyEntity(&world.Entity{Identifier: "boat", Transform: &moved}),
		world.AddEntity(&world.Entity{Identifier: "house-1", Shape: &world.Shape{Primitive: "box", Size: spatial.Vector{2, 3, 2}, Color: &red}}),
		world.AddEntity(&world.Entity{Identifier: "flag", Parent: "boat", Shape: &world.Shape{Mesh: "hull"}}),
		world.RemoveEntity("boat#2"),
		world.ChangeEnvironment(&world.Environment{TideAngle: &angle}),
	}})
	if err != nil {
		t.Fatal(err)
	}
	if drawn.Nodes[0].Transform.Apply(graphics.Vec4{0, 0, 0, 1})[0] != 3 {
		t.Fatalf("the boat should move")
	}
	if len(drawn.Nodes) != 2 || drawn.Nodes[1].Name != "house-1" || drawn.Nodes[1].Mesh.VertexCount() != 24 {
		t.Fatalf("boat#2 should go and house-1 come, nodes: %d", len(drawn.Nodes))
	}
	if len(drawn.Nodes[0].Children) != 2 || drawn.Nodes[0].Children[1].Mesh.Name != "hull" {
		t.Fatalf("the flag hangs from the boat with the hull mesh")
	}
	if drawn.Environment.Water.Variation.Angle != 2 {
		t.Fatalf("the tide angle should change")
	}
	unknown := world.ChangeSet{Changes: []world.Change{world.AddEntity(&world.Entity{Identifier: "x", Shape: &world.Shape{Mesh: "kraken"}})}}
	if err := bridge.Apply(unknown); err == nil {
		t.Fatalf("an unknown mesh should be an error")
	}
}
