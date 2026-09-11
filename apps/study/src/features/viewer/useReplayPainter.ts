/**
 * useReplayPainter — WHEN the map is painted, and where its clock is.
 *
 * The split with `paintReplay.ts` is the one that matters: that file knows what a frame looks
 * like, this one knows when to make another. Everything below is scheduling — a resize
 * observer, an offscreen floor that must not be repainted per frame, an animation loop, and
 * the ref that holds the playing position.
 *
 * THE POSITION IS A REF, AND THAT IS THE WHOLE PERFORMANCE STORY. The canvas advances at
 * screen cadence; publishing that back to React sixty times a second would re-render eight
 * player cards for a number that has barely changed. So the frame lives here, is published
 * every 150 ms, and a position COMMANDED from outside arrives as an anchor whose NONCE says
 * "this one is new, snap to it" — which is what tells a fresh seek apart from the echo of this
 * loop's own last report.
 */
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'

import { resolveToken } from '@/lib/accessibility/resolveToken'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import { useColorPaletteVersion } from '@/lib/accessibility/useColorPaletteVersion'

import { readInk } from '../replay/canvasInk'
import { buildFloorGrid, type FloorGrid } from '../replay/mapFloor'
import { fitWidth, isAliveAt, msToFrames, sceneBounds } from '../replay/replayLogic'
import type { ReplayDocumentReady } from '../replay/replayNormalize'
import type { CanvasView, MarkerTiming } from '../replay/replayMarkers'

import { sceneCanvasHeight } from './canvasFraming'
import { floorSurfacesOf } from './floorScope'
import { arcThrowers, ARC_ORIGIN_WINDOW_MS } from './grenadeArcs'
import {
  calibrationFor,
  floorSourceOf,
  type FloorSource,
  type MapImageCalibration,
} from './mapCalibration'
import { MAP_IMAGES } from './mapImages.config'
import { COPIED_TRAIL_OFF, paintReplay, sizeCanvas, type PaintLayers, type PaintStyle } from './paintReplay'
import { advanceBy } from './playbackLogic'
import { useFloorImage, useMapImage } from './useFloorImage'

/** Map floor: a neutral token, with no directional connotation — the subject is the players. */
const GEOMETRY_TOKEN: SemanticToken = 'divergent-neutral'
/**
 * Point events. A shot borrows the alert token (it DEALT damage — the film records no other
 * kind), a throw an informational one: two natures, two readings.
 */
const SHOT_TOKEN: SemanticToken = 'destructive'
const GRENADE_TOKEN: SemanticToken = 'info'

/** How long a point event lingers, in real time. 1.4 s is the convention the origin settled on. */
const EVENT_HOLD_MS = 1_400

const CANVAS_PAD = 24

/**
 * Timing of the player layer, in REAL TIME — never in frames: the sampling cadence is chosen
 * at build time and can change without playback changing. Values from
 * `apps/web/src/features/match-replay/ReplayCanvas.tsx`, where they were tuned on screen; the
 * measurements behind them are in `replayMarkers.ts`.
 *
 * The origin's trailing window (7 s) is NOT here: this app draws its own trail, over a window
 * the reader picks, and the copied layer's flat one is turned off (cf. `COPIED_TRAIL_OFF`).
 */
const TIMING_MS = {
  aimHold: 5_000,
  shieldHold: 2_000,
  death: 1_500,
  spawn: 800,
} as const

/** Below this drop (metres) the map counts as flat: no floor filter is offered. */
const MIN_FLOOR_SPAN = 1

/** How often the current frame is published to React. Cf. the header. */
const FRAME_PUBLISH_MS = 150

export interface FrameAnchor {
  frame: number
  /** Increments on every commanded position. A change here, and only here, snaps the clock. */
  nonce: number
}

export interface ReplayPainterOptions {
  doc: ReplayDocumentReady
  /** One colour per track, aligned with `doc.tracks`. */
  inks: string[]
  inkOfSlotAt: PaintLayers['inkOfSlotAt']
  playing: boolean
  speed: number
  anchor: FrameAnchor
  onFrameChange: (frame: number) => void
  showAim: boolean
  showShield: boolean
  showShots: boolean
  showGrenades: boolean
  /** Trailing window in real time; `null` is the whole life so far. Cf. `trailLogic.ts`. */
  trailWindowMs: number | null
  /**
   * The archive's key for the map this match was played on, or null when it is not known.
   *
   * IT CANNOT COME FROM THE ARTIFACT: a replay document carries its match, its title and its
   * bounds, and no map at all. It is read from the archive (`GET /matches/{id}`), and it is
   * what the calibrated-image fallback looks a floor up by.
   */
  mapModule: string | null
  /** Selected altitude band, or null for all. Ignored on a map with no relief. */
  floor: number | null
}

export interface ReplayPainter {
  containerRef: React.RefObject<HTMLDivElement | null>
  canvasRef: React.RefObject<HTMLCanvasElement | null>
  /** Written to directly at screen cadence: the count of players on the map. */
  aliveRef: React.RefObject<HTMLSpanElement | null>
  /** Drawing width, 0 until the container has been measured. */
  renderWidth: number
  /** Drawing height — the scene's own, cf. `canvasFraming.ts`. Never 0. */
  renderHeight: number
  /** Whether this map has enough relief for a floor filter to mean anything. */
  hasFloors: boolean
  /** Which floor is actually under the match — what the reader is told, cf. `mapCalibration`. */
  floorSource: FloorSource
}

export function useReplayPainter(o: ReplayPainterOptions): ReplayPainter {
  const { doc, playing, speed, anchor, onFrameChange } = o
  const containerRef = useRef<HTMLDivElement>(null)
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const aliveRef = useRef<HTMLSpanElement>(null)
  const frameRef = useRef(anchor.frame)
  const publishedAtRef = useRef(0)
  const appliedNonceRef = useRef(anchor.nonce)
  /** The pre-painted floor. Held as a ref because `draw` reads it and `useFloorImage` writes
   *  it — a value would put the two in a declaration cycle for no gain. */
  const floorImage = useRef<HTMLCanvasElement | null>(null)

  const width = useObservedWidth(containerRef)
  const { style, reducedMotion } = useCanvasInks()
  // The calibration is looked up from the MAP, and the image is fetched from the calibration:
  // both before the scene, which is what avoids a cycle between "what floor do we draw" and
  // "has its image arrived".
  const calibration = useMemo(() => calibrationFor(o.mapModule, MAP_IMAGES), [o.mapModule])
  const mapImage = useMapImage(calibration)
  const scene = useReplayScene(doc, width, calibration, mapImage)

  /**
   * The trailing window in frames. `null` — the full life path — becomes `Infinity` rather
   * than the frame count: a window is compared against a sample's AGE, and a life that started
   * before the window opened is the case the comparison exists for.
   */
  const trailFrames = useMemo(
    () => (o.trailWindowMs === null ? Number.POSITIVE_INFINITY : msToFrames(o.trailWindowMs, doc)),
    [o.trailWindowMs, doc],
  )

  const layers = useMemo<PaintLayers>(
    () => ({
      inks: o.inks,
      inkOfSlotAt: o.inkOfSlotAt,
      showAim: o.showAim,
      showShield: o.showShield,
      showShots: o.showShots,
      showGrenades: o.showGrenades,
      trailFrames,
      floor: scene.hasFloors ? o.floor : null,
      reducedMotion,
    }),
    [
      o.inks,
      o.inkOfSlotAt,
      o.showAim,
      o.showShield,
      o.showShots,
      o.showGrenades,
      trailFrames,
      o.floor,
      scene.hasFloors,
      reducedMotion,
    ],
  )

  const draw = useCallback(() => {
    const canvas = canvasRef.current
    if (!canvas || scene.view.width === 0) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return
    const dpr = window.devicePixelRatio || 1
    sizeCanvas(canvas, ctx, scene.view.width, scene.view.height, dpr)
    const frame = frameRef.current
    paintReplay(ctx, doc, {
      view: scene.view,
      frame,
      dpr,
      floorImage: floorImage.current,
      floorSource: scene.floorSource,
      style,
      timing: scene.timing,
      zRange: scene.zRange,
      eventHoldFrames: scene.eventHoldFrames,
      arcThrowers: scene.arcThrowers,
      layers,
    })

    if (aliveRef.current) {
      aliveRef.current.textContent = String(doc.tracks.reduce((n, tr) => n + (isAliveAt(tr, frame) ? 1 : 0), 0))
    }
    const now = performance.now()
    if (now - publishedAtRef.current >= FRAME_PUBLISH_MS) {
      publishedAtRef.current = now
      onFrameChange(Math.floor(frame))
    }
  }, [doc, scene, style, layers, onFrameChange])

  // ORDER IS LOAD-BEARING between these two, and it is the reason the floor effect does not
  // call `draw` itself. Every input the floor depends on (`scene.floorGrid`, `scene.view`,
  // `scene.mapImage`, `style.floor`) is part of `scene` or `style`, which `draw` also depends
  // on — so a commit that repaints the floor is always a commit where `draw` changed too, and
  // effects run in declaration order. That is also what makes a map image arriving LATE show
  // up: it changes the scene, so the floor is repainted and the frame redrawn, in that order.
  // Letting the floor effect take `draw` as a dependency instead was the bug this ordering
  // replaces: it made every layer toggle re-rasterise the whole grid.
  useFloorImage(floorImage, {
    view: scene.view,
    height: scene.view.height,
    floorGrid: scene.floorGrid,
    calibration: scene.calibration,
    mapImage: scene.mapImage,
    style,
  })

  // Redraw outside the animation: theme, resize, data, pause, floor filter, focus.
  useEffect(() => {
    draw()
  }, [draw])

  useSnapToAnchor(anchor, appliedNonceRef, frameRef, draw)
  usePlaybackLoop({ playing, speed, doc, ready: scene.view.width > 0, frameRef, draw })

  /**
   * WHEN THE CLOCK STOPS, IT SAYS EXACTLY WHERE IT STOPPED.
   *
   * The report inside `draw` is throttled to 150 ms, which is right while the replay runs and
   * wrong the moment it does not: at 4x the loop can be six frames past its last report when
   * the reader hits pause, leaving the scrubber and the clock visibly behind the picture. One
   * unthrottled report on the way down closes that gap.
   */
  useEffect(() => {
    if (playing) return
    onFrameChange(Math.floor(frameRef.current))
  }, [playing, onFrameChange])

  return {
    containerRef,
    canvasRef,
    aliveRef,
    renderWidth: scene.view.width,
    renderHeight: scene.view.height,
    hasFloors: scene.hasFloors,
    floorSource: scene.floorSource,
  }
}

/** useObservedWidth reports the container's width in CSS pixels, 0 until it is measured. */
function useObservedWidth(ref: React.RefObject<HTMLElement | null>): number {
  const [width, setWidth] = useState(0)
  useEffect(() => {
    const el = ref.current
    if (!el) return
    const ro = new ResizeObserver((entries) => {
      setWidth(Math.max(Math.floor(entries[0]?.contentRect.width ?? 0), 0))
    })
    ro.observe(el)
    return () => ro.disconnect()
  }, [ref])
  return width
}

/**
 * useCanvasInks resolves the tokens the map is painted with.
 *
 * A canvas cannot read a CSS variable, so these have to be concrete values — and re-resolved
 * when the theme or the palette changes, which is what `useColorPaletteVersion` is for.
 */
function useCanvasInks(): { style: PaintStyle; reducedMotion: boolean } {
  const paletteVersion = useColorPaletteVersion()
  const style = useMemo<PaintStyle>(() => {
    void paletteVersion // re-resolve on a theme change: these read the DOM
    return {
      geometry: resolveToken(GEOMETRY_TOKEN),
      shot: resolveToken(SHOT_TOKEN),
      grenade: resolveToken(GRENADE_TOKEN),
      // Floor layout inks follow the theme, not the accessibility palette.
      floor: { fill: resolveToken(GEOMETRY_TOKEN), edge: readInk('--muted-foreground') },
    }
  }, [paletteVersion])

  /** REDUCED MOTION. The stylesheet honours it for the DOM; a canvas is reached by no CSS rule. */
  const reducedMotion = useMemo(
    () => typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches,
    [],
  )
  return { style, reducedMotion }
}

interface ReplayScene {
  view: CanvasView
  zRange: { min: number; max: number }
  hasFloors: boolean
  timing: MarkerTiming
  eventHoldFrames: number
  /** Slot that threw each projectile, aligned with `doc.projectiles` (null where unknown). */
  arcThrowers: (number | null)[]
  floorGrid: FloorGrid | null
  /** The calibrated image for this map, when one is configured AND has loaded. */
  mapImage: HTMLImageElement | null
  calibration: MapImageCalibration | null
  /** Which of the three floors is actually painted. */
  floorSource: FloorSource
}

/**
 * useReplayScene derives everything that depends on the document and the available width, and
 * on nothing that changes per frame.
 *
 * That distinction is what keeps the animation cheap: the altitude grid is built once for a
 * document, the frame timings are converted once from real time, and the framing is recomputed
 * only on a resize.
 */
function useReplayScene(
  doc: ReplayDocumentReady,
  width: number,
  calibration: MapImageCalibration | null,
  mapImage: HTMLImageElement | null,
): ReplayScene {
  const floorGrid = useMemo(() => {
    // A real match's structure is not automatically confined to where players actually
    // went — cf. floorScope.ts, whose header measures how much of it was not, on the
    // first real capture this app rendered. buildFloorGrid clamps an out-of-range
    // coordinate onto the nearest edge cell rather than dropping it, so an unfiltered
    // list does not fail loudly: it piles distant geometry onto the grid's border.
    const onFloor = floorSurfacesOf(doc.structure, doc.bounds)
    return onFloor.length > 0 ? buildFloorGrid(onFloor, doc.bounds) : null
  }, [doc.structure, doc.bounds])
  const bounds = useMemo(() => sceneBounds(doc), [doc])
  /**
   * Height first, from the scene's OWN shape — not a fixed constant — then width fit to
   * that height. A scene at least as wide as it is tall gets exactly what every scene
   * rendered at before this; only a portrait one grows past it. Cf. canvasFraming.ts for
   * why that has to be a branch on the scene's own aspect ratio and not a formula that
   * merely clamps toward it.
   */
  const view = useMemo<CanvasView>(() => {
    const height = sceneCanvasHeight(bounds, width, CANVAS_PAD)
    return {
      bounds,
      width: width === 0 ? 0 : Math.floor(fitWidth(bounds, width, height, CANVAS_PAD)),
      height,
      pad: CANVAS_PAD,
    }
  }, [bounds, width])
  const zRange = useMemo(
    () => ({ min: doc.bounds.minZ ?? 0, max: doc.bounds.maxZ ?? 0 }),
    [doc.bounds.minZ, doc.bounds.maxZ],
  )
  const timing = useMemo<MarkerTiming>(
    () => ({
      trail: COPIED_TRAIL_OFF,
      aimHold: msToFrames(TIMING_MS.aimHold, doc),
      shieldHold: msToFrames(TIMING_MS.shieldHold, doc),
      death: msToFrames(TIMING_MS.death, doc),
      spawn: msToFrames(TIMING_MS.spawn, doc),
    }),
    [doc],
  )
  const eventHoldFrames = useMemo(() => msToFrames(EVENT_HOLD_MS, doc), [doc])

  /**
   * WHOSE ARC IS WHOSE, computed once per document: the pairing depends on the throws and the
   * flights and on nothing that changes per frame, and it walks both lists.
   */
  const throwers = useMemo(
    () => arcThrowers(doc.grenades, doc.projectiles, msToFrames(ARC_ORIGIN_WINDOW_MS, doc)),
    [doc],
  )

  /**
   * THE WRAPPER IS MEMOISED TOO, and that is not tidiness — it is the whole point of the file.
   *
   * `draw` depends on this object, and three effects depend on `draw`. Returning a fresh
   * literal would give it a new identity on every render, so every 150 ms report would
   * re-rasterise the 45 000-cell floor, repaint outside the animation, AND tear down and
   * re-request the animation frame — which also drops the elapsed time between the two, making
   * playback run slow by an amount that varies with how often React happens to render.
   */
  return useMemo(
    () => ({
      view,
      zRange,
      hasFloors: zRange.max - zRange.min > MIN_FLOOR_SPAN,
      timing,
      eventHoldFrames,
      arcThrowers: throwers,
      floorGrid,
      mapImage,
      calibration,
      floorSource: floorSourceOf(floorGrid !== null, mapImage !== null, calibration?.preferOverStructure),
    }),
    [view, zRange, timing, eventHoldFrames, throwers, floorGrid, mapImage, calibration],
  )
}

/**
 * useSnapToAnchor moves the clock to a COMMANDED position, and ignores the echo of our own
 * report.
 *
 * The nonce is the whole test. Comparing frames instead would be wrong in both directions: a
 * seek to the frame we are already on would be ignored, and a published frame would drag the
 * clock backwards by up to 150 ms on every render.
 */
function useSnapToAnchor(
  anchor: FrameAnchor,
  appliedNonce: React.RefObject<number>,
  frame: React.RefObject<number>,
  draw: () => void,
): void {
  useEffect(() => {
    if (appliedNonce.current === anchor.nonce) return
    appliedNonce.current = anchor.nonce
    frame.current = anchor.frame
    draw()
  }, [anchor, appliedNonce, frame, draw])
}

/** usePlaybackLoop advances the clock at screen cadence, and only while the replay is playing. */
function usePlaybackLoop(o: {
  playing: boolean
  speed: number
  doc: ReplayDocumentReady
  /** False until the canvas has a width: there is nothing to animate onto yet. */
  ready: boolean
  frameRef: React.RefObject<number>
  draw: () => void
}): void {
  const { playing, speed, doc, ready, frameRef, draw } = o
  useEffect(() => {
    if (!playing || !ready) return
    let raf = 0
    let last = 0
    const step = (ts: number) => {
      if (last === 0) last = ts
      const dtMs = ts - last
      last = ts
      frameRef.current = advanceBy(frameRef.current, speed, dtMs, doc)
      draw()
      raf = requestAnimationFrame(step)
    }
    raf = requestAnimationFrame(step)
    return () => cancelAnimationFrame(raf)
  }, [playing, speed, doc, ready, frameRef, draw])
}
