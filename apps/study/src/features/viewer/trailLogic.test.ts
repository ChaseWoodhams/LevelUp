/**
 * trailLogic.test.ts — the trail is the right length, in the right order, aged correctly.
 *
 * The cases are the ones a fading trail can actually get wrong: an age derived from a position
 * in the array rather than from the sample's own timestamp (which the film breaks by skipping
 * samples), a window that silently becomes the whole life, and a full-path mode that grades its
 * fade against nothing.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayPoint } from '@/lib/api/types'

import {
  DEFAULT_TRAIL_WINDOW_MS,
  TRAIL_OLDEST_SHARE,
  TRAIL_WINDOW_MS,
  trailFade,
  trailFadeSpan,
  trailPath,
} from './trailLogic'

/** A walk east, one sample every 10 frames. */
const WALK: ReplayPoint[] = [
  { t: 0, x: 0, y: 0 },
  { t: 10, x: 1, y: 0 },
  { t: 20, x: 2, y: 0 },
  { t: 30, x: 3, y: 0 },
  { t: 40, x: 4, y: 0 },
]

describe('trailPath', () => {
  it('keeps only the samples inside the window, oldest first', () => {
    const path = trailPath(WALK, 40, 20)
    expect(path.map((v) => v.x)).toEqual([2, 3, 4])
  })

  it('ages each vertex from its own timestamp, not from its rank', () => {
    // The gap is what a rank-based age gets wrong: the film only transmits what changed, so
    // two consecutive samples can be five seconds apart.
    const sparse: ReplayPoint[] = [
      { t: 0, x: 0, y: 0 },
      { t: 45, x: 1, y: 0 },
      { t: 50, x: 2, y: 0 },
    ]
    expect(trailPath(sparse, 50, 60).map((v) => v.age)).toEqual([50, 5, 0])
  })

  it('ends on the interpolated head so the trail meets the marker', () => {
    const path = trailPath(WALK, 35, 20)
    const head = path[path.length - 1]
    expect(head).toEqual({ x: 3.5, y: 0, age: 0 })
  })

  it('does not repeat the head when a sample already sits on the frame', () => {
    const path = trailPath(WALK, 40, 20)
    expect(path.filter((v) => v.x === 4)).toHaveLength(1)
  })

  it('draws the whole life so far when the window is infinite', () => {
    expect(trailPath(WALK, 40, Number.POSITIVE_INFINITY)).toHaveLength(5)
  })

  it('yields nothing rather than a one-vertex path', () => {
    expect(trailPath(WALK, 0, 20)).toEqual([])
    expect(trailPath(WALK, 40, 0)).toEqual([])
    expect(trailPath([], 40, 20)).toEqual([])
  })

  it('never reaches past the frame being drawn', () => {
    expect(trailPath(WALK, 25, Number.POSITIVE_INFINITY).every((v) => v.age >= 0)).toBe(true)
  })
})

describe('trailFadeSpan', () => {
  it('is the window itself when there is one, so age reads the same on every player', () => {
    expect(trailFadeSpan(trailPath(WALK, 40, 20), 20)).toBe(20)
    expect(trailFadeSpan(trailPath(WALK, 15, 20), 20)).toBe(20)
  })

  it('is the oldest vertex in full-path mode, where there is no window to grade against', () => {
    const path = trailPath(WALK, 40, Number.POSITIVE_INFINITY)
    expect(trailFadeSpan(path, Number.POSITIVE_INFINITY)).toBe(40)
  })

  it('never divides by zero', () => {
    expect(trailFadeSpan([], Number.POSITIVE_INFINITY)).toBe(1)
    expect(trailFadeSpan([], 0)).toBe(1)
  })
})

describe('trailFade', () => {
  it('is full at the head and a visible floor at the far end', () => {
    expect(trailFade(0, 20)).toBe(1)
    expect(trailFade(20, 20)).toBeCloseTo(TRAIL_OLDEST_SHARE)
  })

  it('clamps rather than going transparent past the window', () => {
    expect(trailFade(1_000, 20)).toBeCloseTo(TRAIL_OLDEST_SHARE)
    expect(trailFade(-5, 20)).toBe(1)
  })
})

describe('the offered windows', () => {
  it('include the default, and one full-path entry', () => {
    expect(TRAIL_WINDOW_MS).toContain(DEFAULT_TRAIL_WINDOW_MS)
    expect(TRAIL_WINDOW_MS.filter((w) => w === null)).toHaveLength(1)
  })
})
