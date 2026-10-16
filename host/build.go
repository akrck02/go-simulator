package host

import (
	"errors"
	"fmt"
	"strings"

	"github.com/akrck02/go-simulator/configuration"
	"github.com/akrck02/go-simulator/protocol"
	"github.com/akrck02/go-simulator/world"
)

// Builtin is a plugin the host program carries inside; rules files name it with "builtin: true".
type Builtin func(*protocol.Connection) error

// Build starts the plugins of a rules file, attaches its rules, checks that their contracts agree
// (see contracts.go) and returns the engine, ready to Start. The world is the initial state (for example, built from a scene); nil means empty.
func Build(rules *configuration.Configuration, initial *world.World, builtins map[string]Builtin) (*Engine, error) {
	if initial == nil {
		initial = world.New()
	}
	if err := AddAnchors(rules, initial); err != nil {
		return nil, err
	}
	engine := &Engine{World: initial}
	if err := engine.launchPlugins(rules, builtins); err != nil {
		_ = engine.Close()
		return nil, err
	}
	if err := engine.attachStages(rules); err != nil {
		_ = engine.Close()
		return nil, err
	}
	warnings, err := engine.checkContracts()
	engine.Warnings = warnings
	if err != nil {
		_ = engine.Close()
		return nil, err
	}
	return engine, nil
}

func (engine *Engine) launchPlugins(rules *configuration.Configuration, builtins map[string]Builtin) error {
	for _, settings := range rules.Plugins {
		plugin, err := launchPlugin(rules, settings, builtins)
		if err != nil {
			return err
		}
		engine.Plugins = append(engine.Plugins, plugin)
	}
	return nil
}

func launchPlugin(rules *configuration.Configuration, settings configuration.Plugin, builtins map[string]Builtin) (*Plugin, error) {
	if !settings.Builtin {
		return Launch(settings.Name, rules.CommandPath(settings), rules.WorkingDirectory(settings), rules.Directory)
	}
	serve, found := builtins[settings.Name]
	if !found {
		return nil, fmt.Errorf("this program has no built-in plugin %q", settings.Name)
	}
	return Connect(settings.Name, serve)
}

func (engine *Engine) attachStages(rules *configuration.Configuration) error {
	names := map[string]bool{}
	for _, settings := range rules.Stages {
		stage := &Stage{Name: settings.Name, Clock: Clock(settings.Clock), Snapshots: settings.Snapshots, SnapshotFields: settings.SnapshotFields,
			Objectives: settings.Objectives}
		for _, ruleSettings := range settings.Rules {
			rule, err := engine.attachRule(stage, ruleSettings, names)
			if err != nil {
				return fmt.Errorf("stage %q: %w", stage.Name, err)
			}
			stage.Rules = append(stage.Rules, rule)
		}
		engine.Stages = append(engine.Stages, stage)
	}
	return nil
}

func (engine *Engine) attachRule(stage *Stage, settings configuration.Rule, names map[string]bool) (*RuleInstance, error) {
	plugin, ruleName, err := engine.findRule(settings.Rule)
	if err != nil {
		return nil, err
	}
	instance := uniqueName(instanceName(stage, settings, ruleName), names)
	if err := plugin.Attach(instance, ruleName, settings.With); err != nil {
		return nil, err
	}
	return &RuleInstance{Name: instance, Plugin: plugin, Rule: ruleName, Enabled: settings.Enabled == nil || *settings.Enabled,
		Every: settings.Every, At: settings.At, ToggleKey: strings.ToUpper(settings.ToggleKey)}, nil
}

// findRule finds the plugin that offers a rule, written "rule" or "plugin/rule".
func (engine *Engine) findRule(written string) (*Plugin, string, error) {
	if pluginName, ruleName, qualified := strings.Cut(written, "/"); qualified {
		for _, plugin := range engine.Plugins {
			if plugin.Name == pluginName {
				return plugin, ruleName, nil
			}
		}
		return nil, "", fmt.Errorf("rule %q: no plugin %q", written, pluginName)
	}
	var offering []*Plugin
	for _, plugin := range engine.Plugins {
		if _, offers := plugin.Rule(written); offers {
			offering = append(offering, plugin)
		}
	}
	switch len(offering) {
	case 0:
		return nil, "", fmt.Errorf("no plugin offers rule %q", written)
	case 1:
		return offering[0], written, nil
	}
	return nil, "", errors.New(ambiguousRule(written, offering))
}

func ambiguousRule(rule string, offering []*Plugin) string {
	names := make([]string, 0, len(offering))
	for _, plugin := range offering {
		names = append(names, plugin.Name+"/"+rule)
	}
	return fmt.Sprintf("several plugins offer rule %q: write one of %s", rule, strings.Join(names, ", "))
}

func instanceName(stage *Stage, settings configuration.Rule, rule string) string {
	if settings.Name != "" {
		return settings.Name
	}
	return stage.Name + "/" + rule
}

func uniqueName(name string, taken map[string]bool) string {
	unique := name
	for number := 2; taken[unique]; number++ {
		unique = fmt.Sprintf("%s#%d", name, number)
	}
	taken[unique] = true
	return unique
}
