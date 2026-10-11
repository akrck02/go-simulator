package configuration

import (
	"fmt"

	"github.com/akrck02/go-simulator/conditions"
)

// Anchor is a place the rules file fixes because the story or a decision says it is there: a gate,
// a spring, an inn, an old road. The host adds it to the world before the simulation starts, as a
// hidden entity with the property "anchor": true, and plugins build around it, never over it.
// It is a point, a line or an outline in ground coordinates (x, z), or a GeoJSON file with one
// anchor per feature (the feature's properties "id", "kind" and "role" are used when present).
//
//	anchors:
//	  - {id: main_gate, role: gate, at: [-2.5, 1.2], source: "chapter 3 of the story"}
//	  - {id: spring, role: obstacle, outline: [[x, z], …]}
//	  - {from: anchors.geojson}
type Anchor struct {
	Identifier string         `yaml:"id"`
	Kind       string         `yaml:"kind"` // default "anchor"
	Role       string         `yaml:"role"` // what it means to the plugins: "gate", "obstacle", "road"…
	At         *[2]float32    `yaml:"at"`
	Line       [][2]float32   `yaml:"line"`
	Width      float32        `yaml:"width"`
	Outline    [][2]float32   `yaml:"outline"`
	From       string         `yaml:"from"`
	Properties map[string]any `yaml:"properties"`
	Source     string         `yaml:"source"` // where it comes from, for people: a text, a decision
}

// Objective is something a run must reach: what the story or a decision fixes about the result.
// Runs without a window check every objective when its stage reaches its time and report the
// ones that fail. It either counts the entities a condition matches, or reads a value from the
// last event of a name:
//
//	objectives:
//	  - name: walled by year 60
//	    at: 60
//	    count: {kind: wall}            # at least one, unless min, max or near say otherwise
//	  - name: about 6,000 people in year 100
//	    at: 100
//	    event: indicators
//	    value: population
//	    near: 6000
//	    tolerance: 0.05                # ±5 %
type Objective struct {
	Name      string         `yaml:"name"`
	At        float64        `yaml:"at"` // stage time
	Count     map[string]any `yaml:"count"`
	Event     string         `yaml:"event"`
	Value     string         `yaml:"value"` // name of the value in the event's data
	Minimum   *float64       `yaml:"min"`
	Maximum   *float64       `yaml:"max"`
	Near      *float64       `yaml:"near"`
	Tolerance float64        `yaml:"tolerance"` // share of near (0.05 = ±5 %)
	Source    string         `yaml:"source"`
}

// Counts tells whether the objective counts entities (otherwise it reads an event).
func (objective Objective) Counts() bool { return objective.Count != nil }

func checkAnchor(anchor Anchor, index int) []error {
	label := anchor.Identifier
	if label == "" {
		label = fmt.Sprintf("number %d", index+1)
	}
	ways := 0
	for _, given := range []bool{anchor.At != nil, len(anchor.Line) > 0, len(anchor.Outline) > 0, anchor.From != ""} {
		if given {
			ways++
		}
	}
	var problems []error
	if ways != 1 {
		problems = append(problems, fmt.Errorf("anchor %s: give exactly one of at, line, outline or from", label))
	}
	if anchor.From == "" && anchor.Identifier == "" {
		problems = append(problems, fmt.Errorf("anchor %s: needs an id", label))
	}
	if len(anchor.Line) == 1 || (len(anchor.Outline) > 0 && len(anchor.Outline) < 3) {
		problems = append(problems, fmt.Errorf("anchor %s: a line needs 2 points and an outline 3", label))
	}
	return problems
}

func checkObjective(stage string, objective Objective) []error {
	label := fmt.Sprintf("stage %q, objective %q", stage, objective.Name)
	var problems []error
	if objective.Name == "" {
		problems = append(problems, fmt.Errorf("stage %q: an objective has no name", stage))
	}
	if objective.Counts() == (objective.Event != "") {
		problems = append(problems, fmt.Errorf("%s: give either count or event", label))
	}
	if objective.Event != "" && objective.Value == "" {
		problems = append(problems, fmt.Errorf("%s: an event objective needs the value it reads", label))
	}
	if objective.Event != "" && objective.Minimum == nil && objective.Maximum == nil && objective.Near == nil {
		problems = append(problems, fmt.Errorf("%s: give min, max or near", label))
	}
	if objective.Near != nil && objective.Tolerance < 0 {
		problems = append(problems, fmt.Errorf("%s: tolerance cannot be negative", label))
	}
	if err := conditions.Check(objective.Count, nil); err != nil {
		problems = append(problems, fmt.Errorf("%s: %w", label, err))
	}
	return problems
}
