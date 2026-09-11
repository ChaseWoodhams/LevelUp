/**
 * studyDraw.test.ts — the two layers this app draws itself, observed through a recording
 * context (cf. `src/test/recordingContext.ts`).
 *
 * The properties asserted are the ones the layers EXIST for, and each of them is a thing a flat
 * copy of the origin's layer would get wrong: a trail whose opacity says which end is recent, a
 * full-life mode that really reaches the spawn, and an arc that is drawn in its thrower's colour
 * only when the film names one.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayProjectileReady, ReplayTrackReady } from '../replay/replayNormalize'
import type { CanvasView } from '../replay/replayMarkers'

import { countOf, recordingContext, stringsOf, valuesOf } from '@/test/recordingContext'

import {
  drawGrenadeArcsLayer,
  drawGridLayer,
  drawMapImage,
  drawTrailsLayer,
  GRID_STEP_M,
  type TrailLayerStyle,
} from './studyDraw'

const VIEW: CanvasView = {
  bounds: { minX: 0, minY: 0, maxX: 40, maxY: 30 },
  width: 400,
  height: 300,
  pad: 0,
}

/**
 * A life walking east, sampled every 10 frames, from frame 0 to frame 100.
 *
 * `team` is -1 because the FILM carries no team: the artifact hard-wires the field, and a
 * fixture that pretended otherwise would be testing against data this app never sees.
 */
const WALKER: ReplayTrackReady = {
  slot: 1,
  team: -1,
  startFrame: 0,
  endFrame: 100,
  points: Array.from({ length: 11 }, (_, i) => ({ t: i * 10, x: i, y: 5, z: 0 })),
}

const INK = 'rgb(10, 20, 30)'

function trailStyle(over: Partial<TrailLayerStyle> = {}): TrailLayerStyle {
  return {
    colors: [INK],
    frame: 100,
    windowFrames: 60,
    k: 1,
    floor: null,
    z: { min: 0, max: 6 },
    ...over,
  }
}

function drawTrail(tracks: ReplayTrackReady[], over: Partial<TrailLayerStyle> = {}) {
  const { ops, ctx } = recordingContext()
  drawTrailsLayer(ctx, tracks, VIEW, trailStyle(over))
  return ops
}

describe('drawTrailsLayer', () => {
  it('strokes the trail segment by segment, because the opacity changes along it', () => {
    const ops = drawTrail([WALKER])
    expect(countOf(ops, 'stroke')).toBeGreaterThan(1)
    expect(countOf(ops, 'moveTo')).toBe(countOf(ops, 'stroke'))
  })

  it('fades with age: the oldest segment is the faintest, the newest the strongest', () => {
    const alphas = valuesOf(drawTrail([WALKER]), 'globalAlpha').filter((a) => a < 1)
    expect(alphas.length).toBeGreaterThan(1)
    expect(alphas[0]).toBeLessThan(alphas[alphas.length - 1])
  })

  it('draws more of the path in full-life mode than inside a window', () => {
    const windowed = countOf(drawTrail([WALKER], { windowFrames: 20 }), 'stroke')
    const whole = countOf(drawTrail([WALKER], { windowFrames: Number.POSITIVE_INFINITY }), 'stroke')
    expect(whole).toBeGreaterThan(windowed)
  })

  it('draws in the colour it was handed, and invents none', () => {
    expect(new Set(stringsOf(drawTrail([WALKER]), 'strokeStyle'))).toEqual(new Set([INK]))
  })

  it('draws nothing for a life that has ended — the death mark is the other layer’s job', () => {
    expect(countOf(drawTrail([WALKER], { frame: 400 }), 'stroke')).toBe(0)
  })

  it('draws nothing for a life whose colour is empty', () => {
    expect(countOf(drawTrail([WALKER], { colors: [''] }), 'stroke')).toBe(0)
  })

  it('dims a life that is off the selected altitude band rather than hiding it', () => {
    // The trailing `globalAlpha = 1` is the layer handing the context back clean, not a
    // segment, so it is dropped before the two runs are compared.
    const drawn = (floor: number) => valuesOf(drawTrail([WALKER], { floor }), 'globalAlpha').slice(0, -1)
    const on = drawn(0)
    const off = drawn(2)
    expect(off.length).toBe(on.length)
    expect(Math.max(...off)).toBeLessThan(Math.max(...on))
    expect(Math.max(...off)).toBeGreaterThan(0)
  })

  it('scales the line with the device pixel ratio, and the positions not at all', () => {
    const at1 = valuesOf(drawTrail([WALKER], { k: 1 }), 'lineWidth')[0]
    const at2 = valuesOf(drawTrail([WALKER], { k: 2 }), 'lineWidth')[0]
    expect(at2).toBeCloseTo(at1 * 2)
  })
})

/** A flight of four grid points, born at (10, 10) on frame `t0`. */
function flight(t0: number): ReplayProjectileReady {
  return {
    t0,
    p: [
      [0, 10, 10],
      [1, 11, 12],
      [2, 12, 13],
      [3, 13, 13],
    ],
  }
}

const THROWER_INK = 'rgb(1, 2, 3)'
const NEUTRAL = 'rgb(9, 9, 9)'

function drawArcs(
  projectiles: ReplayProjectileReady[],
  throwers: (number | null)[],
  frame: number,
) {
  const { ops, ctx } = recordingContext()
  drawGrenadeArcsLayer(ctx, projectiles, VIEW, {
    frame,
    throwers,
    inkOfSlot: (slot) => (slot === 4 ? THROWER_INK : null),
    fallback: NEUTRAL,
    k: 1,
  })
  return ops
}

describe('drawGrenadeArcsLayer', () => {
  it('draws the arc in its thrower’s colour when the film names one', () => {
    expect(stringsOf(drawArcs([flight(0)], [4], 3), 'strokeStyle')).toEqual([THROWER_INK])
  })

  it('keeps the neutral ink for a flight nobody is known to have thrown', () => {
    expect(stringsOf(drawArcs([flight(0)], [null], 3), 'strokeStyle')).toEqual([NEUTRAL])
  })

  it('keeps the neutral ink when the named slot has no living owner at this frame', () => {
    expect(stringsOf(drawArcs([flight(0)], [7], 3), 'strokeStyle')).toEqual([NEUTRAL])
  })

  it('draws the flight only as far as it has flown', () => {
    const early = countOf(drawArcs([flight(0)], [4], 1), 'lineTo')
    const late = countOf(drawArcs([flight(0)], [4], 3), 'lineTo')
    expect(early).toBe(1)
    expect(late).toBe(3)
  })

  it('has not started before its first point', () => {
    expect(countOf(drawArcs([flight(50)], [4], 10), 'stroke')).toBe(0)
  })

  it('fades out after the last replicated point instead of ending on a burst', () => {
    const atRest = valuesOf(drawArcs([flight(0)], [4], 3), 'globalAlpha')[0]
    const after = valuesOf(drawArcs([flight(0)], [4], 7), 'globalAlpha')[0]
    expect(after).toBeGreaterThan(0)
    expect(after).toBeLessThan(atRest)
    // Long after the last point there is nothing left to draw at all.
    expect(countOf(drawArcs([flight(0)], [4], 40), 'stroke')).toBe(0)
  })

  it('skips a flight with fewer than two points', () => {
    expect(countOf(drawArcs([{ t0: 0, p: [[0, 1, 1]] }], [4], 0), 'stroke')).toBe(0)
  })
})

/**
 * THE LAST TWO FLOORS OF THE FALLBACK CHAIN. Which one is chosen is `mapCalibration.ts`'s
 * decision and is tested there, with no canvas in sight; what is checked here is that each of
 * them, once chosen, is drawn in the WORLD's frame rather than the canvas's — the property the
 * whole chain rests on, because a floor placed in the canvas's frame would rescale itself for
 * every match and quietly misstate every distance read off it.
 */
describe('drawGridLayer', () => {
  const INK = { line: 'rgb(4, 4, 4)', label: 'rgb(5, 5, 5)' }

  function drawGrid(view = VIEW) {
    const { ops, ctx } = recordingContext()
    drawGridLayer(ctx, view, INK)
    return ops
  }

  it('rules a line every step of the world, on both axes', () => {
    const ops = drawGrid()
    // 40 m across and 30 m up, on a 10 m lattice: x at 0/10/20/30/40, y at 0/10/20/30.
    expect(countOf(ops, 'stroke')).toBe(5 + 4)
  })

  it('is aligned on the world origin, not on the corner of whatever area was played', () => {
    // The lattice of a scene offset by half a step still falls on the multiples of the step.
    const offset = { ...VIEW, bounds: { minX: 5, minY: 5, maxX: 45, maxY: 35 } }
    const labels = drawGrid(offset)
      .filter((o) => o.op === 'fillText')
      .map((o) => Number(o.args[0]))
    expect(labels.every((v) => v % GRID_STEP_M === 0)).toBe(true)
  })

  it('writes the world coordinate on each line — the grid is the ruler a calibration is read off', () => {
    const labels = drawGrid()
      .filter((o) => o.op === 'fillText')
      .map((o) => Number(o.args[0]))
    expect(labels).toContain(10)
    expect(labels).toContain(30)
  })

  it('picks the world axes out from the rest', () => {
    // A scene straddling the origin has an x = 0 and a y = 0 line, and they are stronger.
    const across = { ...VIEW, bounds: { minX: -20, minY: -20, maxX: 20, maxY: 20 } }
    const alphas = new Set(valuesOf(drawGrid(across), 'globalAlpha'))
    expect(alphas.size).toBeGreaterThan(2) // grid, axis, and the label opacity
  })
})

describe('drawMapImage', () => {
  /** A stand-in for an image element: `drawImage` only ever passes it through. */
  const IMAGE = { width: 512, height: 512 } as unknown as CanvasImageSource

  function place(world: { minX: number; minY: number; maxX: number; maxY: number }) {
    const { ops, ctx } = recordingContext()
    drawMapImage(ctx, IMAGE, world, VIEW)
    const call = ops.find((o) => o.op === 'drawImage')
    return call?.args.slice(1) as number[]
  }

  it('places the image by its world corners: the top-left is (minX, maxY)', () => {
    // The scene is 40 x 30 m on 400 x 300 px with no padding, so one metre is ten pixels and
    // the world's +Y is the canvas's -Y.
    const [x, y, w, h] = place({ minX: 10, minY: 5, maxX: 30, maxY: 25 })
    expect(x).toBeCloseTo(100)
    expect(y).toBeCloseTo(50)
    expect(w).toBeCloseTo(200)
    expect(h).toBeCloseTo(200)
  })

  it('does not fit itself to the canvas — a bigger rectangle draws bigger, and is clipped', () => {
    const [, , inside] = place({ minX: 0, minY: 0, maxX: 20, maxY: 20 })
    const [, , outside] = place({ minX: -20, minY: 0, maxX: 60, maxY: 20 })
    expect(outside).toBeCloseTo(inside * 4)
  })

  it('leaves the context opacity as it found it', () => {
    const { ops, ctx } = recordingContext()
    drawMapImage(ctx, IMAGE, { minX: 0, minY: 0, maxX: 10, maxY: 10 }, VIEW)
    expect(valuesOf(ops, 'globalAlpha').at(-1)).toBe(1)
  })
})
