/**
 * ReplayTimeline — the transport: where the replay is, and every way of moving it.
 *
 * NO LOGIC LIVES HERE. Every position this component can produce comes out of
 * `playbackLogic.ts` — including the answer to "is there a next death?", which is what
 * disables the control rather than a rule written twice. The component's whole job is to put
 * the actions on screen and to say, honestly, which of them lead somewhere.
 *
 * A CONTROL THAT LEADS NOWHERE IS DISABLED, NOT HIDDEN. A match with no objective actions is
 * the ordinary case (every slayer match), and a jump button that silently does nothing teaches
 * the reader to distrust the row. Disabled with a reason in its tooltip says which of the two
 * it is: no objectives in this match, or no more of them ahead.
 */
import { Button } from '@/components/ui/button'
import { CompactAction, Toggle } from '@/components/ui/controls'
import type { Locale } from '@/lib/i18n/locale'

import { REPLAY_TEXT } from '../replay/i18n'
import { formatClock, frameToMs } from '../replay/replayLogic'
import type { ReplayDocumentReady } from '../replay/replayNormalize'

import { VIEWER_TEXT } from './i18n'
import {
  FOCUSABLE_PLAYERS,
  lastFrameOf,
  nextStop,
  SPEEDS,
  type PlaybackAction,
  type PlaybackState,
  type Speed,
  type Timeline,
} from './playbackLogic'

interface ReplayTimelineProps {
  doc: ReplayDocumentReady
  timeline: Timeline
  state: PlaybackState
  dispatch: (action: PlaybackAction) => void
  locale: Locale
  /** A player's displayed name, or null when neither the film nor the archive names them. */
  nameOf: (xuid: string) => string | null
}

export function ReplayTimeline({
  doc,
  timeline,
  state,
  dispatch,
  locale,
  nameOf,
}: ReplayTimelineProps) {
  const v = VIEWER_TEXT[locale]
  return (
    <div className="flex flex-col gap-2 rounded-lg border border-border bg-card p-3">
      <TransportRow doc={doc} state={state} dispatch={dispatch} locale={locale} />
      <NavigationRow
        timeline={timeline}
        state={state}
        dispatch={dispatch}
        locale={locale}
        nameOf={nameOf}
      />
      <p className="text-xs text-muted-foreground">{v.shortcuts}</p>
    </div>
  )
}

/** TransportRow — play, restart, the match clock, and the scrubber. */
function TransportRow({
  doc,
  state,
  dispatch,
  locale,
}: {
  doc: ReplayDocumentReady
  state: PlaybackState
  dispatch: (action: PlaybackAction) => void
  locale: Locale
}) {
  const t = REPLAY_TEXT[locale]
  const v = VIEWER_TEXT[locale]
  const total = formatClock(doc.durationMs ?? frameToMs(doc.frameCount, doc))
  return (
    <div className="flex flex-wrap items-center gap-3">
      <Button variant="default" size="sm" onClick={() => dispatch({ type: 'toggle' })} className="h-8 w-24">
        {state.playing ? t.pause : t.play}
      </Button>
      <Button variant="ghost" size="sm" onClick={() => dispatch({ type: 'restart' })} className="h-8">
        {t.restart}
      </Button>
      {/* The clock has its OWN accessible name: two elements sharing one are indistinguishable
          to anyone driving by keyboard or by voice, and one of these is read while the other
          is dragged. */}
      <span className="min-w-[6rem] font-mono text-xs tabular-nums text-muted-foreground" aria-label={v.clock}>
        {formatClock(frameToMs(state.frame, doc))} / {total}
      </span>
      <input
        type="range"
        min={0}
        max={lastFrameOf(doc.frameCount)}
        value={state.frame}
        onChange={(e) => dispatch({ type: 'seek', frame: Number(e.currentTarget.value) })}
        className="min-w-[12rem] flex-1"
        aria-label={t.time}
      />
    </div>
  )
}

/** NavigationRow — stepping, the jumps, the speeds, and who the view is following. */
function NavigationRow({
  timeline,
  state,
  dispatch,
  locale,
  nameOf,
}: {
  timeline: Timeline
  state: PlaybackState
  dispatch: (action: PlaybackAction) => void
  locale: Locale
  nameOf: (xuid: string) => string | null
}) {
  const t = REPLAY_TEXT[locale]
  const v = VIEWER_TEXT[locale]
  const lastFrame = lastFrameOf(timeline.frameCount)
  return (
    <div className="flex flex-wrap items-center gap-1">
      <StepButton
        glyph="−1"
        label={v.stepBack}
        hint={v.stepHint}
        onClick={() => dispatch({ type: 'step', delta: -1 })}
        enabled={state.frame > 0}
      />
      <StepButton
        glyph="+1"
        label={v.stepForward}
        hint={v.stepHint}
        onClick={() => dispatch({ type: 'step', delta: 1 })}
        enabled={state.frame < lastFrame}
      />
      <Separator />
      <JumpControls
        stops={timeline.deaths}
        frame={state.frame}
        onJump={(direction) => dispatch({ type: 'jumpDeath', direction })}
        prevLabel={v.prevDeath}
        nextLabel={v.nextDeath}
        hint={timeline.deaths.length === 0 ? v.noDeaths : v.deathHint}
        count={`${timeline.deaths.length} ${v.deathsLabel}`}
      />
      <Separator />
      <JumpControls
        stops={timeline.objectives}
        frame={state.frame}
        onJump={(direction) => dispatch({ type: 'jumpObjective', direction })}
        prevLabel={v.prevObjective}
        nextLabel={v.nextObjective}
        hint={v.objectiveHint}
        count={`${timeline.objectives.length} ${v.objectivesLabel}`}
      />
      <Separator />
      <span className="mr-1 text-xs text-muted-foreground">{t.speed}</span>
      <SpeedControls speed={state.speed} onPick={(speed) => dispatch({ type: 'speed', speed })} />
      <Separator />
      <FocusPicker
        players={timeline.players}
        nameOf={nameOf}
        focus={state.focus}
        onPick={(index) => dispatch({ type: 'focus', index })}
        onRelease={() => dispatch({ type: 'focusXUID', xuid: null })}
        locale={locale}
      />
    </div>
  )
}

/** SpeedControls — the playback multipliers. `1×` is the match's own clock, not a frame rate. */
function SpeedControls({ speed, onPick }: { speed: Speed; onPick: (speed: Speed) => void }) {
  return (
    <>
      {SPEEDS.map((m) => (
        <Toggle key={m} on={speed === m} onClick={() => onPick(m)}>
          {m < 1 ? `${m.toFixed(1)}×` : `${m.toFixed(0)}×`}
        </Toggle>
      ))}
    </>
  )
}

/**
 * FocusPicker — which digit follows whom, on screen.
 *
 * WHY THE DIGITS ARE DRAWN AND NOT JUST DOCUMENTED. `1`–`8` address the players in the film's
 * roster order, and nothing else on the screen shows that order: the roster panel is copied
 * verbatim from `apps/web` and cannot be given a number or a click handler from here. A
 * shortcut whose mapping is invisible is a shortcut nobody uses, so the mapping IS the control
 * — each digit is a button carrying its player's name, and pressing the key does what clicking
 * the button does.
 *
 * A ninth player has no digit and no button, and that is stated in the tooltip rather than
 * hidden: the archive holds other people's matches, and one of them may be bigger than an
 * arena match.
 */
function FocusPicker({
  players,
  nameOf,
  focus,
  onPick,
  onRelease,
  locale,
}: {
  players: string[]
  /** The player's displayed name, or null when neither the film nor the archive names them. */
  nameOf: (xuid: string) => string | null
  focus: string | null
  onPick: (index: number) => void
  onRelease: () => void
  locale: Locale
}) {
  const v = VIEWER_TEXT[locale]
  const label = (xuid: string) => nameOf(xuid) ?? REPLAY_TEXT[locale].unknownPlayer
  const addressable = players.slice(0, FOCUSABLE_PLAYERS)
  return (
    <>
      <span className="mr-1 text-xs text-muted-foreground" title={v.focusHint}>
        {v.focus}
      </span>
      {addressable.map((xuid, i) => (
        <Toggle
          key={xuid}
          mono
          on={focus === xuid}
          onClick={() => onPick(i)}
          title={`${i + 1} — ${label(xuid)}`}
          label={`${v.focus}: ${label(xuid)}`}
        >
          {i + 1}
        </Toggle>
      ))}
      {focus === null ? (
        <span className="ml-1 text-xs text-muted-foreground">{v.focusNone}</span>
      ) : (
        <CompactAction variant="outline" onClick={onRelease} title={v.focusClear}>
          {label(focus)} ×
        </CompactAction>
      )}
    </>
  )
}

/** Separator — a hairline between groups of controls, and nothing a reader has to hear. */
function Separator() {
  return <span aria-hidden className="mx-1 h-4 w-px bg-border" />
}

/** StepButton — one frame in one direction. Disabled at the end of the axis it points at. */
function StepButton({
  glyph,
  label,
  hint,
  onClick,
  enabled,
}: {
  glyph: string
  label: string
  hint: string
  onClick: () => void
  enabled: boolean
}) {
  return (
    <CompactAction mono onClick={onClick} enabled={enabled} title={hint} label={label}>
      {glyph}
    </CompactAction>
  )
}

/**
 * JumpControls — previous and next of one kind of stop, with how many there are.
 *
 * THE COUNT IS PART OF THE CONTROL. "0 objective actions" is an answer about the match; a pair
 * of greyed arrows on their own would read as a broken feature.
 */
function JumpControls({
  stops,
  frame,
  onJump,
  prevLabel,
  nextLabel,
  hint,
  count,
}: {
  stops: number[]
  frame: number
  onJump: (direction: 1 | -1) => void
  prevLabel: string
  nextLabel: string
  hint: string
  count: string
}) {
  return (
    <>
      <CompactAction
        onClick={() => onJump(-1)}
        enabled={nextStop(stops, frame, -1) !== null}
        title={hint}
        label={prevLabel}
      >
        ◀
      </CompactAction>
      <span className="px-1 text-xs text-muted-foreground" title={hint}>
        {count}
      </span>
      <CompactAction
        onClick={() => onJump(1)}
        enabled={nextStop(stops, frame, 1) !== null}
        title={hint}
        label={nextLabel}
      >
        ▶
      </CompactAction>
    </>
  )
}
