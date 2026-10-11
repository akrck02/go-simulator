package bridge

import (
	"testing"

	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-simulator/world"
)

// roofBuilder draws houses only, and remembers what it was given.
type roofBuilder struct {
	entities, context int
}

func (builder *roofBuilder) Handles(kind string, _ map[string]any) bool {
	return kind == "house" || kind == "street"
}

func (builder *roofBuilder) Build(entities, context []*world.Entity, ground func(x, z float32) (float32, bool)) []*scene.Node {
	builder.entities, builder.context = len(entities), len(context)
	return []*scene.Node{{Name: "roofs", Mesh: &scene.Mesh{Name: "roofs", Positions: []float32{0, 0, 0, 1, 0, 0, 0, 0, 1}}}}
}

func groundEntity(identifier, kind, batch string) *world.Entity {
	return &world.Entity{Identifier: world.Identifier(identifier), Kind: kind,
		Shape: &world.Shape{Outline: [][2]float32{{0, 0}, {0.01, 0}, {0.01, 0.01}}, Batch: batch}}
}

func TestBuilderDrawsTheBatchesItHandles(t *testing.T) {
	_, shown := FromScene(scene.New())
	builder := &roofBuilder{}
	shown.UseBuilder(builder)
	set := world.ChangeSet{Changes: []world.Change{
		{Operation: world.Add, Entity: groundEntity("house-1", "house", "houses")},
		{Operation: world.Add, Entity: groundEntity("house-2", "house", "houses")},
		{Operation: world.Add, Entity: groundEntity("street-1", "street", "streets")},
		{Operation: world.Add, Entity: groundEntity("field-1", "field", "fields")},
	}}
	if err := shown.Apply(set); err != nil {
		t.Fatal(err)
	}
	houses := shown.batches["houses"].node
	if houses.Mesh != nil || len(houses.Children) != 1 || houses.Children[0].Mesh.Name != "roofs" {
		t.Fatalf("the builder should draw the houses as children: %+v", houses)
	}
	if builder.entities == 0 || builder.context == 0 {
		t.Errorf("a batch is built with the others as context: %d entities, %d context", builder.entities, builder.context)
	}
	if fields := shown.batches["fields"].node; fields.Mesh == nil || len(fields.Mesh.Indices) == 0 {
		t.Errorf("the bridge keeps drawing what the builder does not handle")
	}
}
