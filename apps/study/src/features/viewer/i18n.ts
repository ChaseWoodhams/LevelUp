/**
 * i18n.ts — the strings of the study viewer's own controls, FR and EN.
 *
 * The copied replay modules bring their own table (`features/replay/i18n.ts`) and it still
 * covers everything they draw: layers, floors, the coverage banner, the player cards. What is
 * here is only what this app added — the transport controls, the jumps, and the focus — held
 * in the same shape and by the same rule: parity by typing, so a string cannot exist in one
 * language only.
 */
import type { Locale } from '@/lib/i18n/locale'

import type { FloorSource } from './mapCalibration'

/** What the floor under a match is, and where it came from. One entry per fallback. */
interface FloorSourceText {
  label: string
  hint: string
}

interface ViewerText {
  /**
   * Le chronomètre, en lecture seule.
   *
   * IL A SON PROPRE LIBELLÉ et ne partage pas celui du curseur : deux éléments portant le
   * même nom accessible sont indiscernables pour qui navigue au clavier ou à la voix — l'un
   * se lit, l'autre se manipule.
   */
  clock: string
  /** L'heure de l'horloge du MATCH, distincte de la position sur l'axe du document. */
  matchClock: string
  matchClockLabel: string
  matchClockHint: string
  /** Transport. */
  stepBack: string
  stepForward: string
  stepHint: string
  prevDeath: string
  nextDeath: string
  deathHint: string
  prevObjective: string
  nextObjective: string
  objectiveHint: string
  noDeaths: string
  /** Focus. */
  focus: string
  focusNone: string
  focusHint: string
  focusClear: string
  /** The shortcut sheet, one line under the transport. */
  shortcuts: string
  /** Counters beside the scrubber. */
  deathsLabel: string
  objectivesLabel: string
  /**
   * Le calque de traînée : sa fenêtre, et ce qu'elle veut dire.
   *
   * `trailWindow` prend les secondes plutôt que de figer trois libellés : la liste des
   * fenêtres offertes vit dans `trailLogic.ts`, et un libellé par valeur les ferait diverger.
   */
  trail: string
  trailWindow: (seconds: number) => string
  trailFull: string
  trailHint: string
  /** Calques d'événements ponctuels. */
  layerShotsHint: string
  layerGrenadesHint: string
  /**
   * Le SOL : lequel des trois est sous le match, dit explicitement.
   *
   * Un repli n'est pas un détail d'implémentation : une grille et une géométrie reconstruite
   * ne se lisent pas de la même façon, et un lecteur qui prend l'une pour l'autre mesure des
   * distances sur un fond qui ne les porte pas.
   */
  floorLine: (source: string) => string
  floorSource: Record<FloorSource, FloorSourceText>
}

export const VIEWER_TEXT: Record<Locale, ViewerText> = {
  en: {
    clock: 'Replay clock',
    matchClock: 'Match clock',
    matchClockLabel: 'match',
    matchClockHint:
      'Time on the match clock. Frame 0 of the replay is not the start of the match: the film ' +
      'begins at the first replicated position, during the pre-match.',
    stepBack: 'Previous frame',
    stepForward: 'Next frame',
    stepHint: 'Steps one frame. With Shift: one second of match time.',
    prevDeath: 'Previous death',
    nextDeath: 'Next death',
    deathHint:
      'The end of a life the film named. A life the film never named, or one that runs to the last frame, is not a death and is not offered.',
    prevObjective: 'Previous objective action',
    nextObjective: 'Next objective action',
    objectiveHint:
      'A capture, a return, a zone secured — what a player DID, dated by the film. Absent outside objective modes.',
    noDeaths: 'No readable death in this film',
    focus: 'Focus',
    focusNone: 'All players',
    focusHint:
      'Brings one player forward: the others stay on the map, held back. Keys 1 to 8; the same key releases.',
    focusClear: 'Release the focus',
    shortcuts:
      'Space: play/pause · Arrows: frame by frame (Shift: one second) · , and .: between deaths · 1 to 8: follow a player (0 releases)',
    deathsLabel: 'deaths',
    objectivesLabel: 'objective actions',
    trail: 'Trail',
    trailWindow: (seconds) => `${seconds} s`,
    trailFull: 'Full life',
    trailHint:
      'How much of the path behind each player is drawn. The line fades with age: the bright end is now, the pale end is the start of the window. "Full life" reaches back to the spawn.',
    layerShotsHint:
      'Every shot points from the shooter along their aim, never at another player: the film does not record who was hit. It only records shots that DEAL damage — there is no such thing as a missed shot in this data, and nothing here reads as accuracy.',
    layerGrenadesHint:
      'The dot marks the throw, the arc is the projectile flight as replicated. An arc takes its thrower’s colour when the film lets the two be matched, and stays neutral otherwise. The last point is not a detonation: replication stops before the fuse does.',
    floorLine: (source) => `Floor: ${source}`,
    floorSource: {
      structure: {
        label: 'reconstructed geometry',
        hint: 'The floor comes from the map structure the artifact carries: the same data that carries the trajectories, a median 8 mm from the positions themselves.',
      },
      image: {
        label: 'calibrated image',
        hint: 'No structure in the artifact: the background is a top-down image, placed by hand on the world coordinates of its corners (mapImages.config.ts). It is a landmark, not a measurement.',
      },
      grid: {
        label: 'grid',
        hint: 'Neither structure nor a calibrated image for this map: the background is a plain metric grid, aligned on the world origin. It shows no walls — it gives the scale, and it is what a calibration’s four coordinates are read off.',
      },
    },
  },
}
