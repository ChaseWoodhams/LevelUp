/**
 * studyApi.test.ts — THE BOUNDARY, EXERCISED WITH FIXTURE PAYLOADS.
 *
 * Everything below runs against a stub `fetch` and the hand-written artifact the viewer
 * already draws, so what is under test is the boundary itself: the version ruling, the
 * normalisation, the roster widening, and the naming of each failure. No server, no canvas,
 * no React.
 *
 * THE FAILURE CASES ARE THE POINT. A replay route that works is one assertion; a replay route
 * that tells apart "unknown identifier", "recorded but never built", "a capture holds the
 * archive" and "the server is not running" is the difference between a study tool and a
 * spinner. Each of those is a separate expectation here because each is a separate screen.
 */
import { describe, expect, it } from 'vitest'

import { FIXTURE_REPLAY_DOCUMENT } from '../replay/fixtures/replayFixture'

import { loadArchivedMatch, toScoreboardRow, type ParticipantRow } from './studyApi'

const MATCH = '000d5950'

const PARTICIPANTS: ParticipantRow[] = [
  { xuid: '2533274800000001', gamertag: 'Aigle-01', team_side: 't0', kills: 14, deaths: 8, assists: 5 },
  { xuid: '2533274800000003', gamertag: 'Cobra-01', team_side: 't1', kills: 10, deaths: 9, assists: 4 },
  // The archive names no team for this one: the row is published with a null, and the viewer
  // is expected to leave him ungrouped rather than guess.
  { xuid: '2533274800000005', gamertag: 'Sans-equipe', team_side: null, kills: null, deaths: null, assists: null },
]

/** A JSON answer, in the shape `getJSON` reads. */
function ok(body: unknown): Response {
  return new Response(JSON.stringify(body), { status: 200, headers: { 'content-type': 'application/json' } })
}

/** An error answer in the app's own contract: `{code, message, retryable}`. */
function fail(status: number, code: string): Response {
  return new Response(JSON.stringify({ code, message: code, retryable: status >= 500 }), { status })
}

/**
 * stubFetch answers each path from a table and RECORDS what was asked.
 *
 * The record is not decoration: one of the properties under test is that a match with no
 * artifact never goes on to ask for its roster.
 */
function stubFetch(routes: Record<string, () => Response>) {
  const calls: string[] = []
  const impl = (async (input: RequestInfo | URL) => {
    const url = String(input)
    calls.push(url)
    const route = Object.entries(routes).find(([suffix]) => url.endsWith(suffix))
    if (!route) throw new TypeError(`fetch failed: no stub for ${url}`)
    return route[1]()
  }) as unknown as typeof fetch
  return { impl, calls }
}

const replayPath = `/matches/${MATCH}/replay`
const participantsPath = `/matches/${MATCH}/participants`

describe('loadArchivedMatch', () => {
  it('normalises the artifact once and widens the roster', async () => {
    const { impl } = stubFetch({
      [replayPath]: () => ok(FIXTURE_REPLAY_DOCUMENT),
      [participantsPath]: () => ok({ participants: PARTICIPANTS }),
    })

    const load = await loadArchivedMatch(MATCH, { fetch: impl })
    expect(load.kind).toBe('ready')
    if (load.kind !== 'ready') return

    // The frontier did its work: the arrays the contract types as nullable are present, and
    // the ones the fixture omits entirely are empty rather than undefined.
    expect(Array.isArray(load.doc.tracks)).toBe(true)
    expect(load.doc.objectives).toEqual([])
    expect(load.doc.geometry).toEqual([])
    expect(load.scoreboard.map((r) => r.xuid)).toEqual(PARTICIPANTS.map((p) => p.xuid))
    expect(load.scoreboard[2].team_side).toBeNull()
  })

  it('refuses an artifact of an unrecognised schema version, and reports which', async () => {
    const { impl, calls } = stubFetch({
      [replayPath]: () => ok({ ...FIXTURE_REPLAY_DOCUMENT, schemaVersion: 99 }),
      [participantsPath]: () => ok({ participants: PARTICIPANTS }),
    })

    const load = await loadArchivedMatch(MATCH, { fetch: impl })
    expect(load).toEqual({ kind: 'unsupported', version: 99 })
    // Nothing was read from a document this viewer cannot read — including its roster.
    expect(calls.some((c) => c.endsWith(participantsPath))).toBe(false)
  })

  it('reads a match with no artifact as an empty state, and asks for nothing more', async () => {
    const { impl, calls } = stubFetch({
      [replayPath]: () => fail(404, 'replay_not_available'),
      [participantsPath]: () => ok({ participants: PARTICIPANTS }),
    })

    expect(await loadArchivedMatch(MATCH, { fetch: impl })).toEqual({ kind: 'no-artifact' })
    expect(calls).toEqual([`/study${replayPath}`])
  })

  it('tells an unknown identifier apart from a missing artifact', async () => {
    const { impl } = stubFetch({ [replayPath]: () => fail(404, 'match_not_found') })
    expect(await loadArchivedMatch(MATCH, { fetch: impl })).toEqual({ kind: 'not-found' })
  })

  it('reads a held archive as busy, not as a fault', async () => {
    const { impl } = stubFetch({ [replayPath]: () => fail(503, 'archive_busy') })
    expect(await loadArchivedMatch(MATCH, { fetch: impl })).toEqual({ kind: 'busy' })
  })

  it('fails the screen when the roster cannot be read — an empty roster would be a claim', async () => {
    const { impl } = stubFetch({
      [replayPath]: () => ok(FIXTURE_REPLAY_DOCUMENT),
      [participantsPath]: () => fail(404, 'replay_not_available'),
    })

    // The same code that means "empty state" on the artifact must NOT mean it here: every
    // player would land in the ungrouped bucket, which is how this viewer says the archive
    // has no row for someone.
    expect(await loadArchivedMatch(MATCH, { fetch: impl })).toMatchObject({ kind: 'failed' })
  })

  it('names an unreachable server rather than hiding it behind a spinner', async () => {
    const impl = (async () => {
      throw new TypeError('Failed to fetch')
    }) as unknown as typeof fetch

    const load = await loadArchivedMatch(MATCH, { fetch: impl })
    expect(load.kind).toBe('failed')
    if (load.kind === 'failed') expect(load.message).toContain('Failed to fetch')
  })

  it('survives an error body that is not the error contract', async () => {
    const { impl } = stubFetch({
      // What a proxy in front of a dead server actually returns.
      [replayPath]: () => new Response('<html>502</html>', { status: 502 }),
    })
    expect(await loadArchivedMatch(MATCH, { fetch: impl })).toMatchObject({ kind: 'failed' })
  })

  it('escapes the identifier it is handed', async () => {
    const { impl, calls } = stubFetch({ '/replay': () => fail(404, 'match_not_found') })
    await loadArchivedMatch('a b/c', { fetch: impl })
    expect(calls[0]).toBe('/study/matches/a%20b%2Fc/replay')
  })
})

describe('toScoreboardRow', () => {
  it('states what it knows, and leaves the rest as gaps', () => {
    const row = toScoreboardRow(PARTICIPANTS[0])
    expect(row).toMatchObject({
      xuid: '2533274800000001',
      gamertag: 'Aigle-01',
      team_side: 't0',
      kills: 14,
      deaths: 8,
      assists: 5,
      // Studying an archive is always a view from outside it: there is no viewer in there.
      is_me: false,
    })
    // A counter the archive does not record is a gap, never a zero — a zero would be a
    // measurement, and the roster card renders the two differently on purpose.
    expect(row.accuracy).toBeNull()
    expect(row.damage_dealt).toBeNull()
    expect(row.rank).toBeNull()
  })

  it('carries an unknown team through as null rather than picking one', () => {
    expect(toScoreboardRow(PARTICIPANTS[2]).team_side).toBeNull()
  })
})
