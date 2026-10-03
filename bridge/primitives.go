package bridge

import (
	"fmt"

	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-simulator/spatial"
	"github.com/akrck02/go-simulator/world"
)

// primitiveKey identifies a primitive mesh; entities with the same primitive share one mesh.
type primitiveKey struct {
	primitive string
	size      spatial.Vector
	offset    spatial.Vector
	color     [4]float32
}

var defaultColor = [4]float32{0.8, 0.8, 0.8, 1}

func (bridge *Bridge) primitive(shape *world.Shape) (*scene.Mesh, error) {
	key := keyOf(shape)
	if mesh, built := bridge.primitives[key]; built {
		return mesh, nil
	}
	if key.primitive != "box" {
		return nil, fmt.Errorf("unknown primitive %q (known: box)", key.primitive)
	}
	mesh := boxMesh(key)
	bridge.primitives[key] = mesh
	return mesh, nil
}

func keyOf(shape *world.Shape) primitiveKey {
	key := primitiveKey{primitive: shape.Primitive, size: shape.Size, color: defaultColor}
	if key.size == (spatial.Vector{}) {
		key.size = spatial.Vector{1, 1, 1}
	}
	key.offset = spatial.Vector{0, key.size[1] / 2, 0} // resting on its transform's origin
	if shape.Offset != nil {
		key.offset = *shape.Offset
	}
	if shape.Color != nil {
		key.color = *shape.Color
	}
	return key
}

// boxFaces lists each face as its normal and two edge directions.
var boxFaces = [6][3]spatial.Vector{
	{{1, 0, 0}, {0, 0, -1}, {0, 1, 0}}, {{-1, 0, 0}, {0, 0, 1}, {0, 1, 0}},
	{{0, 1, 0}, {1, 0, 0}, {0, 0, -1}}, {{0, -1, 0}, {1, 0, 0}, {0, 0, 1}},
	{{0, 0, 1}, {1, 0, 0}, {0, 1, 0}}, {{0, 0, -1}, {-1, 0, 0}, {0, 1, 0}},
}

// boxMesh builds a box with flat faces: four vertices and two triangles per face.
func boxMesh(key primitiveKey) *scene.Mesh {
	mesh := &scene.Mesh{Name: fmt.Sprintf("box %v", key.size), Material: &scene.Material{BaseColor: [4]float32{1, 1, 1, 1}}}
	half := key.size.Scale(0.5)
	for _, face := range boxFaces {
		addBoxFace(mesh, face, half, key)
	}
	return mesh
}

func addBoxFace(mesh *scene.Mesh, face [3]spatial.Vector, half spatial.Vector, key primitiveKey) {
	normal, across, up := face[0], face[1], face[2]
	first := uint32(mesh.VertexCount())
	for _, corner := range [4][2]float32{{-1, -1}, {1, -1}, {1, 1}, {-1, 1}} {
		point := scaleEach(normal.Add(across.Scale(corner[0])).Add(up.Scale(corner[1])), half).Add(key.offset)
		mesh.Positions = append(mesh.Positions, point[0], point[1], point[2])
		mesh.Normals = append(mesh.Normals, normal[0], normal[1], normal[2])
		mesh.Colors = append(mesh.Colors, key.color[0], key.color[1], key.color[2], key.color[3])
	}
	mesh.Indices = append(mesh.Indices, first, first+1, first+2, first, first+2, first+3)
}

func scaleEach(vector, factors spatial.Vector) spatial.Vector {
	return spatial.Vector{vector[0] * factors[0], vector[1] * factors[1], vector[2] * factors[2]}
}
