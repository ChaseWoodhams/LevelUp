/**
 * replayFixture.test.ts — the fixture actually PAINTS.
 *
 * WHY THIS EXISTS AND THE SMOKE TEST IS NOT ENOUGH. `App.test.tsx` proves the canvas
 * element mounts; jsdom hands it no 2D context, so nothing there can tell a document
 * that draws from one that silently draws nothing — a fixture with an empty floor, a
 * mistyped slot or an event outside every hold window would pass it unchanged.
 *
 * So the layers are driven here against a recording context: an object that executes
 * nothing and notes every call. The assertions are on WORK EMITTED, never on pixels —
 * a pixel assertion would be an anti-aliasing test. The shapes themselves are already
 * covered, one layer down, by the copied `canvasRecording.test.ts`.
 */
import { describe, expect, it } from 'vitest'

import { buildFloorGrid } from '../mapFloor'
import { drawFloorLayer, drawGrenadesLayer, drawShotsLayer } from '../replayDraw'
import { msToFrames } from '../replayLogic'
import { drawProjectilesLayer, drawTracksLayer } from '../replayMarkers'
import { normalizeReplayDocument } from '../replayNormalize'

import { FIXTURE_REPLAY_DOCUMENT, FIXTURE_SCOREBOARD } from './replayFixture'

/** A context that runs nothing and counts what it was asked to do. */
function recorder(): { ops: string[]; ctx: CanvasRenderingContext2D } {
  const ops: string[] = []
  const proxy = new Proxy(
    {},
    {
      get(_t, prop) {
        if (typeof prop !== 'string') return undefined
        // The aim cone is the one call whose RESULT is used: it wants something with
        // `addColorStop`. An inert token keeps the call in the trace.
        if (prop === 'createRadialGradient') {
          return (...args: unknown[]) => {
            ops.push(`${prop}(${args.length})`)
            return { addColorStop: () => ops.push('addColorStop') }
          }
        }
        return (...args: unknown[]) => ops.push(`${prop}(${args.length})`)
      },
      set(_t, prop) {
        if (typeof prop === 'string') ops.push(`set ${prop}`)
        return true
      },
    },
  )
  return { ops, ctx: proxy as unknown as CanvasRenderingContext2D }
}

const doc = normalizeReplayDocument(FIXTURE_REPLAY_DOCUMENT)
const view = { bounds: doc.bounds, width: 900, height: 480, pad: 24 }
/** Same 1.4 s event persistence the canvas uses. */
const HOLD = msToFrames(1_400, doc)
/** A frame with two shots inside the hold window. */
const SHOT_FRAME = 208
/** A frame with a grenade throw and its projectile in flight. */
const GRENADE_FRAME = 232

const strokes = (ops: string[]): number => ops.filter((o) => o.startsWith('stroke')).length
const fills = (ops: string[]): number => ops.filter((o) => o.startsWith('fill')).length

describe('the sample artifact', () => {
  it('crosses the frontier with no nullable array left', () => {
    expect(doc.tracks.length).toBeGreaterThan(0)
    expect(doc.structure.length).toBeGreaterThan(0)
    for (const track of doc.tracks) expect(track.points.length).toBeGreaterThan(0)
  })

  it('rasterises into a floor with real relief, not one flat wash', () => {
    const grid = buildFloorGrid(doc.structure, doc.bounds)
    expect(grid.filled).toBeGreaterThan(0)
    // Calibration bounds that coincide would mean every cell sits at one altitude —
    // the floor would draw, and no step or ledge would ever appear on it.
    expect(grid.zHi).toBeGreaterThan(grid.zLo)
  })

  it('paints the floor layer', () => {
    const { ops, ctx } = recorder()
    drawFloorLayer(ctx, buildFloorGrid(doc.structure, doc.bounds), view, {
      fill: 'rgb(1 2 3)',
      edge: 'rgb(4 5 6)',
    })
    expect(fills(ops)).toBeGreaterThan(0)
    expect(strokes(ops)).toBeGreaterThan(0)
  })

  it('draws the lives that are alive at the read frame', () => {
    const { ops, ctx } = recorder()
    drawTracksLayer(ctx, doc.tracks, view, {
      colors: doc.tracks.map(() => 'rgb(1 2 3)'),
      ink: 'rgb(4 5 6)',
      frame: SHOT_FRAME,
      timing: {
        trail: msToFrames(7_000, doc),
        aimHold: msToFrames(5_000, doc),
        shieldHold: msToFrames(2_000, doc),
        death: msToFrames(1_500, doc),
        spawn: msToFrames(800, doc),
      },
      floor: null,
      z: { min: doc.bounds.minZ ?? 0, max: doc.bounds.maxZ ?? 0 },
      k: 1,
      showAim: true,
      showShield: true,
    })
    expect(strokes(ops)).toBeGreaterThan(0)
    // The aim cone only draws off a heading the film actually replicated: seeing the
    // gradient proves the fixture publishes headings the way the record does.
    expect(ops.some((o) => o.startsWith('createRadialGradient'))).toBe(true)
  })

  it('draws the shots of the current window, and only those', () => {
    const { ops, ctx } = recorder()
    const style = {
      colorOfSlot: () => 'rgb(1 2 3)',
      fallback: 'rgb(4 5 6)',
      effectOf: (id: string | undefined) => (id ? doc.weaponLabels?.[id]?.fx : undefined),
      reducedMotion: false,
    }
    drawShotsLayer(ctx, doc.shots, view, { frame: SHOT_FRAME, hold: HOLD }, style)
    expect(strokes(ops) + fills(ops)).toBeGreaterThan(0)

    // A frame with no shot within the hold window must emit no shot geometry at all —
    // otherwise the layer would be painting events that are not happening.
    const quiet = recorder()
    drawShotsLayer(quiet.ctx, doc.shots, view, { frame: 320, hold: HOLD }, style)
    expect(strokes(quiet.ops) + fills(quiet.ops)).toBe(0)
  })

  it('draws a grenade throw and the projectile in flight', () => {
    const { ops, ctx } = recorder()
    drawGrenadesLayer(ctx, doc.grenades, view, { frame: GRENADE_FRAME, hold: HOLD }, 'rgb(1 2 3)')
    expect(fills(ops)).toBeGreaterThan(0)

    const proj = recorder()
    drawProjectilesLayer(proj.ctx, doc.projectiles, view, GRENADE_FRAME, 'rgb(1 2 3)')
    expect(strokes(proj.ops) + fills(proj.ops)).toBeGreaterThan(0)
  })

  it('gives every scoreboard row a life in the film', () => {
    const filmed = new Set(doc.tracks.map((t) => t.xuid))
    for (const row of FIXTURE_SCOREBOARD) expect(filmed.has(row.xuid)).toBe(true)
  })

  it('declares a balanced coverage: attached + rejects = available', () => {
    // A sum that does not close is the LEAK the banner exists to surface. A fixture
    // that leaked would make the banner report a defect of the fixture as a defect of
    // the decoder — the one thing that panel must never be caught doing.
    const shots = doc.coverage?.shots
    expect(shots).toBeDefined()
    const { attached, noSlot, ambiguous, outOfWindow, unpublished, available } = shots!
    expect(attached + noSlot + ambiguous + outOfWindow + unpublished).toBe(available)
  })
})
