// Package scenebuilder lets the bridge draw batches with go-scene-builder: entities become map
// features (kind, properties, outline or line) and the builder's pieces become scene meshes, so a
// live simulation shows the same roofs, gates and bridges as a scene built from a map. It needs
// GEOS, like go-scene-builder.
package scenebuilder

import (
	"math"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-scene-builder/build"
	"github.com/akrck02/go-scene-builder/mapdata"
	"github.com/akrck02/go-scene-builder/mesh"
	"github.com/akrck02/go-scene-builder/models"
	"github.com/akrck02/go-scene-builder/style"
	"github.com/akrck02/go-simulator/world"
)

// detail is the longest side (km) of the triangles laid on the ground, so they follow it.
const detail = 0.0125

// Builder draws the entities its style rules match.
type Builder struct {
	rules     *style.Rules
	library   *models.Library
	materials map[string]*scene.Material
}

// Load reads the style rules and the model libraries they name.
func Load(path string) (*Builder, error) {
	rules, err := style.Load(path)
	if err != nil {
		return nil, err
	}
	library, err := models.Load(rules.ModelFiles()...)
	if err != nil {
		return nil, err
	}
	return &Builder{rules: rules, library: library, materials: map[string]*scene.Material{}}, nil
}

// Handles tells whether some rule gives entities of this kind a shape.
func (builder *Builder) Handles(kind string, properties map[string]any) bool {
	resolved, matched := builder.rules.Resolve(kind, properties)
	return matched && resolved.Section("shape").Text("role", "none") != "none"
}

// Build turns entities into nodes (meshes and instanced models), with context entities taking part
// in the relations.
func (builder *Builder) Build(entities, context []*world.Entity, ground func(x, z float32) (float32, bool)) []*scene.Node {
	heights := build.HeightFunction(func(east, north float64) float64 {
		height, found := ground(float32(east), float32(-north))
		if !found {
			return 0
		}
		return float64(height) * 1000
	})
	origin := mapdata.Point{0, 0} // scene x is east and z is south: the frame needs no shift
	result := build.Build(build.Input{Map: &mapdata.Map{Features: featuresOf(entities), Origin: &origin}, Rules: builder.rules,
		Heights: heights, Detail: detail, Context: featuresOf(context), Models: builder.library})
	nodes := make([]*scene.Node, 0, len(result.Pieces)+len(result.Instances))
	for _, piece := range result.Pieces {
		nodes = append(nodes, &scene.Node{Name: piece.Name, Transform: graphics.Identity(), Mesh: builder.meshOf(piece)})
	}
	for _, copies := range result.Instances {
		instances := instancesOf(copies)
		for _, piece := range copies.Model {
			nodes = append(nodes, &scene.Node{Name: copies.Name, Transform: graphics.Identity(), Mesh: builder.meshOf(piece), Instances: instances})
		}
	}
	return nodes
}

// instancesOf turns the builder's copies into the renderer's instances.
func instancesOf(copies *mesh.Instances) *scene.Instances {
	instances := &scene.Instances{}
	for index := range copies.Count() {
		half := float64(copies.Turns[index]) / 2
		position := graphics.Vec4{copies.Positions[3*index], copies.Positions[3*index+1], copies.Positions[3*index+2], 1}
		rotation := graphics.Vec4{0, float32(math.Sin(half)), 0, float32(math.Cos(half))}
		scale := copies.Scales[index]
		instances.Transforms = append(instances.Transforms, graphics.FromTranslationRotationScale(position, rotation, graphics.Vec4{scale, scale, scale, 1}))
		tint := copies.Tints[4*index : 4*index+4]
		instances.Colors = append(instances.Colors, graphics.Vec4{tint[0], tint[1], tint[2], tint[3]})
	}
	return instances
}

// featuresOf turns entities with ground shapes into map features (km east and north).
func featuresOf(entities []*world.Entity) []*mapdata.Feature {
	features := make([]*mapdata.Feature, 0, len(entities))
	for _, entity := range entities {
		if entity.Shape == nil || entity.Shape.Hidden {
			continue
		}
		feature := &mapdata.Feature{Identifier: string(entity.Identifier), Kind: entity.Kind, Properties: propertiesOf(entity)}
		if len(entity.Shape.Outline) >= 3 {
			feature.Polygons = []mapdata.Polygon{{Exterior: mapdata.Ring(pointsOf(entity.Shape.Outline))}}
		}
		if len(entity.Shape.Line) >= 2 {
			feature.Lines = [][]mapdata.Point{pointsOf(entity.Shape.Line)}
		}
		features = append(features, feature)
	}
	return features
}

// propertiesOf are the entity's properties, plus the shape's width (km) when they lack one.
func propertiesOf(entity *world.Entity) map[string]any {
	properties := make(map[string]any, len(entity.Properties)+2)
	for key, value := range entity.Properties {
		properties[key] = value
	}
	properties["id"] = string(entity.Identifier) // stable choices (palettes, plants) follow the entity
	if _, present := properties["width"]; !present && entity.Shape.Width > 0 {
		properties["width"] = float64(entity.Shape.Width)
	}
	return properties
}

func pointsOf(scenePoints [][2]float32) []mapdata.Point {
	points := make([]mapdata.Point, len(scenePoints))
	for index, point := range scenePoints {
		points[index] = mapdata.Point{float64(point[0]), -float64(point[1])}
	}
	return points
}

func (builder *Builder) meshOf(piece *mesh.Piece) *scene.Mesh {
	return &scene.Mesh{Name: piece.Name, Positions: piece.Positions, Normals: piece.Normals, Colors: piece.Colors,
		Indices: piece.Indices, Material: builder.material(piece.Material), Ground: piece.Ground}
}

func (builder *Builder) material(material mesh.Material) *scene.Material {
	if existing, known := builder.materials[material.Name]; known {
		return existing
	}
	made := &scene.Material{Name: material.Name, Kind: scene.KindLit, BaseColor: [4]float32{1, 1, 1, 1},
		DoubleSided: material.DoubleSided, DepthBias: float32(material.DepthBias),
		Pattern: scene.PatternNamed(material.Pattern), PatternScale: float32(material.PatternScale / 1000)}
	if material.Kind == "glow" {
		made.Kind = scene.KindGlow
	}
	builder.materials[material.Name] = made
	return made
}
