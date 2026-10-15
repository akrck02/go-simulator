# 📚 go-simulator documentation

![Four simulators on the same made-up island: its relief, its climate, its vegetation after 200 years and a town after 100](images/chain.jpg)

*Four simulators taking turns on the same made-up island: the terrain gives the ground, the climate
works out heat, rain and biomes from it, the flora grows forests where they fit, and a town grows on
the plain. None of them knows the others: they only share the world.*

go-simulator is the engine those simulators share. It keeps the world, moves the clock forward and
passes each simulator what it needs. The simulators themselves are **plugins**: small separate
programs, in any language, that you can mix like building blocks.

| | Page | What is in it |
|---|---|---|
| 🏛️ | [Architecture](architecture.md) | How it all fits together: the world, the seven changes, one step from start to end, interests, events, clocks and the packages. |
| 📜 | [Rules files](rules-files.md) | Setting up a simulation: every key, with includes, anchors and objectives, and a rules file's life. |
| ▶️ | [Running](running.md) | `simulate` and `simulator`, with or without a window, the files they write and the bridge. |
| 🏷️ | [Labels, header and notices](overlay.md) | The words the live viewer writes over the scene, with pictures. |
| 🔌 | [Writing plugins](writing-plugins.md) | A simulator of your own: the Go toolkit, testing, and other languages. |
| 📖 | [Vocabulary and conditions](vocabulary.md) | The names plugins share, their units, and the little condition language. |
| 📡 | [Protocol](protocol.md) | Talking to the host from another language: frames, messages and the JSON of every change. |

New here? Start with the [project README](../README.md), then [Architecture](architecture.md).

Plugins that exist today: [terrain](https://github.com/akrck02/go-simulator-terrain), [climate](https://github.com/akrck02/go-simulator-climate),
[flora](https://github.com/akrck02/go-simulator-flora), [urban](https://github.com/akrck02/go-simulator-urban) and
[motion](https://github.com/akrck02/go-simulator-motion).

## 🚧 Not done yet

- The bridge does not draw fields yet: a scene builder can turn them into terrain or vegetation.
- Ground shapes take their height from the scene's walkable ground; the first time, the renderer
  builds its ground index, which can take a few seconds on a large terrain.
- Removed meshes leave the scene, but go-renderer has no call yet to free their GPU memory.
- Plugins resolve go-renderer through this module's requirements, so a plugin's `go.mod` needs the
  same `replace` during development.
