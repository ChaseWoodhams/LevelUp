/**
 * ReplayViewer — the whole screen of one match: the map, the transport, the roster, and the
 * banner that says what is missing from all three.
 *
 * WHAT THIS COMPONENT OWNS: the playback state, the keyboard, and the resolution of a team
 * TOKEN into an actual colour. Nothing else — the map draws, the panel reads, the banner
 * counts, and each of them is handed what it needs rather than asked to work it out.
 *
 * WHY THE COLOURS ARE RESOLVED HERE AND NOT IN THE CANVAS. Two surfaces have to show the same
 * player in the same colour: the canvas, which needs a concrete value because a canvas cannot
 * read a CSS variable, and the roster panel, which uses `var(--ac-*)` directly and reacts to a
 * theme change on its own. Resolving once at the point where both are composed is what makes
 * "the same colour" a fact rather than a coincidence between two call sites.
 *
 * THE COVERAGE BANNER IS STUCK BESIDE THE MAP, on purpose. It says how much of each layer
 * could be attached, and a reader scrolled down to the roster with the banner off screen is
 * exactly the reader who will mistake what was attached for everything that happened. It stays
 * next to the layers it describes.
 */
import { useCallback, useEffect, useMemo, useReducer } from 'react'

import { resolveToken } from '@/lib/accessibility/resolveToken'
import { useColorPaletteVersion } from '@/lib/accessibility/useColorPaletteVersion'
import type { Locale } from '@/lib/i18n/locale'
import type { MatchScoreboardRow } from '@/lib/api/types'

import { readInk } from '../replay/canvasInk'
import { ReplayCoverageBanner } from '../replay/ReplayCoverage'
import { ReplayTeams } from '../replay/ReplayTeams'
import type { ReplayDocumentReady } from '../replay/replayNormalize'
import { playerName } from '../replay/rosterLogic'

import { actionForKey, focusOwnsKey } from './keyboard'
import {
  initialPlayback,
  playbackReducer,
  timelineOf,
  type PlaybackAction,
  type PlaybackState,
} from './playbackLogic'
import { ReplayTimeline } from './ReplayTimeline'
import { StudyReplayCanvas } from './StudyReplayCanvas'
import {
  buildRosterColoring,
  focusInk,
  trackInks,
  xuidOfSlotAt,
  type RosterColoring,
} from './teamColors'

interface ReplayViewerProps {
  doc: ReplayDocumentReady
  scoreboard: MatchScoreboardRow[]
  locale: Locale
  /**
   * The archive's key for the map, or null when it is not known.
   *
   * IT IS NOT IN THE ARTIFACT and cannot be: a replay document carries its match, its title and
   * its bounds, and no map. It arrives from the archive's own row for the match, and the floor's
   * calibrated-image fallback is what needs it. Null is an ordinary case — the sample artifact
   * belongs to no archived match at all — and it simply means the floor falls through.
   */
  mapModule?: string | null
  /**
   * Zéro de l'horloge du MATCH sur l'axe du document, en ms ; null = non mesuré.
   *
   * IL ARRIVE À CÔTÉ DU DOCUMENT, ET NON DEDANS. `replayNormalize.ts` est un fichier COPIÉ,
   * byte-identique à `apps/web` : ce viewer ne peut pas y ajouter un champ dont l'app n'a pas
   * besoin sans passer d'abord en amont. `parseReplayPayload` le lit donc sur le document brut
   * et le porte à part (cf. features/archive/schemaVersion.ts).
   *
   * Null est le cas ordinaire d'un artefact dont le fil des morts n'a rien apparié : l'écran
   * n'affiche alors pas d'heure de partie, plutôt qu'une heure fausse.
   */
  matchClockZeroMs?: number | null
}

export function ReplayViewer({
  doc,
  scoreboard,
  locale,
  mapModule = null,
  matchClockZeroMs = null,
}: ReplayViewerProps) {
  const coloring = useMemo(() => buildRosterColoring(doc, scoreboard), [doc, scoreboard])
  const timeline = useMemo(() => timelineOf(doc), [doc])

  // The reducer is rebound when the document changes; React always calls the one from the
  // current render, so a jump can never be resolved against another match's death list.
  const reduce = useCallback(
    (state: PlaybackState, action: PlaybackAction) => playbackReducer(state, action, timeline),
    [timeline],
  )
  const [state, dispatch] = useReducer(reduce, undefined, initialPlayback)

  const inks = useTeamInks(coloring)

  const trackColors = useMemo(
    () =>
      trackInks(doc.tracks, {
        inkOfXUID: (xuid) => inks.byXUID.get(xuid),
        anonymous: inks.anonymous,
        focus: state.focus,
      }),
    [doc.tracks, inks, state.focus],
  )

  const inkOfSlotAt = useCallback(
    (slot: number, frame: number, hold: number) => {
      const xuid = xuidOfSlotAt(doc.tracks, slot, frame, hold)
      if (xuid === null) return null
      return focusInk(inks.byXUID.get(xuid) ?? inks.anonymous, xuid, state.focus)
    },
    [doc.tracks, inks, state.focus],
  )

  useReplayKeyboard(dispatch)

  const anchor = useMemo(
    () => ({ frame: state.frame, nonce: state.seekNonce }),
    [state.frame, state.seekNonce],
  )
  const onFrameChange = useCallback((frame: number) => dispatch({ type: 'published', frame }), [])

  const nameOf = useCallback(
    (xuid: string) => {
      const player = coloring.players.find((p) => p.xuid === xuid)
      return player ? playerName(player) : null
    },
    [coloring.players],
  )

  return (
    <div className="flex flex-col gap-3">
      <div className="grid gap-3 lg:grid-cols-[minmax(0,3fr)_minmax(0,1fr)]">
        <div className="flex flex-col gap-3">
          <StudyReplayCanvas
            doc={doc}
            locale={locale}
            inks={trackColors}
            inkOfSlotAt={inkOfSlotAt}
            playing={state.playing}
            speed={state.speed}
            anchor={anchor}
            onFrameChange={onFrameChange}
            mapModule={mapModule}
          />
          <ReplayTimeline
            doc={doc}
            matchClockZeroMs={matchClockZeroMs}
            timeline={timeline}
            state={state}
            dispatch={dispatch}
            locale={locale}
            nameOf={nameOf}
          />
        </div>
        <div className="self-start lg:sticky lg:top-3">
          <ReplayCoverageBanner doc={doc} locale={locale} />
        </div>
      </div>
      <ReplayTeams doc={doc} scoreboard={scoreboard} frame={state.frame} locale={locale} />
    </div>
  )
}

/**
 * useTeamInks resolves each team's token into the colour a canvas can actually paint with.
 *
 * The roster panel reads `var(--ac-*)` and follows a theme change on its own; the canvas
 * cannot, so the value is resolved here — at the one place both surfaces are composed, which
 * is what makes "the same player, the same colour" a fact rather than a coincidence.
 */
function useTeamInks(coloring: RosterColoring): { byXUID: Map<string, string>; anonymous: string } {
  const paletteVersion = useColorPaletteVersion()
  return useMemo(() => {
    void paletteVersion // re-resolve on a theme change: these read the DOM
    const byXUID = new Map<string, string>()
    coloring.tokenOfXUID.forEach((token, xuid) => byXUID.set(xuid, resolveToken(token)))
    return { byXUID, anonymous: readInk('--muted-foreground') }
  }, [coloring, paletteVersion])
}

/**
 * useReplayKeyboard binds the shortcuts to the window rather than to a focusable region.
 *
 * The reader's hands are on the keys and their eyes are on the map; asking them to click the
 * canvas first would defeat the shortcut. `focusOwnsKey` is what stops that reach from taking
 * keys the focused element needs — every key from a text field or the scrubber, and Space and
 * Enter from a focused button.
 */
function useReplayKeyboard(dispatch: (action: PlaybackAction) => void): void {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (focusOwnsKey(e.target, e.key)) return
      const action = actionForKey(e)
      if (action === null) return
      e.preventDefault()
      dispatch(action)
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [dispatch])
}
