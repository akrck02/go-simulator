// Package plugin is the toolkit for writing a simulator plugin in Go. A plugin is a program that
// offers rules; the host starts it, attaches rules with the parameters of a rules file and asks it
// to step them. Plugins never touch a renderer: they read their copy of the world and describe
// what changes.
//
//	func main() {
//		motion := &plugin.Plugin{Name: "motion", Version: "1.0.0", Rules: []plugin.Definition{dayCycle}}
//		if err := motion.Serve(); err != nil {
//			log.Fatal(err)
//		}
//	}
//
// Logs go to standard error: standard output carries the protocol.
package plugin

import (
	"github.com/akrck02/go-simulator/protocol"
	"github.com/akrck02/go-simulator/world"
)

// Plugin is a set of rules with a name and a version.
type Plugin struct {
	Name    string
	Version string
	Summary string
	Rules   []Definition
	// AfterStep, when set, runs once after the rules of every start and step request, with the
	// same context (Instance is empty). Plugins whose rules share state use it to describe the
	// result once instead of after every rule.
	AfterStep func(context *Context) error
}

// Definition describes a rule and builds an instance of it from its parameters.
type Definition struct {
	Description protocol.RuleDescription
	New         func(parameters Parameters) (Rule, error)
}

// Rule is one use of a rule. Step runs every time the host steps it; it reads context.World and
// describes its changes through the context.
type Rule interface {
	Step(context *Context) error
}

// Starter is a rule that also does something once, when the simulation starts.
type Starter interface {
	Start(context *Context) error
}

// describe returns what the host needs to know about the plugin.
func (plugin *Plugin) describe() protocol.PluginDescription {
	description := protocol.PluginDescription{Protocol: protocol.Version, Name: plugin.Name, Version: plugin.Version, Summary: plugin.Summary}
	for _, definition := range plugin.Rules {
		description.Rules = append(description.Rules, definition.Description)
	}
	return description
}

func (plugin *Plugin) definition(rule string) (Definition, bool) {
	for _, definition := range plugin.Rules {
		if definition.Description.Name == rule {
			return definition, true
		}
	}
	return Definition{}, false
}

// Context is what a rule sees during a step: the plugin's copy of the world and the time. The
// changes a rule makes are applied to that copy at once, so the rule and the rules after it see
// them, and are sent to the host at the end of the step.
type Context struct {
	World    *world.World
	Stage    string
	Time     float64 // simulated time after the step
	Step     float64 // time the step covers
	Instance string  // name of the rule instance being run
	changes  []world.Change
	failure  error
}

// Add adds an entity.
func (context *Context) Add(entity *world.Entity) { context.record(world.AddEntity(entity)) }

// Remove removes an entity and everything attached to it.
func (context *Context) Remove(identifier world.Identifier) {
	context.record(world.RemoveEntity(identifier))
}

// Modify patches an entity: set members replace, properties merge (nil removes one).
func (context *Context) Modify(patch *world.Entity) { context.record(world.ModifyEntity(patch)) }

// SetField creates a field or sets some of its cells.
func (context *Context) SetField(patch *world.FieldPatch) {
	context.record(world.SetFieldValues(patch))
}

// SetEnvironment patches the environment.
func (context *Context) SetEnvironment(patch *world.Environment) {
	context.record(world.ChangeEnvironment(patch))
}

// ClearSceneArea hides what some nodes of the drawn scene show in an area (see world.Area).
func (context *Context) ClearSceneArea(area *world.Area) {
	context.record(world.ClearSceneArea(area))
}

// Emit tells that something happened.
func (context *Context) Emit(name string, data map[string]any) {
	context.record(world.EmitEvent(name, data))
}

func (context *Context) record(change world.Change) {
	if context.failure != nil {
		return
	}
	if err := context.World.ApplyChange(change); err != nil {
		context.failure = err
		return
	}
	context.changes = append(context.changes, change)
}
