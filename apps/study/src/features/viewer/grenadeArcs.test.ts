/**
 * grenadeArcs.test.ts — the arc is attributed only when the film actually says so.
 *
 * The cases that matter are the refusals: this pairing decides whose colour an arc is drawn in,
 * and every wrong answer puts a grenade in somebody's hand who never threw it.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayGrenade, ReplayProjectile } from '@/lib/api/types'

import { ARC_ORIGIN_RADIUS_M, arcThrowers } from './grenadeArcs'

function grenade(slot: number, t: number, x: number, y: number): ReplayGrenade {
  return { i: 0, rank: 0, s: 'frag', slot, t, x, y }
}

/** A flight born at (x, y) on frame `t0`, drifting east. */
function flight(t0: number, x: number, y: number): ReplayProjectile {
  return {
    t0,
    p: [
      [0, x, y],
      [1, x + 1, y + 1],
      [2, x + 2, y + 1],
    ],
  }
}

const WINDOW = 3

describe('arcThrowers', () => {
  it('attributes a flight born where and when a throw happened', () => {
    expect(arcThrowers([grenade(4, 100, 10, 10)], [flight(100, 10, 10)], WINDOW)).toEqual([4])
  })

  it('tolerates the gap between the hand and the first replicated position', () => {
    const near = ARC_ORIGIN_RADIUS_M - 0.5
    expect(arcThrowers([grenade(4, 100, 10, 10)], [flight(102, 10 + near, 10)], WINDOW)).toEqual([4])
  })

  it('refuses a throw too far away on the ground', () => {
    const far = ARC_ORIGIN_RADIUS_M + 0.5
    expect(arcThrowers([grenade(4, 100, 10, 10)], [flight(100, 10 + far, 10)], WINDOW)).toEqual([null])
  })

  it('refuses a throw outside the time window', () => {
    expect(arcThrowers([grenade(4, 100, 10, 10)], [flight(100 + WINDOW + 1, 10, 10)], WINDOW)).toEqual([
      null,
    ])
  })

  it('refuses to choose between two throwers at the same place and moment', () => {
    const throws = [grenade(4, 100, 10, 10), grenade(7, 101, 11, 10)]
    expect(arcThrowers(throws, [flight(100, 10, 10)], WINDOW)).toEqual([null])
  })

  it('still attributes when one player threw twice into the same window', () => {
    const throws = [grenade(4, 100, 10, 10), grenade(4, 101, 10.5, 10)]
    expect(arcThrowers(throws, [flight(100, 10, 10)], WINDOW)).toEqual([4])
  })

  it('answers null for a flight no throw explains, rather than the nearest one', () => {
    expect(arcThrowers([grenade(4, 500, 30, 30)], [flight(100, 10, 10)], WINDOW)).toEqual([null])
  })

  it('answers one entry per flight, in order', () => {
    const throws = [grenade(1, 100, 10, 10), grenade(2, 300, 20, 20)]
    const flights = [flight(300, 20, 20), flight(100, 10, 10), flight(700, 0, 0)]
    expect(arcThrowers(throws, flights, WINDOW)).toEqual([2, 1, null])
  })

  it('survives a flight with no points at all', () => {
    expect(arcThrowers([grenade(4, 100, 10, 10)], [{ t0: 100, p: [] }], WINDOW)).toEqual([null])
  })

  it('attributes nothing when the artifact carries no throws', () => {
    expect(arcThrowers([], [flight(100, 10, 10)], WINDOW)).toEqual([null])
  })
})
