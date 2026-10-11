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
	nodes := builder.Build([]*world.Entity{house}, nil, ground)
	if len(nodes) != 2 {
		t.Fatalf("walls and roof expected, got %d nodes", len(nodes))
	}
	highest, southmost := float32(0), float32(-1)
	for _, node := range nodes {
		built := node.Mesh
		for index := 0; index+2 < len(built.Positions); index += 3 {
			highest = max(highest, built.Positions[index+1])
			southmost = max(southmost, built.Positions[index+2])
		}
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
