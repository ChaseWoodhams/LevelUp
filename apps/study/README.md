# `apps/study` — the replay study tool

A separate Vite + React + TypeScript app for watching archived match films: open any match in
the study archive — yours or somebody else's — and watch it play out top-down with team
colouring, trails, facing direction, shots and grenades.

It is **not** part of `apps/web`. Its own `package.json`, its own dev server, its own port
(5174), its own module graph. Nothing here is built, linted or bundled with the web app, and
nothing here edits it.

## Run it

```bash
npm install       # first time — .npmrc pins legacy-peer-deps, same reason as apps/web
npm run dev       # http://localhost:5174
npm run typecheck # tsc -b
npm run test:run  # vitest, one pass
npm run build     # production bundle into dist/
```

`npm run generate-types` rebuilds `src/lib/api/generated.ts` from
`apps/go-api/api/openapi.yaml` — the same contract, and the same command shape, as the web
app's. Run it after a change to the replay document schema on the Go side.

## What is on screen today

A hand-written artifact from `src/features/replay/fixtures/`, drawn on canvas: a reconstructed
floor with steps, eight lives across five players, shots with and without a readable heading,
grenade throws, keyframe loadouts and inventories, and the coverage banner. **No real match data is
involved** — every number in that file was written to put a layer on screen.

The archive browser and the fetch layer against `study-server` are separate work; the fixture
is what lets the viewer be built and reviewed before either exists.

## Layout

| Path | What lives there |
|---|---|
| `src/features/replay/` | The replay view — **copied** from `apps/web`, see its `README.md` |
| `src/features/replay/fixtures/` | The sample artifact, written here |
| `src/lib/`, `src/components/ui/` | Shared modules copied from `apps/web`, same paths as there |
| `src/styles/tokens.css` | Design-system variables + the `--ac-*` semantic colour tokens |
| `src/app/i18n.ts` | This app's own chrome strings, FR + EN |
| `src/*.guard.test.ts` | The guard-rails below |

## Guard-rails

This app sits outside the reach of the repository's own linters and CI job — both are pinned
to `apps/web`. Three properties that would otherwise rest on prose are therefore asserted in
its own test suite:

- `src/copies.guard.test.ts` — every copied file is byte-identical to its origin, and its
  header SHA is really that origin's last-modifying commit. It also checks the `--ac-*`
  values in `styles/tokens.css` against the palette they were extracted from. (It earned its
  keep on the first run: `--ac-divergent-neutral`, this viewer's map-floor colour, had been
  taken from the stale `--ac-*` fallback in `globals.css` instead of the palette.)
- `src/colors.guard.test.ts` — no hex value and no Tailwind colour class in this app's code,
  the rule `tools/lint-no-hardcoded-colors.mjs` enforces for `apps/web/src` only.
- `src/lib/api/generated-types-fresh.guard.test.ts` — `generated.ts` still derives from the
  current `openapi.yaml`, so `replayContract.test.ts` keeps checking the replay document
  against today's contract rather than last month's.

Colours follow the same rule as the rest of the fork: **semantic tokens only**, no hex value
and no Tailwind colour class in `features/` or `components/`.
