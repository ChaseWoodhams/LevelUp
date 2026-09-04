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
shell store). The study tool fetches from `study-server`, and that layer is its own work.

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
