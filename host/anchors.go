package host

import (
	"fmt"
	"maps"
	"path/filepath"
	"strings"

	"github.com/akrck02/go-simulator/configuration"
	"github.com/akrck02/go-simulator/snapshot"
	"github.com/akrck02/go-simulator/spatial"
	"github.com/akrck02/go-simulator/world"
)

// AddAnchors adds the anchors of a rules file to a world, before the simulation starts. Each one
// becomes a hidden entity with the property "anchor": true, its role and its source.
func AddAnchors(rules *configuration.Configuration, initial *world.World) error {
	for _, anchor := range rules.Anchors {
		entities, err := anchorEntities(rules, anchor)
		if err != nil {
			return err
		}
		for _, entity := range entities {
			if err := initial.ApplyChange(world.AddEntity(entity)); err != nil {
				return fmt.Errorf("anchor %q: %w", entity.Identifier, err)
			}
		}
	}
	return nil
}

func anchorEntities(rules *configuration.Configuration, anchor configuration.Anchor) ([]*world.Entity, error) {
	if anchor.From == "" {
		return []*world.Entity{anchorEntity(anchor)}, nil
	}
	features, err := snapshot.ReadFeatures(rules.Resolve(anchor.From))
	if err != nil {
		return nil, fmt.Errorf("anchors: %w", err)
	}
	entities := make([]*world.Entity, 0, len(features))
	for index, feature := range features {
		entities = append(entities, anchorEntity(anchorFromFeature(anchor, feature, index)))
	}
	return entities, nil
}

// anchorFromFeature turns a GeoJSON feature into an anchor; the feature's id, kind and role win
// over the ones the rules file gives for the whole file.
func anchorFromFeature(file configuration.Anchor, feature snapshot.Feature, index int) configuration.Anchor {
	anchor := configuration.Anchor{
		Identifier: textOr(feature.Properties["id"], fmt.Sprintf("%s-%d", fileStem(file.From), index+1)),
		Kind:       textOr(feature.Properties["kind"], file.Kind),
		Role:       textOr(feature.Properties["role"], file.Role),
		At:         feature.Point, Line: feature.Line, Outline: feature.Outline,
		Width:      float32(numberOr(feature.Properties["width"], float64(file.Width))),
		Properties: maps.Clone(file.Properties),
		Source:     textOr(feature.Properties["source"], file.Source),
	}
	if anchor.Properties == nil {
		anchor.Properties = map[string]any{}
	}
	for name, value := range feature.Properties {
		if name != "id" && name != "kind" {
			anchor.Properties[name] = value
		}
	}
	return anchor
}

func anchorEntity(anchor configuration.Anchor) *world.Entity {
	properties := maps.Clone(anchor.Properties)
	if properties == nil {
		properties = map[string]any{}
	}
	properties[world.PropertyAnchor] = true
	if anchor.Role != "" {
		properties[world.PropertyRole] = anchor.Role
	}
	if anchor.Source != "" {
		properties["source"] = anchor.Source
	}
	entity := &world.Entity{Identifier: world.Identifier(anchor.Identifier), Kind: anchor.Kind, Properties: properties}
	if entity.Kind == "" {
		entity.Kind = world.KindAnchor
	}
	if anchor.At != nil {
		position := spatial.Translation(spatial.Vector{anchor.At[0], 0, anchor.At[1]})
		entity.Transform = &position
		return entity
	}
	entity.Shape = &world.Shape{Line: anchor.Line, Width: anchor.Width, Outline: anchor.Outline, Hidden: true}
	return entity
}

func textOr(value any, fallback string) string {
	if text, isText := value.(string); isText && text != "" {
		return text
	}
	return fallback
}

func numberOr(value any, fallback float64) float64 {
	if number, isNumber := value.(float64); isNumber {
		return number
	}
	return fallback
}

func fileStem(path string) string {
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}
