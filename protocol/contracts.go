package protocol

import (
	"fmt"
	"slices"
)

// Contracts: a rule says what it publishes and what it needs from what it reads, so the host can
// check, before the first step, that the plugins of a rules file agree on names and units. The
// names stay free text: a plugin chooses them, and the host only compares them.

// What an output or an input is.
const (
	WhatField       = "field"       // a grid over the ground
	WhatKind        = "kind"        // entities of a kind
	WhatEvent       = "event"       // an announcement
	WhatEnvironment = "environment" // a member of the environment
)

var whats = []string{WhatField, WhatKind, WhatEvent, WhatEnvironment}

// Units are compared as text and never converted: two plugins that write the same thing in
// different units are an error to fix in one of them. "1" is a share or a number without unit,
// "index" a category (its names come in an event), "world" the scene's world units.
var Units = []string{"m", "km", "m²", "km²", "mm", "°C", "m/m", "1", "index", "world", "s", "year"}

// OutputDescription is one thing a rule publishes.
type OutputDescription struct {
	Name       string              `json:"name"`
	What       string              `json:"what"`            // field, kind, event or environment
	Unit       string              `json:"unit,omitempty"`  // one of Units; fields need one
	Names      string              `json:"names,omitempty"` // "index" fields: the event that names their values
	Summary    string              `json:"summary,omitempty"`
	Properties []OutputDescription `json:"properties,omitempty"` // kinds and events: their members (What is left empty)
}

// InputDescription is one thing a rule reads and what it needs from it.
type InputDescription struct {
	Name     string `json:"name"`
	What     string `json:"what"`
	Unit     string `json:"unit,omitempty"`
	Required bool   `json:"required,omitempty"` // the rule cannot work without it
}

// Described tells whether a rule describes what it publishes: a rule of a version 2 plugin that
// publishes nothing describes that too, but one that only lists names in Writes does not.
func (rule RuleDescription) Described() bool {
	return len(rule.Publishes) > 0 || len(rule.Writes) == 0
}

// Check finds what is wrong in the contracts a rule describes.
func (rule RuleDescription) Check() error {
	for _, output := range rule.Publishes {
		if err := output.check(); err != nil {
			return fmt.Errorf("rule %q publishes %w", rule.Name, err)
		}
	}
	for _, input := range rule.Expects {
		if err := checkNameWhatUnit(input.Name, input.What, input.Unit); err != nil {
			return fmt.Errorf("rule %q expects %w", rule.Name, err)
		}
	}
	return nil
}

func (output OutputDescription) check() error {
	if err := checkNameWhatUnit(output.Name, output.What, output.Unit); err != nil {
		return err
	}
	if output.What == WhatField && output.Unit == "" {
		return fmt.Errorf("field %q without a unit (one of %v)", output.Name, Units)
	}
	if len(output.Properties) > 0 && output.What != WhatKind && output.What != WhatEvent {
		return fmt.Errorf("%s %q with properties: only kinds and events have them", output.What, output.Name)
	}
	for _, property := range output.Properties {
		if property.Name == "" {
			return fmt.Errorf("%s %q: a property without a name", output.What, output.Name)
		}
		if property.Unit != "" && !slices.Contains(Units, property.Unit) {
			return fmt.Errorf("%s %q: property %q in unknown unit %q (known: %v)", output.What, output.Name, property.Name, property.Unit, Units)
		}
	}
	return nil
}

func checkNameWhatUnit(name, what, unit string) error {
	switch {
	case name == "":
		return fmt.Errorf("something without a name")
	case !slices.Contains(whats, what):
		return fmt.Errorf("%q as %q, which is not one of %v", name, what, whats)
	case unit != "" && !slices.Contains(Units, unit):
		return fmt.Errorf("%s %q in unknown unit %q (known: %v)", what, name, unit, Units)
	}
	return nil
}
