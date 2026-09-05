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
 */
import { drawGeometryLayer, drawGrenadesLayer, drawShotsLayer } from '../replay/replayDraw'
import type { ReplayDocumentReady } from '../replay/replayNormalize'
import { drawProjectilesLayer, drawTracksLayer, type CanvasView, type MarkerTiming } from '../replay/replayMarkers'

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
  /** The selected altitude band, or null for all of them. */
  floor: number | null
  reducedMotion: boolean
}

export interface PaintOptions {
  view: CanvasView
  frame: number
  /** Device pixel ratio: everything addressed to the eye is scaled by it. */
  dpr: number
  /** The pre-painted floor, when the map has a reconstructed one. */
  floorImage: HTMLCanvasElement | null
  style: PaintStyle
  timing: MarkerTiming
  zRange: { min: number; max: number }
  /** How long a point event stays on screen, in frames. */
  eventHoldFrames: number
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
  } else if (doc.geometry.length > 0) {
    // A FALLBACK, not a duplicate: without a frozen structure file the map has no
    // reconstructed floor, and the Forge props are the only landmark left. They cover 3.4 %
    // of the ground — which is little, and better than an empty background.
    drawGeometryLayer(ctx, doc.geometry, o.view, { color: o.style.geometry, z: o.zRange })
  }
  // Projectiles go UNDER the players: they are objects on the ground, not the subject.
  if (doc.projectiles.length > 0) {
    drawProjectilesLayer(ctx, doc.projectiles, o.view, o.frame, o.style.grenade)
  }
}

/** paintEvents puts the point events on top: they are read against the trails, not under them. */
function paintEvents(ctx: CanvasRenderingContext2D, doc: ReplayDocumentReady, o: PaintOptions): void {
  const win = { frame: o.frame, hold: o.eventHoldFrames }
  if (doc.shots.length > 0) {
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
  if (doc.grenades.length > 0) {
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
