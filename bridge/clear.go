package bridge

import (
	"slices"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-simulator/world"
)

// clearArea removes, from the named scene nodes, the triangles whose middle falls inside the area
// and the instances standing in it.
func (bridge *Bridge) clearArea(area *world.Area) {
	if area == nil || len(area.Outlines) == 0 {
		return
	}
	polygons := newGroundPolygons(area.Outlines)
	polygons.spared = newGroundPolygons(area.Keep).rings
	touchedGround := false
	bridge.scene.Walk(func(node *scene.Node, transform graphics.Mat4) {
		if !slices.Contains(area.Nodes, node.Name) || node.Mesh == nil {
			return
		}
		if node.Instances != nil {
			clearInstances(node.Instances, transform, polygons)
			return
		}
		for level := range node.Mesh.LevelCount() {
			clearTriangles(node.Mesh.Level(level), transform, polygons)
		}
		touchedGround = touchedGround || node.Mesh.Ground
	})
	if touchedGround {
		bridge.scene.InvalidateGround()
	}
}

// clearTriangles keeps the triangles whose middle, in world coordinates, is outside the polygons.
func clearTriangles(mesh *scene.Mesh, transform graphics.Mat4, polygons groundPolygons) {
	kept := make([]uint32, 0, len(mesh.Indices))
	removed := false
	mesh.ForEachTriangle(func(first, second, third uint32) {
		middle := transform.Apply(middleOf(mesh, first, second, third))
		if polygons.contain(middle[0], middle[2]) {
			removed = true
			return
		}
		kept = append(kept, first, second, third)
	})
	if removed {
		mesh.Indices, mesh.Dirty = kept, true
	}
}

func middleOf(mesh *scene.Mesh, first, second, third uint32) graphics.Vec4 {
	a, b, c := mesh.Vertex(first), mesh.Vertex(second), mesh.Vertex(third)
	return graphics.Vec4{(a[0] + b[0] + c[0]) / 3, (a[1] + b[1] + c[1]) / 3, (a[2] + b[2] + c[2]) / 3, 1}
}

// clearInstances drops the instances whose origin, in world coordinates, is inside the polygons.
func clearInstances(instances *scene.Instances, transform graphics.Mat4, polygons groundPolygons) {
	transforms := instances.Transforms[:0]
	colors := instances.Colors[:0]
	removed := false
	for index, instance := range instances.Transforms {
		origin := transform.Multiply(instance).Apply(graphics.Vec4{0, 0, 0, 1})
		if polygons.contain(origin[0], origin[2]) {
			removed = true
			continue
		}
		transforms = append(transforms, instance)
		if index < len(instances.Colors) {
			colors = append(colors, instances.Colors[index])
		}
	}
	if removed {
		instances.Transforms, instances.Colors, instances.Dirty = transforms, colors, true
	}
}

// groundPolygons tests points against polygons on the ground plane, with a box around each.
type groundPolygons struct {
	rings           [][][2]float32
	lowest, highest [][2]float32
	spared          [][][2]float32 // points inside these are never contained
}

func newGroundPolygons(outlines [][][2]float32) groundPolygons {
	polygons := groundPolygons{}
	for _, ring := range outlines {
		if len(ring) < 3 {
			continue
		}
		low, high := ring[0], ring[0]
		for _, point := range ring {
			low = [2]float32{min(low[0], point[0]), min(low[1], point[1])}
			high = [2]float32{max(high[0], point[0]), max(high[1], point[1])}
		}
		polygons.rings, polygons.lowest, polygons.highest = append(polygons.rings, ring), append(polygons.lowest, low), append(polygons.highest, high)
	}
	return polygons
}

func (polygons groundPolygons) contain(x, z float32) bool {
	for _, ring := range polygons.spared {
		if insideRing(ring, x, z) {
			return false
		}
	}
	for index, ring := range polygons.rings {
		low, high := polygons.lowest[index], polygons.highest[index]
		if x < low[0] || x > high[0] || z < low[1] || z > high[1] {
			continue
		}
		if insideRing(ring, x, z) {
			return true
		}
	}
	return false
}

// insideRing is the even-odd rule: a ray towards +x crosses the ring an odd number of times.
func insideRing(ring [][2]float32, x, z float32) bool {
	inside := false
	for index, point := range ring {
		previous := ring[(index+len(ring)-1)%len(ring)]
		if (point[1] > z) != (previous[1] > z) {
			crossingX := point[0] + (z-point[1])*(previous[0]-point[0])/(previous[1]-point[1])
			if x < crossingX {
				inside = !inside
			}
		}
	}
	return inside
}
