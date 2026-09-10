/**
 * StudyReplayCanvas — the map, drawn at a frame somebody else owns.
 *
 * DERIVED FROM apps/web/src/features/match-replay/ReplayCanvas.tsx @ 0afd83f7e — and
 * DELIBERATELY NOT A COPY, which is why it carries no `COPIED FILE` header and no drift guard.
 * The layer order, the offscreen floor, the device-pixel handling and the timing constants are
 * that component's, verbatim in spirit; two things had to change, and neither can be expressed
 * through its props:
 *
 *   1. THE COLOURS ARE GIVEN, not derived. The origin paints each TRACK from the chart series
 *      palette — a colour per life, so a player is repainted at every respawn. This viewer
 *      paints by TEAM, from the same tokens the roster panel uses, and the mapping needs the
 *      archive's participants rows, which the origin has never heard of.
 *   2. THE CLOCK IS OUTSIDE. The origin owns play, pause, speed and position; the study tool
 *      adds frame stepping, jumps between deaths and objectives, and keyboard control, all of
 *      which have to act on the same clock. A component that owns its own position cannot be
 *      driven from outside it.
 *
 * `git diff 0afd83f7e HEAD -- apps/web/src/features/match-replay/ReplayCanvas.tsx` is still the
 * right thing to read when the origin moves: everything it learns about DRAWING applies here.
 *
 * WHAT THIS FILE IS LEFT WITH: the markup, and the two view choices that belong to it. Painting
 * is `paintReplay.ts`, scheduling is `useReplayPainter.ts`.
 *
 * THE TOGGLES AND THE FLOOR FILTER STAY LOCAL. They change what is drawn and nothing else — no
 * keyboard shortcut, no URL, no other panel reads them — so hoisting them would be state moved
 * away from its only reader.
 */
import { useState } from 'react'

import { Toggle } from '@/components/ui/controls'

import { REPLAY_TEXT, type ReplayLocale } from '../replay/i18n'
import { FLOOR_BANDS } from '../replay/replayLogic'
import type { ReplayDocumentReady } from '../replay/replayNormalize'

import { VIEWER_TEXT } from './i18n'
import type { PaintLayers } from './paintReplay'
import { DEFAULT_TRAIL_WINDOW_MS, TRAIL_WINDOW_MS } from './trailLogic'
import { useReplayPainter, type FrameAnchor } from './useReplayPainter'

interface StudyReplayCanvasProps {
  doc: ReplayDocumentReady
  locale: ReplayLocale
  /** One colour per track, aligned with `doc.tracks`. Cf. `teamColors.trackInks`. */
  inks: string[]
  /** The colour of whoever owned a slot at a frame, within a mark's lingering window. */
  inkOfSlotAt: PaintLayers['inkOfSlotAt']
  playing: boolean
  speed: number
  anchor: FrameAnchor
  /** Called at reduced cadence with the current frame: the panels outside the canvas read it. */
  onFrameChange: (frame: number) => void
  /** The archive's key for the map, for the floor's calibrated-image fallback. */
  mapModule: string | null
}

export function StudyReplayCanvas({
  doc,
  locale,
  inks,
  inkOfSlotAt,
  playing,
  speed,
  anchor,
  onFrameChange,
  mapModule,
}: StudyReplayCanvasProps) {
  const t = REPLAY_TEXT[locale]
  const v = VIEWER_TEXT[locale]
  const [floor, setFloor] = useState<number | null>(null)
  const [showAim, setShowAim] = useState(true)
  const [showShield, setShowShield] = useState(true)
  const [showShots, setShowShots] = useState(true)
  const [showGrenades, setShowGrenades] = useState(true)
  const [trailWindowMs, setTrailWindowMs] = useState<number | null>(DEFAULT_TRAIL_WINDOW_MS)

  const painter = useReplayPainter({
    doc,
    inks,
    inkOfSlotAt,
    playing,
    speed,
    anchor,
    onFrameChange,
    showAim,
    showShield,
    showShots,
    showGrenades,
    trailWindowMs,
    mapModule,
    floor,
  })
  const floorText = v.floorSource[painter.floorSource]

  return (
    <div ref={painter.containerRef} className="rounded-lg border border-border bg-card">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-border px-3 py-2">
        <div className="flex items-baseline gap-2 text-sm">
          <span className="font-medium">
            {doc.tracks.length} {t.livesSuffix}
          </span>
          <span className="text-xs text-muted-foreground">
            {/* Written to directly at screen cadence — the one number that changes per frame. */}
            <span ref={painter.aliveRef} /> {t.aliveSuffix}
          </span>
        </div>
        <div className="flex flex-wrap items-center gap-1">
          <span className="mr-1 text-xs text-muted-foreground">{t.layers}</span>
          <Toggle on={showAim} onClick={() => setShowAim((on) => !on)} title={t.layerAimHint}>
            {t.layerAim}
          </Toggle>
          <Toggle on={showShield} onClick={() => setShowShield((on) => !on)} title={t.layerShieldHint}>
            {t.layerShield}
          </Toggle>
          <Toggle on={showShots} onClick={() => setShowShots((on) => !on)} title={v.layerShotsHint}>
            {t.layerShots}
          </Toggle>
          <Toggle
            on={showGrenades}
            onClick={() => setShowGrenades((on) => !on)}
            title={v.layerGrenadesHint}
          >
            {t.layerGrenades}
          </Toggle>
          <TrailWindow chosen={trailWindowMs} onPick={setTrailWindowMs} text={v} />
          {painter.hasFloors && (
            <FloorFilter floor={floor} onPick={setFloor} labels={floorLabels(t)} allLabel={t.floorAll} title={t.floor} />
          )}
        </div>
      </div>
      <div className="p-3">
        <canvas
          ref={painter.canvasRef}
          className="mx-auto block"
          style={{ width: painter.renderWidth || '100%', height: painter.renderHeight }}
        />
        <p className="mt-2 text-xs text-muted-foreground">
          {t.note}
          {doc.geometry.length > 0 ? ` ${doc.geometry.length} ${t.propsSuffix}.` : ''}
        </p>
        {/* WHICH FLOOR IS UNDER THE MATCH, always said. A grid and a reconstructed geometry do
            not carry the same claims, and a reader who takes one for the other measures
            distances on a background that does not support them. */}
        <p className="mt-1 text-xs text-muted-foreground" title={floorText.hint}>
          {v.floorLine(floorText.label)}
        </p>
      </div>
    </div>
  )
}

/**
 * TrailWindow — how far back each player's path is drawn, including all of it.
 *
 * THE LABELS ARE DERIVED FROM THE WINDOWS, not written beside them: the list lives in
 * `trailLogic.ts`, and a hand-written label per value would be a second list to keep in step.
 */
function TrailWindow({
  chosen,
  onPick,
  text,
}: {
  chosen: number | null
  onPick: (ms: number | null) => void
  text: (typeof VIEWER_TEXT)['fr']
}) {
  return (
    <>
      <span aria-hidden className="mx-2 h-4 w-px bg-border" />
      <span className="mr-1 text-xs text-muted-foreground">{text.trail}</span>
      {TRAIL_WINDOW_MS.map((ms) => (
        <Toggle
          key={ms ?? 'full'}
          on={chosen === ms}
          onClick={() => onPick(ms)}
          title={text.trailHint}
        >
          {ms === null ? text.trailFull : text.trailWindow(Math.round(ms / 1000))}
        </Toggle>
      ))}
    </>
  )
}

/** floorLabels names the altitude bands, in the order `floorOf` numbers them. */
function floorLabels(t: (typeof REPLAY_TEXT)['fr']): string[] {
  return [t.floorLow, t.floorMid, t.floorHigh].slice(0, FLOOR_BANDS)
}

/** FloorFilter — one band at a time, or all of them. Offered only on a map with relief. */
function FloorFilter({
  floor,
  onPick,
  labels,
  allLabel,
  title,
}: {
  floor: number | null
  onPick: (floor: number | null) => void
  labels: string[]
  allLabel: string
  title: string
}) {
  return (
    <>
      <span aria-hidden className="mx-2 h-4 w-px bg-border" />
      <span className="mr-1 text-xs text-muted-foreground">{title}</span>
      <Toggle on={floor === null} onClick={() => onPick(null)}>
        {allLabel}
      </Toggle>
      {labels.map((label, i) => (
        <Toggle key={label} on={floor === i} onClick={() => onPick(i)}>
          {label}
        </Toggle>
      ))}
    </>
  )
}

