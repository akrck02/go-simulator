package host

import (
	"errors"
	"fmt"

	"github.com/akrck02/go-simulator/configuration"
	"github.com/akrck02/go-simulator/world"
)

// Engine runs the stages of a simulation over one world.
type Engine struct {
	World   *world.World
	Plugins []*Plugin
	Stages  []*Stage
	Speed   float64 // multiplies the time of real-time stages (0 = 1)
	Paused  bool    // real-time stages do not advance while paused
	// Warnings are what the contracts of the plugins tell before the first step without being
	// wrong (see contracts.go); whoever drives the engine shows them.
	Warnings []string
	started  bool
	// lastEvents holds the last event of every name, for objectives that read them.
	lastEvents map[string]world.Event
}

// Stage is a group of rules sharing a clock.
type Stage struct {
	Name      string
	Clock     Clock
	Rules     []*RuleInstance
	Time      float64   // current time in the clock's unit
	Snapshots []float64 // times at which a run without a window saves the world
	// SnapshotFields are the fields the snapshots save (none: every field).
	SnapshotFields []string
	// Objectives are what the run must reach (see CheckObjective).
	Objectives []configuration.Objective
	waited     float64 // real seconds since the last paced step
}

// Clock says how a stage's time advances. A real-time clock advances with the seconds that pass;
// a stepped clock advances by Step from From to To, one step each time it is asked to or, with a
// Pace, one step every Pace real seconds while the simulation is shown.
type Clock struct {
	Realtime bool
	Unit     string // "second", "year", "step"…: only for people and plugins to read
	From     float64
	To       float64
	Step     float64
	Pace     float64
}

// RuleInstance is one use of a rule in a stage.
type RuleInstance struct {
	Name      string // unique in the simulation; the plugin knows the instance by it
	Plugin    *Plugin
	Rule      string
	Enabled   bool
	Every     float64  // run when this much stage time passed since its last run (0 = every step)
	At        *float64 // run once, when the stage time reaches it
	ToggleKey string   // key that turns it on and off in an interactive simulation
	lastRun   float64
	hasRun    bool
}

// Start gives every plugin its part of the world and applies what their rules do when they start.
func (engine *Engine) Start() ([]world.ChangeSet, error) {
	if engine.started {
		return nil, errors.New("the simulation already started")
	}
	engine.started = true
	for _, stage := range engine.Stages {
		stage.Time = stage.Clock.From
	}
	var sets []world.ChangeSet
	for _, plugin := range engine.Plugins {
		snapshot := engine.World.Snapshot(plugin.interest)
		snapshot.Changes = append(snapshot.Changes, engine.lastEventsFor(plugin.interest)...)
		set, err := plugin.Start(snapshot)
		if err != nil {
			return sets, err
		}
		if err := engine.apply(plugin, &set); err != nil {
			return sets, err
		}
		sets = appendIfAny(sets, set)
	}
	return sets, nil
}

// Advance moves every real-time stage forward by some seconds, scaled by Speed, and gives paced
// stepped stages the steps those seconds are worth.
func (engine *Engine) Advance(seconds float64) ([]world.ChangeSet, error) {
	if engine.Paused {
		return nil, nil
	}
	scaled := seconds * engine.speed()
	var sets []world.ChangeSet
	for _, stage := range engine.Stages {
		stepped, err := engine.advanceStage(stage, scaled)
		sets = append(sets, stepped...)
		if err != nil {
			return sets, err
		}
	}
	return sets, nil
}

func (engine *Engine) advanceStage(stage *Stage, seconds float64) ([]world.ChangeSet, error) {
	if stage.Clock.Realtime {
		return engine.runStage(stage, seconds)
	}
	if stage.Clock.Pace <= 0 {
		return nil, nil
	}
	var sets []world.ChangeSet
	for stage.waited += seconds; stage.waited >= stage.Clock.Pace && !stage.Finished(); stage.waited -= stage.Clock.Pace {
		stepped, err := engine.runStage(stage, stage.Clock.Step)
		sets = append(sets, stepped...)
		if err != nil {
			return sets, err
		}
	}
	return sets, nil
}

// StepStage moves a stepped stage forward by one step of its clock.
func (engine *Engine) StepStage(name string) ([]world.ChangeSet, error) {
	stage := engine.Stage(name)
	if stage == nil {
		return nil, fmt.Errorf("no stage %q", name)
	}
	if stage.Clock.Realtime {
		return nil, fmt.Errorf("stage %q runs in real time: use Advance", name)
	}
	if stage.Finished() {
		return nil, nil
	}
	return engine.runStage(stage, stage.Clock.Step)
}

// Finished tells whether a stepped stage reached the end of its clock.
func (stage *Stage) Finished() bool {
	return !stage.Clock.Realtime && stage.Clock.Step > 0 && stage.Time >= stage.Clock.To
}

// Stage returns a stage by name, or nil.
func (engine *Engine) Stage(name string) *Stage {
	for _, stage := range engine.Stages {
		if stage.Name == name {
			return stage
		}
	}
	return nil
}

// Toggle turns on or off the rule instances bound to a key; it returns the ones it changed.
func (engine *Engine) Toggle(key string) []*RuleInstance {
	var toggled []*RuleInstance
	for _, stage := range engine.Stages {
		for _, rule := range stage.Rules {
			if rule.ToggleKey != "" && rule.ToggleKey == key {
				rule.Enabled = !rule.Enabled
				toggled = append(toggled, rule)
			}
		}
	}
	return toggled
}

// Close stops every plugin.
func (engine *Engine) Close() error {
	var failures []error
	for _, plugin := range engine.Plugins {
		if err := plugin.Close(); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

// runStage advances a stage's time and steps the rules that are due, in the order of the rules
// file. Consecutive rules of the same plugin go in one request.
func (engine *Engine) runStage(stage *Stage, step float64) ([]world.ChangeSet, error) {
	stage.Time += step
	var sets []world.ChangeSet
	for _, batch := range dueBatches(stage) {
		set, err := batch.plugin.Step(stage.Name, stage.Time, step, batch.instances)
		if err != nil {
			return sets, err
		}
		if err := engine.apply(batch.plugin, &set); err != nil {
			return sets, err
		}
		sets = appendIfAny(sets, set)
	}
	return sets, nil
}

// batch is a run of consecutive due rules that belong to the same plugin.
type batch struct {
	plugin    *Plugin
	instances []string
}

func dueBatches(stage *Stage) []batch {
	var batches []batch
	for _, rule := range stage.Rules {
		if !rule.isDue(stage.Time) {
			continue
		}
		rule.lastRun, rule.hasRun = stage.Time, true
		if count := len(batches); count > 0 && batches[count-1].plugin == rule.Plugin {
			batches[count-1].instances = append(batches[count-1].instances, rule.Name)
			continue
		}
		batches = append(batches, batch{plugin: rule.Plugin, instances: []string{rule.Name}})
	}
	return batches
}

func (rule *RuleInstance) isDue(time float64) bool {
	if !rule.Enabled {
		return false
	}
	if rule.At != nil {
		return !rule.hasRun && time >= *rule.At
	}
	if rule.Every > 0 && rule.hasRun {
		return time-rule.lastRun >= rule.Every-1e-9
	}
	return true
}

// apply applies a plugin's changes to the world one by one, and queues for every other plugin
// the ones that concern it (judged on the world as it was just before each change).
func (engine *Engine) apply(origin *Plugin, set *world.ChangeSet) error {
	set.Origin = origin.Name
	for index, change := range set.Changes {
		engine.forward(origin, change)
		engine.rememberEvent(change)
		if err := engine.World.ApplyChange(change); err != nil {
			return fmt.Errorf("change %d (%s) from plugin %q: %w", index, change.Operation, origin.Name, err)
		}
	}
	if set.Time > engine.World.Time {
		engine.World.Time = set.Time
	}
	return nil
}

func (engine *Engine) forward(origin *Plugin, change world.Change) {
	for _, plugin := range engine.Plugins {
		if plugin != origin && plugin.interest.Covers(change, engine.World) {
			plugin.pending = append(plugin.pending, change)
		}
	}
}

func (engine *Engine) speed() float64 {
	if engine.Speed <= 0 {
		return 1
	}
	return engine.Speed
}

func appendIfAny(sets []world.ChangeSet, set world.ChangeSet) []world.ChangeSet {
	if len(set.Changes) == 0 {
		return sets
	}
	return append(sets, set)
}
