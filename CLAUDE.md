# go-simulator

A plugin-based simulation engine: a neutral world (entities, fields, environment), a host that
runs plugins (separate programs, any language) on clocks and stages from a rules file, a binary
protocol, and two programs: `cmd/simulate` (headless, writes results and objectives) and
`cmd/simulator` (live, in go-renderer's viewer). Start with `docs/3-how-it-fits-together.md`.

## Commands

```bash
go vet ./...
go test ./...
go run ./cmd/simulate -rules rules.yaml -output results/
go run ./cmd/simulator -rules rules.yaml scene.glb   # needs a display (go-renderer, cgo)
```

Sibling checkouts are required: `go.mod` replaces the akrck02 modules with `../go-simulator`,
`../go-renderer` and the like, so clone them next to this one.

## Rules

- **Generic only.** This project knows nothing about any particular world, game or story. Never put
  names, places, dates, numbers or settings from a user's own project in code, comments, tests,
  examples, docs or commit messages. Examples use made-up places (like the example island in the docs)
  and round, neutral numbers.
- **Readable code.** Full-word names (no acronyms, no one-letter variables), small functions that do
  one thing, early returns instead of nested branches, comments that say *why*. Match the code
  around you.
- **Tests with every change.** `go vet ./...` and `go test ./...` must pass before committing.
- **Docs live in `docs/`**, written for people: explanatory, friendly, with 2D images where they
  help. Keep the README short and point to `docs/`. Update the docs in the same commit as the
  behaviour they describe.
- **AI disclosure.** Commits written with an AI assistant end with a `Co-Authored-By: Claude …`
  line, and the *AI disclosure* section of the README stays true. The
  code is pending human review: say so, don't overclaim.
- **Git.** Commit only when asked; never push, force-push, rewrite history or delete branches or
  the repository without the author's explicit permission. Do not change the global git config.

## Engine rules

- The engine stays neutral: no plugin's vocabulary (cities, plants, rain…) in `world`, `host` or
  `protocol`. Plugins reach the world only through the seven changes.
- A protocol change is a version change: update `docs/5-reference/2-protocol.md` and every plugin with it.
