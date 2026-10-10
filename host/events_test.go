package host

import (
	"testing"

	"github.com/akrck02/go-simulator/configuration"
	"github.com/akrck02/go-simulator/plugin"
	"github.com/akrck02/go-simulator/protocol"
	"github.com/akrck02/go-simulator/world"
)

// announcer emits the names of some categories when it starts; listener remembers the events it
// receives, at the start and at every step.
type announcer struct{}

func (rule *announcer) Start(context *plugin.Context) error {
	context.Emit("categories", map[string]any{"names": []any{"forest", "meadow"}})
	return nil
}

func (rule *announcer) Step(*plugin.Context) error { return nil }

type listener struct{ atStart, atSteps *[]world.Event }

func (rule *listener) Start(context *plugin.Context) error {
	*rule.atStart = append(*rule.atStart, context.Events...)
	return nil
}

func (rule *listener) Step(context *plugin.Context) error {
	*rule.atSteps = append(*rule.atSteps, context.Events...)
	return nil
}

func TestALaterPluginLearnsEarlierEvents(t *testing.T) {
	var atStart, atSteps []world.Event
	announcing := &plugin.Plugin{Name: "climate", Version: "1", Rules: []plugin.Definition{{
		Description: protocol.RuleDescription{Name: "announce"},
		New:         func(plugin.Parameters) (plugin.Rule, error) { return &announcer{}, nil },
	}}}
	listening := &plugin.Plugin{Name: "flora", Version: "1", Rules: []plugin.Definition{{
		Description: protocol.RuleDescription{Name: "listen", Reads: world.Interest{Events: []string{"categories"}}},
		New: func(plugin.Parameters) (plugin.Rule, error) {
			return &listener{atStart: &atStart, atSteps: &atSteps}, nil
		},
	}}}
	settings, err := configuration.Parse([]byte(`
plugins: [{name: climate, builtin: true}, {name: flora, builtin: true}]
stages:
  - {name: year, clock: {from: 0, to: 2, step: 1}, rules: [{rule: announce}, {rule: listen}]}
`))
	if err != nil {
		t.Fatal(err)
	}
	engine, err := Build(settings, nil, map[string]Builtin{"climate": announcing.ServeConnection, "flora": listening.ServeConnection})
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	if _, err := engine.Start(); err != nil {
		t.Fatal(err)
	}
	if len(atStart) != 1 || atStart[0].Name != "categories" {
		t.Fatalf("at the start the listener got %v, want the announcer's categories", atStart)
	}
	for range 2 {
		if _, err := engine.StepStage("year"); err != nil {
			t.Fatal(err)
		}
	}
	if len(atSteps) != 1 {
		t.Errorf("over the steps the listener got %d events, want the one sent while it started", len(atSteps))
	}
}
