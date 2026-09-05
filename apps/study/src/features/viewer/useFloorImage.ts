/**
 * useFloorImage.ts — GETTING A FLOOR UNDER THE MATCH, exactly once.
 *
 * Two hooks and one decision, split out of `useReplayPainter.ts` when the floor's fallback
 * chain arrived and pushed that file to the edge of this repository's 500-line rule. The split
 * is not only arithmetic: what is here is about the GROUND — where it comes from, whether it
 * has arrived, and how it gets painted once instead of sixty times a second — while what is
 * left there is about the clock and the frame.
 *
 * THE CHAIN LIVES HERE, and it is the only place it is decided in pixels: measured structure
 * geometry, then a calibrated top-down image, then a plain metric grid. `mapCalibration.ts`
 * decides the same order in words, against the same two facts, so what the reader is told and
 * what is drawn cannot disagree.
 */
import { useEffect, useState } from 'react'

import { drawFloorLayer } from '../replay/replayDraw'
import type { FloorGrid } from '../replay/mapFloor'
import type { CanvasView } from '../replay/replayMarkers'

import type { MapImageCalibration } from './mapCalibration'
import type { PaintStyle } from './paintReplay'
import { drawGridLayer, drawMapImage } from './studyDraw'

/** Images already reported as unreachable, so one broken calibration warns once, not per render. */
const reportedBrokenImages = new Set<string>()

/**
 * useMapImage loads a calibrated map's image, and answers null until it is actually usable.
 *
 * NULL IS A COMPLETE ANSWER, and it is the reason this hook returns an element rather than a
 * boolean: while the image is in flight — and forever, if its file is not there — the floor
 * chain falls through to the grid and the line under the map SAYS grid. A viewer that claimed
 * an image floor it never drew would be the one failure mode a fallback chain must not have.
 *
 * A file that cannot be fetched is reported once rather than swallowed: a calibration pointing
 * at a missing image looks exactly like a map nobody has calibrated, and the operator who just
 * added the entry deserves to know which of the two they are looking at.
 */
export function useMapImage(calibration: MapImageCalibration | null): HTMLImageElement | null {
  const src = calibration?.image ?? null
  const [loaded, setLoaded] = useState<HTMLImageElement | null>(null)
  useEffect(() => {
    setLoaded(null)
    if (src === null) return
    const el = new Image()
    let live = true
    el.onload = () => {
      if (live) setLoaded(el)
    }
    el.onerror = () => {
      if (!live || reportedBrokenImages.has(src)) return
      reportedBrokenImages.add(src)
      console.warn(`[study] calibrated map image not reachable, falling back to the grid: ${src}`)
    }
    el.src = src
    return () => {
      live = false
    }
  }, [src])
  return loaded
}

/** Everything the floor is made of, and nothing that changes per frame. */
export interface FloorInputs {
  view: CanvasView
  /** Height of the drawing surface, in CSS pixels. Owned by the painter, not by the floor. */
  height: number
  /** The rasterised structure floor, when the artifact carries one. */
  floorGrid: FloorGrid | null
  /** The calibrated image entry for this map, when one is configured. */
  calibration: MapImageCalibration | null
  /** That image, once it has actually loaded. */
  mapImage: HTMLImageElement | null
  style: PaintStyle
}

/**
 * useFloorImage paints THE FLOOR — whichever of the three it is — once onto an offscreen
 * canvas, and repaints it only when its own geometry, its framing or its inks change: never per
 * frame, and never for a layer drawn OVER it.
 *
 * That is what makes 45 000 cells cost nothing to the animation: the frame loop blits an image
 * instead of rasterising a grid. All three floors go through the same offscreen canvas, so the
 * frame loop has ONE case rather than three, and none of them is cheap enough per frame to be
 * worth a second path (an image blit is the cheapest, and it is still a blit).
 *
 * IT DEPENDS ON THE FIELDS, NOT ON THE OBJECT. `inputs` is a fresh literal on every render of
 * the caller; the dependency list below is its FIELDS, each of which the caller memoises. Taking
 * the object would repaint the floor on every render, which is the exact cost this hook exists
 * to avoid.
 *
 * It deliberately does not depend on the draw callback, which changes for reasons that have
 * nothing to do with the floor (a layer toggled, a player focused, a frame reported). The caller
 * redraws right after — cf. the note at its call site.
 */
export function useFloorImage(
  target: React.RefObject<HTMLCanvasElement | null>,
  inputs: FloorInputs,
): void {
  const { view, height, floorGrid, calibration, mapImage, style } = inputs
  useEffect(() => {
    if (view.width === 0) {
      target.current = null
      return
    }
    const dpr = window.devicePixelRatio || 1
    const off = document.createElement('canvas')
    off.width = Math.round(view.width * dpr)
    off.height = Math.round(height * dpr)
    const ctx = off.getContext('2d')
    if (!ctx) return
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
    // THE FALLBACK CHAIN, in one place and in one order: measured geometry, then a calibrated
    // image, then a grid that claims nothing.
    if (floorGrid) {
      drawFloorLayer(ctx, floorGrid, view, style.floor)
    } else if (calibration && mapImage) {
      drawMapImage(ctx, mapImage, calibration.world, view)
    } else {
      drawGridLayer(ctx, view, { line: style.floor.edge, label: style.floor.edge })
    }
    target.current = off
  }, [target, view, height, floorGrid, calibration, mapImage, style.floor])
}
