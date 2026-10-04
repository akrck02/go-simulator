package scenebuilder

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/akrck02/go-simulator/world"
)

const rules = `
units: m
rules:
  - when: {kind: building}
    shape: {role: volume, height: "3 * floors", sink: 0}
    roof: {recipe: gable, pitch: 45, eaves: 0}
`

func TestEntitiesBecomeMeshes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "style.yaml")
	if err := os.WriteFile(path, []byte(rules), 0o644); err != nil {
		t.Fatal(err)
	}
	builder, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !builder.Handles("building", nil) || builder.Handles("tree", nil) {
		t.Fatal("only buildings are styled")
	}
	house := &world.Entity{Identifier: "house", Kind: "building", Properties: map[string]any{"floors": 2.0},
		Shape: &world.Shape{Outline: [][2]float32{{0, 0}, {0.006, 0}, {0.006, 0.01}, {0, 0.01}}}}
	ground := func(x, z float32) (float32, bool) { return 0.1, true } // 100 m everywhere
	meshes := builder.Build([]*world.Entity{house}, nil, ground)
	if len(meshes) != 1 || len(meshes[0].Indices) == 0 {
		t.Fatalf("one mesh expected, got %d", len(meshes))
	}
	highest, southmost := float32(0), float32(-1)
	for index := 0; index+2 < len(meshes[0].Positions); index += 3 {
		highest = max(highest, meshes[0].Positions[index+1])
		southmost = max(southmost, meshes[0].Positions[index+2])
	}
	// 100 m ground + 6 m of walls + 3 m of roof (a 6 m wide house at 45°)
	if math.Abs(float64(highest)-0.109) > 1e-4 {
		t.Errorf("ridge at %v, want 0.109", highest)
	}
	// the outline is in scene coordinates (z to the south): the house reaches z = 0.01
	if math.Abs(float64(southmost)-0.01) > 1e-4 {
		t.Errorf("house reaches z = %v, want 0.01", southmost)
	}
}
