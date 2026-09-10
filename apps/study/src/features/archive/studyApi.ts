/**
 * studyApi.ts — THE FETCH BOUNDARY, and the only place in this app that knows the archive is
 * on the other side of a socket.
 *
 * WHAT CROSSES HERE AND NOWHERE ELSE. Three transformations happen at this line and are done
 * with by the time anything renders:
 *
 *   1. the SCHEMA VERSION is ruled on (`parseReplayPayload`) — an artifact this viewer does
 *      not recognise never reaches a drawing module at all;
 *   2. the document crosses the NULLABILITY FRONTIER once (`normalizeReplayDocument`, called
 *      from `parseReplayPayload`), so no `?? []` leaks into rendering code — the reason that
 *      frontier exists in the first place;
 *   3. the archive's six participant columns become the SCOREBOARD ROW the copied roster
 *      logic joins on.
 *
 * WHY A PLAIN `fetch` AND NO QUERY LIBRARY. The web app reaches for TanStack Query because it
 * has an app shell, a cache shared between routes, and refetch policies to honour. This app
 * has one screen reading one immutable artifact off a local read-only server: there is nothing
 * to invalidate and nothing to keep warm. The one behaviour that WOULD have justified a
 * library — retry policy — is a requirement in the negative here (never retry on a loop), and
 * "do not retry" costs no dependency.
 *
 * EVERY FAILURE HAS A NAME. The server's routes answer with the app's own error contract
 * (`{code, message, retryable}`), and the codes are the distinctions the screen needs to make:
 * an unknown identifier is not a missing artifact, and neither is a capture holding the file.
 * Collapsing them into "something went wrong" would send the reader looking in the wrong place
 * every time.
 */
import type { MatchScoreboardRow, ReplayDocument } from '@/lib/api/types'

import { parseReplayPayload, type ReplayPayload } from './schemaVersion'

/**
 * Where study-server is reached. A path, not an origin: `vite.config.ts` proxies it to the
 * server's loopback port in dev, exactly as apps/web proxies `/api` to the Go API. Naming an
 * origin here would hardcode a port into the bundle and lose the same-origin property the
 * proxy gives us.
 */
export const STUDY_API_BASE = '/study'

/**
 * ParticipantRow is one row of `GET /matches/{id}/participants`, field for field.
 *
 * SIX COLUMNS, AND THE SERVER SAYS WHY (cf. `cmd/study-server/participants.go`): the archive
 * records these and no more, and publishing the fifteen other scoreboard fields as nulls
 * would invent a scoreboard it does not have. An unknown value is `null`, never an absent key.
 */
export interface ParticipantRow {
  xuid: string
  gamertag: string
  team_side: string | null
  kills: number | null
  deaths: number | null
  assists: number | null
}

/** The envelope `GET /matches/{id}/participants` answers with. */
interface ParticipantsBody {
  participants: ParticipantRow[]
}

/**
 * MatchSummary is one row of `GET /matches`, and the whole body of `GET /matches/{id}`.
 *
 * Field for field the server's `matchSummary` (`cmd/study-server/archive.go`). Everything but
 * the two identifiers is `omitempty` on the Go side, which is why almost every field here is
 * optional: an archive row whose match stats named no map really does carry no map name, and
 * the browser has to render that row rather than crash on it.
 *
 * `coverage` is the fraction of lives the decoder could NAME, 0..1 (ADR 0006's unit) — and
 * ABSENT when the artifact reported no lives at all, which is "unknown", not "zero".
 */
export interface MatchSummary {
  /** Final game scores for Eagle (t0) and Cobra (t1); absent in older archives. */
  team0_score?: number
  team1_score?: number
  match_id: string
  short_id: string
  played_at?: string
  map_name?: string
  /** The archive's stable key for the map — what a floor calibration is looked up by. */
  map_module?: string
  mode?: string
  playlist?: string
  duration_ms?: number
  /** Whose archiving pass DISCOVERED the match. Provenance, never ownership. */
  source_gamertag?: string
  built_at?: string
  coverage?: number
  named_lives: number
  total_lives: number
  tracks: number
  points: number
  shots: number
  /**
   * Who played. Present on the LIST route, absent on the single-match one — the server says why
   * (`archive.go`): the replay screen has its own roster route, whose failure must fail that
   * screen, while this summary is allowed to be missing.
   */
  participants?: ParticipantRow[]
}

/** The app's error contract, as `internal/api/humacore` writes it. */
interface ApiErrorBody {
  code?: string
  message?: string
}

/**
 * MatchLoad — everything one screen needs to know about one archived match, as ONE value.
 *
 * The screen is a switch over this union and nothing else, which is what keeps the honest
 * distinctions from eroding: "no artifact yet" and "identifier unknown" cannot collapse into
 * a shared empty state by accident, because they are different variants and the compiler
 * counts them.
 */
export type MatchLoad =
  | {
      kind: 'ready'
      doc: ReplayPayloadReady['doc']
      /** Where the MATCH clock's zero sits on the document's axis, in ms; null = not measured. */
      matchClockZeroMs: ReplayPayloadReady['matchClockZeroMs']
      scoreboard: MatchScoreboardRow[]
      /** The archive's row for this match, or null when it could not be read (cf. below). */
      summary: MatchSummary | null
    }
  /** The artifact is of a schema version this viewer does not read. Nothing is drawn. */
  | { kind: 'unsupported'; version: number }
  /** The archive knows this match but holds no artifact for it — recorded, never built. */
  | { kind: 'no-artifact' }
  /** No archived match under that identifier, in either the full or the short form. */
  | { kind: 'not-found' }
  /** A capture is holding the archive. Nothing is broken; coming back later works. */
  | { kind: 'busy' }
  /** Anything else: the server said so, or it could not be reached at all. */
  | { kind: 'failed'; message: string }

type ReplayPayloadReady = Extract<ReplayPayload, { kind: 'ready' }>

/**
 * loadArchivedMatch fetches one match and answers with the state of its screen.
 *
 * THE REQUESTS ARE SEQUENTIAL, AND THAT IS THE DECISION TREE, NOT A MISSED OPTIMISATION.
 * The artifact decides what the screen is: without one there is nothing to draw, and a roster
 * beside an empty map would be a panel about a match the reader cannot watch. Only once the
 * document is in hand — and readable — is there a reason to ask who played, and only once
 * there is a map to put a floor under is there a reason to ask which map it was. On a loopback
 * server those round trips cost less than the branches they save.
 */
export async function loadArchivedMatch(
  matchId: string,
  init: { fetch?: typeof fetch; signal?: AbortSignal } = {},
): Promise<MatchLoad> {
  const doFetch = init.fetch ?? fetch
  let payload: ReplayPayload
  try {
    payload = await getReplay(doFetch, matchId, init.signal)
  } catch (err) {
    return failureOf(err, 'no-artifact')
  }
  if (payload.kind === 'unsupported') return { kind: 'unsupported', version: payload.version }

  let rows: ParticipantRow[]
  try {
    rows = await getParticipants(doFetch, matchId, init.signal)
  } catch (err) {
    // A ROSTER THAT FAILED IS NOT AN EMPTY ROSTER. Rendering the map with no scoreboard would
    // put every player in the ungrouped bucket — the exact shape this viewer uses to say "the
    // archive has no row for this person". A transport failure must never be able to make
    // that claim, so it fails the screen instead.
    return failureOf(err, 'failed')
  }

  return {
    kind: 'ready',
    doc: payload.doc,
    matchClockZeroMs: payload.matchClockZeroMs,
    scoreboard: rows.map(toScoreboardRow),
    summary: await getSummaryOrNull(doFetch, matchId, init.signal),
  }
}

/**
 * getSummaryOrNull reads the archive's row for the match, and DEGRADES where the roster fails.
 *
 * The asymmetry is deliberate and it is about what a missing answer would make the screen say.
 * A missing roster makes the viewer claim eight players have no team; a missing summary costs
 * the floor its calibrated image, and the floor then says "grid" — which is true. So this one
 * is worth a replay the reader can still watch, and the failure is reported rather than
 * swallowed.
 */
async function getSummaryOrNull(
  doFetch: typeof fetch,
  matchId: string,
  signal?: AbortSignal,
): Promise<MatchSummary | null> {
  try {
    return await getJSON<MatchSummary>(doFetch, `/matches/${encodeURIComponent(matchId)}`, signal)
  } catch (err) {
    console.warn(`[study] no archive row for ${matchId}; the floor falls back to the grid`, err)
    return null
  }
}

/** The server's per-request ceiling; larger archives are read in successive pages. */
export const ARCHIVE_PAGE_LIMIT = 1000

/**
 * ArchiveLoad — the browser's screen as one value, on the same principle as `MatchLoad`.
 *
 * `busy` is its own answer rather than a failure because it is the ordinary state of an archive
 * during an hourly capture: nothing is broken, and the reader's move is to come back.
 */
export type ArchiveLoad =
  | { kind: 'ready'; matches: MatchSummary[]; total: number }
  | { kind: 'busy' }
  | { kind: 'changed' }
  | { kind: 'failed'; message: string }

/** The envelope `GET /matches` answers with. */
interface MatchPageBody {
  matches: MatchSummary[] | null
  total: number
}

/**
 * Read all pages before exposing rows to the local filters and sort. Requests are sequential
 * so each releases the archive before the next opens it. A failed page discards the partial
 * result; a changed count or empty intermediate page asks the reader to reload, without retries.
 */
export async function listArchivedMatches(
  init: { fetch?: typeof fetch; signal?: AbortSignal } = {},
): Promise<ArchiveLoad> {
  const doFetch = init.fetch ?? fetch
  try {
    const matches: MatchSummary[] = []
    let total: number | undefined
    do {
      const offset = matches.length === 0 ? '' : `&offset=${matches.length}`
      const body = await getJSON<MatchPageBody>(
        doFetch, `/matches?limit=${ARCHIVE_PAGE_LIMIT}${offset}`, init.signal,
      )
      if (total !== undefined && body.total !== total) {
        return { kind: 'changed' }
      }
      total = body.total
      if (!body.matches?.length && matches.length < total) {
        return { kind: 'changed' }
      }
      matches.push(...(body.matches ?? []))
    } while (matches.length < total)
    return { kind: 'ready', matches, total }
  } catch (err) {
    if (err instanceof StudyApiError && err.code === 'archive_busy') return { kind: 'busy' }
    return { kind: 'failed', message: err instanceof Error ? err.message : String(err) }
  }
}

/** getReplay reads the artifact and rules on its schema version. */
async function getReplay(
  doFetch: typeof fetch,
  matchId: string,
  signal?: AbortSignal,
): Promise<ReplayPayload> {
  const raw = await getJSON<ReplayDocument>(doFetch, `/matches/${encodeURIComponent(matchId)}/replay`, signal)
  return parseReplayPayload(raw)
}

/** getParticipants reads the roster the archiver recorded beside the film. */
async function getParticipants(
  doFetch: typeof fetch,
  matchId: string,
  signal?: AbortSignal,
): Promise<ParticipantRow[]> {
  const body = await getJSON<ParticipantsBody>(
    doFetch,
    `/matches/${encodeURIComponent(matchId)}/participants`,
    signal,
  )
  return body.participants ?? []
}

/**
 * StudyApiError carries the server's own error CODE, not just its status.
 *
 * Two of the three 404s this app can meet mean different things (`match_not_found` against
 * `replay_not_available`), so a status alone cannot tell the screen which one it is holding.
 */
export class StudyApiError extends Error {
  readonly code: string
  readonly status: number

  constructor(code: string, status: number, message: string) {
    super(message)
    this.name = 'StudyApiError'
    this.code = code
    this.status = status
  }
}

/**
 * getJSON performs one request and turns a non-2xx into a `StudyApiError`.
 *
 * `cache: 'no-store'` because the artifact of a given match never changes, but the ARCHIVE
 * does: a match browsed before its film was rebuilt must not be served from a stale 404.
 */
async function getJSON<T>(doFetch: typeof fetch, path: string, signal?: AbortSignal): Promise<T> {
  const res = await doFetch(`${STUDY_API_BASE}${path}`, { signal, cache: 'no-store' })
  if (!res.ok) throw new StudyApiError(await errorCodeOf(res), res.status, `${res.status} on ${path}`)
  return (await res.json()) as T
}

/**
 * errorCodeOf reads the `code` of an error body, and never throws while doing it.
 *
 * A body that is not the error contract at all — a proxy's HTML page, an empty 502 — is a real
 * possibility on the way to a local server, and it must degrade to "unnamed failure" rather
 * than replacing the server's failure with a JSON parse error nobody can act on.
 */
async function errorCodeOf(res: Response): Promise<string> {
  try {
    const body = (await res.json()) as ApiErrorBody
    return typeof body?.code === 'string' ? body.code : ''
  } catch {
    return ''
  }
}

/**
 * failureOf names a thrown failure.
 *
 * `missing` is what a 404 with the `replay_not_available` code means TO THE CALLER: on the
 * artifact it is an empty state, on the roster it is a broken screen — the same code, two
 * readings, so the caller supplies the one that applies.
 */
function failureOf(err: unknown, missing: 'no-artifact' | 'failed'): MatchLoad {
  if (err instanceof StudyApiError) {
    switch (err.code) {
      case 'match_not_found':
        return { kind: 'not-found' }
      case 'replay_not_available':
        return missing === 'no-artifact' ? { kind: 'no-artifact' } : { kind: 'failed', message: err.message }
      case 'archive_busy':
        return { kind: 'busy' }
    }
    return { kind: 'failed', message: err.message }
  }
  // A rejected fetch is the server being absent, not an answer from it — the study server is
  // started by hand, so this is the ordinary first-run failure and deserves its own words.
  return { kind: 'failed', message: err instanceof Error ? err.message : String(err) }
}

/**
 * toScoreboardRow widens an archive row into the shape `rosterLogic.ts` joins on.
 *
 * THE FIELDS THIS FILLS IN ARE NOT INVENTED DATA. `is_me` is FALSE as a statement of fact:
 * studying an archive is always looking at a match from outside it, and there is no viewer in
 * the archive to point at. Every counter the archive does not record is `null`, which is the
 * value the roster component already renders as a gap. `outcome_label` is empty because the
 * archive stores Halo's raw outcome code and turning a code into words is the title adapter's
 * job against a versioned TOML — a label written here would be a French or English string
 * baked into the client, which the repository's first rule forbids. Nothing in the replay view
 * reads it.
 */
export function toScoreboardRow(p: ParticipantRow): MatchScoreboardRow {
  return {
    xuid: p.xuid,
    gamertag: p.gamertag,
    team_side: p.team_side,
    is_me: false,
    rank: null,
    score: null,
    kills: p.kills,
    deaths: p.deaths,
    assists: p.assists,
    shots_fired: null,
    shots_hit: null,
    accuracy: null,
    damage_dealt: null,
    damage_taken: null,
    average_life: null,
    headshot_kills: null,
    max_killing_spree: null,
    perfect_kills: null,
    power_weapon_kills: null,
    melee_kills: null,
    outcome_label: '',
  }
}
