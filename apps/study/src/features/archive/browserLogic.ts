/**
 * browserLogic.ts — FINDING THE MATCH TO STUDY: what a filter means, what an order means, and
 * what a row is worth looking at.
 *
 * PURE, AND THAT IS THE POINT OF THE FILE. Everything here takes rows and gives rows. The
 * browser's table has one job — narrowing a growing archive down to the match somebody wants to
 * watch — and every rule it does that by is stated here, against fixture rows, with no network,
 * no React and no server. That is the seam the ticket asks for, and it is also where the
 * mistakes live: an empty filter that quietly excludes everything, a coverage floor that treats
 * "unknown" as zero, a sort that reorders rows the reader cannot see.
 *
 * COVERAGE IS THE COLUMN THE BROWSER EXISTS FOR. It is the share of lives the decoder could
 * NAME — how much of the match the artifact can attribute to a player — and it separates a
 * match worth studying from one whose data is sparse. It is deliberately shown BEFORE the match
 * is opened, and `undefined` is "unknown" and never zero: an artifact that reported no lives at
 * all has no coverage figure, and a floor of any value must not admit it (the server's SQL
 * takes the same position, in as many words).
 */
import type { MatchSummary, ParticipantRow } from './studyApi'

/**
 * ArchiveFilter — the reader's narrowing, as plain values.
 *
 * Empty string and null both mean "do not narrow on this". A single meaning for "no filter"
 * matters more than it looks: the browser's controls hand back `''` when cleared, and a rule
 * that treated an empty map name as a map nobody played on would empty the table.
 */
export interface ArchiveFilter {
  map: string
  mode: string
  /** A gamertag or an xuid: the archive identifies by one and people type the other. */
  player: string
  /** Inclusive, as `YYYY-MM-DD`. */
  from: string
  /** Inclusive of the whole day, as `YYYY-MM-DD` — cf. `withinDates`. */
  to: string
  /** A fraction of 1 (0.85, not 85), matching ADR 0006's unit and the server's own. */
  minCoverage: number | null
}

export const EMPTY_FILTER: ArchiveFilter = {
  map: '',
  mode: '',
  player: '',
  from: '',
  to: '',
  minCoverage: null,
}

/** Which column an archive is ordered by. */
export type SortKey = 'played_at' | 'map_name' | 'mode' | 'coverage'

export interface ArchiveSort {
  key: SortKey
  direction: 'asc' | 'desc'
}

/** Newest first: what a reader wants before they have said anything. */
export const DEFAULT_SORT: ArchiveSort = { key: 'played_at', direction: 'desc' }

/** filterMatches keeps the rows that satisfy every part of the filter. */
export function filterMatches(rows: readonly MatchSummary[], filter: ArchiveFilter): MatchSummary[] {
  return rows.filter(
    (row) =>
      sameText(row.map_name, filter.map) &&
      sameText(row.mode, filter.mode) &&
      hasPlayer(row, filter.player) &&
      withinDates(row, filter) &&
      meetsCoverage(row, filter.minCoverage),
  )
}

/** sameText compares a row's value against a chosen one, ignoring case and surrounding space. */
function sameText(value: string | undefined, chosen: string): boolean {
  const want = chosen.trim().toLowerCase()
  if (want === '') return true
  return (value ?? '').trim().toLowerCase() === want
}

/**
 * hasPlayer matches on EITHER key, because the archive identifies players by xuid and a gamertag
 * is the only one of the two anybody types. Case-insensitive on the gamertag alone: an xuid has
 * no case to fold.
 *
 * A row whose roster the archive never recorded matches nobody — which is right, and is not the
 * same as matching everybody.
 */
function hasPlayer(row: MatchSummary, player: string): boolean {
  const want = player.trim()
  if (want === '') return true
  const folded = want.toLowerCase()
  return (row.participants ?? []).some(
    (p) => p.xuid === want || (p.gamertag ?? '').trim().toLowerCase() === folded,
  )
}

/**
 * withinDates applies the range.
 *
 * A BARE DATE IS A DAY, NOT AN INSTANT, and `to` therefore includes the whole of its day — the
 * server's own rule, and the reason `from=D&to=D` means "that day" rather than nothing at all.
 * The comparison is on the ISO date prefix, so it never depends on the reader's time zone: the
 * archive stores UTC, and a match played at 23:40 UTC must not move to the next day because the
 * browser happens to be east of Greenwich.
 */
function withinDates(row: MatchSummary, filter: ArchiveFilter): boolean {
  const from = filter.from.trim()
  const to = filter.to.trim()
  if (from === '' && to === '') return true
  const day = (row.played_at ?? '').slice(0, 10)
  // A match whose stats named no start time cannot satisfy a date range, and is not smuggled
  // through one: "unknown" is not "inside".
  if (day === '') return false
  if (from !== '' && day < from) return false
  if (to !== '' && day > to) return false
  return true
}

/**
 * meetsCoverage applies the floor.
 *
 * AN UNKNOWN COVERAGE NEVER SATISFIES A FLOOR, whatever the floor is. `undefined` here means the
 * artifact reported no lives at all — there was nothing to attach anything to — which is a
 * different fact from "nothing was attached", and only the second one is a coverage of zero.
 * SQL's three-valued logic gives the server the same answer for free; this is that answer
 * written out.
 */
function meetsCoverage(row: MatchSummary, floor: number | null): boolean {
  if (floor === null) return true
  return row.coverage !== undefined && row.coverage >= floor
}

/**
 * sortMatches orders the rows, stably.
 *
 * STABLE, AND TIE-BROKEN ON THE MATCH ID. Two matches played in the same second, or on the same
 * map, must not swap places between two renders of the same table — a row that moves while
 * being clicked is a row the reader opens by accident.
 *
 * A MISSING VALUE SINKS, in both directions. A match whose stats named no map is not "the first
 * map alphabetically" and not "the last"; it is unknown, and it belongs at the end of the list
 * either way rather than heading a table the reader is scanning.
 */
export function sortMatches(rows: readonly MatchSummary[], sort: ArchiveSort): MatchSummary[] {
  const sign = sort.direction === 'asc' ? 1 : -1
  return [...rows].sort((a, b) => {
    const known = rank(a, sort.key) - rank(b, sort.key)
    if (known !== 0) return known
    const by = compare(a, b, sort.key) * sign
    return by !== 0 ? by : a.match_id.localeCompare(b.match_id)
  })
}

/** rank sinks the rows with no value for the sorted column, whichever way the column is sorted. */
function rank(row: MatchSummary, key: SortKey): number {
  return valueOf(row, key) === undefined ? 1 : 0
}

function valueOf(row: MatchSummary, key: SortKey): string | number | undefined {
  switch (key) {
    case 'played_at':
      return row.played_at
    case 'map_name':
      return row.map_name
    case 'mode':
      return row.mode
    case 'coverage':
      return row.coverage
  }
}

function compare(a: MatchSummary, b: MatchSummary, key: SortKey): number {
  const left = valueOf(a, key)
  const right = valueOf(b, key)
  if (left === undefined || right === undefined) return 0
  if (typeof left === 'number' && typeof right === 'number') return left - right
  return String(left).localeCompare(String(right))
}

/** The distinct values the loaded archive actually contains, for the filter controls. */
export interface ArchiveFacets {
  maps: string[]
  modes: string[]
  players: string[]
}

/**
 * facetsOf lists what is actually IN the archive, sorted, without repeats.
 *
 * The filters are built from this rather than from a fixed list of Halo maps: a control offering
 * a map nobody has captured is a control that can only ever empty the table, and one missing a
 * map that IS in the archive hides matches from the reader.
 */
export function facetsOf(rows: readonly MatchSummary[]): ArchiveFacets {
  const maps = new Set<string>()
  const modes = new Set<string>()
  const players = new Set<string>()
  for (const row of rows) {
    if (row.map_name) maps.add(row.map_name)
    if (row.mode) modes.add(row.mode)
    for (const p of row.participants ?? []) {
      players.add(p.gamertag || p.xuid)
    }
  }
  const sorted = (s: Set<string>) => [...s].sort((a, b) => a.localeCompare(b))
  return { maps: sorted(maps), modes: sorted(modes), players: sorted(players) }
}

/** One side of a match, as the browser shows it: who was on it. */
export interface ArchiveTeam {
  /** The archive's own `team_side` label, or null for players it named no team for. */
  side: string | null
  players: ParticipantRow[]
}

/**
 * archiveTeamsOf groups a row's players by side, IN THE SAME ORDER THE REPLAY VIEW USES.
 *
 * That order is what makes the chips' colours mean the same thing on both screens: the replay
 * paints the Nth group with the Nth comparison token (`teamColors.teamTokenAt`), so a browser
 * that grouped in another order would show a player in one colour in the table and another on
 * the map. The rule — sides sorted by label, the team-less bucket last — is `rosterLogic`'s
 * `groupByTeam`, and `browserLogic.test.ts` pins the two together against the same input.
 */
export function archiveTeamsOf(row: MatchSummary): ArchiveTeam[] {
  const bySide = new Map<string, ArchiveTeam>()
  for (const p of row.participants ?? []) {
    const side = p.team_side ?? null
    const key = side ?? ''
    let team = bySide.get(key)
    if (!team) {
      team = { side, players: [] }
      bySide.set(key, team)
    }
    team.players.push(p)
  }
  // '￿' sorts after every real label, which puts the team-less bucket last — the same
  // trick, and the same outcome, as `groupByTeam`.
  return [...bySide.values()].sort((a, b) => (a.side ?? '￿').localeCompare(b.side ?? '￿'))
}
