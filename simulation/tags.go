package simulation

import (
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"
)

// FromTags builds the motion systems that nodes ask for in their tags (glTF node extras):
//
//	"bob":  {"height": h, "period": seconds, "roll": radians}
//	"spin": {"axis": [x, y, z], "speed": radians per second}
//	"path": {"points": [[x, y, z], ...], "speed": units per second, "loop": true}
func FromTags(world *scene.Scene) []System {
	var systems []System
	world.Walk(func(node *scene.Node, _ graphics.Mat4) {
		systems = append(systems, systemsForNode(node)...)
	})
	return systems
}

func systemsForNode(node *scene.Node) []System {
	var systems []System
	if values, found := node.Tags["path"].(map[string]any); found {
		systems = append(systems, NewPathFollower(node, points(values["points"]), number(values, "speed", 1), flag(values, "loop", true)))
	}
	if values, found := node.Tags["spin"].(map[string]any); found {
		systems = append(systems, NewSpin(node, vector(values, "axis", graphics.Vec4{0, 1, 0, 0}), float64(number(values, "speed", 1))))
	}
	if values, found := node.Tags["bob"].(map[string]any); found {
		systems = append(systems, NewBobbing(node, number(values, "height", 0.001), float64(number(values, "period", 4)), float64(number(values, "roll", 0.03))))
	}
	return systems
}

func number(values map[string]any, key string, fallback float32) float32 {
	if value, isNumber := values[key].(float64); isNumber {
		return float32(value)
	}
	return fallback
}

func flag(values map[string]any, key string, fallback bool) bool {
	if value, isBool := values[key].(bool); isBool {
		return value
	}
	return fallback
}

func vector(values map[string]any, key string, fallback graphics.Vec4) graphics.Vec4 {
	list, isList := values[key].([]any)
	if !isList {
		return fallback
	}
	result := fallback
	for component := 0; component < len(list) && component < 3; component++ {
		if value, isNumber := list[component].(float64); isNumber {
			result[component] = float32(value)
		}
	}
	return result
}

func points(value any) []graphics.Vec4 {
	list, _ := value.([]any)
	result := make([]graphics.Vec4, 0, len(list))
	for _, entry := range list {
		result = append(result, vector(map[string]any{"point": entry}, "point", graphics.Vec4{}))
	}
	return result
}
