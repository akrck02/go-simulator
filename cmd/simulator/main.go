// Command simulator shows a glTF 2.0 scene with go-renderer's viewer while a simulation changes it:
// the motions its nodes ask for in their extras (bob, spin, path), wind gusts, drifting clouds, the
// tides (when the scene's water has a level variation) and the passing of the day.
//
//	go run ./cmd/simulator scene.glb
//	go run ./cmd/simulator -day-length 300 -day scene.glb
//
// Keys, besides the viewer's: P pauses, T starts or stops the day cycle, M starts or stops the
// tides, , and . halve or double the speed of time.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/akrck02/go-renderer/input"
	"github.com/akrck02/go-renderer/loaders"
	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-renderer/viewer"
	"github.com/akrck02/go-simulator/simulation"
)

// settings are the simulator's own options.
type settings struct {
	dayLength     float64
	dayCycle      bool
	windVariation float64
	windVeering   float64
	tidePeriod    float64
	noTides       bool
}

func main() {
	options := viewer.DefaultOptions()
	options.RegisterFlags(flag.CommandLine)
	own := registerSettings(flag.CommandLine)
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: simulator", viewer.Usage, "[-day-length S] [-day] [-wind-variation F] [-wind-veering R] [-tide-period S] [-no-tides] scene.glb")
		os.Exit(2)
	}
	scenePath := flag.Arg(0)
	loadingStarted := time.Now()
	world, err := loaders.LoadScene(scenePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot load scene:", err)
		os.Exit(1)
	}
	session := newSession(world, own)
	if err := viewer.Run(world, options, "Simulator · "+scenePath, session.update, time.Since(loadingStarted)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func registerSettings(flags *flag.FlagSet) *settings {
	own := &settings{}
	flags.Float64Var(&own.dayLength, "day-length", 120, "seconds a whole day lasts")
	flags.BoolVar(&own.dayCycle, "day", false, "start with the day cycle running")
	flags.Float64Var(&own.windVariation, "wind-variation", 0.5, "how much the wind strength changes in gusts (0..1)")
	flags.Float64Var(&own.windVeering, "wind-veering", 0.35, "how far the wind direction swings, in radians")
	flags.Float64Var(&own.tidePeriod, "tide-period", 0, "seconds per tidal cycle (0 = 12.42 h in proportion to the day length)")
	flags.BoolVar(&own.noTides, "no-tides", false, "start with the tides stopped")
	return own
}

// session is the running simulation and the keys that control it.
type session struct {
	simulation *simulation.Simulation
	dayCycle   *simulation.DayCycle
	tides      *simulation.Tides
}

func newSession(world *scene.Scene, own *settings) *session {
	dayCycle := &simulation.DayCycle{SecondsPerDay: own.dayLength, Enabled: own.dayCycle}
	running := simulation.New(simulation.FromTags(world)...)
	running.Add(&simulation.Gusts{Variation: float32(own.windVariation), Veering: float32(own.windVeering)})
	running.Add(&simulation.CloudDrift{})
	running.Add(dayCycle)
	tides := &simulation.Tides{PeriodSeconds: tidePeriodFor(own), Enabled: !own.noTides}
	running.Add(tides)
	return &session{simulation: running, dayCycle: dayCycle, tides: tides}
}

// hoursPerTide is the semidiurnal tidal period in hours; the tide keeps that proportion to the day.
const hoursPerTide = 12.42

func tidePeriodFor(own *settings) float64 {
	if own.tidePeriod > 0 {
		return own.tidePeriod
	}
	return own.dayLength * hoursPerTide / 24
}

// update handles the simulator keys and advances the simulation by one fixed step.
func (current *session) update(world *scene.Scene, state *input.State, step float64) {
	current.handleKeys(state)
	current.simulation.Advance(world, step)
}

func (current *session) handleKeys(state *input.State) {
	running := current.simulation
	if state.Pressed(input.KeyP) {
		running.Paused = !running.Paused
	}
	if state.Pressed(input.KeyT) {
		current.dayCycle.Enabled = !current.dayCycle.Enabled
	}
	if state.Pressed(input.KeyM) {
		current.tides.Enabled = !current.tides.Enabled
	}
	if state.Pressed(input.KeyComma) {
		running.Speed = currentSpeed(running) / 2
		fmt.Printf("time speed ×%g\n", running.Speed)
	}
	if state.Pressed(input.KeyPeriod) {
		running.Speed = currentSpeed(running) * 2
		fmt.Printf("time speed ×%g\n", running.Speed)
	}
}

func currentSpeed(running *simulation.Simulation) float64 {
	if running.Speed <= 0 {
		return 1
	}
	return running.Speed
}
