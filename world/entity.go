// Package world is the neutral description of what a simulation knows about: entities, fields
// and the environment. It does not depend on any renderer; the bridge package maps it onto a
// go-renderer scene. Worlds only change by applying changes (see Change), so every change can be
// recorded, forwarded to plugins and replayed.
package world

import (
	"slices"

	"github.com/akrck02/go-simulator/spatial"
)

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

// Shape says how an entity looks. Only one way is used: an existing mesh by name, a primitive, an
// outline on the ground or a line on the ground.
//
// Outlines and lines are drawn in world coordinates on the ground plane (x, z) and follow the
// terrain; the entity's transform does not move them. An outline is a flat area (a parcel, a
// square) or, with a height, a volume (a house, a tower); a line is a strip of some width (a
// street) or, with a height, a wall. Shapes with the same Batch are drawn as one mesh, which keeps
// thousands of small things cheap to draw.
type Shape struct {
	Mesh      string          `json:"mesh,omitempty"`      // reuse a mesh the scene already has
	Primitive string          `json:"primitive,omitempty"` // "box"
	Size      spatial.Vector  `json:"size,omitempty"`      // primitive size (width, height, depth)
	Offset    *spatial.Vector `json:"offset,omitempty"`    // primitive centre relative to the transform; nil = resting on it
	Outline   [][2]float32    `json:"outline,omitempty"`   // ground polygon (x, z), without repeating the first point
	Line      [][2]float32    `json:"line,omitempty"`      // ground polyline (x, z)
	Width     float32         `json:"width,omitempty"`     // width of a line
	Height    float32         `json:"height,omitempty"`    // height of an outline or a line above the ground (0 = flat)
	Lift      float32         `json:"lift,omitempty"`      // how far a flat shape floats over the ground, so it does not flicker
	Batch     string          `json:"batch,omitempty"`     // shapes of the same batch become one mesh
	Color     *[4]float32     `json:"color,omitempty"`     // linear RGBA
	Hidden    bool            `json:"hidden,omitempty"`
}

// Copy returns a deep copy of the shape.
func (shape *Shape) Copy() *Shape {
	copied := *shape
	copied.Outline = slices.Clone(shape.Outline)
	copied.Line = slices.Clone(shape.Line)
	if shape.Color != nil {
		color := *shape.Color
		copied.Color = &color
	}
	if shape.Offset != nil {
		offset := *shape.Offset
		copied.Offset = &offset
	}
	return &copied
}

// OnTheGround tells whether the shape is drawn from ground coordinates (an outline or a line).
func (shape *Shape) OnTheGround() bool { return len(shape.Outline) > 0 || len(shape.Line) > 0 }

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
		copied.Shape = entity.Shape.Copy()
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
		entity.Shape = patch.Shape.Copy()
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
