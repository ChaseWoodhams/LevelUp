# `features/replay/` — copied from `apps/web`, on purpose

Every file in this folder except `fixtures/` and this README is a **verbatim copy** of a
file under `apps/web/src/features/match-replay/`, under a header naming its origin path
and the commit it was copied at.

## Why copied and not imported

There is no workspace in this repository wiring cross-app imports, and `apps/web/**` is
never edited from here — which rules out adding exports there to make a live import work.
Copying was decided up front rather than attempted and abandoned.

The shared modules these files import came across the same way and kept their original
paths, so the import lines are byte-identical to the origin's:

| Path in this app | Origin |
|---|---|
| `src/lib/accessibility/*` | `apps/web/src/lib/accessibility/*` |
| `src/lib/i18n/locale.ts` | `apps/web/src/lib/i18n/locale.ts` |
| `src/components/ui/button.tsx` | `apps/web/src/components/ui/button.tsx` |
| `src/lib/api/types.ts` | extract of `apps/web/src/lib/api/types.ts` |

`src/lib/api/types.ts` is the one that is not whole-file: the origin is 2 500 lines of API
surface and only one corner of it is read here. The declarations it does carry are
reproduced verbatim, and the `Replay*` aliases still resolve through `./generated`, which
`npm run generate-types` builds from the SAME contract as the web app's
(`apps/go-api/api/openapi.yaml`). `replayContract.test.ts` therefore still checks the
nullability frontier against the contract, not against a mirror of it.

## What did NOT come across

`queries.ts` — it is the web app's fetch layer (its API client, its query keys, its app
shell store). The study tool fetches from `study-server`, and that layer is its own work
(`features/archive/`).

## What came across and then left

`ReplayCanvas.tsx` was copied here and has since been **deleted**, replaced by
`features/viewer/StudyReplayCanvas.tsx` — DERIVED from it, not a copy, and carrying no drift
guard. Two things the origin cannot express through its props forced the fork:

1. it paints one colour per TRACK from the chart-series palette, so a player is repainted at
   every respawn; this viewer paints by TEAM, which needs the archive's participants rows;
2. it owns its own playback position, and the study tool drives that position from outside —
   frame stepping, jumps between deaths, keyboard.

The derived file names its origin and the commit it was derived at, so
`git diff <sha> HEAD -- apps/web/src/features/match-replay/ReplayCanvas.tsx` still says what the
origin has learnt about DRAWING. Keeping the unused copy beside it would have been a dead
module with green tests, which is the shape of debt this repository names first in its own list
of anti-patterns. `lib/accessibility/plotlyColorscale.ts` left with it, for the same reason: the
chart-series palette had no other reader here.

## What is copied, kept, and no longer called

Two behaviours of `replayMarkers.ts` are deliberately bypassed by this app's own layers
(`features/viewer/studyDraw.ts`). The files stay byte-identical — a copy is a copy — and the
bypass happens at the call site:

- **The trail.** `drawTracksLayer` strokes one flat polyline per life at a single opacity: it
  shows where somebody has been and cannot say in which direction. This app draws a trail that
  fades with age, over a window the reader picks, and turns the copied one off by handing that
  layer a trailing window of zero (`paintReplay.COPIED_TRAIL_OFF`). Drawing both would put a flat
  line under a graded one, at a length nobody chose.
- **The projectile flights.** `drawProjectilesLayer` paints every flight on the map in ONE ink,
  because the archetype it reads carries no player. `drawGrenadeArcsLayer` paints each in its
  thrower's colour where the film lets the throw and the flight be matched, and in the neutral
  ink where it does not. The copied function is no longer called; it is not dead code of this
  app's making, it is part of a file that is a copy.

Both remain the right thing to read when the origin moves: a change to either upstream is a
change this app's replacements should be measured against.

## Keeping a copy honest

A copy that drifts silently is worse than no copy, so the property is **asserted, not asked
for**: `src/copies.guard.test.ts` checks that every file carrying a `COPIED FILE — origin:`
header is byte-identical to that origin below the header, and that the SHA in the header
really is the origin's last-modifying commit.

A failure there does not mean this app is broken. It means the origin moved. Two rules:

1. **A fix goes upstream first.** If a bug is in copied code, it is in `apps/web` too.
   Fix it there, then re-copy the file here and bump the SHA in its header.
2. **The header SHA is the diffing handle.** `git diff <sha> HEAD -- <origin path>` says
   exactly what the origin has learned since the copy was taken — which is what you read
   before deciding whether the change belongs here as well.

## Colours

The copied components assume two colour systems are in scope, and neither is a literal:
the design-system layout variables (`--border`, `--card`, `--foreground`,
`--muted-foreground`) and the semantic accessibility tokens (`--ac-*`). Both live in
`src/styles/tokens.css`. No hex value and no Tailwind colour class belongs in this folder —
`src/colors.guard.test.ts` enforces that, because the repository's own colour linter is
hardcoded to `apps/web/src` and does not reach this app.

Take the `--ac-*` values from `apps/web/src/lib/accessibility/palettes/default.ts`, **not**
from the `--ac-*` block in `globals.css`: that block is a fallback the web app overwrites at
runtime, and it has already gone stale on at least one token. `copies.guard.test.ts` checks
this app's values against the palette.
