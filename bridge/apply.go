package bridge

import (
	"fmt"
	"slices"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-simulator/world"
)

// Apply shows a change set in the scene. Call it on the thread that draws, between frames.
func (bridge *Bridge) Apply(set world.ChangeSet) error {
	return bridge.ApplyAll([]world.ChangeSet{set})
}

// ApplyAll shows several change sets, rebuilding the batches that changed only once at the end.
func (bridge *Bridge) ApplyAll(sets []world.ChangeSet) error {
	defer bridge.rebuildChangedBatches()
	for _, set := range sets {
		for index, change := range set.Changes {
			if err := bridge.applyChange(change); err != nil {
				return fmt.Errorf("showing change %d (%s) from %q: %w", index, change.Operation, set.Origin, err)
			}
		}
	}
	return nil
}

func (bridge *Bridge) applyChange(change world.Change) error {
	switch change.Operation {
	case world.Add:
		return bridge.add(change.Entity)
	case world.Remove:
		return bridge.remove(change.Identifier)
	case world.Modify:
		return bridge.modify(change.Entity)
	case world.SetEnvironment:
		bridge.setEnvironment(change.Environment)
		return nil
	case world.SetField:
		bridge.Ignored[change.Operation]++
		return nil
	case world.ClearArea:
		bridge.clearArea(change.Area)
		return nil
	}
	return nil
}

func (bridge *Bridge) add(entity *world.Entity) error {
	node := &scene.Node{Name: string(entity.Identifier), Transform: graphics.Identity(), Tags: copyTags(entity.Properties)}
	bridge.kinds[entity.Identifier] = entity.Kind
	if entity.Transform != nil {
		node.Transform = graphics.Mat4(*entity.Transform)
	}
	parent, err := bridge.parentNode(entity.Parent)
	if err != nil {
		return err
	}
	if err := bridge.shape(entity.Identifier, node, entity.Shape); err != nil {
		return err
	}
	bridge.attach(node, parent)
	bridge.nodes[entity.Identifier] = node
	return nil
}

func (bridge *Bridge) parentNode(identifier world.Identifier) (*scene.Node, error) {
	if identifier == "" {
		return nil, nil
	}
	parent, found := bridge.nodes[identifier]
	if !found {
		return nil, fmt.Errorf("no entity %q to attach to", identifier)
	}
	return parent, nil
}

func (bridge *Bridge) attach(node, parent *scene.Node) {
	bridge.parents[node] = parent
	if parent == nil {
		bridge.scene.Nodes = append(bridge.scene.Nodes, node)
	} else {
		parent.Children = append(parent.Children, node)
	}
	if node.Mesh != nil && node.Mesh.Ground {
		bridge.scene.InvalidateGround()
	}
}

// remove takes a node and its children out of the scene.
func (bridge *Bridge) remove(identifier world.Identifier) error {
	node, found := bridge.nodes[identifier]
	if !found {
		return fmt.Errorf("no entity %q", identifier)
	}
	parent := bridge.parents[node]
	if parent == nil {
		bridge.scene.Nodes = slices.DeleteFunc(bridge.scene.Nodes, func(candidate *scene.Node) bool { return candidate == node })
	} else {
		parent.Children = slices.DeleteFunc(parent.Children, func(candidate *scene.Node) bool { return candidate == node })
	}
	if holdsGround(node) {
		bridge.scene.InvalidateGround()
	}
	bridge.forget(node)
	return nil
}

// forget drops a node and its children from the bridge's records.
func (bridge *Bridge) forget(node *scene.Node) {
	for _, child := range node.Children {
		bridge.forget(child)
	}
	delete(bridge.parents, node)
	for identifier, candidate := range bridge.nodes {
		if candidate == node {
			delete(bridge.nodes, identifier)
			bridge.leaveBatch(identifier)
		}
	}
}

// holdsGround tells whether a node or its children walk on a ground mesh.
func holdsGround(node *scene.Node) bool {
	if node.Mesh != nil && node.Mesh.Ground {
		return true
	}
	for _, child := range node.Children {
		if holdsGround(child) {
			return true
		}
	}
	return false
}

func (bridge *Bridge) modify(patch *world.Entity) error {
	node, found := bridge.nodes[patch.Identifier]
	if !found {
		return fmt.Errorf("no entity %q", patch.Identifier)
	}
	if patch.Transform != nil {
		node.Transform = graphics.Mat4(*patch.Transform)
	}
	if patch.Shape != nil {
		if err := bridge.shape(patch.Identifier, node, patch.Shape); err != nil {
			return err
		}
	}
	node.Tags = mergeTags(node.Tags, patch.Properties)
	return nil
}

// shape gives a node the mesh a shape asks for.
func (bridge *Bridge) shape(identifier world.Identifier, node *scene.Node, shape *world.Shape) error {
	if shape == nil {
		return nil
	}
	if shape.OnTheGround() {
		bridge.placeGroundShape(identifier, node, shape)
		return nil
	}
	bridge.leaveBatch(identifier)
	node.Hidden = shape.Hidden
	switch {
	case shape.Mesh != "":
		mesh, found := bridge.meshes[shape.Mesh]
		if !found {
			return fmt.Errorf("the scene has no mesh %q", shape.Mesh)
		}
		node.Mesh = mesh
	case shape.Primitive != "":
		mesh, err := bridge.primitive(shape)
		if err != nil {
			return err
		}
		node.Mesh = mesh
	}
	return nil
}

func (bridge *Bridge) setEnvironment(patch *world.Environment) {
	environment := &bridge.scene.Environment
	if patch.SunDirection != nil {
		environment.SunDirection = vec4Of(*patch.SunDirection).Normalize()
	}
	if patch.Wind != nil {
		environment.Wind = scene.Wind{Direction: vec4Of(patch.Wind.Direction), Strength: patch.Wind.Strength}
	}
	if patch.CloudOffset != nil && environment.Sky != nil {
		environment.Sky.Clouds.Offset = vec4Of(*patch.CloudOffset)
	}
	if patch.CelestialPole != nil && environment.Sky != nil {
		environment.Sky.CelestialPole = vec4Of(*patch.CelestialPole).Normalize()
	}
	if patch.TideAngle != nil && environment.Water != nil && environment.Water.Variation != nil {
		environment.Water.Variation.Angle = *patch.TideAngle
	}
}

func mergeTags(tags, patch map[string]any) map[string]any {
	if len(patch) == 0 {
		return tags
	}
	if tags == nil {
		tags = map[string]any{}
	}
	for name, value := range patch {
		if value == nil {
			delete(tags, name)
			continue
		}
		tags[name] = value
	}
	return tags
}
