// Package bridge connects the neutral world of a simulation to a go-renderer scene: it builds the
// initial world from a scene and applies every change set to the scene, so the renderer draws the
// simulation as it goes. Plugins never see the renderer; only this package does.
package bridge

import (
	"fmt"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-simulator/spatial"
	"github.com/akrck02/go-simulator/world"
)

// Bridge keeps track of which scene node shows each entity.
type Bridge struct {
	scene      *scene.Scene
	nodes      map[world.Identifier]*scene.Node
	parents    map[*scene.Node]*scene.Node // nil value = a top-level node
	meshes     map[string]*scene.Mesh      // the scene's meshes by name, for shapes that reuse one
	primitives map[primitiveKey]*scene.Mesh
	batches    map[string]*batch
	batchOf    map[world.Identifier]string // batch each batched entity is drawn in
	kinds      map[world.Identifier]string // kind of every entity added, for the builder
	builder    BatchBuilder                // draws the batches it handles; nil = the bridge draws them all
	Ignored    map[world.Operation]int     // changes the scene has no way to show yet, counted
}

// FromScene describes a scene as a world: every node becomes an entity and the environment
// comes along. It returns the world and the bridge that keeps the scene in step with it.
func FromScene(drawn *scene.Scene) (*world.World, *Bridge) {
	bridge := &Bridge{scene: drawn, nodes: map[world.Identifier]*scene.Node{}, parents: map[*scene.Node]*scene.Node{}, kinds: map[world.Identifier]string{},
		meshes: map[string]*scene.Mesh{}, primitives: map[primitiveKey]*scene.Mesh{}, batches: map[string]*batch{},
		batchOf: map[world.Identifier]string{}, Ignored: map[world.Operation]int{}}
	initial := world.New()
	for _, node := range drawn.Nodes {
		bridge.describeNode(initial, node, nil, "")
	}
	initial.Environment = environmentOf(drawn.Environment)
	return initial, bridge
}

// describeNode adds a node and its children to the world as entities.
func (bridge *Bridge) describeNode(described *world.World, node, parent *scene.Node, parentIdentifier world.Identifier) {
	identifier := bridge.identifierFor(node)
	bridge.nodes[identifier], bridge.parents[node] = node, parent
	if node.Mesh != nil && node.Mesh.Name != "" {
		bridge.meshes[node.Mesh.Name] = node.Mesh
	}
	_ = described.ApplyChange(world.AddEntity(entityOf(node, identifier, parentIdentifier)))
	for _, child := range node.Children {
		bridge.describeNode(described, child, node, identifier)
	}
}

// identifierFor names a node after itself; unnamed or repeated names get a number.
func (bridge *Bridge) identifierFor(node *scene.Node) world.Identifier {
	base := node.Name
	if base == "" {
		base = "node"
	}
	identifier := world.Identifier(base)
	for number := 2; bridge.nodes[identifier] != nil; number++ {
		identifier = world.Identifier(fmt.Sprintf("%s#%d", base, number))
	}
	return identifier
}

func entityOf(node *scene.Node, identifier, parent world.Identifier) *world.Entity {
	transform := spatial.Matrix(node.Transform)
	if transform.IsZero() {
		transform = spatial.Identity()
	}
	entity := &world.Entity{Identifier: identifier, Kind: "node", Parent: parent, Transform: &transform, Properties: copyTags(node.Tags)}
	if kind, named := node.Tags["kind"].(string); named {
		entity.Kind = kind
	}
	if node.Mesh != nil || node.Hidden {
		entity.Shape = &world.Shape{Hidden: node.Hidden}
		if node.Mesh != nil {
			entity.Shape.Mesh = node.Mesh.Name
		}
	}
	return entity
}

func environmentOf(environment scene.Environment) world.Environment {
	sun := vectorOf(environment.SunDirection)
	pole := vectorOf(environment.EffectiveSky().CelestialPole)
	wind := world.Wind{Direction: vectorOf(environment.Wind.Direction), Strength: environment.Wind.Strength}
	described := world.Environment{SunDirection: &sun, CelestialPole: &pole, Wind: &wind}
	if environment.Sky != nil {
		offset := vectorOf(environment.Sky.Clouds.Offset)
		described.CloudOffset = &offset
	}
	if environment.Water != nil && environment.Water.Variation != nil {
		angle := environment.Water.Variation.Angle
		described.TideAngle = &angle
	}
	return described
}

func vectorOf(vector graphics.Vec4) spatial.Vector {
	return spatial.Vector{vector[0], vector[1], vector[2]}
}

func vec4Of(vector spatial.Vector) graphics.Vec4 {
	return graphics.Vec4{vector[0], vector[1], vector[2], 0}
}

func copyTags(tags map[string]any) map[string]any {
	if len(tags) == 0 {
		return nil
	}
	copied := make(map[string]any, len(tags))
	for name, value := range tags {
		copied[name] = value
	}
	return copied
}
