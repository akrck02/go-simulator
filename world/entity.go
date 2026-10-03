// Package world is the neutral description of what a simulation knows about: entities, fields
// and the environment. It does not depend on any renderer; the bridge package maps it onto a
// go-renderer scene. Worlds only change by applying changes (see Change), so every change can be
// recorded, forwarded to plugins and replayed.
package world

import "github.com/akrck02/go-simulator/spatial"

// Identifier names an entity; it is stable for the whole life of the entity.
type Identifier string

// Entity is anything that exists in the world: a boat, a house, a street, a parcel.
type Entity struct {
	Identifier Identifier      `json:"id"`
	Kind       string          `json:"kind,omitempty"`   // what it is ("boat", "house"); free text chosen by plugins
	Parent     Identifier      `json:"parent,omitempty"` // entity it is attached to; empty = the world
	Transform  *spatial.Matrix `json:"transform,omitempty"`
	Shape      *Shape          `json:"shape,omitempty"`
	Properties map[string]any  `json:"properties,omitempty"` // free data: "floors": 3, "use": "shop"
}

// Shape says how an entity looks. Only one way is used: an existing mesh by name or a primitive.
type Shape struct {
	Mesh      string          `json:"mesh,omitempty"`      // reuse a mesh the scene already has
	Primitive string          `json:"primitive,omitempty"` // "box"
	Size      spatial.Vector  `json:"size,omitempty"`      // primitive size (width, height, depth)
	Color     *[4]float32     `json:"color,omitempty"`     // linear RGBA
	Hidden    bool            `json:"hidden,omitempty"`
	Offset    *spatial.Vector `json:"offset,omitempty"` // primitive centre relative to the transform; nil = resting on it
}

// HasProperty tells whether the entity has a property, whatever its value.
func (entity *Entity) HasProperty(name string) bool {
	_, found := entity.Properties[name]
	return found
}

// Copy returns a deep copy of the entity, so the copy can change without touching the original.
func (entity *Entity) Copy() *Entity {
	copied := *entity
	if entity.Transform != nil {
		transform := *entity.Transform
		copied.Transform = &transform
	}
	if entity.Shape != nil {
		shape := *entity.Shape
		copied.Shape = &shape
	}
	copied.Properties = copyProperties(entity.Properties)
	return &copied
}

// merge applies a patch: a set transform, shape, kind or parent replaces the current one;
// properties are merged one by one and a nil value removes the property.
func (entity *Entity) merge(patch *Entity) {
	if patch.Kind != "" {
		entity.Kind = patch.Kind
	}
	if patch.Parent != "" {
		entity.Parent = patch.Parent
	}
	if patch.Transform != nil {
		transform := *patch.Transform
		entity.Transform = &transform
	}
	if patch.Shape != nil {
		shape := *patch.Shape
		entity.Shape = &shape
	}
	entity.Properties = mergeProperties(entity.Properties, patch.Properties)
}

func mergeProperties(current, patch map[string]any) map[string]any {
	if len(patch) == 0 {
		return current
	}
	if current == nil {
		current = map[string]any{}
	}
	for name, value := range patch {
		if value == nil {
			delete(current, name)
			continue
		}
		current[name] = value
	}
	return current
}

func copyProperties(properties map[string]any) map[string]any {
	if properties == nil {
		return nil
	}
	copied := make(map[string]any, len(properties))
	for name, value := range properties {
		copied[name] = value
	}
	return copied
}
