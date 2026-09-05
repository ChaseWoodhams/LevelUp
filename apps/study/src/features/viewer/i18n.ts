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

interface ViewerText {
  /**
   * Le chronomètre, en lecture seule.
   *
   * IL A SON PROPRE LIBELLÉ et ne partage pas celui du curseur : deux éléments portant le
   * même nom accessible sont indiscernables pour qui navigue au clavier ou à la voix — l'un
   * se lit, l'autre se manipule.
   */
  clock: string
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
}

export const VIEWER_TEXT: Record<Locale, ViewerText> = {
  fr: {
    clock: 'Chronomètre du match',
    stepBack: 'Image précédente',
    stepForward: 'Image suivante',
    stepHint: 'Avance d’une image. Avec Maj : une seconde de match.',
    prevDeath: 'Mort précédente',
    nextDeath: 'Mort suivante',
    deathHint:
      'Fin d’une vie nommée par le film. Une vie que le film n’a pas nommée, ou qui dure jusqu’à la dernière image, n’est pas une mort : elle n’est pas proposée.',
    prevObjective: 'Action d’objectif précédente',
    nextObjective: 'Action d’objectif suivante',
    objectiveHint:
      'Capture, retour, prise de zone — ce que le joueur a FAIT, daté par le film. Absent hors mode à objectifs.',
    noDeaths: 'Aucune mort lisible dans ce film',
    focus: 'Focus',
    focusNone: 'Tous les joueurs',
    focusHint:
      'Met un joueur en avant : les autres restent dessinés, en retrait. Touches 1 à 8, la même touche relâche.',
    focusClear: 'Relâcher le focus',
    shortcuts:
      'Espace : lecture/pause · Flèches : image par image (Maj : une seconde) · , et . : entre les morts · 1 à 8 : suivre un joueur (0 relâche)',
    deathsLabel: 'morts',
    objectivesLabel: 'actions d’objectif',
  },
  en: {
    clock: 'Match clock',
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
  },
}
