# go-simulator

The library that lets [go-renderer](../go-renderer) show simulations. Simulators are **plugins**:
separate programs, in their own repositories and in any language, that the host starts at
runtime. A **rules file** says which plugins to use and which of their rules to run, with what
parameters and on which clock. Plugins never touch the renderer: they read the world and answer
with **changes** (add, remove, modify, set a field, change the environment, emit an event, clear
an area of the drawn scene). The
host applies them to the world, forwards them to the plugins that read that part of the world,
and the bridge shows them in the scene.

```
go-renderer      draws a scene; knows nothing about simulations
   ▲
go-simulator     world model · change protocol · host engine · rules files · renderer bridge
   ▲   (plugins use the plugin, world and spatial packages; never the renderer)
plugins          go-simulator-terrain · go-simulator-climate · go-simulator-motion · go-simulator-urban · …
```

During development the renderer is used from the folder next to this one (`replace` in `go.mod`).

## Run

```bash
go run ./cmd/simulator -rules ../go-simulator-motion/examples/living-scene.yaml scene.glb
```

It accepts all the viewer flags (`-walk`, `-night`, `-capture`, `-benchmark`…), and `-advance N`
steps every stepped stage N times before showing (to start, or capture a frame, further on). Keys, besides the
viewer's: `P` pauses, `,` `.` halve or double the speed of time, and every `toggle_key` in the
rules file turns its rule on and off. Without `-rules` it only shows the scene.

To run the stepped stages without a window, to the end of their clocks, and save the world at the
times each stage lists in `snapshots` (GeoJSON, in ground coordinates [x, z]) with every emitted
event in `events.jsonl` and the stages' objectives checked in `objectives.json`:

```bash
go run ./cmd/simulate -rules rules.yaml [-scene scene.glb] -output results/
```

## Rules files

```yaml
plugins:
  - name: motion
    command: [go-simulator-motion]   # program and arguments; relative paths start at this file
    directory: ..                     # optional: where the program runs
stages:
  - name: motion
    clock: realtime                   # or {unit: year, from: 0, to: 100, step: 1, pace: 0.4}
    snapshots: [0, 50, 100]           # stepped stages: times a run without a window saves
    rules:
      - rule: day_cycle               # or motion/day_cycle when several plugins offer it
        with: {seconds_per_day: 120}  # parameters
        enabled: false
        toggle_key: T
      - rule: tides
        every: 0.5                    # run when this much stage time passed (default: every step)
      - rule: entity_motion
        at: 10                        # run once at this stage time
        name: late_motion             # instance name (default: stage/rule)
```

A rule's parameters can come from files: `with: {include: scenario.yaml}` (or a list of files)
reads their YAML maps in order and the rule's own parameters win over them. A scenario can then
live in its own file, with comments and notes for people (`story`, `interpretation`…), which
plugins ignore because they only read the parameters they know. An included file cannot include
others, and paths inside it still start at the rules file's folder.

Rules run in the file's order; consecutive rules of one plugin go in a single request.
Real-time stages advance with the viewer (`Engine.Advance`); stepped stages advance one step at a
time (`Engine.StepStage`) until their `to`, or, with a `pace`, one step every `pace` real seconds
while shown. Plugin programs get the rules file's folder in `GO_SIMULATOR_RULES_DIRECTORY`, and
`plugin.Path` resolves the paths of their parameters against it; `~/` works in commands,
directories and paths.

## Anchors and objectives

A simulator is generic: it knows nothing about the world it runs on. What makes its result the
one a story needs comes from the rules file, in two ways.

**Anchors** are places the story fixes. The host adds them to the world before the simulation
starts, as hidden entities with `"anchor": true`, their `role` and their `source`, so every plugin
sees them and builds around them, never over them:

```yaml
anchors:
  - {id: main_gate, role: gate, at: [-2.5, 1.2], source: "chapter 3 of the story"}
  - {id: spring, role: obstacle, outline: [[x, z], …]}
  - {id: old_road, role: road, line: [[x, z], …], width: 0.01}
  - {from: anchors.geojson, role: obstacle}   # one per feature; its id, kind and role win
```

**Objectives** are what a run must reach. Each stage lists them; `simulate` checks each one when
its stage reaches its time, prints it, saves every result in `objectives.json` and fails if any
was missed (`-objectives warn` only reports them). An objective counts the entities a condition
matches or reads a value from the last event of a name:

```yaml
    objectives:
      - name: walled by year 60
        at: 60
        count: {kind: wall}                  # at least one, unless min, max or near say otherwise
      - name: about 6,000 people in year 100
        at: 100
        event: indicators
        value: inhabitants
        near: 6000
        tolerance: 0.05                      # ±5 %
        source: "the author's notes"
```

With a fixed seed, a run that reaches its objectives gives the same concrete result every time.

## Conditions

Objectives, biome rules and scene styles say when something applies with the same small
language (package `conditions`): a map from a fact to what it must be, every entry holding.

| Written | Holds when the fact is |
|---|---|
| `kind: building` | equal to the value (numbers by value, the rest by text) |
| `use: [shop, workshop]` | one of the values |
| `floors: {min: 2, max: 3}` | a number in the range (either end may be left out) |
| `river: true` | that yes/no value |

A fact that does not exist never holds. For entities, the facts are `kind` and their properties.

## Vocabulary

Plugins read each other's results through the world, never through each other's files, so the
names they share are written down in `world/vocabulary.go`: fields (`elevation`, `water_depth`,
`slope`, `temperature`, `precipitation`, `biome`), kinds (`street`, `parcel`, `building`, `wall`,
`gate`, `square`, `bridge`, `river`, `tree`, `anchor`) and properties (`year`, `use`, `floors`,
`class`, `sector`, `species`, `anchor`, `role`). Positions and the sizes of shapes are in world
units, the unit of the scene, on the ground plane (x, z) with y up: they are what gets drawn.
Fields and properties are data in physical units written next to their names (the elevation in
metres, the temperature in °C), so models read them without knowing the scene's scale. A name
another plugin may want to read belongs in the vocabulary. The ground itself comes from the
terrain plugin ([go-simulator-terrain](../go-simulator-terrain)), which publishes `elevation` and
`slope` from a relief.

## Shapes

Entities are drawn from their shape: an existing mesh by name, a `box` primitive, or a ground
shape in world coordinates on the ground plane that follows the terrain:

| Shape | Drawn as |
|---|---|
| `outline` | a flat area (a plot, a square) or, with `height`, a volume (a house, a tower) |
| `line` + `width` | a strip (a street) or, with `height`, a wall |

A simulation that takes over part of the drawn scene (an old town drawn into the island, the forest
where a city grows) sends `clear_area`: the triangles of the named scene nodes whose middle falls
inside its outlines, and the instances standing there, are removed; `keep` spares holes in it.

Shapes with the same `batch` become one mesh, rebuilt when any of them changes, so thousands of
houses cost a handful of draw calls. Flat shapes float `lift` over the ground and are pulled
towards the camera so the terrain does not hide them from far away.

## Packages

| Package | What it is |
|---|---|
| `world` | The neutral state: entities (id, kind, parent, transform, shape, properties), fields (named grids), the environment (sun, wind, clouds, tide), changes and change sets, interests (what a rule reads) and the shared vocabulary |
| `conditions` | The condition language of objectives, biome rules and styles |
| `protocol` | Messages between host and plugins and their framing over standard input and output |
| `plugin` | The toolkit for writing a plugin in Go: `Plugin`, `Definition`, `Rule`, `Context`, `Parameters`, `Serve` |
| `spatial` | Renderer-free vectors and matrices for plugins (same layout as go-renderer) |
| `host` | Starts plugins (`Launch` for programs, `Connect` for built-ins), adds the anchors, attaches rules, runs the stages (`Engine`) and checks objectives |
| `configuration` | Reads and checks rules files: includes, anchors and objectives |
| `bridge` | Builds the initial world from a go-renderer scene and shows every change set in it (ground shapes, batches) |
| `snapshot` | Writes a world's entities as GeoJSON and fields as grids; reads GeoJSON features |

## Writing a plugin in Go

```go
type lighthouse struct{ angle float64 }

func (rule *lighthouse) Step(context *plugin.Context) error {
	rule.angle += context.Step
	beam := spatial.RotationY(rule.angle)
	context.Modify(&world.Entity{Identifier: "lighthouse-beam", Transform: &beam})
	return nil
}

func main() {
	turn := plugin.Definition{
		Description: protocol.RuleDescription{Name: "turn_beam", Reads: world.Interest{Kinds: []string{"beam"}}},
		New:         func(plugin.Parameters) (plugin.Rule, error) { return &lighthouse{}, nil },
	}
	coast := &plugin.Plugin{Name: "coast", Version: "0.1.0", Rules: []plugin.Definition{turn}}
	if err := coast.Serve(); err != nil {
		log.Fatal(err)
	}
}
```

A rule can also implement `Start(context)` to do something once at the beginning. `context.Events`
holds the events of other plugins the rule reads (its interest's `Events`) received with the current
request; at the start, it holds the last event of each of those names, so a plugin that starts later
still learns, for example, the names of the biomes the climate announced. Changes a rule
makes are applied at once to the plugin's copy of the world, so it and the next rules see them.
A plugin whose rules share state can set `AfterStep`, run once after the rules of each request, to
describe the result once (go-simulator-urban publishes its cities there).
Logs go to standard error; standard output carries the protocol.

## Writing a plugin in another language

The protocol is plain bytes, so any language works. Every message is a frame: a 4-byte
little-endian length, a JSON header of that length — `{"kind", "body", "attachments": [byte
lengths]}` — and then the attachments. Field values travel as attachments of little-endian
float32, one per `set_field` change, in the order those changes appear in the message.

| Host sends | Plugin answers |
|---|---|
| `describe` | `description`: `{protocol: 1, name, version, rules: [{name, parameters, reads, writes}]}` |
| `attach` `{instance, rule, parameters}` | `attached` `{instance}` |
| `start` `{snapshot: change set}` | `changes`: a change set |
| `step` `{stage, time, step, instances, incoming: [changes]}` | `changes`: a change set |
| `stop` | (exits) |

Any answer can instead be `error` `{message}`; the host stops the simulation with it. See
`protocol` and `world` for the exact JSON of changes.

## Known limits

- A field change (`set_field`) is kept in the world and forwarded to plugins, but the bridge does
  not draw fields yet: they will become terrain and water when the landscape plugin needs them.
- Ground shapes take their height from the scene's walkable ground; the first time, the renderer
  builds its ground index, which can take a few seconds on a large terrain.
- Removed meshes leave the scene, but go-renderer has no call yet to free their GPU memory; a
  simulation that adds and removes thousands of different meshes will need one.
- Plugins still resolve go-renderer through this module's requirements, so a plugin's `go.mod`
  needs the same `replace` for it during development.

## Scene styles

A rules file can name a style file of [go-scene-builder](../go-scene-builder): the viewer then
draws the batches of ground shapes its rules match with the scene builder (walls, roofs, gates,
level bridges…) instead of plain extruded outlines; the other batches are drawn as before.

```yaml
scene_style: style.yaml      # relative to the rules file
plugins: …
```

The bridge only knows the `BatchBuilder` interface; the `scenebuilder` package implements it with
go-scene-builder, which needs GEOS (`pkg-config geos`) to build the simulator.
