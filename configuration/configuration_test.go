package configuration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, folder, name, contents string) string {
	t.Helper()
	path := filepath.Join(folder, name)
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestIncludesBringParametersFromFiles(t *testing.T) {
	folder := t.TempDir()
	writeFile(t, folder, "scenario.yaml", "# the scenario, with comments\nseed: 7\npopulation: {to: 6000}\nstory: ignored by the plugins\n")
	writeFile(t, folder, "seasons.yaml", "seed: 8\nwarmest: 0.35\n")
	rules := writeFile(t, folder, "rules.yaml", `
plugins: [{name: urban, command: [urban]}]
stages:
  - name: city
    clock: {from: 0, to: 3, step: 1}
    rules:
      - rule: city
        with: {include: [scenario.yaml, seasons.yaml], warmest: 0.4}
`)
	configuration, err := Load(rules)
	if err != nil {
		t.Fatal(err)
	}
	with := configuration.Stages[0].Rules[0].With
	if with["seed"] != 8 || with["warmest"] != 0.4 || with["story"] != "ignored by the plugins" {
		t.Errorf("with = %v: later files replace earlier ones and the rule's own parameters win", with)
	}
	if population, isMap := with["population"].(map[string]any); !isMap || population["to"] != 6000 {
		t.Errorf("population = %v, want the included map", with["population"])
	}
	if _, kept := with["include"]; kept {
		t.Error("the include parameter must go once expanded")
	}
}

func TestIncludedFilesCannotIncludeOthers(t *testing.T) {
	folder := t.TempDir()
	writeFile(t, folder, "scenario.yaml", "include: other.yaml\n")
	rules := writeFile(t, folder, "rules.yaml", `
plugins: [{name: urban, command: [urban]}]
stages:
  - {name: city, clock: {from: 0, to: 3, step: 1}, rules: [{rule: city, with: {include: scenario.yaml}}]}
`)
	if _, err := Load(rules); err == nil {
		t.Error("a nested include must fail")
	}
}

func TestCheckFindsBadAnchorsAndObjectives(t *testing.T) {
	_, err := Parse([]byte(`
plugins: [{name: urban, command: [urban]}]
anchors:
  - {id: gate}
  - {id: spring, at: [0, 0], outline: [[0, 0], [1, 0], [1, 1]]}
stages:
  - name: city
    clock: {from: 0, to: 3, step: 1}
    objectives:
      - {name: both, at: 1, count: {kind: wall}, event: indicators}
      - {name: no bounds, at: 1, event: indicators, value: population}
      - {name: bad range, at: 1, count: {floors: {maximum: 3}}}
`))
	if err == nil {
		t.Fatal("bad anchors and objectives must fail")
	}
	for _, expected := range []string{"anchor gate", "anchor spring", "objective \"both\"", "objective \"no bounds\"", "objective \"bad range\""} {
		if !strings.Contains(err.Error(), expected) {
			t.Errorf("the error does not mention %s:\n%v", expected, err)
		}
	}
}
