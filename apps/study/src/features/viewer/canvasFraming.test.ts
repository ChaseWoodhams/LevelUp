/**
 * canvasFraming.test.ts — a landscape scene renders exactly as it always did; a portrait
 * one gets to use the width it has, within a bound.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayBounds } from '@/lib/api/types'

import { MAX_CANVAS_HEIGHT, MIN_CANVAS_HEIGHT, sceneCanvasHeight } from './canvasFraming'

const PAD = 24

function bounds(width: number, height: number): ReplayBounds {
  return { minX: 0, minY: 0, maxX: width, maxY: height }
}

describe('sceneCanvasHeight', () => {
  it('gives every landscape scene exactly the old fixed height', () => {
    // The fixture's own scene (40 x 30) and a wide arena, at a typical container width.
    expect(sceneCanvasHeight(bounds(40, 30), 1000, PAD)).toBe(MIN_CANVAS_HEIGHT)
    expect(sceneCanvasHeight(bounds(120, 20), 1000, PAD)).toBe(MIN_CANVAS_HEIGHT)
  })

  it('does not grow an ordinary landscape scene in a wide container', () => {
    // The bug this file's own header records: a naive "full-width height, clamped to a
    // minimum" formula would have grown THIS scene too, since 1000 px / 480 px is a wider
    // reference ratio (~2.08:1) than 40 x 30 (1.33:1) — on nothing more than the
    // container being wide, which is the ordinary case.
    expect(sceneCanvasHeight(bounds(40, 30), 2000, PAD)).toBe(MIN_CANVAS_HEIGHT)
  })

  it('grows for a portrait scene, up to the maximum', () => {
    // 0e97be38's own bounds, the case this file exists for.
    const got = sceneCanvasHeight(bounds(28.45, 40.7), 1000, PAD)
    expect(got).toBeGreaterThan(MIN_CANVAS_HEIGHT)
    expect(got).toBeLessThanOrEqual(MAX_CANVAS_HEIGHT)
  })

  it('never exceeds the maximum, however extreme the aspect ratio', () => {
    expect(sceneCanvasHeight(bounds(5, 500), 1000, PAD)).toBe(MAX_CANVAS_HEIGHT)
  })

  it('answers the minimum before the container has been measured', () => {
    expect(sceneCanvasHeight(bounds(28.45, 40.7), 0, PAD)).toBe(MIN_CANVAS_HEIGHT)
  })

  it('never divides by zero on a degenerate (zero-width) bounds', () => {
    expect(Number.isFinite(sceneCanvasHeight(bounds(0, 40), 1000, PAD))).toBe(true)
  })
})
