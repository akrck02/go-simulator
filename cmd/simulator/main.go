// Command simulator shows a glTF 2.0 scene with go-renderer's viewer while the plugins of a rules
// file change it.
//
//	go run ./cmd/simulator -rules rules.yaml scene.glb
//	go run ./cmd/simulator -rules rules.yaml -advance 80 -capture city.png scene.glb   (80 steps on, one frame)
//
// Keys, besides the viewer's: P pauses, , and . halve or double the speed of time, and every key
// a rule declares as its toggle_key turns that rule on and off.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/akrck02/go-renderer/loaders"
	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-renderer/viewer"
	"github.com/akrck02/go-simulator/bridge"
	"github.com/akrck02/go-simulator/configuration"
	"github.com/akrck02/go-simulator/host"
	"github.com/akrck02/go-simulator/scenebuilder"
)

func main() {
	options := viewer.DefaultOptions()
	options.RegisterFlags(flag.CommandLine)
	rulesPath := flag.String("rules", "", "rules file with the plugins and rules to run (none = just show the scene)")
	advance := flag.Int("advance", 0, "steps every stepped stage this many times before showing (to start, or capture, further on)")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: simulator", viewer.Usage, "[-rules rules.yaml] [-advance N] scene.glb")
		os.Exit(2)
	}
	if err := run(flag.Arg(0), *rulesPath, *advance, options); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(scenePath, rulesPath string, advance int, options viewer.Options) error {
	loadingStarted := time.Now()
	drawn, err := loaders.LoadScene(scenePath)
	if err != nil {
		return fmt.Errorf("cannot load scene: %w", err)
	}
	running, err := startSession(drawn, rulesPath)
	if err != nil {
		return err
	}
	defer running.close()
	if advance > 0 && running.engine != nil {
		if err := running.advance(advance); err != nil {
			return err
		}
	}
	return viewer.Run(drawn, options, "Simulator · "+scenePath, running.update, time.Since(loadingStarted))
}

func startSession(drawn *scene.Scene, rulesPath string) (*session, error) {
	initial, shown := bridge.FromScene(drawn)
	if rulesPath == "" {
		fmt.Fprintln(os.Stderr, "no rules file: showing the scene without simulating")
		return &session{}, nil
	}
	rules, err := configuration.Load(rulesPath)
	if err != nil {
		return nil, err
	}
	if rules.SceneStyle != "" {
		styled, err := scenebuilder.Load(rules.Resolve(rules.SceneStyle))
		if err != nil {
			return nil, fmt.Errorf("scene style: %w", err)
		}
		shown.UseBuilder(styled)
	}
	engine, err := host.Build(rules, initial, nil)
	if err != nil {
		return nil, err
	}
	running := &session{engine: engine, bridge: shown, keys: toggleKeysOf(engine)}
	started, err := engine.Start()
	if err == nil {
		err = running.show(started)
	}
	if err != nil {
		_ = engine.Close()
		return nil, err
	}
	return running, nil
}
