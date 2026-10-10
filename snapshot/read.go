package snapshot

import (
	"encoding/json"
	"fmt"
	"os"
)

// Feature is one feature read from a GeoJSON file, in ground coordinates [x, z]. Only one of
// Point, Line and Outline is set; an outline does not repeat its first point.
type Feature struct {
	Point      *[2]float32
	Line       [][2]float32
	Outline    [][2]float32
	Properties map[string]any
}

// ReadFeatures reads the points, line strings and polygons (their outer ring) of a GeoJSON
// feature collection.
func ReadFeatures(path string) ([]Feature, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var collection struct {
		Features []struct {
			Geometry struct {
				Type        string          `json:"type"`
				Coordinates json.RawMessage `json:"coordinates"`
			} `json:"geometry"`
			Properties map[string]any `json:"properties"`
		} `json:"features"`
	}
	if err := json.Unmarshal(contents, &collection); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	features := make([]Feature, 0, len(collection.Features))
	for index, read := range collection.Features {
		feature, err := featureFrom(read.Geometry.Type, read.Geometry.Coordinates)
		if err != nil {
			return nil, fmt.Errorf("%s, feature %d: %w", path, index+1, err)
		}
		feature.Properties = read.Properties
		features = append(features, feature)
	}
	return features, nil
}

func featureFrom(kind string, coordinates json.RawMessage) (Feature, error) {
	switch kind {
	case "Point":
		var point [2]float32
		err := json.Unmarshal(coordinates, &point)
		return Feature{Point: &point}, err
	case "LineString":
		var line [][2]float32
		err := json.Unmarshal(coordinates, &line)
		return Feature{Line: line}, err
	case "Polygon":
		var rings [][][2]float32
		if err := json.Unmarshal(coordinates, &rings); err != nil || len(rings) == 0 {
			return Feature{}, fmt.Errorf("a polygon needs an outer ring: %v", err)
		}
		return Feature{Outline: withoutClosingPoint(rings[0])}, nil
	}
	return Feature{}, fmt.Errorf("geometry %q is not supported (Point, LineString or Polygon)", kind)
}

func withoutClosingPoint(ring [][2]float32) [][2]float32 {
	if count := len(ring); count > 1 && ring[0] == ring[count-1] {
		return ring[:count-1]
	}
	return ring
}
