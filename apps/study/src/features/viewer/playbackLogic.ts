/**
 * playbackLogic.ts — DRIVING THE REPLAY, with no canvas and no React in sight.
 *
 * WHAT THIS OWNS AND WHY IT IS SEPARATE. Everything about WHERE the replay is and how it is
 * moving — playing or not, at what speed, at which frame, jumping to the next death, focused
 * on one player — is a state machine over integers. Leaving it inside the canvas component
 * would make it testable only by mounting a canvas, which jsdom does not have; here it is a
 * reducer and a handful of functions over frame indices, and the whole of it is exercised
 * directly.
 *
 * THE FRAME LIVES IN TWO PLACES, ON PURPOSE, AND THIS IS THE ONE SUBTLETY WORTH READING.
 * During playback the canvas advances its own clock at screen cadence — sixty times a second —
 * and publishes it back here at a tenth of that (cf. `StudyReplayCanvas`). Re-rendering eight
 * player cards sixty times a second would spend the whole animation budget on a panel whose
 * numbers barely change. So:
 *
 *   - `frame` is the last position PUBLISHED by the canvas, or the last one COMMANDED here;
 *   - `seekNonce` increments only on a commanded position, and it is the canvas's signal to
 *     snap its own clock to `frame` rather than carry on from where it was.
 *
 * The consequence to know about: a step or a jump issued mid-playback starts from a position
 * up to 150 ms old. It is why both of them PAUSE first — as an editor does — after which the
 * canvas settles and reports its exact frame, and every subsequent step is precise.
 *
 * FRAME ARITHMETIC IS BORROWED, NEVER REWRITTEN. `advanceFrame`, `framesPerSecond` and
 * `trackWindow` already exist in `replayLogic.ts` and are already tested; the rule for this
 * file is that it composes them and adds no second opinion about what a frame is.
 */
import {
  advanceFrame,
  framesPerSecond,
  trackWindow,
} from '../replay/replayLogic'
import type { ReplayDocumentReady } from '../replay/replayNormalize'

/** Playback speeds, as multiples of real match time. `1` follows the film's own clock. */
export const SPEEDS = [0.5, 1, 2, 4] as const
export type Speed = (typeof SPEEDS)[number]

/**
 * How many players the digit keys can address. Eight is the size of a Halo Infinite arena
 * match and the number of digits a hand reaches without moving.
 *
 * A NINTH PLAYER CANNOT BE FOCUSED AT ALL, and that is a real limit rather than a fallback:
 * the roster panel is copied verbatim from `apps/web` and carries no click handler, so there
 * is nowhere else to select from. It does not arise in an arena match; it would in a Big Team
 * one, which this viewer does not claim to serve. The picker in the transport row shows which
 * digit addresses whom, so the boundary is visible rather than discovered.
 */
export const FOCUSABLE_PLAYERS = 8

export interface PlaybackState {
  /** Last frame published by the canvas, or last frame commanded here. */
  frame: number
  playing: boolean
  speed: Speed
  /** xuid of the player the view is focused on; null = everyone. */
  focus: string | null
  /** Bumped on every COMMANDED position: the canvas snaps its clock when it changes. */
  seekNonce: number
}

/**
 * Timeline — the fixed facts about one artifact that the reducer needs.
 *
 * Derived once from the document (`timelineOf`) rather than recomputed per keystroke: a jump
 * across a 600-frame replay must not walk 99 tracks to find out where the deaths are.
 */
export interface Timeline {
  frameCount: number
  /** Frames at which a named life ends — sorted, unique. Cf. `deathFrames`. */
  deaths: number[]
  /** Frames carrying an objective action — sorted, unique. Empty outside objective modes. */
  objectives: number[]
  /** The players the digit keys address, in the document's own roster order. */
  players: string[]
}

export type PlaybackAction =
  | { type: 'toggle' }
  | { type: 'speed'; speed: Speed }
  /** A position the reader chose: the scrubber, or a click on the timeline. */
  | { type: 'seek'; frame: number }
  /** One frame at a time, in either direction. Pauses. */
  | { type: 'step'; delta: number }
  /** To the next or previous death. Pauses. */
  | { type: 'jumpDeath'; direction: 1 | -1 }
  /** To the next or previous objective action. Pauses. */
  | { type: 'jumpObjective'; direction: 1 | -1 }
  /** Focus the Nth player of the roster, 0-based. Out of range = no change. */
  | { type: 'focus'; index: number }
  | { type: 'focusXUID'; xuid: string | null }
  | { type: 'restart' }
  /** The canvas reporting where it has got to. Never bumps the nonce. */
  | { type: 'published'; frame: number }

export function initialPlayback(): PlaybackState {
  return { frame: 0, playing: true, speed: 1, focus: null, seekNonce: 0 }
}

/**
 * timelineOf reads the fixed positions of one document.
 *
 * Cheap enough to call per document and far too expensive to call per keystroke, which is the
 * only reason it is a separate step.
 */
export function timelineOf(doc: ReplayDocumentReady): Timeline {
  return {
    frameCount: doc.frameCount,
    deaths: deathFrames(doc),
    objectives: objectiveFrames(doc),
    players: rosterXUIDs(doc),
  }
}

/**
 * deathFrames gives the frames where somebody died.
 *
 * TWO LIVES ARE NOT DEATHS AND ARE LEFT OUT. A life the film never NAMED (no xuid) is one the
 * death feed did not close — on the reference film, 15 of 105, four of them starting before
 * the match and six of them survivors — so its end is the end of what was recorded, not an
 * event. And a life that runs to the last frame ended with the match, not with a death.
 * Including either would send `,` and `.` to positions where nothing happened.
 */
export function deathFrames(doc: ReplayDocumentReady): number[] {
  const last = doc.frameCount - 1
  const out = new Set<number>()
  for (const track of doc.tracks) {
    if (!track.xuid) continue
    const end = trackWindow(track).end
    if (end < last) out.add(end)
  }
  return [...out].sort((a, b) => a - b)
}

/** objectiveFrames gives the frames carrying an objective action, sorted and deduplicated. */
export function objectiveFrames(doc: ReplayDocumentReady): number[] {
  return [...new Set(doc.objectives.map((o) => o.t))].sort((a, b) => a - b)
}

/**
 * rosterXUIDs is the order the digit keys address.
 *
 * THE FILM'S ROSTER FIRST, because it is an order the document states rather than one this
 * app invented, and it is the same one `buildPlayers` uses — so the digit and the card agree.
 * A player who has lives but no roster entry follows, in order of first appearance.
 */
export function rosterXUIDs(doc: ReplayDocumentReady): string[] {
  const seen = new Set<string>()
  const out: string[] = []
  for (const entry of doc.roster) {
    if (seen.has(entry.xuid)) continue
    seen.add(entry.xuid)
    out.push(entry.xuid)
  }
  for (const track of doc.tracks) {
    if (!track.xuid || seen.has(track.xuid)) continue
    seen.add(track.xuid)
    out.push(track.xuid)
  }
  return out
}

/**
 * advanceBy gives the next playing position after `dtMs` of real time.
 *
 * `framesPerSecond` is what makes 1× the match's own clock rather than a fixed frame rate: it
 * reads `frameIntervalMs` from the artifact, and only falls back to a nominal cadence for the
 * older artifacts that carry no time scale at all. `advanceFrame` owns the wrap at the end.
 *
 * IT TAKES A FRAME AND NOT THE STATE, because the frame it advances is the canvas's own clock
 * — the one that ticks at screen cadence between two publications — and that clock is
 * deliberately not in React state (cf. the header).
 */
export function advanceBy(
  frame: number,
  speed: number,
  dtMs: number,
  doc: ReplayDocumentReady,
): number {
  const fps = framesPerSecond(doc) * speed
  return advanceFrame(frame, (dtMs / 1000) * fps, doc.frameCount)
}

/**
 * lastFrameOf is the highest position the axis carries.
 *
 * `frameCount` counts frames; the axis is indexed from zero, and an empty document must not
 * yield -1. One function rather than the same `Math.max` at every call site, which is where an
 * off-by-one gets to differ between the scrubber's bound and the clamp that guards it.
 */
export function lastFrameOf(frameCount: number): number {
  return Math.max(frameCount - 1, 0)
}

/** clampFrame keeps a position on the axis the document publishes. */
export function clampFrame(frame: number, frameCount: number): number {
  if (!Number.isFinite(frame)) return 0
  return Math.min(Math.max(Math.round(frame), 0), lastFrameOf(frameCount))
}

/**
 * nextStop gives the nearest position of `stops` strictly beyond `frame` in `direction`, or
 * null when there is none.
 *
 * STRICTLY BEYOND, so that pressing the key twice at a death moves on rather than landing on
 * the same frame for ever. Null rather than a wrap: arriving at the last death and being sent
 * back to the first one reads as a bug, and a disabled control says more.
 */
export function nextStop(stops: number[], frame: number, direction: 1 | -1): number | null {
  if (direction === 1) return stops.find((s) => s > frame) ?? null
  for (let i = stops.length - 1; i >= 0; i--) if (stops[i] < frame) return stops[i]
  return null
}

/**
 * playbackReducer applies one action.
 *
 * EVERY DELIBERATE MOVE PAUSES. Stepping a frame or jumping to a death while the replay keeps
 * running would leave the reader looking at a moment they have already left; every editor
 * pauses on a scrub for the same reason. Only the scrubber itself is exempt — dragging it
 * during playback is a legitimate way to skim.
 */
export function playbackReducer(
  state: PlaybackState,
  action: PlaybackAction,
  timeline: Timeline,
): PlaybackState {
  switch (action.type) {
    case 'toggle':
      return { ...state, playing: !state.playing }
    case 'speed':
      return { ...state, speed: action.speed }
    case 'published': {
      // A REPORT IS ALWAYS TAKEN, PAUSED OR NOT, and it cannot undo a seek: the canvas reads
      // its clock at the instant it draws, and a commanded position snaps that clock BEFORE
      // the draw that reports it — so what comes back after a seek is the seeked frame.
      // Refusing reports while paused was the tempting guard and it was wrong in a way the
      // reader feels: pausing at 4x left the scrubber and the clock up to six frames behind
      // the picture, and the next step then snapped the map backwards.
      const frame = clampFrame(action.frame, timeline.frameCount)
      return frame === state.frame ? state : { ...state, frame }
    }
    case 'seek':
      return commit(state, clampFrame(action.frame, timeline.frameCount), state.playing)
    case 'step':
      return commit(state, clampFrame(state.frame + action.delta, timeline.frameCount), false)
    case 'jumpDeath':
      return jump(state, timeline.deaths, action.direction, timeline)
    case 'jumpObjective':
      return jump(state, timeline.objectives, action.direction, timeline)
    case 'focus':
      // Out of range is a no-op rather than a release: the digit keys are bounded by
      // `FOCUSABLE_PLAYERS` where they are read, and a press addressing nobody should leave
      // the reader exactly where they were.
      return focusOn(state, timeline.players[action.index] ?? null)
    case 'focusXUID':
      return { ...state, focus: action.xuid }
    case 'restart':
      return commit(state, 0, true)
  }
}

/** commit records a COMMANDED position: the nonce is what makes the canvas snap to it. */
function commit(state: PlaybackState, frame: number, playing: boolean): PlaybackState {
  return { ...state, frame, playing, seekNonce: state.seekNonce + 1 }
}

/** jump moves to a stop, or stays put — and stays PLAYING when there is nowhere to go. */
function jump(
  state: PlaybackState,
  stops: number[],
  direction: 1 | -1,
  timeline: Timeline,
): PlaybackState {
  const target = nextStop(stops, state.frame, direction)
  if (target === null) return state
  return commit(state, clampFrame(target, timeline.frameCount), false)
}

/**
 * focusOn selects a player, and a second press on the same digit releases them.
 *
 * The toggle is what makes the digit usable without a second key to learn: the hand that
 * pressed 3 to follow someone presses 3 again to let go.
 */
function focusOn(state: PlaybackState, xuid: string | null): PlaybackState {
  if (xuid === null) return state
  return { ...state, focus: state.focus === xuid ? null : xuid }
}
