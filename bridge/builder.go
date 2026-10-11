package bridge

import (
	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-simulator/world"
)

// BatchBuilder draws whole batches of ground shapes in place of the bridge: a scene builder that
// turns entities into finished geometry (roofs, gates…) following style rules. The bridge keeps
// drawing the batches it does not handle.
type BatchBuilder interface {
	// Handles tells whether the builder draws the entities of this kind and properties.
	Handles(kind string, properties map[string]any) bool
	// Build returns the nodes (meshes, or instanced models) of a batch's entities. context are the
	// entities of the other batches the builder handles (a house looks onto a street of another
	// batch); ground is the height of the scene's walkable ground at a point, in world units, and
	// whether there is ground there.
	Build(entities, context []*world.Entity, ground func(x, z float32) (float32, bool)) []*scene.Node
}

// UseBuilder hands the batches a builder handles over to it.
func (bridge *Bridge) UseBuilder(builder BatchBuilder) {
	bridge.builder = builder
	for _, group := range bridge.batches {
		group.changed = true
	}
}

// entityOfMember is the entity a batch member stands for, with its current properties.
func (bridge *Bridge) entityOfMember(identifier world.Identifier, shape *world.Shape) *world.Entity {
	entity := &world.Entity{Identifier: identifier, Kind: bridge.kinds[identifier], Shape: shape}
	if node, found := bridge.nodes[identifier]; found {
		entity.Properties = node.Tags
	}
	return entity
}

// handledBy tells whether the builder draws a batch (judged by its first member).
func (bridge *Bridge) handledBy(group *batch) bool {
	if bridge.builder == nil {
		return false
	}
	for _, identifier := range group.order {
		if _, member := group.shapes[identifier]; member {
			entity := bridge.entityOfMember(identifier, nil)
			return bridge.builder.Handles(entity.Kind, entity.Properties)
		}
	}
	return false
}

// buildWithBuilder replaces a batch's mesh with the builder's meshes, as children of its node.
func (bridge *Bridge) buildWithBuilder(group *batch) {
	var entities, context []*world.Entity
	for _, other := range bridge.batches {
		if other != group && !bridge.handledBy(other) {
			continue
		}
		for _, identifier := range other.order {
			shape, member := other.shapes[identifier]
			if !member {
				continue
			}
			if other == group {
				entities = append(entities, bridge.entityOfMember(identifier, shape))
			} else {
				context = append(context, bridge.entityOfMember(identifier, shape))
			}
		}
	}
	bridge.scene.GroundHeight(0, 0) // builds the ground index now: the builder asks from several goroutines
	group.node.Mesh = nil
	group.node.Children = group.node.Children[:0]
	for _, node := range bridge.builder.Build(entities, context, bridge.scene.GroundHeight) {
		if node.Mesh != nil {
			node.Mesh.Dirty = true
		}
		if node.Instances != nil {
			node.Instances.Dirty = true
		}
		group.node.Children = append(group.node.Children, node)
	}
}
