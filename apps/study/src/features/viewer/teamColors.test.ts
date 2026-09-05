/**
 * teamColors.test.ts — the map and the panel agree, and nobody is given a team they do not
 * have.
 */
import { describe, expect, it } from 'vitest'

import { FIXTURE_REPLAY_DOCUMENT, FIXTURE_SCOREBOARD } from '../replay/fixtures/replayFixture'
import { normalizeReplayDocument } from '../replay/replayNormalize'
import { buildPlayers, groupByTeam } from '../replay/rosterLogic'

import { rosterXUIDs } from './playbackLogic'
import {
  buildRosterColoring,
  teamTokenAt,
  trackInks,
  UNFOCUSED_ALPHA,
  xuidOfSlotAt,
  TEAM_TOKENS,
} from './teamColors'

const doc = normalizeReplayDocument(FIXTURE_REPLAY_DOCUMENT)
const coloring = buildRosterColoring(doc, FIXTURE_SCOREBOARD)

/**
 * A stand-in for `resolveToken`. It has to answer with a real colour NOTATION and not with
 * the token name: `fadeColor` hands back untouched anything it cannot parse, so a stub that
 * returned `'compare-a'` would make the dimming assertions pass for the wrong reason.
 */
const inkOfXUID = (xuid: string) => {
  const token = coloring.tokenOfXUID.get(xuid)
  return token === undefined ? undefined : `rgb(${10 + TEAM_TOKENS.indexOf(token)}, 20, 30)`
}
const ANONYMOUS = 'rgb(90, 90, 90)'

describe('buildRosterColoring', () => {
  it('gives every player of a team the same token', () => {
    const eagle = coloring.groups.find((g) => g.side === 'Eagle')!
    const tokens = new Set(eagle.players.map((p) => coloring.tokenOfXUID.get(p.xuid)))
    expect(tokens.size).toBe(1)
  })

  it('gives two teams two different tokens', () => {
    const [eagle, cobra] = ['Eagle', 'Cobra'].map(
      (side) => coloring.groups.find((g) => g.side === side)!.players[0].xuid,
    )
    expect(coloring.tokenOfXUID.get(eagle)).not.toBe(coloring.tokenOfXUID.get(cobra))
  })

  it('groups the player the archive has no row for on his own, and never in a team', () => {
    const ungrouped = coloring.groups.filter((g) => g.side === null)
    expect(ungrouped).toHaveLength(1)
    expect(ungrouped[0].players.map((p) => p.xuid)).toEqual(['2533274800000005'])
  })

  it('uses the same group order as the roster panel, so the two show the same colour', () => {
    // The panel colours its heading by the group's index in `groupByTeam`; this is that
    // index, computed from the same pair of functions rather than from a parallel rule.
    const groups = groupByTeam(buildPlayers(doc, FIXTURE_SCOREBOARD))
    groups.forEach((group, index) => {
      for (const player of group.players) {
        expect(coloring.tokenOfXUID.get(player.xuid)).toBe(teamTokenAt(index))
      }
    })
  })

  it('addresses players in the same order the digit keys do', () => {
    // A digit that focused a different player from the one on the Nth card would be worse
    // than no shortcut at all.
    expect(coloring.players.map((p) => p.xuid)).toEqual(rosterXUIDs(doc))
  })

  it('cycles the tokens rather than running out on a match with many groups', () => {
    expect(teamTokenAt(TEAM_TOKENS.length)).toBe(TEAM_TOKENS[0])
  })
})

describe('trackInks', () => {
  it('paints every life of a player in that player colour, respawn after respawn', () => {
    const inks = trackInks(doc.tracks, { inkOfXUID, anonymous: ANONYMOUS, focus: null })
    const byXUID = new Map<string, Set<string>>()
    doc.tracks.forEach((track, i) => {
      if (!track.xuid) return
      const seen = byXUID.get(track.xuid) ?? new Set<string>()
      seen.add(inks[i])
      byXUID.set(track.xuid, seen)
    })
    // Eight lives across five players in the fixture, and nobody changes colour on a respawn.
    expect([...byXUID.values()].every((s) => s.size === 1)).toBe(true)
  })

  it('draws a life the film never named in the neutral ink, never in a team colour', () => {
    const anonymousDoc = normalizeReplayDocument({
      ...FIXTURE_REPLAY_DOCUMENT,
      tracks: [{ slot: 9, team: -1, points: [{ t: 0, x: 1, y: 1 }] }],
    })
    expect(trackInks(anonymousDoc.tracks, { inkOfXUID, anonymous: ANONYMOUS, focus: null })).toEqual([
      ANONYMOUS,
    ])
  })

  it('never yields an empty ink — every life that happened is drawn', () => {
    const inks = trackInks(doc.tracks, { inkOfXUID, anonymous: ANONYMOUS, focus: null })
    expect(inks).toHaveLength(doc.tracks.length)
    expect(inks.every((c) => c !== '')).toBe(true)
  })

  it('dims everyone but the focused player, and leaves them all on the map', () => {
    const focus = coloring.players[0].xuid
    const inks = trackInks(doc.tracks, { inkOfXUID, anonymous: ANONYMOUS, focus })
    doc.tracks.forEach((track, i) => {
      if (track.xuid === focus) expect(inks[i]).toBe(inkOfXUID(focus))
      else expect(inks[i]).toContain(String(UNFOCUSED_ALPHA))
    })
    // Context is dimmed, not removed: a fight is two sides.
    expect(inks.some((c) => c === '')).toBe(false)
  })
})

describe('xuidOfSlotAt', () => {
  it('names the owner of a slot at a given frame', () => {
    // Slot 0 is the first life of the first player, alive from frame 0 to 260.
    expect(xuidOfSlotAt(doc.tracks, 0, 100)).toBe('2533274800000001')
  })

  it('answers null once the life holding that slot has ended', () => {
    // A slot is reassigned at every respawn: a map from slot to player, built once, would
    // attribute half a match to the wrong person.
    expect(xuidOfSlotAt(doc.tracks, 0, 400)).toBeNull()
  })

  it('still names the owner inside the lingering window of a mark they left', () => {
    // The trade kill: the shooter died at 260, and their shot is still on screen at 268.
    // Without the look back the mark would lose its colour exactly when a fight resolves.
    expect(xuidOfSlotAt(doc.tracks, 0, 268, 14)).toBe('2533274800000001')
  })

  it('does not reach back further than the window it was given', () => {
    expect(xuidOfSlotAt(doc.tracks, 0, 400, 14)).toBeNull()
  })

  it('answers null for a slot the film never used', () => {
    expect(xuidOfSlotAt(doc.tracks, 42, 100)).toBeNull()
  })
})
