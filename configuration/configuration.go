// Package configuration reads rules files: which plugins a simulation uses, its stages and, in
// each stage, the rules with their parameters and schedule.
//
//	plugins:
//	  - name: motion
//	    command: [go-simulator-motion]
//	stages:
//	  - name: motion
//	    clock: realtime
//	    rules:
//	      - rule: day_cycle
//	        with: {seconds_per_day: 120}
//	        enabled: false
//	        toggle_key: T
//	  - name: city
//	    clock: {unit: year, from: 0, to: 100, step: 1}
//	    rules:
//	      - rule: urban/prune_empty_streets
//	        every: 5
package configuration

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Configuration is a whole rules file.
type Configuration struct {
	Plugins   []Plugin `yaml:"plugins"`
	Stages    []Stage  `yaml:"stages"`
	Directory string   `yaml:"-"` // folder of the file; relative paths start there
}

// Plugin is a plugin program the simulation uses.
type Plugin struct {
	Name      string   `yaml:"name"`
	Command   []string `yaml:"command"`   // program and arguments; a relative program path starts at the file's folder
	Directory string   `yaml:"directory"` // where the program runs (default: the file's folder)
	Builtin   bool     `yaml:"builtin"`   // a plugin the host program carries inside, found by name
}

// Stage is a group of rules sharing a clock.
type Stage struct {
	Name  string `yaml:"name"`
	Clock Clock  `yaml:"clock"`
	Rules []Rule `yaml:"rules"`
}

// Clock is "realtime" or a stepped clock {unit, from, to, step}.
type Clock struct {
	Realtime bool    `yaml:"realtime"`
	Unit     string  `yaml:"unit"`
	From     float64 `yaml:"from"`
	To       float64 `yaml:"to"`
	Step     float64 `yaml:"step"`
}

// Rule is one use of a rule.
type Rule struct {
	Rule      string         `yaml:"rule"` // "name", or "plugin/name" when several plugins offer it
	Name      string         `yaml:"name"` // instance name (default: stage/rule)
	With      map[string]any `yaml:"with"` // parameters
	Enabled   *bool          `yaml:"enabled"`
	Every     float64        `yaml:"every"`
	At        *float64       `yaml:"at"`
	ToggleKey string         `yaml:"toggle_key"`
}

// UnmarshalYAML accepts "clock: realtime" as well as the full form.
func (clock *Clock) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		if node.Value != "realtime" {
			return fmt.Errorf("line %d: a clock is \"realtime\" or {unit, from, to, step}", node.Line)
		}
		clock.Realtime, clock.Unit = true, "second"
		return nil
	}
	type plainClock Clock
	return node.Decode((*plainClock)(clock))
}

// Load reads a rules file.
func Load(path string) (*Configuration, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	configuration, err := Parse(contents)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	configuration.Directory = filepath.Dir(absolute)
	return configuration, nil
}

// Parse reads a rules file's contents and checks them.
func Parse(contents []byte) (*Configuration, error) {
	var configuration Configuration
	if err := yaml.Unmarshal(contents, &configuration); err != nil {
		return nil, err
	}
	return &configuration, configuration.Check()
}

// Check finds mistakes that would only show up later.
func (configuration *Configuration) Check() error {
	var problems []error
	plugins := map[string]bool{}
	for _, plugin := range configuration.Plugins {
		problems = append(problems, checkPlugin(plugin, plugins)...)
		plugins[plugin.Name] = true
	}
	stages := map[string]bool{}
	for _, stage := range configuration.Stages {
		problems = append(problems, checkStage(stage, stages)...)
		stages[stage.Name] = true
	}
	return errors.Join(problems...)
}

func checkPlugin(plugin Plugin, seen map[string]bool) []error {
	var problems []error
	if plugin.Name == "" {
		problems = append(problems, errors.New("a plugin has no name"))
	}
	if seen[plugin.Name] {
		problems = append(problems, fmt.Errorf("plugin %q appears twice", plugin.Name))
	}
	if !plugin.Builtin && len(plugin.Command) == 0 {
		problems = append(problems, fmt.Errorf("plugin %q has no command", plugin.Name))
	}
	return problems
}

func checkStage(stage Stage, seen map[string]bool) []error {
	var problems []error
	if stage.Name == "" || seen[stage.Name] {
		problems = append(problems, fmt.Errorf("stage %q: stages need a unique name", stage.Name))
	}
	if !stage.Clock.Realtime && (stage.Clock.Step <= 0 || stage.Clock.To < stage.Clock.From) {
		problems = append(problems, fmt.Errorf("stage %q: a stepped clock needs a positive step and to ≥ from", stage.Name))
	}
	for index, rule := range stage.Rules {
		if rule.Rule == "" {
			problems = append(problems, fmt.Errorf("stage %q, rule %d: missing \"rule\"", stage.Name, index+1))
		}
		if rule.Every < 0 {
			problems = append(problems, fmt.Errorf("stage %q, rule %q: \"every\" cannot be negative", stage.Name, rule.Rule))
		}
	}
	return problems
}

// CommandPath resolves a plugin's program: a relative path with a folder in it starts at the
// rules file's folder; a bare name is looked up in the PATH as usual.
func (configuration *Configuration) CommandPath(plugin Plugin) []string {
	command := append([]string(nil), plugin.Command...)
	if len(command) > 0 && !filepath.IsAbs(command[0]) && filepath.Base(command[0]) != command[0] {
		command[0] = filepath.Join(configuration.Directory, command[0])
	}
	return command
}

// WorkingDirectory is where a plugin's program runs.
func (configuration *Configuration) WorkingDirectory(plugin Plugin) string {
	if plugin.Directory == "" {
		return configuration.Directory
	}
	if filepath.IsAbs(plugin.Directory) {
		return plugin.Directory
	}
	return filepath.Join(configuration.Directory, plugin.Directory)
}
