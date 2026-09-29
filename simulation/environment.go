package simulation

import (
	"math"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"
)

// DayCycle turns the sun around the celestial pole; the stars follow it.
type DayCycle struct {
	SecondsPerDay float64 // length of a whole day (0 = 120)
	Enabled       bool
}

// Step turns the sun by the share of the day that passed.
func (cycle *DayCycle) Step(world *scene.Scene, step float64) {
	if !cycle.Enabled {
		return
	}
	environment := &world.Environment
	pole := environment.EffectiveSky().CelestialPole
	turn := graphics.Rotate(pole, 2*math.Pi*step/cycle.secondsPerDay())
	environment.SunDirection = turn.Apply(environment.SunDirection).Normalize()
}

func (cycle *DayCycle) secondsPerDay() float64 {
	if cycle.SecondsPerDay <= 0 {
		return 120
	}
	return cycle.SecondsPerDay
}

// Gusts makes the wind vary around its initial direction and strength.
type Gusts struct {
	Variation float32 // how much the strength changes (0..1)
	Veering   float32 // how far the direction swings, in radians
	elapsed   float64
	base      *scene.Wind
}

// Step changes the wind with smooth, non-repeating waves.
func (gusts *Gusts) Step(world *scene.Scene, step float64) {
	if gusts.base == nil {
		initial := world.Environment.Wind
		gusts.base = &initial
	}
	gusts.elapsed += step
	strength := gusts.base.Strength * (1 + gusts.Variation*float32(smoothWaves(gusts.elapsed*0.23)))
	angle := float64(gusts.Veering) * smoothWaves(gusts.elapsed*0.07+11)
	direction := graphics.RotateY(angle).Apply(gusts.base.Direction).Normalize()
	world.Environment.Wind = scene.Wind{Direction: direction, Strength: float32(math.Max(0, float64(strength)))}
}

// smoothWaves is a sum of incommensurate sines, between -1 and 1.
func smoothWaves(time float64) float64 {
	return (math.Sin(time) + 0.6*math.Sin(time*2.31+1.7) + 0.3*math.Sin(time*5.13+4.2)) / 1.9
}

// CloudDrift moves the clouds with the wind.
type CloudDrift struct {
	Speed float32 // cloud-pattern units per second at wind strength 1 (0 = 0.05)
}

// Step moves the cloud offset along the wind.
func (drift *CloudDrift) Step(world *scene.Scene, step float64) {
	sky := world.Environment.Sky
	if sky == nil {
		return
	}
	speed := drift.Speed
	if speed <= 0 {
		speed = 0.05
	}
	wind := world.Environment.Wind
	distance := speed * (0.2 + wind.Strength) * float32(step)
	sky.Clouds.Offset[0] += wind.Direction[0] * distance
	sky.Clouds.Offset[2] += wind.Direction[2] * distance
}

// Tides raise and lower the water: they turn the angle of its level variation a full turn per
// period. The heights and the phase at every place come with the scene (Water.Variation).
type Tides struct {
	PeriodSeconds float64 // seconds per tidal cycle (0 = 60)
	Enabled       bool
}

// Step advances the tide by the share of the cycle that passed.
func (tides *Tides) Step(world *scene.Scene, step float64) {
	water := world.Environment.Water
	if !tides.Enabled || water == nil || water.Variation == nil {
		return
	}
	period := tides.PeriodSeconds
	if period <= 0 {
		period = 60
	}
	angle := float64(water.Variation.Angle) + 2*math.Pi*step/period
	water.Variation.Angle = float32(math.Mod(angle, 2*math.Pi))
}
