// Package conditions is the small language rules files use to say when something applies: which
// entities an objective counts, which cells get a biome, which shapes a style draws. A condition is
// a map from the name of a fact to what it must be:
//
//	kind: building                 equal to a value
//	use: [shop, workshop]          equal to one of the values
//	floors: {min: 2, max: 3}       a number within a range (either end may be left out)
//	river: true                    a yes/no fact
//
// Every entry must hold. A fact that does not exist never holds. Numbers are compared by value
// whatever their type (YAML gives int, JSON float64); everything else by its text.
package conditions

import (
	"errors"
	"fmt"
	"slices"
	"sort"

	"github.com/akrck02/go-simulator/world"
)

// Facts gives the value of a fact by name, and whether it exists.
type Facts func(name string) (any, bool)

// Holds tells whether every entry of a condition holds for the facts. An empty condition always
// holds.
func Holds(when map[string]any, facts Facts) bool {
	for name, wanted := range when {
		value, exists := facts(name)
		if !exists || !matches(wanted, value) {
			return false
		}
	}
	return true
}

// OfEntity gives an entity's facts: "kind" and its properties.
func OfEntity(entity *world.Entity) Facts {
	return func(name string) (any, bool) {
		if name == "kind" {
			return entity.Kind, entity.Kind != ""
		}
		value, exists := entity.Properties[name]
		return value, exists
	}
}

// OfMap gives the facts of a map, such as the data of an event.
func OfMap(values map[string]any) Facts {
	return func(name string) (any, bool) {
		value, exists := values[name]
		return value, exists
	}
}

// Check finds mistakes in a condition: ranges with keys other than min and max, bounds that are
// not numbers and, when known is not empty, facts that are not among the known names.
func Check(when map[string]any, known []string) error {
	var problems []error
	for _, name := range sortedNames(when) {
		if len(known) > 0 && !slices.Contains(known, name) {
			problems = append(problems, fmt.Errorf("unknown condition %q (known: %v)", name, known))
		}
		if bounds, isRange := when[name].(map[string]any); isRange {
			problems = append(problems, checkRange(name, bounds)...)
		}
	}
	return errors.Join(problems...)
}

func checkRange(name string, bounds map[string]any) []error {
	var problems []error
	for _, key := range sortedNames(bounds) {
		if key != "min" && key != "max" {
			problems = append(problems, fmt.Errorf("condition %q: a range takes min and max, not %q", name, key))
			continue
		}
		if _, isNumber := Number(bounds[key]); !isNumber {
			problems = append(problems, fmt.Errorf("condition %q: %s must be a number", name, key))
		}
	}
	return problems
}

func matches(wanted, value any) bool {
	switch typed := wanted.(type) {
	case []any:
		return slices.ContainsFunc(typed, func(option any) bool { return Equal(option, value) })
	case map[string]any:
		return InRange(typed, value)
	}
	return Equal(wanted, value)
}

// InRange tells whether a value is a number within {min, max}; a missing end does not limit.
func InRange(bounds map[string]any, value any) bool {
	number, isNumber := Number(value)
	if !isNumber {
		return false
	}
	if minimum, present := Number(bounds["min"]); present && number < minimum {
		return false
	}
	if maximum, present := Number(bounds["max"]); present && number > maximum {
		return false
	}
	return true
}

// Equal compares a written value with an actual one: numbers by value, the rest by their text.
func Equal(written, actual any) bool {
	first, firstIsNumber := Number(written)
	second, secondIsNumber := Number(actual)
	if firstIsNumber && secondIsNumber {
		return first == second
	}
	return fmt.Sprint(written) == fmt.Sprint(actual)
}

// Number reads a number of any of the types YAML and JSON give.
func Number(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case uint64:
		return float64(typed), true
	}
	return 0, false
}

func sortedNames[Value any](values map[string]Value) []string {
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
