/**
 * paintReplay.ts — ONE FRAME OF THE MAP, laid down layer by layer.
 *
 * DERIVED FROM the draw body of `apps/web/src/features/match-replay/ReplayCanvas.tsx`
 * @ 0afd83f7e. Not a copy — the colours arrive as arguments instead of being derived from a
 * chart palette — so it carries no drift guard; `git diff 0afd83f7e HEAD -- <origin>` is still
 * what to read when the origin learns something about drawing.
 *
 * NO REACT AND NO STATE. Everything the frame needs is an argument, which is what lets the
 * component above it be a component and the hook beside it be about scheduling. It is also why
 * the drawing modules it calls are testable against a recording context without any of this.
 *
 * TWO LAYERS ARE THIS APP'S OWN (`studyDraw.ts`) and take the place of the copied versions:
 * the trail, because it fades with age and can run the whole life, and the grenade arc, because
 * it is drawn in its thrower's colour. Neither is reachable through the copied modules' props,
 * and those modules are byte-identical to `apps/web` and not editable from here.
 */
import { drawGeometryLayer, drawGrenadesLayer, drawShotsLayer } from '../replay/replayDraw'
import type { ReplayDocumentReady } from '../replay/replayNormalize'
import { drawTracksLayer, type CanvasView, type MarkerTiming } from '../replay/replayMarkers'

import type { FloorSource } from './mapCalibration'
import { drawGrenadeArcsLayer, drawTrailsLayer } from './studyDraw'

/**
 * The trailing window to hand the COPIED player layer, and the way that layer's own flat trail
 * is turned off: `trailAt(points, frame, 0)` spans no samples, so it returns the head alone and
 * the layer draws nothing. The trail on screen is `drawTrailsLayer`'s; drawing both would put a
 * flat line under a graded one, at a different length and at a length the reader did not pick.
 *
 * It is applied where the timing is built (`useReplayPainter`), not re-applied here, so there
 * is one assignment rather than two that could disagree.
 */
export const COPIED_TRAIL_OFF = 0

/** The inks a frame is painted with, all of them already resolved from semantic tokens. */
export interface PaintStyle {
  geometry: string
  shot: string
  grenade: string
  floor: { fill: string; edge: string }
}

/** What the reader has chosen to see, and in whose colours. */
export interface PaintLayers {
  /** One colour per track, aligned with `doc.tracks`. */
  inks: string[]
  /**
   * The colour of whoever owned a slot at a frame. `hold` is the lingering window of the mark
   * being drawn: a shot outlives its instant, and often its shooter.
   */
  inkOfSlotAt: (slot: number, frame: number, hold: number) => string | null
  showAim: boolean
  showShield: boolean
  showShots: boolean
  showGrenades: boolean
  /** Trailing window in frames; `Infinity` draws the whole life so far. */
  trailFrames: number
  /** The selected altitude band, or null for all of them. */
  floor: number | null
  reducedMotion: boolean
}

export interface PaintOptions {
  view: CanvasView
  frame: number
  /** Device pixel ratio: everything addressed to the eye is scaled by it. */
  dpr: number
  /** The pre-painted floor — whichever of the three it is. Null until the canvas is measured. */
  floorImage: HTMLCanvasElement | null
  /** Which floor that image holds. Cf. `mapCalibration.ts`. */
  floorSource: FloorSource
  style: PaintStyle
  timing: MarkerTiming
  zRange: { min: number; max: number }
  /** How long a point event stays on screen, in frames. */
  eventHoldFrames: number
  /** Slot that threw each projectile, aligned with `doc.projectiles`. Cf. `grenadeArcs.ts`. */
  arcThrowers: readonly (number | null)[]
  layers: PaintLayers
}

/**
 * paintReplay lays the layers down, back to front: the floor carries the trails, which carry
 * the events. Inverting the order would drown the players.
 */
export function paintReplay(
  ctx: CanvasRenderingContext2D,
  doc: ReplayDocumentReady,
  o: PaintOptions,
): void {
  paintGround(ctx, doc, o)
  drawTrailsLayer(ctx, doc.tracks, o.view, {
    colors: o.layers.inks,
    frame: o.frame,
    windowFrames: o.layers.trailFrames,
    k: o.dpr,
    floor: o.layers.floor,
    z: o.zRange,
  })
  drawTracksLayer(ctx, doc.tracks, o.view, {
    colors: o.layers.inks,
    ink: o.style.floor.edge,
    frame: o.frame,
    timing: o.timing,
    floor: o.layers.floor,
    z: o.zRange,
    k: o.dpr,
    showAim: o.layers.showAim,
    showShield: o.layers.showShield,
  })
  paintEvents(ctx, doc, o)
}

/** paintGround puts down the floor and everything that belongs to the terrain rather than to a player. */
function paintGround(ctx: CanvasRenderingContext2D, doc: ReplayDocumentReady, o: PaintOptions): void {
  if (o.floorImage) {
    ctx.drawImage(o.floorImage, 0, 0, o.view.width, o.view.height)
  }
  // THE PROPS ARE A LANDMARK, AND NO LONGER THE LAST RESORT. Before the fallback chain they
  // were drawn only when there was no floor at all — which was also the only case in which
  // anything was drawn under the players. Now the grid catches that case, and the props keep
  // the job they were actually good at: on a map with no reconstructed structure they are the
  // only real thing on screen (3.4 % of the ground, which is little and is measured), so they
  // go ON TOP of whichever fallback is underneath. Over a reconstructed floor they would just
  // be noise on data that is already better.
  if (o.floorSource !== 'structure' && doc.geometry.length > 0) {
    drawGeometryLayer(ctx, doc.geometry, o.view, { color: o.style.geometry, z: o.zRange })
  }
  // Arcs go UNDER the players: they are objects in the air over the ground, not the subject.
  if (o.layers.showGrenades && doc.projectiles.length > 0) {
    drawGrenadeArcsLayer(ctx, doc.projectiles, o.view, {
      frame: o.frame,
      throwers: o.arcThrowers,
      inkOfSlot: (slot) => o.layers.inkOfSlotAt(slot, o.frame, o.eventHoldFrames),
      fallback: o.style.grenade,
      k: o.dpr,
    })
  }
}

/** paintEvents puts the point events on top: they are read against the trails, not under them. */
function paintEvents(ctx: CanvasRenderingContext2D, doc: ReplayDocumentReady, o: PaintOptions): void {
  const win = { frame: o.frame, hold: o.eventHoldFrames }
  if (o.layers.showShots && doc.shots.length > 0) {
    drawShotsLayer(ctx, doc.shots, o.view, win, {
      // The shooter is resolved from the pair (slot, frame) and never from a slot-to-player
      // map: a slot is reassigned at every respawn. The hold goes with it because a mark
      // outlives its instant — and, often enough, its shooter.
      colorOfSlot: (slot) => o.layers.inkOfSlotAt(slot, o.frame, o.eventHoldFrames),
      fallback: o.style.shot,
      effectOf: (id) => (id ? doc.weaponLabels?.[id]?.fx : undefined),
      reducedMotion: o.layers.reducedMotion,
    })
  }
  if (o.layers.showGrenades && doc.grenades.length > 0) {
    drawGrenadesLayer(ctx, doc.grenades, o.view, win, o.style.grenade)
  }
}

/**
 * sizeCanvas matches the backing store to the device pixel ratio, then works in CSS pixels.
 *
 * The size is only assigned when it actually changed: writing `canvas.width` clears the
 * surface even when the value is identical, which would erase the frame on every redraw.
 */
export function sizeCanvas(
  canvas: HTMLCanvasElement,
  ctx: CanvasRenderingContext2D,
  width: number,
  height: number,
  dpr: number,
): void {
  const pw = Math.round(width * dpr)
  const ph = Math.round(height * dpr)
  if (canvas.width !== pw || canvas.height !== ph) {
    canvas.width = pw
    canvas.height = ph
  }
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
  ctx.clearRect(0, 0, width, height)
}
