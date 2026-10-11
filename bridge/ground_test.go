package bridge

import (
	"testing"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-simulator/world"
)

// slopedScene has a walkable ground that rises 0.1 per unit towards +x, from x = 0 to 10.
func slopedScene() *scene.Scene {
	drawn := scene.New()
	ground := &scene.Mesh{Name: "terrain", Ground: true,
		Positions: []float32{0, 0, 0, 10, 1, 0, 10, 1, 10, 0, 0, 10}, Indices: []uint32{0, 2, 1, 0, 3, 2}}
	drawn.Nodes = []*scene.Node{{Name: "terrain", Mesh: ground}}
	return drawn
}

func faceUp(mesh *scene.Mesh, first, second, third uint32) bool {
	a, b, c := mesh.Vertex(first), mesh.Vertex(second), mesh.Vertex(third)
	ux, uz, vx, vz := b[0]-a[0], b[2]-a[2], c[0]-a[0], c[2]-a[2]
	return uz*vx-ux*vz > 0 // y of (b − a) × (c − a)
}

func TestTrianglesFaceUpWhateverTheWinding(t *testing.T) {
	square := [][2]float32{{0, 0}, {1, 0}, {1, 1}, {0, 1}}
	notch := [][2]float32{{0, 0}, {4, 0}, {4, 4}, {2, 1}, {0, 4}}
	for _, outline := range [][][2]float32{square, {{0, 1}, {1, 1}, {1, 0}, {0, 0}}, notch} {
		builder := &meshBuilder{}
		builder.addFlatOutline(outline, 0, defaultColor, func(float32, float32) float32 { return 0 })
		mesh := &scene.Mesh{}
		builder.fill(mesh)
		if len(mesh.Indices) != 3*(len(outline)-2) {
			t.Fatalf("%d corners should give %d triangles, got %d", len(outline), len(outline)-2, len(mesh.Indices)/3)
		}
		mesh.ForEachTriangle(func(first, second, third uint32) {
			if !faceUp(mesh, first, second, third) {
				t.Fatalf("a triangle of %v faces down", outline)
			}
		})
	}
}

func TestVolumesStandOnTheGroundUnderThem(t *testing.T) {
	drawn := slopedScene()
	_, bridge := FromScene(drawn)
	house := &world.Shape{Outline: [][2]float32{{2, 2}, {4, 2}, {4, 4}, {2, 4}}, Height: 0.5}
	if err := bridge.Apply(world.ChangeSet{Changes: []world.Change{world.AddEntity(&world.Entity{Identifier: "house", Shape: house})}}); err != nil {
		t.Fatal(err)
	}
	mesh := drawn.Nodes[1].Mesh
	lowest, highest := float32(99), float32(-99)
	for index := range mesh.VertexCount() {
		y := mesh.Vertex(uint32(index))[1]
		lowest, highest = min(lowest, y), max(highest, y)
	}
	if lowest > 0.2-sinkIntoGround+1e-4 || highest < 0.4+0.5-1e-4 {
		t.Fatalf("the house should go from under the ground at x=2 (0.2) to 0.5 over x=4 (0.4): %v..%v", lowest, highest)
	}
}

func TestBatchesMergeShapesAndForgetRemovedOnes(t *testing.T) {
	drawn := slopedScene()
	_, bridge := FromScene(drawn)
	parcel := func(x float32) *world.Shape {
		return &world.Shape{Outline: [][2]float32{{x, 1}, {x + 1, 1}, {x + 1, 2}, {x, 2}}, Batch: "parcels"}
	}
	street := &world.Shape{Line: [][2]float32{{0, 5}, {9, 5}}, Width: 0.2, Batch: "streets"}
	err := bridge.Apply(world.ChangeSet{Changes: []world.Change{
		world.AddEntity(&world.Entity{Identifier: "parcel-1", Shape: parcel(1)}),
		world.AddEntity(&world.Entity{Identifier: "parcel-2", Shape: parcel(3)}),
		world.AddEntity(&world.Entity{Identifier: "street-1", Shape: street}),
	}})
	if err != nil {
		t.Fatal(err)
	}
	parcels := bridge.batches["parcels"].node.Mesh
	if parcels.VertexCount() != 8 || !parcels.Dirty {
		t.Fatalf("two parcels in one mesh: %d vertices", parcels.VertexCount())
	}
	if streets := bridge.batches["streets"].node.Mesh; streets.VertexCount() < 4*90 {
		t.Fatalf("a 9-unit street is cut every %v: %d vertices", lineSampleEvery, streets.VertexCount())
	}
	parcels.Dirty = false
	if err := bridge.Apply(world.ChangeSet{Changes: []world.Change{world.RemoveEntity("parcel-1")}}); err != nil {
		t.Fatal(err)
	}
	if parcels.VertexCount() != 4 || !parcels.Dirty {
		t.Fatalf("removing a parcel rebuilds the batch: %d vertices", parcels.VertexCount())
	}
}

func TestClearingAnAreaRemovesTrianglesAndInstances(t *testing.T) {
	drawn := slopedScene()
	town := &scene.Mesh{Name: "construcciones", Positions: []float32{
		1, 0, 1, 2, 0, 1, 2, 0, 2, // inside the area
		7, 0, 7, 8, 0, 7, 8, 0, 8, // outside
	}}
	trees := &scene.Instances{Transforms: []graphics.Mat4{graphics.Translate(graphics.Vec4{2.5, 0, 2.5, 0}), graphics.Translate(graphics.Vec4{9, 0, 9, 0}),
		graphics.Translate(graphics.Vec4{0.5, 0, 0.5, 0})}, Colors: []graphics.Vec4{{1, 0, 0, 1}, {0, 1, 0, 1}, {0, 0, 1, 1}}}
	drawn.Nodes = append(drawn.Nodes, &scene.Node{Name: "construcciones", Mesh: town},
		&scene.Node{Name: "bosque", Mesh: &scene.Mesh{Name: "arbol"}, Instances: trees})
	_, bridge := FromScene(drawn)
	area := &world.Area{Outlines: [][][2]float32{{{0, 0}, {3, 0}, {3, 3}, {0, 3}}}, Nodes: []string{"construcciones", "bosque"},
		Keep: [][][2]float32{{{0.2, 0.2}, {0.8, 0.2}, {0.8, 0.8}, {0.2, 0.8}}}}
	if err := bridge.Apply(world.ChangeSet{Changes: []world.Change{world.ClearSceneArea(area)}}); err != nil {
		t.Fatal(err)
	}
	if len(town.Indices) != 3 || town.Indices[0] != 3 || !town.Dirty {
		t.Fatalf("only the outside triangle should stay: %v", town.Indices)
	}
	if len(trees.Transforms) != 2 || trees.Colors[0] != (graphics.Vec4{0, 1, 0, 1}) || trees.Colors[1] != (graphics.Vec4{0, 0, 1, 1}) || !trees.Dirty {
		t.Fatalf("the outside tree and the spared one should stay, with their colours")
	}
	if drawn.Nodes[0].Mesh.VertexCount() != 4 {
		t.Fatalf("nodes not named in the area are left alone")
	}
}
