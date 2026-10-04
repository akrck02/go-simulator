// Command simulate runs the stepped stages of a rules file without a window, to the end of their
// clocks, and saves the world at the times each stage lists in "snapshots".
//
//	go run ./cmd/simulate -rules rules.yaml -output results/
//	go run ./cmd/simulate -rules rules.yaml -scene scene.glb -output results/
//
// It writes, in the output folder, one <stage> <time>.geojson per snapshot and events.jsonl with
// every event the plugins emitted. Real-time stages do not run.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/akrck02/go-renderer/loaders"
	"github.com/akrck02/go-simulator/bridge"
	"github.com/akrck02/go-simulator/configuration"
	"github.com/akrck02/go-simulator/host"
	"github.com/akrck02/go-simulator/snapshot"
	"github.com/akrck02/go-simulator/world"
)

func main() {
	rulesPath := flag.String("rules", "", "rules file")
	scenePath := flag.String("scene", "", "glTF scene the world starts from (optional)")
	output := flag.String("output", "", "folder for the snapshots and the events")
	flag.Parse()
	if *rulesPath == "" || *output == "" {
		fmt.Fprintln(os.Stderr, "usage: simulate -rules rules.yaml [-scene scene.glb] -output folder")
		os.Exit(2)
	}
	if err := run(*rulesPath, *scenePath, *output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(rulesPath, scenePath, output string) error {
	rules, err := configuration.Load(rulesPath)
	if err != nil {
		return err
	}
	initial, err := initialWorld(scenePath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(output, 0o755); err != nil {
		return err
	}
	engine, err := host.Build(rules, initial, nil)
	if err != nil {
		return err
	}
	defer engine.Close()
	recorder, err := newRecorder(output)
	if err != nil {
		return err
	}
	defer recorder.close()
	return runStages(engine, recorder)
}

func initialWorld(scenePath string) (*world.World, error) {
	if scenePath == "" {
		return world.New(), nil
	}
	drawn, err := loaders.LoadScene(scenePath)
	if err != nil {
		return nil, fmt.Errorf("cannot load scene: %w", err)
	}
	initial, _ := bridge.FromScene(drawn)
	return initial, nil
}

// runStages starts the simulation and steps every stepped stage once per round, in the order of
// the rules file, until all of them reach the end of their clocks.
func runStages(engine *host.Engine, recorder *recorder) error {
	started, err := engine.Start()
	recorder.events(started)
	if err != nil {
		return err
	}
	for _, stage := range steppedStages(engine) {
		if err := recorder.snapshotIfDue(engine, stage, stage.Clock.From-1); err != nil {
			return err
		}
	}
	for {
		running := unfinished(steppedStages(engine))
		if len(running) == 0 {
			return nil
		}
		for _, stage := range running {
			if err := stepAndRecord(engine, stage, recorder); err != nil {
				return err
			}
		}
	}
}

func stepAndRecord(engine *host.Engine, stage *host.Stage, recorder *recorder) error {
	before, began := stage.Time, time.Now()
	sets, err := engine.StepStage(stage.Name)
	recorder.events(sets)
	if err != nil {
		return fmt.Errorf("stage %q at %g: %w", stage.Name, stage.Time, err)
	}
	fmt.Fprintf(os.Stderr, "%s %g (%d changes, %v)\n", stage.Name, stage.Time, countChanges(sets), time.Since(began).Round(time.Millisecond))
	return recorder.snapshotIfDue(engine, stage, before)
}

func steppedStages(engine *host.Engine) []*host.Stage {
	var stepped []*host.Stage
	for _, stage := range engine.Stages {
		if !stage.Clock.Realtime {
			stepped = append(stepped, stage)
		}
	}
	return stepped
}

func unfinished(stages []*host.Stage) []*host.Stage {
	return slices.DeleteFunc(slices.Clone(stages), func(stage *host.Stage) bool { return stage.Finished() })
}

func countChanges(sets []world.ChangeSet) int {
	total := 0
	for _, set := range sets {
		total += len(set.Changes)
	}
	return total
}

// recorder writes snapshots and events into the output folder.
type recorder struct {
	folder string
	log    *os.File
	writer *json.Encoder
}

func newRecorder(folder string) (*recorder, error) {
	log, err := os.Create(filepath.Join(folder, "events.jsonl"))
	if err != nil {
		return nil, err
	}
	return &recorder{folder: folder, log: log, writer: json.NewEncoder(log)}, nil
}

// snapshotIfDue saves the world for every snapshot time the stage passed since "before".
func (recorder *recorder) snapshotIfDue(engine *host.Engine, stage *host.Stage, before float64) error {
	for _, moment := range stage.Snapshots {
		if moment > before && moment <= stage.Time {
			path := filepath.Join(recorder.folder, fmt.Sprintf("%s %g.geojson", stage.Name, moment))
			extra := map[string]any{"stage": stage.Name, "time": stage.Time, "unit": stage.Clock.Unit}
			if err := snapshot.WriteEntities(path, engine.World, extra); err != nil {
				return err
			}
			prefix := fmt.Sprintf("%s %g ", stage.Name, moment)
			if err := snapshot.WriteFields(recorder.folder, prefix, engine.World, extra); err != nil {
				return err
			}
		}
	}
	return nil
}

// events appends every emitted event, with where and when it happened, as one JSON line each.
func (recorder *recorder) events(sets []world.ChangeSet) {
	for _, set := range sets {
		for _, change := range set.Changes {
			if change.Operation == world.Emit && change.Event != nil {
				_ = recorder.writer.Encode(map[string]any{"origin": set.Origin, "stage": set.Stage, "time": set.Time,
					"name": change.Event.Name, "data": change.Event.Data})
			}
		}
	}
}

func (recorder *recorder) close() { _ = recorder.log.Close() }
