# Common commands — LevelUp
> Cheat-sheet for the current stack: Go backend (`apps/go-api`) + React/Vite frontend (`apps/web`).
> Operational tooling is the `levelup` CLI (`apps/go-api/cmd/levelup`). Make targets live in the
> root `Makefile`. DuckDB access requires CGO (see [Tests](#tests)).

---

## Run the app

```bash
make dev          # Go API (air, :8000) + Vite frontend (:5173) — Ctrl+C stops both
make go-api-dev   # Go API only (air hot-reload)
make web          # Frontend only (Vite, :5173)
make stop         # Stop dev servers (kills by port, API + 5173)
make restart      # stop + dev
```

Open http://localhost:5173 once `make dev` is running.

---

## Build

```bash
make go-api-build   # CGO_ENABLED=1 go build -> apps/go-api/bin/server
make install-web    # npm install in apps/web
make generate-types # TypeScript types from apps/go-api/api/openapi.yaml
make check-types    # tsc -b (typecheck only)
```

---

## `levelup` CLI

Built from `apps/go-api/cmd/levelup`. Run via `go run` (CGO required) or build a binary.
Use `LEVELUP_REPO_ROOT` to point at the data repo (auto-detected if omitted).

```bash
cd apps/go-api
CGO_ENABLED=1 go run ./cmd/levelup <command> [flags]
# Per-command help:
CGO_ENABLED=1 go run ./cmd/levelup <command> --help
```

### Sync (Halo API)

```bash
# Delta sync — new matches only
go run ./cmd/levelup sync-delta --gamertag YourGamertag
go run ./cmd/levelup sync-delta --all --max-matches 25
# flags: --match-type all|matchmaking|custom|local  --rps N  --token-pool-size N

# Full sync — walk last N API matches, insert what's missing (fills gaps)
go run ./cmd/levelup sync-full --gamertag YourGamertag --max-matches 500

# Backfill Xbox achievements (admin one-shot)
go run ./cmd/levelup sync-achievements --all [--dry-run]
```

### Backfill (mostly local, Go-only; CSR/weapons need Halo tokens)

```bash
go run ./cmd/levelup backfill --gamertag X --citations        [--force]
go run ./cmd/levelup backfill --all          --lusr           [--force]
go run ./cmd/levelup backfill --gamertag X --perf             [--force]
go run ./cmd/levelup backfill --gamertag X --engagement-scores
go run ./cmd/levelup backfill --gamertag X --csr             [--force]   # Halo tokens
go run ./cmd/levelup backfill --all          --shared-csr     [--dry-run] # Halo tokens
go run ./cmd/levelup backfill --all          --weapons        [--force]   # film CDN
go run ./cmd/levelup backfill --gamertag X --citations-recompute-all
```

### Backup / restore

```bash
go run ./cmd/levelup backup  --gamertag X [--output-dir D] [--compression-level 9]
go run ./cmd/levelup restore --gamertag X --backup-dir D [--replace] [--dry-run] [--tables T1,T2]
go run ./cmd/levelup restore-csr --gamertag X --backup PATH [--dry-run] [--mode preserve|overwrite]
```

### Metadata / seed / migration

```bash
go run ./cmd/levelup seed career-ranks | citation-mappings | medals | rank-translations
go run ./cmd/levelup seed-demo            # generate anonymized demo data (data/demo/)
go run ./cmd/levelup migrate              # migrate data into the multi-title namespace
go run ./cmd/levelup add-title --name "Halo MCC" [--slug s] [--capabilities matchmaking,media] [--xbox-id X] [--steam-id S]
```

### Media

```bash
go run ./cmd/levelup index-media --gamertag X [--force-rescan] [--buffer-min N]
```

### Diagnostics & ops

```bash
go run ./cmd/levelup healthcheck [--verbose]
go run ./cmd/levelup diagnose --db PATH [--verbose]
go run ./cmd/levelup check-env
go run ./cmd/levelup gate-check [--gamertag X] [--json]
go run ./cmd/levelup compare-db --go-db PATH --python-db PATH [--json]
```

### Prestige — coach grammar tuning analyzer

Read-only analyzer (never opens DBs RW). Produces **recommendations** to adjust the
coach synthesis grammar (`config/coach_advisor/synthesis_grammar.toml`) from Prestige
telemetry (completion rate per grammar metric). Application stays **manual**: a human
reads the report and edits the TOML — no auto-PR, no runtime override.

```bash
# All players of a title (default halo_infinite), text report:
go run ./cmd/prestige-tuning-analyze
# Single player, JSON output:
go run ./cmd/prestige-tuning-analyze --player JGtm --format json
# Custom thresholds (rule: completion < min-completion over >= min-sample accepted coach challenges):
go run ./cmd/prestige-tuning-analyze --min-completion 0.30 --min-sample 50 --source coach
# flags: --format text|json  --player SLUG|GAMERTAG  --title SLUG
#        --min-completion 0..1  --min-sample N  --source coach|user|pilot_mode  --grammar PATH
```

Below `--min-sample`: "insufficient data" (no recommendation on noise). A telemetry
metric absent from the grammar is flagged as an orphan (naming drift / legacy challenge).

### Maintenance (server stopped for ART/alias rebuilds)

```bash
go run ./cmd/levelup rebuild-pme-art --all | --gamertag X   # rebuild player_match_enrichment ART index
go run ./cmd/levelup consolidate-aliases                    # merge xbox_aliases into shared.xuid_aliases
go run ./cmd/levelup recompute-friends [--dry-run]          # recompute is_with_friends across player DBs
go run ./cmd/levelup replay-events --gamertag X             # re-parse highlight events
go run ./cmd/levelup reset-bitmasks                         # reset skill/participants/PVE backfill bits
go run ./cmd/levelup engagement-coefs [--with-scores]      # recompute engagement coefficients
```

### Media paths migration (one-shot, standalone binary)

Converts legacy **absolute** media paths to portable relative `{owner_slug}/{rel}` paths in
`shared_social.duckdb` (`media_files.file_path` / `thumbnail_path`, propagated to the
`media_likes.media_path` PK). Idempotent — already-relative paths are skipped, a broken
thumbnail is nulled out so the next `BackfillThumbnailPaths` repoints it. Run with the
**server stopped** (opens `shared_social.duckdb` RW). Already executed in prod for the
existing titles; kept for future legacy imports that could reintroduce absolute paths.

```bash
go run ./cmd/migrate-media-paths --db data/titles/{slug}/warehouse/shared_social.duckdb [--dry-run]
# flags: --db PATH (required)  --captures-base DIR  --settings app_settings.json  --dry-run
# --captures-base defaults to app_settings.json media_captures_base_dir
```

### Notifications

```bash
go run ./cmd/levelup notify-version --version v1.2.3
go run ./cmd/levelup notify-sync --gamertag X --op sync_delta --duration 120s [--matches N]
```

Full list: `go run ./cmd/levelup help`.

---

## Study tool (outside the app)

A local study aid over an archive of match films, captured before their CDN links expire. It
lives beside the app and shares none of its data: one archive at `data/study/`, written by the
archiver and read by the server.

### `study-archiver` — capture (`cmd/study-archiver`)

```bash
# Archive one match: download its whole film into the chunk cache and build the 2D artifact
go run ./cmd/study-archiver fetch-one --xuid <xuid> <matchId>

# One pass over watchlist.toml at the repo root. Exits when done — run it hourly from the
# OS scheduler, not as a daemon.
go run ./cmd/study-archiver watch --xuid <xuid>

# What is in the archive and what went wrong. No network call, no credential.
go run ./cmd/study-archiver status

# Re-assemble one artifact from the chunks already on disk. Offline; never re-downloads.
go run ./cmd/study-archiver rebuild <matchId>
```

Tracked players: `watchlist.toml` at the repo root (git-ignored; model
`watchlist.example.toml`). Exit codes: 0 archived, 3 skipped for a named reason, 1 failure,
2 usage.

**Checking a decoder change against ground truth.** Every build (`fetch-one` or `rebuild`)
compares the replay with Halo's own match stats: each player's named lives against their
official deaths + 1. The archive keeps the result (`matches.gt_*`,
`participants.replay_named_lives`), and `status` prints it under "Replay vs official match
stats". After changing the decoder, rebuild the archived matches and read that section:
over-named lives must stay at 0 (a life named after the wrong player); missing lives and the
lives gap (lives the replay segmented minus lives the stats imply) are the work left.

### `study-server` — serve (`cmd/study-server`)

```bash
# Read-only HTTP over the archive. Loopback by default: it holds other people's matches.
go run ./cmd/study-server [--addr 127.0.0.1:8100] [--title halo_infinite]

# GET /matches?map=&mode=&player=&from=&to=&min_coverage=&limit=&offset=
# GET /matches/{match_id}/replay        the artifact, byte for byte
# GET /matches/{match_id}/participants  xuid, team_side, gamertag, kills, deaths, assists
```

`{match_id}` accepts either the full id or the short film form. `from`/`to` take a plain
`YYYY-MM-DD` date (the range is half-open, so `from=D&to=D` is the whole of day D) or an
RFC 3339 instant. `min_coverage` is a fraction of 1 (`0.85`, not `85`).

**DuckDB is single-instance-per-file across processes.** The server therefore opens nothing at
startup: it borrows the archive while a request is in flight and releases it when the last one
finishes, so it leaves the file free between bursts and an hourly capture always finds a gap.
While a capture holds the archive the server answers `503 archive_busy` with a `Retry-After` —
that is expected, not a fault.

Both binaries link DuckDB, so they need the UCRT toolchain on Windows (cf. CLAUDE.md).

### `apps/study` — the viewer (Vite + React + TS)

Its own app, its own dev server, its own port. It never shares a build with `apps/web`.

```bash
cd apps/study
npm install            # first time (needs .npmrc: legacy-peer-deps)
npm run dev            # http://localhost:5174
npm run typecheck      # tsc -b
npm run test:run       # vitest, once
npm run build          # production bundle into dist/
npm run generate-types # openapi.yaml -> src/lib/api/generated.ts
```

The replay rendering modules under `src/features/replay/` are **copies** of
`apps/web/src/features/match-replay/`, each carrying its origin path and the commit it was
copied at; the folder's `README.md` says why, and how to keep the copy honest.

The URL hash chooses the screen: `#/` is the way in (a field taking a match identifier, short
or full form), `#/match/<id>` opens an archived match, and `#/sample` draws a hand-written
artifact — no real match data, no server, no captured film — so the viewer can be reviewed by
somebody who has archived nothing yet.

**Opening a real match needs `study-server` running.** The dev server proxies `/study` to
`127.0.0.1:8100`; a page reaching the server's origin directly would be a cross-origin request,
and `study-server` deliberately publishes no CORS headers.

```bash
go run ./apps/go-api/cmd/study-server    # terminal 1
cd apps/study && npm run dev             # terminal 2, then open #/match/<id>
```

An artifact whose `schemaVersion` the viewer does not recognise renders a message and draws
NOTHING — the archive keeps documents built by several versions of the builder, and reading one
with the wrong rules would produce a plausible map that is wrong.

---

## Tests

### Go (see [testing.md](testing.md))

```bash
# Fast, no DuckDB (CGO off)
make go-api-test
# or directly:
cd apps/go-api && CGO_ENABLED=0 go test ./internal/domain/... ./internal/analysis/... ./contracttest/... -count=1

# Full suite with DuckDB (CGO on — needs a C toolchain / MinGW on Windows)
cd apps/go-api && CGO_ENABLED=1 LEVELUP_DEMO_MODE=true go test ./... -timeout 5m -count=1

make go-api-coverage   # coverage report
make go-api-lint       # go vet
```

### Frontend (`apps/web`)

```bash
make test-web        # vitest run
make test-e2e        # Playwright (needs `make dev` running)
make test-e2e-ui     # Playwright UI mode
# or via npm in apps/web:
npm run test:run
npm run test:coverage
npm run lint
```

### Local merge gate (`gate-push`)

```bash
make gate-push               # golangci-lint ratchet + web typecheck/lint + test baseline (~25 min)
```

On some Windows dev machines, git-bash's linker fails to resolve DuckDB's
`libduckdb_static` when building CGO test binaries (`undefined reference
__emutls_v._ZSt11__once_call`), which breaks the test-baseline step of
`make gate-push` even though the code itself is fine — native PowerShell links
correctly. Validated workaround (documented in
`.ai/HANDOFF_POST_LOT2_V73.md`): run `scripts/gate-push.ps1` instead. It
reproduces the same 4 links (Go lint, Go integration tests, web typecheck, web
lint) but produces the `go test -json` output from native PowerShell, then
hands it to `scripts/check_test_baseline.sh tests --from-jsonl <file>` (consumer
mode — parses the JSONL, does not re-run the suite). CI remains the authority;
this is a local-only fallback for that specific environment quirk.

```powershell
powershell -File scripts/gate-push.ps1
```

---

## Environment variables

| Variable | Purpose |
|----------|---------|
| `LEVELUP_REPO_ROOT` | Data repo root (auto-detected if absent) |
| `LEVELUP_API_PORT` | Go API port (default `8000`) |
| `LEVELUP_DEMO_MODE` | Demo mode (used by the test targets) |
| `LEVELUP_NOTIFY_VERSIONS` | Set to `1` to enable version notifications in prod |
| `DISCORD_WEBHOOK_URL` | Discord webhook (overrides `app_settings.json`) |
| `CGO_ENABLED` | Must be `1` for any DuckDB-touching build/test |

---

## Data paths

```
data/
  warehouse/metadata.duckdb         # referentials (maps, playlists, medals)
  warehouse/shared_matches_v2.duckdb # shared matches/medals/events/aliases
  warehouse/shared_pve.duckdb       # Firefight stats
  players/{gamertag}/stats.duckdb   # per-player enrichment
  players/{gamertag}/archive/       # Parquet archives
db_profiles.json                    # player profiles (multi-title)
app_settings.json                   # app settings
.env.local                          # Azure tokens / secrets
```

See [ARCHITECTURE_V6.md](ARCHITECTURE_V6.md) for the full data model.
