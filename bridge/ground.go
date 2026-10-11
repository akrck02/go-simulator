package bridge

import (
	"math"

	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-simulator/world"
)

const (
	defaultLift     = 0.0005 // world units a flat shape floats over the ground
	sinkIntoGround  = 0.001  // how far volumes go under the ground, so slopes leave no gap
	lineSampleEvery = 0.01   // lines are cut at least this often to follow the terrain
	flatDepthBias   = 0.001  // pulls flat shapes towards the camera, like roads in exported scenes
)

// groundHeight returns the height of the ground at a point (0 where there is no ground).
type groundHeight func(x, z float32) float32

// meshBuilder collects vertices and triangles for a mesh.
type meshBuilder struct {
	positions, normals, colors []float32
	indices                    []uint32
}

func (builder *meshBuilder) vertex(x, y, z float32, normal [3]float32, color [4]float32) uint32 {
	index := uint32(len(builder.positions) / 3)
	builder.positions = append(builder.positions, x, y, z)
	builder.normals = append(builder.normals, normal[0], normal[1], normal[2])
	builder.colors = append(builder.colors, color[0], color[1], color[2], color[3])
	return index
}

func (builder *meshBuilder) quad(first, second, third, fourth uint32) {
	builder.indices = append(builder.indices, first, second, third, first, third, fourth)
}

// fill writes the collected geometry into a mesh and marks it for upload.
func (builder *meshBuilder) fill(mesh *scene.Mesh) {
	mesh.Positions, mesh.Normals, mesh.Colors, mesh.Indices = builder.positions, builder.normals, builder.colors, builder.indices
	mesh.Dirty = true
}

// addShape adds an outline or a line to the mesh.
func (builder *meshBuilder) addShape(shape *world.Shape, ground groundHeight) {
	if shape.Hidden {
		return
	}
	color := defaultColor
	if shape.Color != nil {
		color = *shape.Color
	}
	switch {
	case len(shape.Outline) >= 3 && shape.Height > 0:
		builder.addVolume(shape.Outline, shape.Height, color, ground)
	case len(shape.Outline) >= 3:
		builder.addFlatOutline(shape.Outline, liftOf(shape), color, ground)
	case len(shape.Line) >= 2 && shape.Width > 0:
		builder.addLine(shape, color, ground)
	}
}

func liftOf(shape *world.Shape) float32 {
	if shape.Lift > 0 {
		return shape.Lift
	}
	return defaultLift
}

var up = [3]float32{0, 1, 0}

// addFlatOutline lays a polygon on the ground, every corner at its own ground height.
func (builder *meshBuilder) addFlatOutline(outline [][2]float32, lift float32, color [4]float32, ground groundHeight) {
	first := uint32(len(builder.positions) / 3)
	for _, point := range outline {
		builder.vertex(point[0], ground(point[0], point[1])+lift, point[1], up, color)
	}
	for _, index := range triangulate(outline) {
		builder.indices = append(builder.indices, first+index)
	}
}

// addVolume raises a polygon into a prism: a flat roof at the height over the highest corner and
// walls down into the ground under the lowest one.
func (builder *meshBuilder) addVolume(outline [][2]float32, height float32, color [4]float32, ground groundHeight) {
	lowest, highest := groundRange(outline, ground)
	bottom, top := lowest-sinkIntoGround, highest+height
	first := uint32(len(builder.positions) / 3)
	for _, point := range outline {
		builder.vertex(point[0], top, point[1], up, color)
	}
	for _, index := range triangulate(outline) {
		builder.indices = append(builder.indices, first+index)
	}
	outward := float32(1)
	if signedArea(outline) < 0 {
		outward = -1
	}
	for index, point := range outline {
		builder.addWall(point, outline[(index+1)%len(outline)], bottom, top, outward, color)
	}
}

// addWall adds one vertical face from a to b, facing outwards.
func (builder *meshBuilder) addWall(from, to [2]float32, bottom, top, outward float32, color [4]float32) {
	normal := wallNormal(from, to, outward)
	lowFrom := builder.vertex(from[0], bottom, from[1], normal, color)
	lowTo := builder.vertex(to[0], bottom, to[1], normal, color)
	highTo := builder.vertex(to[0], top, to[1], normal, color)
	highFrom := builder.vertex(from[0], top, from[1], normal, color)
	if outward > 0 {
		builder.quad(lowFrom, highFrom, highTo, lowTo)
	} else {
		builder.quad(lowFrom, lowTo, highTo, highFrom)
	}
}

// wallNormal is the horizontal normal of an edge pointing out of a polygon whose winding has the
// given sign.
func wallNormal(from, to [2]float32, outward float32) [3]float32 {
	deltaX, deltaZ := to[0]-from[0], to[1]-from[1]
	length := float32(math.Hypot(float64(deltaX), float64(deltaZ)))
	if length == 0 {
		return up
	}
	return [3]float32{deltaZ / length * outward, 0, -deltaX / length * outward}
}

func groundRange(points [][2]float32, ground groundHeight) (lowest, highest float32) {
	lowest, highest = float32(math.Inf(1)), float32(math.Inf(-1))
	for _, point := range points {
		height := ground(point[0], point[1])
		lowest, highest = min(lowest, height), max(highest, height)
	}
	return lowest, highest
}

// addLine lays a strip along a polyline, cut often enough to follow the terrain; with a height it
// becomes a wall with a top and two sides.
func (builder *meshBuilder) addLine(shape *world.Shape, color [4]float32, ground groundHeight) {
	samples := sampleLine(shape.Line, lineSampleEvery)
	half := shape.Width / 2
	var previousLeft, previousRight [2]float32
	for index, sample := range samples {
		side := sideAt(samples, index)
		left := [2]float32{sample[0] + side[0]*half, sample[1] + side[1]*half}
		right := [2]float32{sample[0] - side[0]*half, sample[1] - side[1]*half}
		if index > 0 {
			builder.addLinePiece(previousLeft, previousRight, left, right, shape, color, ground)
		}
		previousLeft, previousRight = left, right
	}
}

// addLinePiece adds the piece of a strip between two cuts.
func (builder *meshBuilder) addLinePiece(fromLeft, fromRight, toLeft, toRight [2]float32, shape *world.Shape, color [4]float32, ground groundHeight) {
	if shape.Height <= 0 {
		lift := liftOf(shape)
		corners := [4][2]float32{fromLeft, toLeft, toRight, fromRight}
		var indices [4]uint32
		for corner, point := range corners {
			indices[corner] = builder.vertex(point[0], ground(point[0], point[1])+lift, point[1], up, color)
		}
		builder.quad(indices[0], indices[1], indices[2], indices[3])
		return
	}
	builder.addWallPiece(fromLeft, fromRight, toLeft, toRight, shape.Height, color, ground)
}

func (builder *meshBuilder) addWallPiece(fromLeft, fromRight, toLeft, toRight [2]float32, height float32, color [4]float32, ground groundHeight) {
	top := func(point [2]float32) float32 { return ground(point[0], point[1]) + height }
	bottom := func(point [2]float32) float32 { return ground(point[0], point[1]) - sinkIntoGround }
	corners := [4][2]float32{fromLeft, toLeft, toRight, fromRight}
	var roof [4]uint32
	for corner, point := range corners {
		roof[corner] = builder.vertex(point[0], top(point), point[1], up, color)
	}
	builder.quad(roof[0], roof[1], roof[2], roof[3])
	for _, side := range [2][2][2]float32{{fromLeft, toLeft}, {toRight, fromRight}} {
		normal := wallNormal(side[0], side[1], -1)
		lowFrom := builder.vertex(side[0][0], bottom(side[0]), side[0][1], normal, color)
		lowTo := builder.vertex(side[1][0], bottom(side[1]), side[1][1], normal, color)
		highTo := builder.vertex(side[1][0], top(side[1]), side[1][1], normal, color)
		highFrom := builder.vertex(side[0][0], top(side[0]), side[0][1], normal, color)
		builder.quad(lowFrom, lowTo, highTo, highFrom)
	}
}

// sampleLine returns the polyline's points plus extra ones so no piece is longer than step.
func sampleLine(line [][2]float32, step float32) [][2]float32 {
	samples := [][2]float32{line[0]}
	for index := 1; index < len(line); index++ {
		from, to := line[index-1], line[index]
		length := float32(math.Hypot(float64(to[0]-from[0]), float64(to[1]-from[1])))
		pieces := max(1, int(math.Ceil(float64(length/step))))
		for piece := 1; piece <= pieces; piece++ {
			amount := float32(piece) / float32(pieces)
			samples = append(samples, [2]float32{from[0] + (to[0]-from[0])*amount, from[1] + (to[1]-from[1])*amount})
		}
	}
	return samples
}

// sideAt is the unit vector to the left of the line at a sample, averaged at corners.
func sideAt(samples [][2]float32, index int) [2]float32 {
	before, after := samples[max(0, index-1)], samples[min(len(samples)-1, index+1)]
	deltaX, deltaZ := after[0]-before[0], after[1]-before[1]
	length := float32(math.Hypot(float64(deltaX), float64(deltaZ)))
	if length == 0 {
		return [2]float32{0, 0}
	}
	return [2]float32{-deltaZ / length, deltaX / length}
}

// groundMaterial is the material of ground shapes: flat ones are pulled towards the camera so the
// terrain does not hide them from far away.
func groundMaterial(shape *world.Shape) *scene.Material {
	material := &scene.Material{Name: "ground shape", BaseColor: [4]float32{1, 1, 1, 1}, DoubleSided: shape.Height <= 0}
	if shape.Height <= 0 {
		material.DepthBias = flatDepthBias
	}
	return material
}
