// Package simulation makes scenes change over time: things that move, the day that passes, the
// wind and the clouds. A Simulation runs a list of systems at a fixed time step; programs call
// Advance from go-renderer's viewer.UpdateFunc (or models.Application.Update), which already run at
// a fixed step. The renderer knows nothing about simulations: it only draws the scene as it is.
//
// Systems change the scene directly: node transforms (drawn as they are every frame), the
// environment (sun, wind, clouds) or geometry and instances, which must then be marked Dirty so
// the backend uploads them again. Heavy per-vertex motion, like plants bending in the wind, is
// done on the GPU from the scene state (Environment.Wind and Material.Sway) instead.
package simulation

import "github.com/akrck02/go-renderer/scene"

// System advances one aspect of a scene by a time step (seconds).
type System interface {
	Step(world *scene.Scene, step float64)
}

// SystemFunc turns a function into a System.
type SystemFunc func(world *scene.Scene, step float64)

// Step calls the function.
func (function SystemFunc) Step(world *scene.Scene, step float64) { function(world, step) }

// Simulation runs systems in order.
type Simulation struct {
	Systems []System
	Speed   float64 // time multiplier (0 = 1)
	Paused  bool
	Time    float64 // simulated seconds since the start
}

// New returns a simulation with the given systems.
func New(systems ...System) *Simulation {
	return &Simulation{Systems: systems}
}

// Add appends a system.
func (simulation *Simulation) Add(system System) {
	simulation.Systems = append(simulation.Systems, system)
}

// Advance runs every system for one step, scaled by Speed; nothing happens while paused.
func (simulation *Simulation) Advance(world *scene.Scene, step float64) {
	if simulation.Paused {
		return
	}
	scaled := step * simulation.speed()
	simulation.Time += scaled
	for _, system := range simulation.Systems {
		system.Step(world, scaled)
	}
}

func (simulation *Simulation) speed() float64 {
	if simulation.Speed <= 0 {
		return 1
	}
	return simulation.Speed
}
