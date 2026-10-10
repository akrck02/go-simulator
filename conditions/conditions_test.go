package conditions

import (
	"testing"

	"github.com/akrck02/go-simulator/world"
)

func TestHoldsForAnEntity(t *testing.T) {
	house := &world.Entity{Identifier: "house-1", Kind: "building", Properties: map[string]any{"use": "shop", "floors": 3.0}}
	cases := []struct {
		name  string
		when  map[string]any
		holds bool
	}{
		{"empty", map[string]any{}, true},
		{"kind", map[string]any{"kind": "building"}, true},
		{"other kind", map[string]any{"kind": "street"}, false},
		{"one of", map[string]any{"use": []any{"housing", "shop"}}, true},
		{"none of", map[string]any{"use": []any{"housing", "workshop"}}, false},
		{"range with an int from YAML", map[string]any{"floors": map[string]any{"min": 2, "max": 3}}, true},
		{"below a range", map[string]any{"floors": map[string]any{"min": 4}}, false},
		{"number by value", map[string]any{"floors": 3}, true},
		{"missing fact", map[string]any{"sector": "forja"}, false},
		{"every entry", map[string]any{"kind": "building", "use": "housing"}, false},
	}
	for _, testCase := range cases {
		if holds := Holds(testCase.when, OfEntity(house)); holds != testCase.holds {
			t.Errorf("%s: holds = %v, want %v", testCase.name, holds, testCase.holds)
		}
	}
}

func TestHoldsForBooleans(t *testing.T) {
	facts := OfMap(map[string]any{"river": true})
	if !Holds(map[string]any{"river": true}, facts) || Holds(map[string]any{"river": false}, facts) {
		t.Error("a yes/no fact must match its value")
	}
}

func TestCheckFindsMistakes(t *testing.T) {
	if err := Check(map[string]any{"temperature": map[string]any{"max": 7}}, []string{"temperature"}); err != nil {
		t.Errorf("a good condition: %v", err)
	}
	if Check(map[string]any{"height": map[string]any{"maximum": 7}}, nil) == nil {
		t.Error("a range with an unknown key must fail")
	}
	if Check(map[string]any{"height": map[string]any{"max": "high"}}, nil) == nil {
		t.Error("a bound that is not a number must fail")
	}
	if Check(map[string]any{"heigth": 3}, []string{"height"}) == nil {
		t.Error("an unknown fact must fail when the known facts are given")
	}
}
