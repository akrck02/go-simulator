package host

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/akrck02/go-simulator/configuration"
	"github.com/akrck02/go-simulator/world"
)

const anchoredTownRules = `
plugins:
  - {name: town, builtin: true}
anchors:
  - {id: spring, role: obstacle, outline: [[0, 0], [1, 0], [1, 1]], source: "the story"}
  - {id: gate, role: gate, at: [2, 3]}
stages:
  - name: growth
    clock: {unit: year, from: 0, to: 10, step: 1}
    rules:
      - rule: build
    objectives:
      - {name: some houses, at: 3, count: {kind: house}, min: 3}
      - {name: too many houses, at: 3, count: {kind: house}, max: 2}
      - {name: a spring, at: 3, count: {kind: anchor, role: obstacle}}
`

func TestAnchorsAreInTheWorldBeforeTheStart(t *testing.T) {
	seen := 0
	engine := buildEngine(t, anchoredTownRules, &seen)
	spring := engine.World.Entity("spring")
	if spring == nil || spring.Properties[world.PropertyAnchor] != true || spring.Properties[world.PropertyRole] != "obstacle" {
		t.Fatalf("spring = %+v, want a hidden obstacle anchor", spring)
	}
	if spring.Shape == nil || !spring.Shape.Hidden || len(spring.Shape.Outline) != 3 {
		t.Errorf("the spring's shape = %+v, want its hidden outline", spring.Shape)
	}
	gate := engine.World.Entity("gate")
	if gate == nil || gate.Transform == nil || gate.Transform.Position() != [3]float32{2, 0, 3} {
		t.Errorf("gate = %+v, want a point at (2, 0, 3)", gate)
	}
}

func TestObjectivesCountEntitiesWhenTheirTimeComes(t *testing.T) {
	seen := 0
	engine := buildEngine(t, anchoredTownRules, &seen)
	stage := engine.Stage("growth")
	for range 3 {
		if _, err := engine.StepStage("growth"); err != nil {
			t.Fatal(err)
		}
	}
	due := stage.DueObjectives(2)
	if len(due) != 3 {
		t.Fatalf("%d objectives due at 3, want 3", len(due))
	}
	wanted := map[string]bool{"some houses": true, "too many houses": false, "a spring": true}
	for _, objective := range due {
		result := engine.CheckObjective(stage, objective)
		if result.Reached != wanted[objective.Name] {
			t.Errorf("%s", result.Describe())
		}
	}
}

func TestObjectivesReadTheLastEvent(t *testing.T) {
	near, engine := 6000.0, &Engine{World: world.New()}
	engine.rememberEvent(world.EmitEvent("indicators", map[string]any{"population": 5640.0}))
	engine.rememberEvent(world.EmitEvent("indicators", map[string]any{"population": 5880.0}))
	objective := configuration.Objective{Name: "people", Event: "indicators", Value: "population", Near: &near, Tolerance: 0.05}
	if result := engine.CheckObjective(&Stage{Name: "city"}, objective); !result.Reached || result.Value != 5880 {
		t.Errorf("%s, want the last event's 5880 within 5 %%", result.Describe())
	}
	objective.Tolerance = 0.01
	if result := engine.CheckObjective(&Stage{Name: "city"}, objective); result.Reached {
		t.Errorf("%s, want it missed within 1 %%", result.Describe())
	}
}

func TestAnchorsFromAGeoJSONFile(t *testing.T) {
	folder := t.TempDir()
	geojson := `{"type": "FeatureCollection", "features": [
	  {"type": "Feature", "geometry": {"type": "Polygon", "coordinates": [[[0, 0], [1, 0], [1, 1], [0, 0]]]}, "properties": {"id": "square", "role": "obstacle"}},
	  {"type": "Feature", "geometry": {"type": "LineString", "coordinates": [[0, 0], [5, 5]]}, "properties": {"width": 0.01}}]}`
	if err := os.WriteFile(filepath.Join(folder, "anchors.geojson"), []byte(geojson), 0o644); err != nil {
		t.Fatal(err)
	}
	rules := &configuration.Configuration{Directory: folder, Anchors: []configuration.Anchor{{From: "anchors.geojson", Role: "road"}}}
	initial := world.New()
	if err := AddAnchors(rules, initial); err != nil {
		t.Fatal(err)
	}
	square, road := initial.Entity("square"), initial.Entity("anchors-2")
	if square == nil || len(square.Shape.Outline) != 3 || square.Properties[world.PropertyRole] != "obstacle" {
		t.Errorf("square = %+v, want an obstacle outline without its closing point", square)
	}
	if road == nil || road.Properties[world.PropertyRole] != "road" || road.Shape.Width != 0.01 {
		t.Errorf("road = %+v, want the file's role and the feature's width", road)
	}
}
