package host

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/akrck02/go-simulator/protocol"
)

// The host compares the contracts of the rules a rules file attaches, before the first step (see
// protocol/contracts.go): the same thing published in two units, a rule expecting another unit
// than the one published, or something a rule cannot work without that nobody publishes, are
// errors. What would only show at run time without being wrong comes back as warnings.

// attachedRule is a rule instance with its description, in the order of the rules file.
type attachedRule struct {
	instance    string
	plugin      string
	description protocol.RuleDescription
	described   bool
}

// publisher is a rule that publishes something, with the unit it gives (empty when it does not say).
type publisher struct {
	rule     int // position among the attached rules
	instance string
	unit     string
}

// subject names what is published: a field, a kind or an event, or a property of one of them.
type subject struct {
	what, name, property string
}

func (thing subject) String() string {
	if thing.property != "" {
		return fmt.Sprintf("%s %q property %q", thing.what, thing.name, thing.property)
	}
	return fmt.Sprintf("%s %q", thing.what, thing.name)
}

// checkContracts compares the contracts of the attached rules; it returns the warnings, or an
// error with every disagreement it found.
func (engine *Engine) checkContracts() ([]string, error) {
	rules := engine.attachedRules()
	published := publishedBy(rules)
	var warnings, problems []string
	warnings = append(warnings, undescribed(rules)...)
	problems = append(problems, unitDisagreements(published)...)
	for position, rule := range rules {
		for _, input := range rule.description.Expects {
			problem, warning := engine.checkInput(position, rule, input, published, rules)
			problems = withText(problems, problem)
			warnings = withText(warnings, warning)
		}
		warnings = append(warnings, unnamedIndexes(rule, published)...)
	}
	if len(problems) > 0 {
		return warnings, errors.New("the plugins disagree:\n  " + strings.Join(problems, "\n  "))
	}
	return warnings, nil
}

func (engine *Engine) attachedRules() []attachedRule {
	var rules []attachedRule
	for _, stage := range engine.Stages {
		for _, instance := range stage.Rules {
			description, _ := instance.Plugin.Rule(instance.Rule)
			described := instance.Plugin.Description.Protocol >= 2 && description.Described()
			rules = append(rules, attachedRule{instance: instance.Name, plugin: instance.Plugin.Name, description: description, described: described})
		}
	}
	return rules
}

// publishedBy lists who publishes what. A rule that does not describe its contracts publishes
// the names of its Writes, in no known unit, so that what it writes is not reported missing.
func publishedBy(rules []attachedRule) map[subject][]publisher {
	published := map[subject][]publisher{}
	for position, rule := range rules {
		if !rule.described {
			for _, name := range rule.description.Writes {
				for _, what := range []string{protocol.WhatField, protocol.WhatKind, protocol.WhatEvent, protocol.WhatEnvironment} {
					key := subject{what: what, name: name}
					published[key] = append(published[key], publisher{rule: position, instance: rule.instance})
				}
			}
			continue
		}
		for _, output := range rule.description.Publishes {
			key := subject{what: output.What, name: output.Name}
			published[key] = append(published[key], publisher{rule: position, instance: rule.instance, unit: output.Unit})
			for _, property := range output.Properties {
				key := subject{what: output.What, name: output.Name, property: property.Name}
				published[key] = append(published[key], publisher{rule: position, instance: rule.instance, unit: property.Unit})
			}
		}
	}
	return published
}

func undescribed(rules []attachedRule) []string {
	var plugins []string
	for _, rule := range rules {
		if !rule.described && !slices.Contains(plugins, rule.plugin) {
			plugins = append(plugins, rule.plugin)
		}
	}
	if len(plugins) == 0 {
		return nil
	}
	return []string{fmt.Sprintf("not checked: plugin %s does not describe what its rules publish (protocol 2)", strings.Join(quoted(plugins), ", "))}
}

// unitDisagreements: the same thing published in two units.
func unitDisagreements(published map[subject][]publisher) []string {
	var problems []string
	for thing, publishers := range published {
		var first *publisher
		for index := range publishers {
			if publishers[index].unit == "" {
				continue
			}
			if first == nil {
				first = &publishers[index]
				continue
			}
			if publishers[index].unit != first.unit {
				problems = append(problems, fmt.Sprintf("%s is published in %q by %q and in %q by %q",
					thing, first.unit, first.instance, publishers[index].unit, publishers[index].instance))
				break
			}
		}
	}
	slices.Sort(problems) // maps have no order; the report should
	return problems
}

// checkInput compares what a rule expects with what is published and with the initial world.
func (engine *Engine) checkInput(position int, rule attachedRule, input protocol.InputDescription, published map[subject][]publisher,
	rules []attachedRule) (problem, warning string) {
	publishers := published[subject{what: input.What, name: input.Name}]
	inWorld := engine.initiallyHas(input)
	if len(publishers) == 0 {
		if input.Required && !inWorld {
			return fmt.Sprintf("%q needs %s %q, which no rule publishes and the world does not have", rule.instance, input.What, input.Name), ""
		}
		return "", ""
	}
	if input.Unit != "" {
		for _, other := range publishers {
			if other.unit != "" && other.unit != input.Unit {
				return fmt.Sprintf("%q expects %s %q in %q, but %q publishes it in %q", rule.instance, input.What, input.Name, input.Unit,
					other.instance, other.unit), ""
			}
		}
	}
	if !inWorld && publishers[0].rule > position {
		first := rules[publishers[0].rule]
		return "", fmt.Sprintf("%q reads %s %q, which only %q publishes, after it: its first step will not have it", rule.instance,
			input.What, input.Name, first.instance)
	}
	return "", ""
}

// initiallyHas tells whether the world has something before any plugin starts.
func (engine *Engine) initiallyHas(input protocol.InputDescription) bool {
	switch input.What {
	case protocol.WhatField:
		return engine.World.Field(input.Name) != nil
	case protocol.WhatKind:
		return len(engine.World.EntitiesOfKind(input.Name)) > 0
	}
	return false
}

// unnamedIndexes: category fields whose names nobody announces.
func unnamedIndexes(rule attachedRule, published map[subject][]publisher) []string {
	var warnings []string
	for _, output := range rule.description.Publishes {
		if output.What != protocol.WhatField || output.Unit != "index" {
			continue
		}
		if output.Names == "" {
			warnings = append(warnings, fmt.Sprintf("%q publishes the categories of field %q without the event that names them", rule.instance, output.Name))
			continue
		}
		if len(published[subject{what: protocol.WhatEvent, name: output.Names}]) == 0 {
			warnings = append(warnings, fmt.Sprintf("%q names the categories of field %q in event %q, which no rule publishes", rule.instance, output.Name, output.Names))
		}
	}
	return warnings
}

func withText(list []string, text string) []string {
	if text == "" {
		return list
	}
	return append(list, text)
}

func quoted(names []string) []string {
	result := make([]string, 0, len(names))
	for _, name := range names {
		result = append(result, fmt.Sprintf("%q", name))
	}
	return result
}
