package host

import (
	"strings"
	"testing"

	"github.com/akrck02/go-simulator/configuration"
	"github.com/akrck02/go-simulator/plugin"
	"github.com/akrck02/go-simulator/protocol"
	"github.com/akrck02/go-simulator/world"
)

type idle struct{}

func (idle) Step(*plugin.Context) error { return nil }

// pluginWith is a built-in plugin offering one rule, described as given, that does nothing.
func pluginWith(name string, description protocol.RuleDescription) Builtin {
	offered := &plugin.Plugin{Name: name, Version: "1", Rules: []plugin.Definition{{
		Description: description,
		New:         func(plugin.Parameters) (plugin.Rule, error) { return idle{}, nil },
	}}}
	return offered.ServeConnection
}

// buildWith builds the rules file "plugins in order, one rule each, in that order".
func buildWith(t *testing.T, initial *world.World, plugins ...struct {
	name        string
	description protocol.RuleDescription
}) (*Engine, error) {
	t.Helper()
	rules := "plugins:\n"
	builtins := map[string]Builtin{}
	for _, offered := range plugins {
		rules += "  - {name: " + offered.name + ", builtin: true}\n"
		builtins[offered.name] = pluginWith(offered.name, offered.description)
	}
	rules += "stages:\n  - name: only\n    clock: {from: 0, to: 1, step: 1}\n    rules:\n"
	for _, offered := range plugins {
		rules += "      - rule: " + offered.name + "/" + offered.description.Name + "\n"
	}
	settings, err := configuration.Parse([]byte(rules))
	if err != nil {
		t.Fatal(err)
	}
	engine, err := Build(settings, initial, builtins)
	if engine != nil {
		t.Cleanup(func() { _ = engine.Close() })
	}
	return engine, err
}

func offered(name string, description protocol.RuleDescription) struct {
	name        string
	description protocol.RuleDescription
} {
	return struct {
		name        string
		description protocol.RuleDescription
	}{name, description}
}

func ground(unit string) protocol.RuleDescription {
	return protocol.RuleDescription{Name: "ground", Publishes: []protocol.OutputDescription{plugin.Field("elevation", unit, "")}}
}

func climate(expects ...protocol.InputDescription) protocol.RuleDescription {
	return protocol.RuleDescription{Name: "climate", Expects: expects,
		Publishes: []protocol.OutputDescription{plugin.Field("temperature", "°C", "")}}
}

func failsWith(t *testing.T, err error, words ...string) {
	t.Helper()
	if err == nil {
		t.Fatalf("the build should fail, mentioning %v", words)
	}
	for _, word := range words {
		if !strings.Contains(err.Error(), word) {
			t.Fatalf("the error should mention %q: %v", word, err)
		}
	}
}

func TestAgreeingPluginsBuildWithoutWarnings(t *testing.T) {
	engine, err := buildWith(t, nil, offered("terrain", ground("m")), offered("climate", climate(plugin.NeedsField("elevation", "m"))))
	if err != nil {
		t.Fatal(err)
	}
	if len(engine.Warnings) > 0 {
		t.Fatalf("no warnings expected: %v", engine.Warnings)
	}
}

func TestTheSameFieldInTwoUnitsIsAnError(t *testing.T) {
	second := ground("km")
	second.Name = "other_ground"
	_, err := buildWith(t, nil, offered("terrain", ground("m")), offered("survey", second))
	failsWith(t, err, `field "elevation"`, `"m"`, `"km"`)
}

func TestTheSamePropertyInTwoUnitsIsAnError(t *testing.T) {
	river := func(name, unit string) protocol.RuleDescription {
		return protocol.RuleDescription{Name: name, Publishes: []protocol.OutputDescription{
			plugin.Kind("river", "", plugin.Property("width", unit, ""))}}
	}
	_, err := buildWith(t, nil, offered("terrain", river("rivers", "world")), offered("town", river("canals", "m")))
	failsWith(t, err, `kind "river" property "width"`)
}

func TestExpectingAnotherUnitIsAnError(t *testing.T) {
	_, err := buildWith(t, nil, offered("terrain", ground("m")), offered("climate", climate(plugin.NeedsField("elevation", "km"))))
	failsWith(t, err, `expects field "elevation" in "km"`, `in "m"`)
}

func TestARequiredInputNobodyPublishesIsAnError(t *testing.T) {
	_, err := buildWith(t, nil, offered("climate", climate(plugin.NeedsField("elevation", "m"))))
	failsWith(t, err, `needs field "elevation"`)
}

func TestTheInitialWorldCanGiveARequiredInput(t *testing.T) {
	initial := world.New()
	patch := &world.FieldPatch{Name: "elevation", Columns: 1, Rows: 1, Extent: &world.Extent{MaximumX: 1, MaximumZ: 1}, Values: []float32{3}}
	if err := initial.ApplyChange(world.SetFieldValues(patch)); err != nil {
		t.Fatal(err)
	}
	if _, err := buildWith(t, initial, offered("climate", climate(plugin.NeedsField("elevation", "m")))); err != nil {
		t.Fatal(err)
	}
}

func TestAnOptionalInputMayBeMissing(t *testing.T) {
	if _, err := buildWith(t, nil, offered("climate", climate(plugin.ReadsField("drainage_area", "km²")))); err != nil {
		t.Fatal(err)
	}
}

func TestReadingWhatALaterRulePublishesIsAWarning(t *testing.T) {
	engine, err := buildWith(t, nil, offered("climate", climate(plugin.NeedsField("elevation", "m"))), offered("terrain", ground("m")))
	if err != nil {
		t.Fatal(err)
	}
	if len(engine.Warnings) != 1 || !strings.Contains(engine.Warnings[0], "first step will not have it") {
		t.Fatalf("one warning about the order expected: %v", engine.Warnings)
	}
}

func TestCategoriesNeedTheEventThatNamesThem(t *testing.T) {
	biomes := func(publishes ...protocol.OutputDescription) protocol.RuleDescription {
		return protocol.RuleDescription{Name: "biomes", Publishes: publishes}
	}
	engine, err := buildWith(t, nil, offered("climate", biomes(plugin.IndexField("biome", "biomes", ""))))
	if err != nil {
		t.Fatal(err)
	}
	if len(engine.Warnings) != 1 || !strings.Contains(engine.Warnings[0], `event "biomes"`) {
		t.Fatalf("a warning about the missing names expected: %v", engine.Warnings)
	}
	engine, err = buildWith(t, nil, offered("climate", biomes(plugin.IndexField("biome", "biomes", ""), plugin.Event("biomes", "", plugin.Property("names", "", "")))))
	if err != nil || len(engine.Warnings) > 0 {
		t.Fatalf("with the event no warning: %v %v", err, engine.Warnings)
	}
}

func TestUndescribedRulesAreNotCheckedButCount(t *testing.T) {
	legacy := protocol.RuleDescription{Name: "ground", Writes: []string{"elevation"}}
	engine, err := buildWith(t, nil, offered("terrain", legacy), offered("climate", climate(plugin.NeedsField("elevation", "m"))))
	if err != nil {
		t.Fatalf("what an undescribed rule writes is not missing: %v", err)
	}
	if len(engine.Warnings) != 1 || !strings.Contains(engine.Warnings[0], `not checked: plugin "terrain"`) {
		t.Fatalf("a warning naming the unchecked plugin expected: %v", engine.Warnings)
	}
}

func TestBadDescriptionsAreRefused(t *testing.T) {
	for _, bad := range []protocol.RuleDescription{
		{Name: "ground", Publishes: []protocol.OutputDescription{{Name: "elevation", What: "field", Unit: "metres"}}},
		{Name: "ground", Publishes: []protocol.OutputDescription{{Name: "elevation", What: "field"}}},
		{Name: "ground", Publishes: []protocol.OutputDescription{{Name: "elevation", What: "grid", Unit: "m"}}},
		{Name: "ground", Expects: []protocol.InputDescription{{Name: "", What: "field"}}},
	} {
		if _, err := buildWith(t, nil, offered("terrain", bad)); err == nil {
			t.Errorf("the description %+v should be refused", bad)
		}
	}
}

// versionOne answers like a plugin of protocol 1: a description without contracts.
func versionOne(connection *protocol.Connection) error {
	for {
		message, err := connection.Receive()
		if err != nil {
			return nil
		}
		switch message.Kind {
		case protocol.Describe:
			err = connection.Send(protocol.Description, protocol.PluginDescription{Protocol: 1, Name: "old", Version: "1",
				Rules: []protocol.RuleDescription{{Name: "ground", Writes: []string{"elevation"}}}})
		case protocol.Attach:
			var request protocol.AttachRequest
			_ = message.Decode(&request)
			err = connection.Send(protocol.Attached, protocol.AttachReply{Instance: request.Instance})
		case protocol.Stop:
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func TestVersionOnePluginsStillRun(t *testing.T) {
	settings, err := configuration.Parse([]byte("plugins: [{name: old, builtin: true}]\nstages:\n  - name: only\n    clock: {from: 0, to: 1, step: 1}\n    rules: [{rule: ground}]\n"))
	if err != nil {
		t.Fatal(err)
	}
	engine, err := Build(settings, nil, map[string]Builtin{"old": versionOne})
	if err != nil {
		t.Fatalf("a protocol 1 plugin must still be accepted: %v", err)
	}
	defer engine.Close()
	if len(engine.Warnings) != 1 || !strings.Contains(engine.Warnings[0], `plugin "old"`) {
		t.Fatalf("its rules are not checked, and the host says so: %v", engine.Warnings)
	}
}
