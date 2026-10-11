// Package snapshot writes the state of a world to files other tools can read: GeoJSON for the
// entities and JSON lines for the events.
//
// GeoJSON coordinates are world coordinates on the ground plane: [x, z]. Entities with an outline
// become polygons, with a line become line strings, and the rest become points at their
// transform's position. Every feature carries the entity's id, kind, parent and properties, and
// the shape's height and width when it has them.
package snapshot

import (
	"encoding/json"
	"os"

	"github.com/akrck02/go-simulator/world"
)

type featureCollection struct {
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties,omitempty"`
	Features   []feature      `json:"features"`
}

type feature struct {
	Type       string         `json:"type"`
	Geometry   geometry       `json:"geometry"`
	Properties map[string]any `json:"properties"`
}

type geometry struct {
	Type        string `json:"type"`
	Coordinates any    `json:"coordinates"`
}

// WriteEntities saves every entity of the world as GeoJSON. Extra goes into the collection's
// properties (the stage and its time, for example).
func WriteEntities(path string, current *world.World, extra map[string]any) error {
	collection := featureCollection{Type: "FeatureCollection", Properties: extra, Features: []feature{}}
	for _, entity := range current.Entities() {
		if shape, found := geometryOf(entity); found {
			collection.Features = append(collection.Features, feature{Type: "Feature", Geometry: shape, Properties: propertiesOf(entity)})
		}
	}
	encoded, err := json.Marshal(collection)
	if err != nil {
		return err
	}
	return os.WriteFile(path, encoded, 0o644)
}

func geometryOf(entity *world.Entity) (geometry, bool) {
	shape := entity.Shape
	switch {
	case shape != nil && len(shape.Outline) >= 3:
		ring := append(append([][2]float32{}, shape.Outline...), shape.Outline[0])
		return geometry{Type: "Polygon", Coordinates: [][][2]float32{ring}}, true
	case shape != nil && len(shape.Line) >= 2:
		return geometry{Type: "LineString", Coordinates: shape.Line}, true
	case entity.Transform != nil:
		position := entity.Transform.Position()
		return geometry{Type: "Point", Coordinates: [2]float32{position[0], position[2]}}, true
	}
	return geometry{}, false
}

func propertiesOf(entity *world.Entity) map[string]any {
	properties := map[string]any{"id": entity.Identifier, "kind": entity.Kind}
	if entity.Parent != "" {
		properties["parent"] = entity.Parent
	}
	if shape := entity.Shape; shape != nil {
		if shape.Height > 0 {
			properties["height"] = shape.Height
		}
		if shape.Width > 0 {
			properties["width"] = shape.Width
		}
	}
	for name, value := range entity.Properties {
		if _, taken := properties[name]; !taken {
			properties[name] = value
		}
	}
	return properties
}
