package main

import (
	"fmt"
	"os"

	"github.com/akrck02/go-renderer/input"
	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-simulator/bridge"
	"github.com/akrck02/go-simulator/host"
	"github.com/akrck02/go-simulator/world"
)

// session is a running simulation drawn by the viewer.
type session struct {
	engine *host.Engine // nil = nothing to simulate
	bridge *bridge.Bridge
	keys   map[string]input.Key // toggle keys the rules declare
	failed bool
}

// update handles the simulator keys and advances the simulation by one viewer step.
func (running *session) update(_ *scene.Scene, state *input.State, step float64) {
	if running.engine == nil || running.failed {
		return
	}
	running.handleKeys(state)
	sets, err := running.engine.Advance(step)
	if err == nil {
		err = running.show(sets)
	}
	if err != nil {
		running.stop(err)
	}
}

func (running *session) show(sets []world.ChangeSet) error {
	for _, set := range sets {
		if err := running.bridge.Apply(set); err != nil {
			return err
		}
	}
	return nil
}

// stop freezes the simulation after an error; the scene stays on screen as it was.
func (running *session) stop(err error) {
	running.failed = true
	fmt.Fprintln(os.Stderr, "simulation stopped:", err)
}

func (running *session) handleKeys(state *input.State) {
	engine := running.engine
	if state.Pressed(input.KeyP) {
		engine.Paused = !engine.Paused
		fmt.Println("paused:", engine.Paused)
	}
	if state.Pressed(input.KeyComma) {
		engine.Speed = currentSpeed(engine) / 2
		fmt.Printf("time speed ×%g\n", engine.Speed)
	}
	if state.Pressed(input.KeyPeriod) {
		engine.Speed = currentSpeed(engine) * 2
		fmt.Printf("time speed ×%g\n", engine.Speed)
	}
	for name, key := range running.keys {
		if state.Pressed(key) {
			reportToggled(engine.Toggle(name))
		}
	}
}

func reportToggled(rules []*host.RuleInstance) {
	for _, rule := range rules {
		fmt.Printf("%s: %v\n", rule.Name, map[bool]string{true: "on", false: "off"}[rule.Enabled])
	}
}

func currentSpeed(engine *host.Engine) float64 {
	if engine.Speed <= 0 {
		return 1
	}
	return engine.Speed
}

func (running *session) close() {
	if running.engine == nil {
		return
	}
	if err := running.engine.Close(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

// toggleKeysOf maps the toggle keys of the rules (letters and digits) to viewer keys.
func toggleKeysOf(engine *host.Engine) map[string]input.Key {
	keys := map[string]input.Key{}
	for _, stage := range engine.Stages {
		for _, rule := range stage.Rules {
			if key, known := keyNamed(rule.ToggleKey); known {
				keys[rule.ToggleKey] = key
			} else if rule.ToggleKey != "" {
				fmt.Fprintf(os.Stderr, "rule %s: toggle_key %q is not a letter or digit; ignored\n", rule.Name, rule.ToggleKey)
			}
		}
	}
	return keys
}

func keyNamed(name string) (input.Key, bool) {
	if len(name) != 1 {
		return input.KeyUnknown, false
	}
	switch character := name[0]; {
	case character >= 'A' && character <= 'Z':
		return input.KeyA + input.Key(character-'A'), true
	case character >= '0' && character <= '9':
		return input.Key0 + input.Key(character-'0'), true
	}
	return input.KeyUnknown, false
}
