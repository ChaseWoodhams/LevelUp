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

Three screens, chosen by the URL hash:

| Hash | Screen |
|---|---|
| `#/` | The way in: a field taking a match identifier, short or full form |
| `#/match/<id>` | One archived match, fetched from `study-server` and drawn |
| `#/sample` | The hand-written artifact, drawn through the very same viewer |

**A real match needs the server running**: `go run ./apps/go-api/cmd/study-server` (loopback,
port 8100). The dev server proxies `/study` to it — see `vite.config.ts` for why a proxy rather
than an origin in the client.

`#/sample` needs neither a server nor a captured film. It draws an artifact from
`src/features/replay/fixtures/`: a reconstructed floor with steps, eight lives across five
players, shots with and without a readable heading, grenade throws, keyframe loadouts and
inventories, and the coverage banner. **No real match data is involved** — every number in that
file was written to put a layer on screen. It exists so the viewer can be looked at and
reviewed by somebody who has captured nothing yet.

The archive browser — the table with filters on map, mode, player and date — is separate work.

## The viewer

- **Team colouring.** Players are coloured by team on the map and in the roster panel, from the
  same `compare-a/b/c` comparison tokens: neutral, because studying an archive is always a
  match watched from outside. The colour belongs to the PLAYER, so it survives every respawn;
  a player the archive has no participants row for keeps his own ungrouped bucket and is never
  folded into a team.
- **Transport.** A scrubber with the match clock, play/pause, 0.5×/1×/2×/4× where 1× follows
  the film's own frame interval, single-frame stepping, and jumps between deaths and between
  objective actions.
- **Keyboard.** Space plays and pauses, the arrows step (Shift for a second at a time), `,` and
  `.` move between deaths, `1`–`8` follow a player and `0` releases. A focused control keeps
  only the keys it actually uses — the scrubber its arrows, a button its Space and Enter — so a
  shortcut never takes away the thing it was meant to make faster.
- **Focus.** Following one player dims the others rather than hiding them: a fight is two sides.
  The digits are drawn as buttons carrying their player's name, because the roster panel is a
  verbatim copy and cannot be given numbers from here — a mapping nobody can see is a shortcut
  nobody uses.
- **Schema-version guard.** An artifact whose `schemaVersion` this viewer does not recognise
  renders a message and draws nothing at all — the refusal is structural, not a banner over a
  map (`src/features/archive/schemaVersion.ts`).

## Layout

| Path | What lives there |
|---|---|
| `src/features/replay/` | The replay modules — **copied** from `apps/web`, see its `README.md` |
| `src/features/replay/fixtures/` | The sample artifact, written here |
| `src/features/archive/` | The fetch boundary against `study-server`, and the version guard |
| `src/features/viewer/` | This app's own screen: playback state, keyboard, team colours, canvas |
| `src/lib/`, `src/components/ui/` | Shared modules copied from `apps/web`, same paths as there — plus `ui/controls.tsx`, this app's own |
| `src/styles/tokens.css` | Design-system variables + the `--ac-*` semantic colour tokens |
| `src/app/` | Routing and this app's own chrome, FR + EN |
| `src/**/*.guard.test.ts` | The guard-rails below |

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
- `src/features/archive/schemaVersion.guard.test.ts` — the schema version this viewer claims to
  read is the one `replay.SchemaVersion` actually holds in the Go source. A number copied out of
  a spec would be right on the day it was copied and silently wrong afterwards.
- `src/features/viewer/teamColors.guard.test.ts` — the map and the roster panel read the same
  list of team tokens. The list exists twice because the panel's copy is private to a file
  copied verbatim from `apps/web`; a drift between them would show as two colours for one team
  and nothing red anywhere.
- `src/features/viewer/normalized.guard.test.ts` — no `?.` or `?? []` on an array
  `normalizeReplayDocument` has already filled. Such a check is not redundant, it is a false
  claim, and it is how the frontier's guarantee erodes one caller at a time. The field list is
  read from the normaliser itself, so a field joining or leaving the frontier moves the guard.
- `src/components/ui/controls.guard.test.ts` — the compact control size is written in
  `controls.tsx` and nowhere else. It had reached eight hand-written copies before it was
  centralised; this is what stops the ninth.

Colours follow the same rule as the rest of the fork: **semantic tokens only**, no hex value
and no Tailwind colour class in `features/` or `components/`.
