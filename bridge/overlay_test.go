package bridge

import (
	"testing"

	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-simulator/spatial"
	"github.com/akrck02/go-simulator/world"
)

func harbourBridge(t *testing.T) (*scene.Scene, *Bridge) {
	t.Helper()
	drawn := scene.New()
	_, bridge := FromScene(drawn)
	return drawn, bridge
}

func applyChanges(t *testing.T, bridge *Bridge, changes ...world.Change) {
	t.Helper()
	if err := bridge.Apply(world.ChangeSet{Changes: changes}); err != nil {
		t.Fatal(err)
	}
}

func TestLabelLifecycle(t *testing.T) {
	drawn, bridge := harbourBridge(t)
	boat := &world.Entity{Identifier: "boat-1", Shape: &world.Shape{Primitive: "box", Size: spatial.Vector{2, 3, 4}},
		Properties: map[string]any{"label": "Boat 1", "icon": "sailboat", "label_color": "#2563eb"}}
	applyChanges(t, bridge, world.AddEntity(boat))
	labels := drawn.Overlay.Labels
	if len(labels) != 1 || labels[0].Text != "Boat 1" || labels[0].Icon != "sailboat" || labels[0].Tint[2] < 0.9 {
		t.Fatalf("the boat gets a label: %+v", labels)
	}
	if labels[0].Node != drawn.Nodes[0] || labels[0].Height != 3 {
		t.Fatalf("anchored to the boat's node, at the top of its box: %+v", labels[0])
	}
	label := labels[0]

	// a property-only modify changes the label in place
	applyChanges(t, bridge, world.ModifyEntity(&world.Entity{Identifier: "boat-1", Properties: map[string]any{"badge": 7.0, "icon": "anchor"}}))
	if len(drawn.Overlay.Labels) != 1 || drawn.Overlay.Labels[0] != label || label.Badge != "7" || label.Icon != "anchor" || label.Text != "Boat 1" {
		t.Fatalf("the same label shows the new badge and icon: %+v", label)
	}

	// null takes a property away; with nothing left the label goes
	applyChanges(t, bridge, world.ModifyEntity(&world.Entity{Identifier: "boat-1", Properties: map[string]any{"badge": nil}}))
	if label.Badge != "" {
		t.Fatalf("the badge goes")
	}
	applyChanges(t, bridge, world.ModifyEntity(&world.Entity{Identifier: "boat-1", Properties: map[string]any{"label": nil, "icon": nil}}))
	if len(drawn.Overlay.Labels) != 0 {
		t.Fatalf("an entity without label, icon or badge has no label")
	}

	// it can come back, and goes with the entity
	applyChanges(t, bridge, world.ModifyEntity(&world.Entity{Identifier: "boat-1", Properties: map[string]any{"label": "Boat 1"}}))
	if len(drawn.Overlay.Labels) != 1 || drawn.Overlay.Labels[0].Height != 3 {
		t.Fatalf("the label comes back over the box")
	}
	applyChanges(t, bridge, world.RemoveEntity("boat-1"))
	if len(drawn.Overlay.Labels) != 0 {
		t.Fatalf("removing the entity removes its label")
	}
}

func TestChildrenTakeTheirLabelsAway(t *testing.T) {
	drawn, bridge := harbourBridge(t)
	applyChanges(t, bridge,
		world.AddEntity(&world.Entity{Identifier: "pier"}),
		world.AddEntity(&world.Entity{Identifier: "crane", Parent: "pier", Properties: map[string]any{"icon": "anchor"}}))
	if len(drawn.Overlay.Labels) != 1 {
		t.Fatalf("the crane has a label")
	}
	applyChanges(t, bridge, world.RemoveEntity("pier"))
	if len(drawn.Overlay.Labels) != 0 {
		t.Fatalf("removing the pier removes the crane and its label")
	}
}

func TestOutlinesAnchorOverTheirMiddle(t *testing.T) {
	drawn, bridge := harbourBridge(t)
	square := &world.Entity{Identifier: "market", Shape: &world.Shape{Outline: [][2]float32{{0, 0}, {10, 0}, {10, 20}, {0, 20}}, Height: 5, Batch: "squares"},
		Properties: map[string]any{"label": "Market square"}}
	applyChanges(t, bridge, world.AddEntity(square))
	label := drawn.Overlay.Labels[0]
	if label.Node != nil || label.Point[0] != 5 || label.Point[2] != 10 || label.Height != 5 {
		t.Fatalf("over the middle of the outline, at its height: %+v", label)
	}
}

func TestPinnedEntitiesGoToTheHeaderInOrder(t *testing.T) {
	drawn, bridge := harbourBridge(t)
	applyChanges(t, bridge,
		world.AddEntity(&world.Entity{Identifier: "boats", Properties: map[string]any{"label": "10 boats", "badge": "3 out", "pinned": 2.0}}),
		world.AddEntity(&world.Entity{Identifier: "clock", Properties: map[string]any{"label": "12:00", "icon": "clock", "pinned": 1.0}}))
	header := drawn.Overlay.Header
	if len(header) != 2 || header[0].Text != "12:00" || header[0].Icon != "clock" || header[1].Text != "10 boats · 3 out" {
		t.Fatalf("clock first, then the boats: %+v", header)
	}
	if len(drawn.Overlay.Labels) != 0 {
		t.Fatalf("pinned entities have no label in the world")
	}
	applyChanges(t, bridge, world.ModifyEntity(&world.Entity{Identifier: "clock", Properties: map[string]any{"label": "12:05"}}))
	if drawn.Overlay.Header[0].Text != "12:05" {
		t.Fatalf("the header follows the property: %+v", drawn.Overlay.Header)
	}
	applyChanges(t, bridge, world.ModifyEntity(&world.Entity{Identifier: "boats", Properties: map[string]any{"pinned": nil}}))
	if len(drawn.Overlay.Header) != 1 || len(drawn.Overlay.Labels) != 1 {
		t.Fatalf("unpinned, the boats go back to a label")
	}
	applyChanges(t, bridge, world.RemoveEntity("clock"))
	if len(drawn.Overlay.Header) != 0 {
		t.Fatalf("removing the clock empties the header")
	}
}

func TestNoticesComeFromEvents(t *testing.T) {
	drawn, bridge := harbourBridge(t)
	drawn.Overlay.Clock = 12
	applyChanges(t, bridge,
		world.EmitEvent("arrivals", map[string]any{"notice": "Boat 2 moored", "icon": "anchor"}),
		world.EmitEvent("arrivals", map[string]any{"count": 3.0}))
	notices := drawn.Overlay.Notices
	if len(notices) != 1 || notices[0].Text != "Boat 2 moored" || notices[0].Icon != "anchor" || notices[0].Added != 12 {
		t.Fatalf("one notice, timed by the viewer's clock: %+v", notices)
	}
}

func TestSceneLabelsBecomeTheBridges(t *testing.T) {
	drawn := scene.New()
	lighthouse := &scene.Node{Name: "lighthouse", Tags: map[string]any{"label": "Lighthouse", "icon": "lighthouse"}}
	drawn.Nodes = []*scene.Node{lighthouse}
	drawn.Overlay.AddLabel(&scene.Label{Node: lighthouse, Text: "Lighthouse"}) // as the loader leaves it
	_, bridge := FromScene(drawn)
	if len(drawn.Overlay.Labels) != 1 || drawn.Overlay.Labels[0].Icon != "lighthouse" {
		t.Fatalf("one label, from the node's properties: %+v", drawn.Overlay.Labels)
	}
	applyChanges(t, bridge, world.ModifyEntity(&world.Entity{Identifier: "lighthouse", Properties: map[string]any{"label": "Lighthouse (lit)"}}))
	if drawn.Overlay.Labels[0].Text != "Lighthouse (lit)" {
		t.Fatalf("and the bridge keeps it up to date")
	}
}
