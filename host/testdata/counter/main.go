// Command counter is a tiny plugin for the host's tests: its rule "count" adds a "mark" entity
// every time it steps.
package main

import (
	"fmt"
	"log"

	"github.com/akrck02/go-simulator/plugin"
	"github.com/akrck02/go-simulator/protocol"
	"github.com/akrck02/go-simulator/world"
)

type counter struct{ count int }

func (rule *counter) Step(context *plugin.Context) error {
	rule.count++
	context.Add(&world.Entity{Identifier: world.Identifier(fmt.Sprintf("mark-%d", rule.count)), Kind: "mark"})
	return nil
}

func main() {
	count := plugin.Definition{
		Description: protocol.RuleDescription{Name: "count", Writes: []string{"mark"}},
		New:         func(plugin.Parameters) (plugin.Rule, error) { return &counter{}, nil },
	}
	counting := &plugin.Plugin{Name: "counter", Version: "1.0.0", Rules: []plugin.Definition{count}}
	if err := counting.Serve(); err != nil {
		log.Fatal(err)
	}
}
