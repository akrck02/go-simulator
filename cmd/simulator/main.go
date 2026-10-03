// Command simulator shows a glTF 2.0 scene with go-renderer's viewer while the plugins of a rules
// file change it.
//
//	go run ./cmd/simulator -rules rules.yaml scene.glb
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
)

func main() {
	options := viewer.DefaultOptions()
	options.RegisterFlags(flag.CommandLine)
	rulesPath := flag.String("rules", "", "rules file with the plugins and rules to run (none = just show the scene)")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: simulator", viewer.Usage, "[-rules rules.yaml] scene.glb")
		os.Exit(2)
	}
	if err := run(flag.Arg(0), *rulesPath, options); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(scenePath, rulesPath string, options viewer.Options) error {
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
