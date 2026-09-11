/**
 * playbackLogic.test.ts — DRIVING THE REPLAY, WITHOUT A CANVAS.
 *
 * The whole point of the module under test is that none of this needs a rendering surface:
 * jsdom has no 2D context, so anything only reachable through a mounted canvas is anything
 * untested. Every behaviour a reader can feel — the pause on a step, the stop at the last
 * death, the second press that releases a focused player — is an assertion over integers here.
 */
import { describe, expect, it } from 'vitest'

import { FIXTURE_REPLAY_DOCUMENT } from '../replay/fixtures/replayFixture'
import { normalizeReplayDocument } from '../replay/replayNormalize'

import {
  advanceBy,
  clampFrame,
  deathFrames,
  initialPlayback,
  nextStop,
  objectiveFrames,
  playbackReducer,
  rosterXUIDs,
  timelineOf,
  type PlaybackAction,
  type PlaybackState,
} from './playbackLogic'

const doc = normalizeReplayDocument(FIXTURE_REPLAY_DOCUMENT)
const timeline = timelineOf(doc)

/** run applies a sequence of actions from the initial state. */
function run(...actions: PlaybackAction[]): PlaybackState {
  return actions.reduce((s, a) => playbackReducer(s, a, timeline), initialPlayback())
}

describe('the fixed positions of a document', () => {
  it('reads deaths as the ends of NAMED lives that are not the end of the match', () => {
    // Two lives run to the last frame — a survivor is not a death — and every life in the
    // fixture carries a xuid, so the rest are all deaths.
    expect(deathFrames(doc)).toEqual([260, 300, 420, 480, 560])
  })

  it('leaves an unnamed life out: the death feed never closed it', () => {
    const anonymous = normalizeReplayDocument({
      ...FIXTURE_REPLAY_DOCUMENT,
      tracks: (FIXTURE_REPLAY_DOCUMENT.tracks ?? []).map((t) => ({ ...t, xuid: undefined })),
    })
    expect(deathFrames(anonymous)).toEqual([])
  })

  it('has no objective stops on a match with no objective actions', () => {
    expect(objectiveFrames(doc)).toEqual([])
  })

  it('sorts and deduplicates objective stops', () => {
    const withObjectives = normalizeReplayDocument({
      ...FIXTURE_REPLAY_DOCUMENT,
      objectives: [
        { t: 300, xuid: '2533274800000001', stat: 'flag_captures', timeMs: 30_000 },
        { t: 120, xuid: '2533274800000003', stat: 'flag_grabs', timeMs: 12_000 },
        { t: 300, xuid: '2533274800000004', stat: 'flag_returns', timeMs: 30_050 },
      ],
    })
    expect(objectiveFrames(withObjectives)).toEqual([120, 300])
  })

  it('addresses players in the film roster order, which is the roster panel order too', () => {
    expect(rosterXUIDs(doc)).toEqual((FIXTURE_REPLAY_DOCUMENT.roster ?? []).map((r) => r.xuid))
  })
})

describe('nextStop', () => {
  const stops = [10, 20, 30]

  it('moves strictly past the current frame, so a repeat press does not stick', () => {
    expect(nextStop(stops, 20, 1)).toBe(30)
    expect(nextStop(stops, 20, -1)).toBe(10)
  })

  it('gives null at either end rather than wrapping round', () => {
    expect(nextStop(stops, 30, 1)).toBeNull()
    expect(nextStop(stops, 10, -1)).toBeNull()
    expect(nextStop([], 5, 1)).toBeNull()
  })
})

describe('clampFrame', () => {
  it('keeps a position on the axis the document publishes', () => {
    expect(clampFrame(-4, 600)).toBe(0)
    expect(clampFrame(9_999, 600)).toBe(599)
    expect(clampFrame(12.6, 600)).toBe(13)
  })

  it('answers 0 for a frame that is not a number', () => {
    expect(clampFrame(Number.NaN, 600)).toBe(0)
  })
})

describe('playbackReducer', () => {
  it('starts playing at the first frame', () => {
    expect(initialPlayback()).toMatchObject({ frame: 0, playing: true, speed: 1, focus: null })
  })

  it('toggles play and pause without moving', () => {
    const paused = run({ type: 'toggle' })
    expect(paused).toMatchObject({ playing: false, frame: 0 })
    expect(playbackReducer(paused, { type: 'toggle' }, timeline).playing).toBe(true)
  })

  it('takes the frame the canvas publishes while playing', () => {
    expect(run({ type: 'published', frame: 137 }).frame).toBe(137)
  })

  it('takes it once paused too — the canvas overshoots the last report before it stops', () => {
    // The canvas advances at screen cadence and reports at a tenth of it, so at 4x it can be
    // six frames past its last report when the reader hits pause. Refusing the settling report
    // would leave the scrubber behind the picture, and the next step would snap the map back.
    const state = run({ type: 'seek', frame: 300 }, { type: 'toggle' })
    expect(playbackReducer(state, { type: 'published', frame: 305 }, timeline).frame).toBe(305)
  })

  it('does not churn on a report of the frame it already holds', () => {
    const state = run({ type: 'seek', frame: 300 })
    expect(playbackReducer(state, { type: 'published', frame: 300 }, timeline)).toBe(state)
  })

  it('never lets a report move the seek nonce — only a command snaps the canvas', () => {
    const state = run({ type: 'seek', frame: 300 })
    const after = playbackReducer(state, { type: 'published', frame: 305 }, timeline)
    expect(after.seekNonce).toBe(state.seekNonce)
  })

  it('seeks without interrupting playback — dragging the scrubber is a way to skim', () => {
    const state = run({ type: 'seek', frame: 250 })
    expect(state).toMatchObject({ frame: 250, playing: true })
    expect(state.seekNonce).toBe(1)
  })

  it('steps one frame in either direction, and pauses doing it', () => {
    const forward = run({ type: 'seek', frame: 100 }, { type: 'step', delta: 1 })
    expect(forward).toMatchObject({ frame: 101, playing: false })
    expect(playbackReducer(forward, { type: 'step', delta: -1 }, timeline).frame).toBe(100)
  })

  it('cannot step off either end of the axis', () => {
    expect(run({ type: 'step', delta: -1 }).frame).toBe(0)
    expect(run({ type: 'seek', frame: 599 }, { type: 'step', delta: 1 }).frame).toBe(599)
  })

  it('bumps the seek nonce on every commanded position, so the canvas snaps to it', () => {
    const state = run({ type: 'seek', frame: 10 }, { type: 'step', delta: 1 }, { type: 'restart' })
    expect(state.seekNonce).toBe(3)
    expect(state.frame).toBe(0)
  })

  it('jumps between deaths, and pauses on arrival', () => {
    const first = run({ type: 'jumpDeath', direction: 1 })
    expect(first).toMatchObject({ frame: 260, playing: false })
    expect(playbackReducer(first, { type: 'jumpDeath', direction: 1 }, timeline).frame).toBe(300)

    const third = run({ type: 'seek', frame: 450 })
    expect(playbackReducer(third, { type: 'jumpDeath', direction: -1 }, timeline).frame).toBe(420)
  })

  it('stops at the first death going backwards rather than falling to the start', () => {
    // The reader asked for a death, and there is not one before this point. Landing on frame
    // 0 would answer a question they did not ask; the control says so by being disabled.
    const first = run({ type: 'jumpDeath', direction: 1 })
    expect(playbackReducer(first, { type: 'jumpDeath', direction: -1 }, timeline)).toBe(first)
  })

  it('stays put — and keeps playing — when there is no death left to jump to', () => {
    const state = run({ type: 'seek', frame: 580 })
    const after = playbackReducer(state, { type: 'jumpDeath', direction: 1 }, timeline)
    expect(after).toBe(state)
  })

  it('does nothing on an objective jump when the match has no objectives', () => {
    const state = run({ type: 'seek', frame: 0 })
    expect(playbackReducer(state, { type: 'jumpObjective', direction: 1 }, timeline)).toBe(state)
  })

  it('jumps between objective actions when the match has them', () => {
    const withObjectives = normalizeReplayDocument({
      ...FIXTURE_REPLAY_DOCUMENT,
      objectives: [{ t: 210, xuid: '2533274800000001', stat: 'zone_secures', timeMs: 21_000 }],
    })
    const line = timelineOf(withObjectives)
    const state = playbackReducer(initialPlayback(), { type: 'jumpObjective', direction: 1 }, line)
    expect(state).toMatchObject({ frame: 210, playing: false })
  })

  it('restarts at the first frame and plays', () => {
    expect(run({ type: 'seek', frame: 400 }, { type: 'toggle' }, { type: 'restart' })).toMatchObject({
      frame: 0,
      playing: true,
    })
  })

  it('focuses the Nth player of the roster, and releases them on a second press', () => {
    const roster = rosterXUIDs(doc)
    const focused = run({ type: 'focus', index: 2 })
    expect(focused.focus).toBe(roster[2])
    expect(playbackReducer(focused, { type: 'focus', index: 2 }, timeline).focus).toBeNull()
  })

  it('moves the focus straight from one player to another', () => {
    const roster = rosterXUIDs(doc)
    const state = run({ type: 'focus', index: 0 }, { type: 'focus', index: 1 })
    expect(state.focus).toBe(roster[1])
  })

  it('ignores a digit beyond the roster rather than focusing nobody', () => {
    const state = run({ type: 'focus', index: 0 })
    expect(playbackReducer(state, { type: 'focus', index: 7 }, timeline).focus).toBe(state.focus)
  })

  it('clears the focus on demand', () => {
    expect(run({ type: 'focus', index: 1 }, { type: 'focusXUID', xuid: null }).focus).toBeNull()
  })

  it('changes speed without moving or stopping', () => {
    expect(run({ type: 'seek', frame: 42 }, { type: 'speed', speed: 4 })).toMatchObject({
      frame: 42,
      speed: 4,
      playing: true,
    })
  })
})

describe('advanceBy', () => {
  it('follows the match clock at 1x, using the artifact own frame interval', () => {
    // 100 ms per frame in the fixture: one second of real time is ten frames.
    expect(advanceBy(0, 1, 1_000, doc)).toBeCloseTo(10)
  })

  it('scales with the chosen speed', () => {
    expect(advanceBy(0, 4, 1_000, doc)).toBeCloseTo(40)
  })

  it('falls back to a nominal cadence when the artifact carries no time scale', () => {
    // Older artifacts publish no `frameIntervalMs`: the axis is then a record index, and
    // `replayLogic` says what "1x" means for it. Nothing here re-decides that.
    const scaleless = normalizeReplayDocument({
      ...FIXTURE_REPLAY_DOCUMENT,
      frameIntervalMs: undefined,
    })
    expect(advanceBy(0, 1, 1_000, scaleless)).toBeCloseTo(60)
  })

  it('wraps to the start at the end of the replay', () => {
    expect(advanceBy(598, 1, 1_000, doc)).toBe(0)
  })
})
