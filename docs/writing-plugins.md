# 🔌 Writing plugins

A plugin is a small program that offers **rules**: things it knows how to do, such as "turn the
sun" or "grow the town for a year". You write the rules; the host decides when to call them and
takes care of everything else (starting your program, sending it the world, forwarding changes,
saving results).

In Go, the `plugin` package does all the talking for you, and a useful plugin fits in a page. In
any other language, the [protocol](protocol.md) is plain JSON over standard input and output.

Here is the whole journey, using a lighthouse whose beam turns.

## In Go

### 1. A rule

A rule is anything with a `Step` method. It reads `context.World` (the plugin's copy of the world)
and describes what changes through the context:

```go
type lighthouse struct{ angle, speed float64 }

func (rule *lighthouse) Step(context *plugin.Context) error {
	rule.angle += rule.speed * context.Step
	beam := spatial.RotationY(rule.angle)
	context.Modify(&world.Entity{Identifier: "lighthouse-beam", Transform: &beam})
	return nil
}
```

A rule that also implements `Start(context *plugin.Context) error` does something once, when the
simulation starts (publish a field, set up its state). Returning an error stops the simulation
with it.

### 2. Its definition

A definition describes the rule to the host and builds an instance from the parameters of one use
in a rules file:

```go
turn := plugin.Definition{
	Description: protocol.RuleDescription{
		Name:    "turn_beam",
		Summary: "Turns the lighthouse beam.",
		Parameters: []protocol.ParameterDescription{
			{Name: "speed", Type: "number", Default: 1, Summary: "radians per second"},
		},
		Reads:  world.Interest{Kinds: []string{"beam"}},
		Writes: []string{"beam"},
		Publishes: []protocol.OutputDescription{
			plugin.Kind("beam", "the lighthouse's turning beam, moved by its transform"),
		},
	},
	New: func(parameters plugin.Parameters) (plugin.Rule, error) {
		return &lighthouse{speed: parameters.Number("speed", 1)}, nil
	},
}
```

Declare every parameter, what the rule reads and what it writes: the host sends the plugin only
what its rules read, and tools build references from the description.

`Publishes` and `Expects` are the rule's [contracts](protocol.md#contracts): what it publishes and
what it needs, with units. The host compares them before the first step and refuses to start when
two plugins disagree, so a wrong unit shows at once instead of as a strange result. The `plugin`
package has one-line helpers: `plugin.Field("elevation", "m", "height of the ground")`,
`plugin.IndexField("biome", "biomes", …)`, `plugin.Kind`, `plugin.Event` and `plugin.Property`
for what it publishes; `plugin.NeedsField("elevation", "m")`, `plugin.ReadsField`,
`plugin.ReadsKind` and `plugin.ReadsEvent` for what it reads.

### 3. The plugin

```go
func main() {
	log.SetFlags(0)
	coast := &plugin.Plugin{Name: "coast", Version: "0.1.0", Rules: []plugin.Definition{turn}}
	if err := coast.Serve(); err != nil {
		log.Fatal(err)
	}
}
```

Logs go to standard error: standard output carries the protocol.

### The context

| Member | Use |
|---|---|
| `World` | The plugin's copy of the world, up to date with other plugins' changes it reads |
| `Stage`, `Time`, `Step`, `Instance` | The stage, the time after the step, the time it covers, the instance being run |
| `Events` | Events of other plugins the rule reads, received with this request; at the start, the last of each name |
| `Add`, `Remove`, `Modify` | Entity changes |
| `SetField`, `SetEnvironment` | Field and environment changes |
| `Emit(name, data)` | Tell something happened |
| `ClearSceneArea(area)` | Hide what some nodes of the drawn scene show in an area |

Changes are applied to the plugin's copy at once, so the rule and the rules after it in the same
request see them, and are sent to the host at the end of the request. A change that cannot apply
(modifying an entity that does not exist) stops the simulation with a clear message.

### Parameters

Parameters arrive as JSON values (numbers are float64, lists `[]any`, maps `map[string]any`).
`Number`, `Boolean`, `Text`, `Vector` and `Map` read them with a fallback; `plugin.Path` turns a
path written in a rules file into one the plugin can open (relative to the rules file, `~/` from
home). For a large set of parameters, marshal them to JSON and decode into a struct, filling the
defaults afterwards.

### Shared state

Rules of one plugin can share state: the rule that sets things up registers it under its instance
name, and the others find it by a parameter (`city: my_town`). When several rules change shared
state, `Plugin.AfterStep` runs once after the rules of every request, with the same context, so
the plugin describes the result once instead of after every rule.

### Testing

The host can run a plugin inside the test process: give `host.Build` a map of built-ins
(`{"coast": coastPlugin.ServeConnection}`) and name them in the rules file with `builtin: true`.
Then `engine.Start()` and `engine.StepStage(stage)` drive it, and `engine.World` shows the result.

```go
settings, _ := configuration.Parse([]byte(rules))
engine, err := host.Build(settings, initialWorld, map[string]host.Builtin{"coast": coast.ServeConnection})
```

## In another language

A plugin must:

1. Read frames from standard input and write frames to standard output (4-byte little-endian
   length, JSON header, attachments).
2. Answer `describe` with its description, `attach` with `attached`, `start` and `step` with a
   change set, and exit on `stop`.
3. Keep its own copy of the world: apply the start snapshot and every step's `incoming` changes.
4. Read and write field values as float32 attachments, one per `set_field`, in order.

See [protocol](protocol.md) for the JSON of every message and change.

## Good plugins

A few habits keep plugins easy to combine:

- Know nothing about any particular world. Everything concrete comes from parameters, anchors and
  fields; use made-up examples in comments, tests and documentation.
- Read other plugins' results from the world by the names of the [vocabulary](vocabulary.md),
  never from their files.
- Publish data, not looks: kinds and properties in physical units; colours and models belong to
  scene styles.
- Be deterministic: take a seed, and derive any randomness from it.
