/**
 * studyDraw.ts — THE TWO LAYERS THIS VIEWER DRAWS ITSELF.
 *
 * Everything else on the map comes from the modules copied out of `apps/web`, which are
 * byte-identical to their origin and cannot be edited from here. These two could not:
 *
 *   1. THE TRAIL FADES WITH AGE. The copied player layer strokes one flat polyline per life:
 *      it shows where somebody has been and refuses to say in which direction. The fade is
 *      what turns a path into a movement, and it is not reachable through that layer's props —
 *      so this file draws the trail and `paintReplay.ts` turns the copied one off.
 *   2. AN ARC IS DRAWN IN ITS THROWER'S COLOUR when the film says whose it was. The copied
 *      projectile layer takes ONE colour for every flight on the map, because the archetype it
 *      reads carries no player; `grenadeArcs.ts` recovers the thrower where the film allows it,
 *      and a flight it cannot attribute keeps the neutral ink here rather than borrowing a team.
 *
 * NO REACT, NO STATE, NO COLOUR LITERAL: a context, geometry, and inks already resolved from
 * semantic tokens — the same contract every drawing module in this app is written to.
 *
 * EVERYTHING ADDRESSED TO THE EYE IS SCALED BY `k`, the device pixel ratio, and everything
 * addressed to the world is not. That is the copied layer's rule and it has to hold here too,
 * or a trail would be half as wide as the marker it ends under on a dense screen.
 */
import type { ReplayProjectileReady, ReplayTrackReady } from '../replay/replayNormalize'
import { altitudeAt, canvasScale, floorOf, isAliveAt, worldToCanvas, type XY } from '../replay/replayLogic'
import type { CanvasView } from '../replay/replayMarkers'

import type { WorldRect } from './mapCalibration'
import { trailFade, trailFadeSpan, trailPath } from './trailLogic'

/** Opacity of the freshest end of a trail. The far end is this times `trailFade`. */
const TRAIL_ALPHA = 0.7
const TRAIL_WIDTH = 2.1

/**
 * Opacity of a life outside the selected altitude band. SECOND COPY of the copied layer's
 * `OFF_FLOOR_ALPHA`, and deliberately the same number: a trail and the marker it ends under
 * must dim together, or the filter would look like it half-worked. A third copy owes a shared
 * helper (CLAUDE.md rule 6); there is nowhere to put one while the first copy sits in a file
 * that cannot be edited from here.
 */
const OFF_FLOOR_ALPHA = 0.12

const ARC_ALPHA = 0.55
const ARC_WIDTH = 1.6
/** A flight stays visible briefly after its last replicated point, then goes. */
const ARC_TAIL_FRAMES = 7

/** What the trail layer needs: a colour per life, the clock, and the reader's choices. */
export interface TrailLayerStyle {
  /** One colour per track, aligned with the array passed in — already focus-adjusted. */
  colors: string[]
  frame: number
  /** Trailing window in frames; `Infinity` draws the whole life so far. */
  windowFrames: number
  /** Device pixel ratio. */
  k: number
  /** Selected altitude band, or null for all of them. */
  floor: number | null
  z: { min: number; max: number }
}

/**
 * drawTrailsLayer strokes one fading trail per LIVING life.
 *
 * A life that has ended draws nothing: its death mark is the copied layer's job, and a trail
 * left behind a dead player would keep them on the map after the film stopped reporting them.
 *
 * SEGMENT BY SEGMENT, because the opacity changes along the line and a canvas has one alpha
 * per stroke. The cost is bounded by what is alive — eight lives of at most a few dozen
 * samples — not by the 99 tracks a match contains.
 */
export function drawTrailsLayer(
  ctx: CanvasRenderingContext2D,
  tracks: readonly ReplayTrackReady[],
  view: CanvasView,
  style: TrailLayerStyle,
): void {
  ctx.lineCap = 'round'
  ctx.lineJoin = 'round'
  ctx.lineWidth = TRAIL_WIDTH * style.k
  tracks.forEach((track, i) => {
    const color = style.colors[i]
    if (!color || !isAliveAt(track, style.frame)) return
    const path = trailPath(track.points, style.frame, style.windowFrames)
    if (path.length < 2) return
    const span = trailFadeSpan(path, style.windowFrames)
    const dim = bandAlpha(track, style)
    ctx.strokeStyle = color
    for (let s = 1; s < path.length; s++) {
      const a = project(path[s - 1], view)
      const b = project(path[s], view)
      ctx.globalAlpha = TRAIL_ALPHA * dim * trailFade(path[s].age, span)
      ctx.beginPath()
      ctx.moveTo(a.x, a.y)
      ctx.lineTo(b.x, b.y)
      ctx.stroke()
    }
  })
  ctx.globalAlpha = 1
}

/** What the arc layer needs to colour each flight. */
export interface ArcLayerStyle {
  frame: number
  /** Slot that threw each flight, aligned with the projectiles — cf. `grenadeArcs.ts`. */
  throwers: readonly (number | null)[]
  /** The colour of a slot's owner at this frame, or null when nobody is known to own it. */
  inkOfSlot: (slot: number) => string | null
  /** Ink for a flight whose thrower the film does not name. */
  fallback: string
  k: number
}

/**
 * drawGrenadeArcsLayer draws each flight in progress, from its origin to where it has got to.
 *
 * THE LAST POINT IS NOT AN IMPACT and nothing here says it is: the film carries no detonation
 * event. Replication of a frag stops about 1.4 s after the throw while its fuse runs to about
 * 3 s, so the arc FADES OUT where the film stops talking rather than ending on a burst.
 */
export function drawGrenadeArcsLayer(
  ctx: CanvasRenderingContext2D,
  projectiles: readonly ReplayProjectileReady[],
  view: CanvasView,
  style: ArcLayerStyle,
): void {
  ctx.lineCap = 'round'
  ctx.lineJoin = 'round'
  ctx.lineWidth = ARC_WIDTH * style.k
  projectiles.forEach((pr, i) => {
    const points = pr.p
    if (points.length < 2) return
    const end = pr.t0 + points[points.length - 1][0]
    if (style.frame < pr.t0 || style.frame > end + ARC_TAIL_FRAMES) return
    const slot = style.throwers[i]
    ctx.strokeStyle = (slot === null || slot === undefined ? null : style.inkOfSlot(slot)) ?? style.fallback
    ctx.globalAlpha =
      ARC_ALPHA * (style.frame > end ? 1 - (style.frame - end) / ARC_TAIL_FRAMES : 1)
    ctx.beginPath()
    let started = false
    for (const [dt, x, y] of points) {
      if (pr.t0 + dt > style.frame) break
      const c = project({ x, y }, view)
      if (started) ctx.lineTo(c.x, c.y)
      else {
        ctx.moveTo(c.x, c.y)
        started = true
      }
    }
    if (started) ctx.stroke()
  })
  ctx.globalAlpha = 1
}

/**
 * GRID_STEP_M — the spacing of the fallback grid, in METRES of the world.
 *
 * Ten metres is a Halo arena's own unit of distance: about a long corridor, a little over a
 * grenade's throw. It is also what makes the grid a RULER rather than decoration — the two
 * things it has to be good for are judging a distance on a map with no floor, and reading the
 * four world coordinates a calibration needs (cf. `mapImages.config.ts`).
 */
export const GRID_STEP_M = 10

const GRID_ALPHA = 0.16
const GRID_AXIS_ALPHA = 0.34
const GRID_LABEL_ALPHA = 0.5
const GRID_LABEL_FONT = '10px system-ui, sans-serif'
const GRID_LABEL_PAD = 3

/**
 * drawGridLayer paints the last fallback: a plain metric grid over the played area.
 *
 * IT IS NOT A MAP AND MUST NOT LOOK LIKE ONE. No walls, no rooms, no shading — nothing that
 * could be read as a feature of the level, because none of it would be true. What it does give
 * is scale and orientation: lines every `GRID_STEP_M` metres, the world axes picked out, and
 * the coordinates written on them.
 *
 * ALIGNED TO THE WORLD ORIGIN, not to the corner of the canvas. A grid starting at the edge of
 * whatever area these particular players covered would move between two matches on the same
 * map, and its labels would be meaningless for calibrating one.
 */
export function drawGridLayer(
  ctx: CanvasRenderingContext2D,
  view: CanvasView,
  style: { line: string; label: string },
): void {
  const b = view.bounds
  ctx.lineWidth = 1
  ctx.font = GRID_LABEL_FONT
  ctx.textBaseline = 'top'
  for (const x of ticks(b.minX, b.maxX)) {
    const top = project({ x, y: b.maxY }, view)
    const bottom = project({ x, y: b.minY }, view)
    strokeGridLine(ctx, style.line, x === 0, top, bottom)
    ctx.fillStyle = style.label
    ctx.globalAlpha = GRID_LABEL_ALPHA
    ctx.fillText(String(x), top.x + GRID_LABEL_PAD, GRID_LABEL_PAD)
  }
  for (const y of ticks(b.minY, b.maxY)) {
    const left = project({ x: b.minX, y }, view)
    const right = project({ x: b.maxX, y }, view)
    strokeGridLine(ctx, style.line, y === 0, left, right)
    ctx.fillStyle = style.label
    ctx.globalAlpha = GRID_LABEL_ALPHA
    ctx.fillText(String(y), GRID_LABEL_PAD, left.y + GRID_LABEL_PAD)
  }
  ctx.globalAlpha = 1
}

/** ticks lists the grid coordinates inside a span, on the world origin's own lattice. */
function ticks(lo: number, hi: number): number[] {
  const out: number[] = []
  const first = Math.ceil(lo / GRID_STEP_M) * GRID_STEP_M
  for (let v = first; v <= hi; v += GRID_STEP_M) out.push(v)
  return out
}

function strokeGridLine(
  ctx: CanvasRenderingContext2D,
  color: string,
  axis: boolean,
  from: XY,
  to: XY,
): void {
  ctx.strokeStyle = color
  ctx.globalAlpha = axis ? GRID_AXIS_ALPHA : GRID_ALPHA
  ctx.beginPath()
  // The half pixel puts a 1 px line ON the pixel grid rather than astride two of them.
  ctx.moveTo(Math.round(from.x) + 0.5, Math.round(from.y) + 0.5)
  ctx.lineTo(Math.round(to.x) + 0.5, Math.round(to.y) + 0.5)
  ctx.stroke()
}

/** How much of itself a calibrated image keeps: a floor, not the subject. */
const MAP_IMAGE_ALPHA = 0.55

/**
 * drawMapImage paints a calibrated top-down image onto the world rectangle it was measured
 * against.
 *
 * THE IMAGE IS PLACED BY ITS CORNERS, never fitted to the canvas. Fitting would rescale the
 * same map differently in every match — the framing follows the area the PLAYERS covered — and
 * a floor whose scale changes per match is a floor that lies about every distance read off it.
 * Placed by its corners, the image and the play area are in the same frame by construction, and
 * an image that reaches beyond the framing is simply clipped, which is correct.
 */
export function drawMapImage(
  ctx: CanvasRenderingContext2D,
  image: CanvasImageSource,
  world: WorldRect,
  view: CanvasView,
): void {
  // Top-left in canvas terms is (minX, maxY): the world's +Y is up and the canvas's is down.
  const topLeft = project({ x: world.minX, y: world.maxY }, view)
  const scale = canvasScale(view.bounds, view.width, view.height, view.pad)
  ctx.globalAlpha = MAP_IMAGE_ALPHA
  ctx.drawImage(
    image,
    topLeft.x,
    topLeft.y,
    (world.maxX - world.minX) * scale,
    (world.maxY - world.minY) * scale,
  )
  ctx.globalAlpha = 1
}

function project(p: XY, view: CanvasView): XY {
  return worldToCanvas(p, view.bounds, view.width, view.height, view.pad)
}

/** bandAlpha dims a life that is not on the selected altitude band. */
function bandAlpha(track: ReplayTrackReady, style: TrailLayerStyle): number {
  if (style.floor === null) return 1
  const z = altitudeAt(track.points, style.frame)
  const band = z === null ? 0 : floorOf(z, style.z.min, style.z.max)
  return band === style.floor ? 1 : OFF_FLOOR_ALPHA
}
