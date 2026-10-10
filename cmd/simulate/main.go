// Command simulate runs the stepped stages of a rules file without a window, to the end of their
// clocks, and saves the world at the times each stage lists in "snapshots".
//
//	go run ./cmd/simulate -rules rules.yaml -output results/
//	go run ./cmd/simulate -rules rules.yaml -scene scene.glb -output results/
//
// It writes, in the output folder, one <stage> <time>.geojson per snapshot and events.jsonl with
// every event the plugins emitted. Real-time stages do not run.
//
// Each stage's objectives are checked when the stage reaches their time and saved in
// objectives.json. A run that misses one fails, unless -objectives warn only reports it.
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
	objectives := flag.String("objectives", "fail", "what a missed objective does: fail or warn")
	flag.Parse()
	if *rulesPath == "" || *output == "" {
		fmt.Fprintln(os.Stderr, "usage: simulate -rules rules.yaml [-scene scene.glb] -output folder")
		os.Exit(2)
	}
	if *objectives != "fail" && *objectives != "warn" {
		fmt.Fprintln(os.Stderr, "-objectives is fail or warn")
		os.Exit(2)
	}
	if err := run(*rulesPath, *scenePath, *output, *objectives == "fail"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(rulesPath, scenePath, output string, failOnMissedObjectives bool) error {
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
	if err := runStages(engine, recorder); err != nil {
		return err
	}
	return recorder.finishObjectives(engine, failOnMissedObjectives)
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
		recorder.checkObjectives(engine, stage, stage.Clock.From-1)
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
	recorder.checkObjectives(engine, stage, before)
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
	folder     string
	log        *os.File
	writer     *json.Encoder
	objectives []host.ObjectiveResult
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

// checkObjectives measures the objectives the stage reached since "before" and reports each one.
func (recorder *recorder) checkObjectives(engine *host.Engine, stage *host.Stage, before float64) {
	for _, objective := range stage.DueObjectives(before) {
		result := engine.CheckObjective(stage, objective)
		recorder.objectives = append(recorder.objectives, result)
		fmt.Fprintln(os.Stderr, result.Describe())
	}
}

// finishObjectives saves every result in objectives.json, reports the objectives no stage reached
// and, when asked to, fails if any objective was missed.
func (recorder *recorder) finishObjectives(engine *host.Engine, failOnMissed bool) error {
	recorder.objectives = append(recorder.objectives, unreachedObjectives(engine)...)
	if len(recorder.objectives) == 0 {
		return nil
	}
	contents, err := json.MarshalIndent(recorder.objectives, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(recorder.folder, "objectives.json"), contents, 0o644); err != nil {
		return err
	}
	missed := countMissed(recorder.objectives)
	fmt.Fprintf(os.Stderr, "objectives: %d of %d reached\n", len(recorder.objectives)-missed, len(recorder.objectives))
	if missed > 0 && failOnMissed {
		return fmt.Errorf("%d objectives not reached (see objectives.json; -objectives warn only reports them)", missed)
	}
	return nil
}

// unreachedObjectives are the objectives whose time is outside their stage's clock.
func unreachedObjectives(engine *host.Engine) []host.ObjectiveResult {
	var results []host.ObjectiveResult
	for _, stage := range steppedStages(engine) {
		for _, objective := range stage.Objectives {
			if objective.At > stage.Time || objective.At <= stage.Clock.From-1 {
				result := host.ObjectiveResult{Stage: stage.Name, Time: objective.At, Name: objective.Name,
					Expected: "a time inside the stage's clock", Source: objective.Source}
				fmt.Fprintln(os.Stderr, result.Describe())
				results = append(results, result)
			}
		}
	}
	return results
}

func countMissed(results []host.ObjectiveResult) int {
	missed := 0
	for _, result := range results {
		if !result.Reached {
			missed++
		}
	}
	return missed
}
