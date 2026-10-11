# go-simulator

Simulations for [go-renderer](https://github.com/akrck02/go-renderer) scenes. The simulators are **plugins**: separate
programs, in their own repositories and in any language, that the host starts at runtime. A
**rules file** says which plugins to use, which of their rules to run, with what parameters, on
which clock and what the run must reach. Plugins never touch the renderer: they read the world and
answer with **changes**; the host applies them, forwards them to the plugins that read that part of
the world, and the bridge shows them in the scene.

```
go-renderer      draws a scene; knows nothing about simulations
   ▲
go-simulator     world model · change protocol · host engine · rules files · renderer bridge
   ▲
plugins          terrain · climate · flora · urban · motion · …
```

## Run

```bash
go run ./cmd/simulator -rules rules.yaml scene.glb          # live, in the viewer
go run ./cmd/simulate -rules rules.yaml -output results/     # without a window, to files
```

```yaml
plugins:
  - {name: motion, command: [go-simulator-motion]}
stages:
  - name: weather
    clock: realtime
    rules:
      - {rule: day_cycle, with: {seconds_per_day: 120}, toggle_key: T}
```

## Documentation

Everything is in [docs/](docs/README.md):

- [Architecture](docs/architecture.md): the world, the seven changes, one step from start to end, interests, events and clocks.
- [Rules files](docs/rules-files.md): every key, with includes, anchors and objectives.
- [Protocol](docs/protocol.md): frames, messages and the JSON of every change.
- [Writing plugins](docs/writing-plugins.md): the Go toolkit and other languages.
- [Vocabulary and conditions](docs/vocabulary.md): shared names, units and the condition language.
- [Running](docs/running.md): `simulate`, `simulator`, the files they write and the bridge.

During development the renderer and the scene builder are used from the folders next to this one
(`replace` in `go.mod`). Building the viewer needs GEOS (`pkg-config geos`) for the scene builder.
