package bridge

import (
	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-simulator/world"
)

// batch draws the ground shapes of many entities as one mesh.
type batch struct {
	node    *scene.Node
	shapes  map[world.Identifier]*world.Shape
	order   []world.Identifier
	changed bool
}

// placeGroundShape draws an entity's outline or line: in its batch's mesh, or in a mesh of its own.
func (bridge *Bridge) placeGroundShape(identifier world.Identifier, node *scene.Node, shape *world.Shape) {
	bridge.leaveBatch(identifier)
	if shape.Batch == "" {
		builder := &meshBuilder{}
		builder.addShape(shape, bridge.groundHeight)
		node.Mesh = &scene.Mesh{Name: string(identifier), Material: groundMaterial(shape)}
		builder.fill(node.Mesh)
		return
	}
	node.Mesh = nil
	group := bridge.batchNamed(shape.Batch, shape)
	if _, member := group.shapes[identifier]; !member {
		group.order = append(group.order, identifier)
	}
	group.shapes[identifier] = shape.Copy()
	group.changed = true
	bridge.batchOf[identifier] = shape.Batch
}

// leaveBatch takes an entity out of the batch it was drawn in, if any.
func (bridge *Bridge) leaveBatch(identifier world.Identifier) {
	name, member := bridge.batchOf[identifier]
	if !member {
		return
	}
	group := bridge.batches[name]
	delete(group.shapes, identifier)
	group.changed = true
	delete(bridge.batchOf, identifier)
}

// batchNamed returns a batch, creating its node the first time; the first shape sets its material.
func (bridge *Bridge) batchNamed(name string, first *world.Shape) *batch {
	if group, exists := bridge.batches[name]; exists {
		return group
	}
	mesh := &scene.Mesh{Name: "batch " + name, Material: groundMaterial(first)}
	group := &batch{node: &scene.Node{Name: "batch " + name, Mesh: mesh}, shapes: map[world.Identifier]*world.Shape{}}
	bridge.scene.Nodes = append(bridge.scene.Nodes, group.node)
	bridge.batches[name] = group
	return group
}

// rebuildChangedBatches builds again the meshes of the batches that changed.
func (bridge *Bridge) rebuildChangedBatches() {
	for _, group := range bridge.batches {
		if group.changed {
			group.rebuild(bridge.groundHeight)
		}
	}
}

func (group *batch) rebuild(ground groundHeight) {
	builder := &meshBuilder{}
	kept := group.order[:0]
	for _, identifier := range group.order {
		shape, member := group.shapes[identifier]
		if !member {
			continue
		}
		kept = append(kept, identifier)
		builder.addShape(shape, ground)
	}
	group.order = kept
	builder.fill(group.node.Mesh)
	group.changed = false
}

// groundHeight is the height of the scene's ground at a point; 0 where there is none.
func (bridge *Bridge) groundHeight(x, z float32) float32 {
	if height, found := bridge.scene.GroundHeight(x, z); found {
		return height
	}
	return 0
}
