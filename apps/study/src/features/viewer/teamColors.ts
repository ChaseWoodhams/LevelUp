/**
 * teamColors.ts — ONE COLOUR PER TEAM, on the map and in the panel, and it is the same one.
 *
 * THE JOIN IS ON XUID AND ON NOTHING ELSE. The film carries no team — `Track.team` is
 * hard-wired to -1, and the artifact says so — so a team can only come from the archive's
 * participants rows, keyed by xuid. A slot is an ORDER inside one film and an index is an
 * order inside one array; neither is an identity, and this project has already published a
 * "player index" that turned out to be its own alphabetical sort. Nothing here joins on a
 * rank.
 *
 * THE COLOUR BELONGS TO THE PLAYER, NOT TO THE LIFE. A slot is reassigned on every respawn —
 * 99 tracks for 8 players on the reference film — so colouring by track would repaint someone
 * every time they came back, and following anyone with the eye would be impossible.
 *
 * COMPARISON TOKENS, NOT ALLY/ENEMY. `compare-a/b/c` say "this is not that" and nothing more.
 * `team-ally` / `team-enemy` would say which side the reader is on, which is a thing this
 * viewer never knows: studying an archive is always a match watched from outside. That is also
 * the rule the copied roster panel already follows, and `teamColors.guard.test.ts` pins the
 * two lists together so the map and the panel cannot drift into different colours.
 *
 * A PLAYER WITH NO PARTICIPANTS ROW IS NOT GIVEN A TEAM. They land in the ungrouped bucket
 * `groupByTeam` already produces, which is a group like any other and gets a colour like any
 * other — what it never gets is somebody else's.
 */
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import type { MatchScoreboardRow } from '@/lib/api/types'

import { isAliveAt, trackWindow } from '../replay/replayLogic'
import type { ReplayDocumentReady, ReplayTrackReady } from '../replay/replayNormalize'
import { buildPlayers, groupByTeam, type ReplayPlayer, type ReplayTeamGroup } from '../replay/rosterLogic'

import { fadeColor } from './fade'

/**
 * The team colours, in the order `groupByTeam` yields its groups.
 *
 * SECOND COPY OF THIS LIST, AND THE LAST ONE ALLOWED. The first is the private `teamColor()`
 * of `ReplayTeams.tsx`, a file copied verbatim from `apps/web` and therefore not editable from
 * here — so the list cannot be centralised today. `teamColors.guard.test.ts` reads that file
 * and fails if the two ever stop agreeing; a third copy owes a shared helper instead.
 */
export const TEAM_TOKENS: readonly SemanticToken[] = ['compare-a', 'compare-b', 'compare-c']

/** How much of its colour a player keeps while somebody else is focused. */
export const UNFOCUSED_ALPHA = 0.18

/** teamTokenAt gives the token of the Nth group, cycling — the roster panel's own rule. */
export function teamTokenAt(groupIndex: number): SemanticToken {
  return TEAM_TOKENS[groupIndex % TEAM_TOKENS.length]
}

/**
 * RosterColoring — who is on the map, in which group, wearing which token.
 *
 * Built from the same `buildPlayers` + `groupByTeam` pair the panel uses, so "the same player"
 * means the same group index on both sides of the screen by construction rather than by
 * agreement between two implementations.
 */
export interface RosterColoring {
  /** Players in the document's roster order — the order the digit keys address. */
  players: ReplayPlayer[]
  /** Teams in `groupByTeam` order; the ungrouped bucket is one of them, always last. */
  groups: ReplayTeamGroup[]
  /** Team token of each player, by xuid. */
  tokenOfXUID: ReadonlyMap<string, SemanticToken>
}

export function buildRosterColoring(
  doc: ReplayDocumentReady,
  scoreboard: MatchScoreboardRow[],
): RosterColoring {
  const players = buildPlayers(doc, scoreboard)
  const groups = groupByTeam(players)
  const tokenOfXUID = new Map<string, SemanticToken>()
  groups.forEach((group, index) => {
    const token = teamTokenAt(index)
    for (const player of group.players) tokenOfXUID.set(player.xuid, token)
  })
  return { players, groups, tokenOfXUID }
}

/** How a track's colour is chosen, once the tokens have been resolved to actual colours. */
export interface TrackInkOptions {
  /** The resolved colour of a player's team, by xuid. */
  inkOfXUID: (xuid: string) => string | undefined
  /**
   * The colour of a life the film never named.
   *
   * IT IS DELIBERATELY NOT A TEAM COLOUR. An unnamed life belongs to nobody the archive can
   * point at — 15 of 105 on the reference film — and painting it in a team's colour would add
   * a player to that team on screen who is not on it in the data. A neutral ink says "this
   * happened, and we cannot say whose it was", which is the whole truth about it.
   */
  anonymous: string
  /** xuid of the focused player, or null when nobody is. */
  focus: string | null
}

/**
 * trackInks gives one colour per track, aligned with `doc.tracks` — the array
 * `drawTracksLayer` indexes.
 *
 * An empty string is how the draw layer is told to skip a track; nothing here produces one,
 * because a life that cannot be attributed is still a life that happened.
 */
export function trackInks(tracks: ReplayTrackReady[], opts: TrackInkOptions): string[] {
  return tracks.map((track) => {
    const ink = (track.xuid ? opts.inkOfXUID(track.xuid) : undefined) ?? opts.anonymous
    return focusInk(ink, track.xuid ?? null, opts.focus)
  })
}

/**
 * focusInk holds a colour back unless it belongs to the focused player.
 *
 * ONE FUNCTION FOR BOTH LAYERS. Trails and shots are drawn by different modules with different
 * signatures, and both have to answer the focus the same way — a shot that stayed bright while
 * its shooter's trail dimmed would point at somebody who is not there.
 */
export function focusInk(ink: string, xuid: string | null, focus: string | null): string {
  if (focus === null || xuid === focus) return ink
  return fadeColor(ink, UNFOCUSED_ALPHA)
}

/**
 * xuidOfSlotAt says who owned a slot at a frame, or owned it in the `hold` frames just before.
 *
 * WHY A LOOKUP AND NOT A MAP FROM SLOT TO PLAYER. A slot is reassigned at every respawn, so
 * "slot 4" names different people at different moments of the same match. A shot carries a
 * slot and a frame, and only the pair identifies a shooter; a slot-to-player map built once
 * would attribute the first half of a match to whoever held the slot last.
 *
 * WHY THE LOOK BACK, AND WHY IT IS BOUNDED. Point events linger on the map for a moment after
 * they happen — 1.4 s for a shot — and the draw layer asks for the shooter's colour at the
 * CURRENT frame, not at the frame of the shot. Without the look back, every trade kill would
 * lose its colour: the shooter died inside the lingering window, so no life holds the slot any
 * more and the mark would fall to the neutral fallback. `hold` is the lingering window itself,
 * which is far shorter than a respawn (~8 s measured), so the life it finds is the one that
 * fired.
 *
 * Returns null when nobody owned that slot in that window — the honest answer, which the draw
 * layer renders in its neutral fallback colour rather than in somebody's team.
 */
export function xuidOfSlotAt(
  tracks: ReplayTrackReady[],
  slot: number,
  frame: number,
  hold = 0,
): string | null {
  let recent: { xuid: string; end: number } | null = null
  for (const track of tracks) {
    if (track.slot !== slot || !track.xuid) continue
    if (isAliveAt(track, frame)) return track.xuid
    const end = trackWindow(track).end
    if (end >= frame - hold && end < frame && (recent === null || end > recent.end)) {
      recent = { xuid: track.xuid, end }
    }
  }
  return recent?.xuid ?? null
}
