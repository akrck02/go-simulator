# go-simulator

Simulations for scenes drawn by [go-renderer](../go-renderer): things that move, wind gusts,
drifting clouds and the passing of the day. The renderer stays agnostic: it draws the scene as it
is, and this project changes the scene between frames through `viewer.Run`'s update function.

During development the renderer is used from the folder next to this one (`replace` in `go.mod`).

## Packages
* `simulation`: a `Simulation` runs `System`s at a fixed time step (`Advance`), with `Speed` and
  `Paused`. Systems change node transforms, the environment, or meshes and instances (marked
  `Dirty` so the renderer uploads them again; when only some instances move, `MarkChanged(indices...)`
  uploads just those). Heavy per-vertex motion is left to the GPU through
  the scene state the renderer reads (`Environment.Wind`, `Material.Sway`).
  * `DayCycle`: turns the sun around the celestial pole; the stars follow it.
  * `Gusts`: varies the wind strength and direction around its initial value.
  * `CloudDrift`: moves the clouds with the wind (`Sky.Clouds.Offset`).
  * `Bobbing`, `Spin`, `PathFollower`: node motions (a boat rocking, sails turning, a ship on a route).
  * `FromTags`: builds the node motions that glTF node extras ask for.

## Node extras read by `FromTags`
| Extra | Motion |
|---|---|
| `"bob": {"height": h, "period": s, "roll": radians}` | Floats up and down with a slight roll |
| `"spin": {"axis": [x, y, z], "speed": radians per second}` | Turns around a local axis |
| `"path": {"points": [[x, y, z], ...], "speed": units per second, "loop": true}` | Moves along the points, facing where it goes (+Z forward); without `loop` it turns around at the end |

Each motion sets the node transform from the transform it had at the start, so a node takes one motion.

## Simulator
```bash
go run ./cmd/simulator scene.glb
go run ./cmd/simulator -day -day-length 300 scene.glb
```
It accepts all the viewer flags (`-walk`, `-night`, `-capture`, `-benchmark`…) and its own:
`-day` (start with the day cycle running), `-day-length` (seconds per day, 120 by default),
`-wind-variation` (0..1) and `-wind-veering` (radians).

| Key | Action |
|---|---|
| `P` | Pause or resume |
| `T` | Start or stop the day cycle |
| `,` `.` | Halve or double the speed of time |

Plus the viewer's controls (see go-renderer's README).
