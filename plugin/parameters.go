package plugin

import "github.com/akrck02/go-simulator/spatial"

// Parameters are the values a rules file gives a rule. They arrive as JSON values: numbers are
// float64, lists are []any and maps are map[string]any.
type Parameters map[string]any

// Number returns a number, or the fallback when it is missing or not a number.
func (parameters Parameters) Number(name string, fallback float64) float64 {
	if value, isNumber := parameters[name].(float64); isNumber {
		return value
	}
	return fallback
}

// Boolean returns a yes/no value, or the fallback.
func (parameters Parameters) Boolean(name string, fallback bool) bool {
	if value, isBoolean := parameters[name].(bool); isBoolean {
		return value
	}
	return fallback
}

// Text returns a text, or the fallback.
func (parameters Parameters) Text(name string, fallback string) string {
	if value, isText := parameters[name].(string); isText {
		return value
	}
	return fallback
}

// Vector returns a vector written as a list of up to three numbers, or the fallback.
func (parameters Parameters) Vector(name string, fallback spatial.Vector) spatial.Vector {
	return VectorFrom(parameters[name], fallback)
}

// Map returns nested parameters, or nil.
func (parameters Parameters) Map(name string) Parameters {
	if value, isMap := parameters[name].(map[string]any); isMap {
		return value
	}
	return nil
}

// VectorFrom reads a list of up to three numbers; missing components keep the fallback's.
func VectorFrom(value any, fallback spatial.Vector) spatial.Vector {
	list, isList := value.([]any)
	if !isList {
		return fallback
	}
	result := fallback
	for component := 0; component < len(list) && component < 3; component++ {
		if number, isNumber := list[component].(float64); isNumber {
			result[component] = float32(number)
		}
	}
	return result
}
