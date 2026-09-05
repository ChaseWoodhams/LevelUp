/**
 * browserLogic.test.ts — the browser's rules, against fixture rows and nothing else.
 *
 * No server, no fetch, no React: the point of putting these rules in pure functions is that the
 * cases below can be the awkward ones. An archive is browsed to FIND something, so every wrong
 * answer here hides a match from its reader — a filter that excludes on an empty value, a
 * coverage floor that reads "unknown" as zero, a sort that shuffles rows between renders.
 */
import { describe, expect, it } from 'vitest'

import type { MatchScoreboardRow } from '@/lib/api/types'

import { groupByTeam, type ReplayPlayer } from '../replay/rosterLogic'

import {
  DEFAULT_SORT,
  EMPTY_FILTER,
  facetsOf,
  filterMatches,
  sortMatches,
  archiveTeamsOf,
  type ArchiveFilter,
} from './browserLogic'
import type { MatchSummary, ParticipantRow } from './studyApi'

function player(xuid: string, gamertag: string, side: string | null, kills: number | null): ParticipantRow {
  return { xuid, gamertag, team_side: side, kills, deaths: null, assists: null }
}

function match(over: Partial<MatchSummary> & { match_id: string }): MatchSummary {
  return {
    short_id: over.match_id.slice(0, 8),
    named_lives: 0,
    total_lives: 0,
    tracks: 0,
    points: 0,
    shots: 0,
    ...over,
  }
}

const CLIFF = match({
  match_id: 'aaaa1111-0000-4000-8000-000000000001',
  played_at: '2026-05-19T20:15:00Z',
  map_name: 'Cliffhanger',
  mode: 'Slayer',
  coverage: 0.857,
  participants: [player('1', 'JGtm', 't0', 15), player('2', 'Rival', 't1', 9)],
})

const STREETS = match({
  match_id: 'bbbb2222-0000-4000-8000-000000000002',
  played_at: '2026-05-20T20:15:00Z',
  map_name: 'Streets',
  mode: 'CTF',
  coverage: 0.4,
  participants: [player('2', 'Rival', 't0', 12), player('3', 'Third', 't1', 11)],
})

/** The row that matters most: built, but the artifact reported no lives, so coverage is UNKNOWN. */
const AQUARIUS = match({
  match_id: 'cccc3333-0000-4000-8000-000000000003',
  played_at: '2026-05-21T20:15:00Z',
  map_name: 'Aquarius',
  mode: 'Slayer',
  participants: [player('1', 'JGtm', 't0', 8), player('9', 'Inconnu', null, null)],
})

/** And the row nothing is known about: no date, no map, no mode, no roster. */
const BARE = match({ match_id: 'dddd4444-0000-4000-8000-000000000004' })

const ROWS = [CLIFF, STREETS, AQUARIUS, BARE]

const filter = (over: Partial<ArchiveFilter> = {}): ArchiveFilter => ({ ...EMPTY_FILTER, ...over })
const ids = (rows: MatchSummary[]) => rows.map((r) => r.map_name ?? '(unknown)')

describe('filterMatches', () => {
  it('narrows on nothing when nothing is chosen', () => {
    expect(filterMatches(ROWS, EMPTY_FILTER)).toHaveLength(ROWS.length)
  })

  it('narrows on map and mode, ignoring case and stray space', () => {
    expect(ids(filterMatches(ROWS, filter({ map: '  cliffhanger ' })))).toEqual(['Cliffhanger'])
    expect(ids(filterMatches(ROWS, filter({ mode: 'SLAYER' })))).toEqual(['Cliffhanger', 'Aquarius'])
  })

  it('narrows on a player by gamertag or by xuid', () => {
    expect(ids(filterMatches(ROWS, filter({ player: 'rival' })))).toEqual(['Cliffhanger', 'Streets'])
    expect(ids(filterMatches(ROWS, filter({ player: '3' })))).toEqual(['Streets'])
  })

  it('matches an xuid exactly — an identifier is not folded or trimmed into another one', () => {
    expect(filterMatches(ROWS, filter({ player: '33' }))).toEqual([])
  })

  it('leaves a match with no recorded roster out of a player filter, not into it', () => {
    expect(ids(filterMatches(ROWS, filter({ player: 'JGtm' })))).toEqual(['Cliffhanger', 'Aquarius'])
  })

  it('reads a bare date as a whole day, so from = to is that day', () => {
    expect(ids(filterMatches(ROWS, filter({ from: '2026-05-20', to: '2026-05-20' })))).toEqual(['Streets'])
  })

  it('reads a range as inclusive at both ends', () => {
    expect(ids(filterMatches(ROWS, filter({ from: '2026-05-19', to: '2026-05-20' })))).toEqual([
      'Cliffhanger',
      'Streets',
    ])
  })

  it('leaves a match with no start time out of any date range', () => {
    expect(filterMatches(ROWS, filter({ from: '2000-01-01', to: '2100-01-01' })).includes(BARE)).toBe(
      false,
    )
  })

  it('applies a coverage floor, and never lets an UNKNOWN coverage satisfy one', () => {
    expect(ids(filterMatches(ROWS, filter({ minCoverage: 0.5 })))).toEqual(['Cliffhanger'])
    // Aquarius has no coverage at all: the artifact reported no lives. That is not a zero, and
    // a floor of zero must not admit it either.
    expect(filterMatches(ROWS, filter({ minCoverage: 0 })).includes(AQUARIUS)).toBe(false)
  })

  it('applies every part of the filter at once', () => {
    const narrow = filter({ mode: 'Slayer', player: 'JGtm', from: '2026-05-19', minCoverage: 0.5 })
    expect(ids(filterMatches(ROWS, narrow))).toEqual(['Cliffhanger'])
  })
})

describe('sortMatches', () => {
  it('puts the most recent first by default', () => {
    expect(ids(sortMatches(ROWS, DEFAULT_SORT))[0]).toBe('Aquarius')
  })

  it('reverses on direction', () => {
    const asc = sortMatches(ROWS, { key: 'played_at', direction: 'asc' })
    expect(ids(asc)[0]).toBe('Cliffhanger')
  })

  it('sorts text by label and numbers by value', () => {
    expect(ids(sortMatches(ROWS, { key: 'map_name', direction: 'asc' })).slice(0, 3)).toEqual([
      'Aquarius',
      'Cliffhanger',
      'Streets',
    ])
    const byCoverage = sortMatches(ROWS, { key: 'coverage', direction: 'desc' })
    expect(ids(byCoverage).slice(0, 2)).toEqual(['Cliffhanger', 'Streets'])
  })

  it('sinks the rows with no value for the column, whichever way it is sorted', () => {
    for (const direction of ['asc', 'desc'] as const) {
      const sorted = sortMatches(ROWS, { key: 'coverage', direction })
      expect(sorted.slice(-2).every((r) => r.coverage === undefined)).toBe(true)
    }
  })

  it('is stable and breaks ties the same way every time', () => {
    // Two matches on the same map, sorted by map: only the id can separate them, and it must
    // separate them identically on every render — a row that moves is a row clicked by mistake.
    const twin = match({ ...CLIFF, match_id: 'aaaa1111-0000-4000-8000-000000000000' })
    const once = sortMatches([CLIFF, twin], { key: 'map_name', direction: 'asc' })
    const again = sortMatches([twin, CLIFF], { key: 'map_name', direction: 'asc' })
    expect(once.map((r) => r.match_id)).toEqual(again.map((r) => r.match_id))
  })

  it('does not disturb the rows it was given', () => {
    const before = [...ROWS]
    sortMatches(ROWS, { key: 'map_name', direction: 'asc' })
    expect(ROWS).toEqual(before)
  })
})

describe('facetsOf', () => {
  it('lets the reader filter by a player whose gamertag was not recorded', () => {
    const unnamed = match({ ...CLIFF, participants: [player('42', '', 't0', 3)] })
    const rows = [unnamed, STREETS]
    expect(facetsOf(rows).players).toEqual(['42', 'Rival', 'Third'])
    expect(filterMatches(rows, filter({ player: '42' }))).toEqual([unnamed])
  })

  it('offers exactly what the archive holds, sorted and without repeats', () => {
    const facets = facetsOf(ROWS)
    expect(facets.maps).toEqual(['Aquarius', 'Cliffhanger', 'Streets'])
    expect(facets.modes).toEqual(['CTF', 'Slayer'])
    expect(facets.players).toEqual(['Inconnu', 'JGtm', 'Rival', 'Third'])
  })

  it('offers nothing for an empty archive rather than a fixed list of maps', () => {
    expect(facetsOf([])).toEqual({ maps: [], modes: [], players: [] })
  })
})

describe('archiveTeamsOf', () => {
  it('groups the players by side', () => {
    const teams = archiveTeamsOf(CLIFF)
    expect(teams.map((t) => t.side)).toEqual(['t0', 't1'])
  })

  it('keeps the players the archive named no team for in their own bucket, last', () => {
    const teams = archiveTeamsOf(AQUARIUS)
    expect(teams.map((t) => t.side)).toEqual(['t0', null])
    expect(teams[1].players.map((p) => p.gamertag)).toEqual(['Inconnu'])
  })

  it('answers nothing for a match whose roster was never recorded', () => {
    expect(archiveTeamsOf(BARE)).toEqual([])
  })

  /**
   * THE GUARD THAT MATTERS: the browser's chips and the replay's roster colour the Nth group
   * with the Nth token, so the two must produce the same groups in the same order or a player
   * changes colour between the table and the map.
   */
  it('groups in the same order the replay view does', () => {
    const board = (xuid: string, side: string | null): MatchScoreboardRow =>
      ({ xuid, gamertag: xuid, team_side: side }) as MatchScoreboardRow
    const players: ReplayPlayer[] = AQUARIUS.participants!.map((p, i) => ({
      xuid: p.xuid,
      board: board(p.xuid, p.team_side),
      lives: [],
      colorIndex: i,
    }))

    expect(archiveTeamsOf(AQUARIUS).map((t) => t.side)).toEqual(groupByTeam(players).map((g) => g.side))
  })
})
