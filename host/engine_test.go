package host

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/akrck02/go-simulator/configuration"
	"github.com/akrck02/go-simulator/plugin"
	"github.com/akrck02/go-simulator/protocol"
	"github.com/akrck02/go-simulator/world"
)

// builder adds a "house" entity each step; watcher counts the houses it knows about.
type builder struct{ built int }

func (rule *builder) Step(context *plugin.Context) error {
	rule.built++
	context.Add(&world.Entity{Identifier: world.Identifier(fmt.Sprintf("house-%d", rule.built)), Kind: "house"})
	return nil
}

type watcher struct{ seen *int }

func (rule *watcher) Step(context *plugin.Context) error {
	*rule.seen = len(context.World.EntitiesOfKind("house"))
	return nil
}

func builtins(seen *int) map[string]Builtin {
	building := &plugin.Plugin{Name: "town", Version: "1", Rules: []plugin.Definition{{
		Description: protocol.RuleDescription{Name: "build", Writes: []string{"house"}},
		New:         func(plugin.Parameters) (plugin.Rule, error) { return &builder{}, nil },
	}}}
	watching := &plugin.Plugin{Name: "census", Version: "1", Rules: []plugin.Definition{{
		Description: protocol.RuleDescription{Name: "count_houses", Reads: world.Interest{Kinds: []string{"house"}}},
		New:         func(plugin.Parameters) (plugin.Rule, error) { return &watcher{seen: seen}, nil },
	}}}
	return map[string]Builtin{"town": building.ServeConnection, "census": watching.ServeConnection}
}

func buildEngine(t *testing.T, rules string, seen *int) *Engine {
	t.Helper()
	settings, err := configuration.Parse([]byte(rules))
	if err != nil {
		t.Fatal(err)
	}
	engine, err := Build(settings, nil, builtins(seen))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = engine.Close() })
	if _, err := engine.Start(); err != nil {
		t.Fatal(err)
	}
	return engine
}

const townRules = `
plugins:
  - {name: town, builtin: true}
  - {name: census, builtin: true}
stages:
  - name: growth
    clock: {unit: year, from: 0, to: 10, step: 1}
    rules:
      - rule: build
      - rule: count_houses
        every: 5
`

func TestStepsApplyChangesAndForwardThemToReaders(t *testing.T) {
	seen := 0
	engine := buildEngine(t, townRules, &seen)
	for range 6 {
		if _, err := engine.StepStage("growth"); err != nil {
			t.Fatal(err)
		}
	}
	if houses := len(engine.World.EntitiesOfKind("house")); houses != 6 {
		t.Fatalf("expected 6 houses in the host's world, got %d", houses)
	}
	if seen != 6 {
		t.Fatalf("the census ran at years 1 and 6 and should see 6 houses the second time, saw %d", seen)
	}
}

func TestSteppedStagesStopAtTheEndOfTheirClock(t *testing.T) {
	seen := 0
	engine := buildEngine(t, townRules, &seen)
	for range 15 {
		if _, err := engine.StepStage("growth"); err != nil {
			t.Fatal(err)
		}
	}
	if stage := engine.Stage("growth"); !stage.Finished() || stage.Time != 10 {
		t.Fatalf("the stage should end at year 10, it is at %v", stage.Time)
	}
}

func TestRulesRunOnceAtTheirTimeAndToggleWithTheirKey(t *testing.T) {
	seen := 0
	engine := buildEngine(t, `
plugins:
  - {name: town, builtin: true}
stages:
  - name: life
    clock: realtime
    rules:
      - rule: build
        at: 2
      - rule: build
        toggle_key: b
        enabled: false
`, &seen)
	for range 4 {
		if _, err := engine.Advance(1); err != nil {
			t.Fatal(err)
		}
	}
	if houses := len(engine.World.Entities()); houses != 1 {
		t.Fatalf("the rule with \"at\" should run once, %d houses", houses)
	}
	if toggled := engine.Toggle("B"); len(toggled) != 1 || !toggled[0].Enabled {
		t.Fatalf("the key should turn the second rule on")
	}
	engine.Paused = true
	_, _ = engine.Advance(1)
	engine.Paused = false
	engine.Speed = 2
	_, _ = engine.Advance(1)
	if stage := engine.Stage("life"); stage.Time != 6 {
		t.Fatalf("pause and speed: the stage should be at 6 s, it is at %v", stage.Time)
	}
}

func TestRuleNamesMustBeFoundAndUnambiguous(t *testing.T) {
	missing, _ := configuration.Parse([]byte("plugins: [{name: town, builtin: true}]\nstages: [{name: s, clock: realtime, rules: [{rule: fly}]}]"))
	if _, err := Build(missing, nil, builtins(new(int))); err == nil {
		t.Fatalf("an unknown rule should fail")
	}
	unknownPlugin, _ := configuration.Parse([]byte("plugins: [{name: town, builtin: true}]\nstages: [{name: s, clock: realtime, rules: [{rule: harbour/build}]}]"))
	if _, err := Build(unknownPlugin, nil, builtins(new(int))); err == nil {
		t.Fatalf("an unknown plugin should fail")
	}
}

func TestProgramsArePlugins(t *testing.T) {
	directory := t.TempDir()
	program := filepath.Join(directory, "counter")
	build := exec.Command("go", "build", "-o", program, "./testdata/counter")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building the test plugin: %v\n%s", err, output)
	}
	rules := fmt.Sprintf("plugins: [{name: counter, command: [%q]}]\nstages: [{name: ticks, clock: realtime, rules: [{rule: count}]}]", program)
	settings, err := configuration.Parse([]byte(rules))
	if err != nil {
		t.Fatal(err)
	}
	settings.Directory, _ = os.Getwd()
	engine, err := Build(settings, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := engine.Start(); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if _, err := engine.Advance(0.1); err != nil {
			t.Fatal(err)
		}
	}
	if marks := len(engine.World.EntitiesOfKind("mark")); marks != 3 {
		t.Fatalf("expected 3 marks from the program, got %d", marks)
	}
	if err := engine.Close(); err != nil {
		t.Fatalf("the program should stop cleanly: %v", err)
	}
}
