package snapshot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/akrck02/go-simulator/spatial"
	"github.com/akrck02/go-simulator/world"
)

func TestEntitiesBecomeFeatures(t *testing.T) {
	current := world.New()
	gate := spatial.Translation(spatial.Vector{5, 0, 6})
	for _, entity := range []*world.Entity{
		{Identifier: "house-1", Kind: "building", Shape: &world.Shape{Outline: [][2]float32{{0, 0}, {1, 0}, {1, 1}}, Height: 0.01}, Properties: map[string]any{"floors": 3.0}},
		{Identifier: "street-1", Kind: "street", Shape: &world.Shape{Line: [][2]float32{{0, 0}, {2, 0}}, Width: 0.006}},
		{Identifier: "gate-1", Kind: "gate", Transform: &gate},
		{Identifier: "idea"},
	} {
		if err := current.ApplyChange(world.AddEntity(entity)); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(t.TempDir(), "city.geojson")
	if err := WriteEntities(path, current, map[string]any{"time": 3.0}); err != nil {
		t.Fatal(err)
	}
	contents, _ := os.ReadFile(path)
	var read struct {
		Properties map[string]any
		Features   []struct {
			Geometry   struct{ Type string }
			Properties map[string]any
		}
	}
	if err := json.Unmarshal(contents, &read); err != nil {
		t.Fatal(err)
	}
	if len(read.Features) != 3 || read.Properties["time"] != 3.0 {
		t.Fatalf("three features with the run's time expected: %s", contents)
	}
	types := []string{read.Features[0].Geometry.Type, read.Features[1].Geometry.Type, read.Features[2].Geometry.Type}
	if types[0] != "Polygon" || types[1] != "LineString" || types[2] != "Point" {
		t.Fatalf("unexpected geometries %v", types)
	}
	if read.Features[0].Properties["floors"] != 3.0 || read.Features[0].Properties["kind"] != "building" {
		t.Fatalf("properties lost: %v", read.Features[0].Properties)
	}
}

func TestFieldsAreWrittenAsGrids(t *testing.T) {
	current := world.New()
	extent := &world.Extent{MinimumX: -1, MaximumX: 1, MinimumZ: -2, MaximumZ: 0}
	if err := current.ApplyChange(world.SetFieldValues(&world.FieldPatch{Name: "biome", Columns: 2, Rows: 1, Extent: extent, Values: []float32{3, -1}})); err != nil {
		t.Fatal(err)
	}
	folder := t.TempDir()
	if err := WriteFields(folder, "city 3 ", current, nil); err != nil {
		t.Fatal(err)
	}
	values, err := os.ReadFile(filepath.Join(folder, "city 3 biome.f32"))
	if err != nil || len(values) != 8 {
		t.Fatalf("two float32 values expected: %v, %v", len(values), err)
	}
	header, _ := os.ReadFile(filepath.Join(folder, "city 3 biome.json"))
	var read map[string]any
	json.Unmarshal(header, &read)
	if read["cell"] != 1.0 || read["north"] != 2.0 || read["west"] != -1.0 {
		t.Errorf("header %v", read)
	}
}
