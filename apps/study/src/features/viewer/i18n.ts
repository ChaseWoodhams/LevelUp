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
    trail: 'Traînée',
    trailWindow: (seconds) => `${seconds} s`,
    trailFull: 'Vie entière',
    trailHint:
      'Longueur du chemin laissé derrière chaque joueur. Le trait pâlit avec l’âge : le bout franc est l’instant présent, le bout pâle le début de la fenêtre. « Vie entière » remonte à l’apparition.',
    layerShotsHint:
      'Chaque tir part du tireur dans la direction de sa visée, jamais vers un autre joueur : le film ne dit pas qui a été touché. Il n’enregistre que les tirs qui INFLIGENT un dégât — il n’y a donc pas de tir manqué dans ces données, et rien ici ne se lit comme une précision.',
    layerGrenadesHint:
      'Le point marque le lancer, l’arc la trajectoire répliquée du projectile. L’arc prend la couleur du lanceur quand le film permet de les rapprocher, et reste neutre sinon. Le dernier point n’est pas une explosion : la réplication s’arrête avant la mèche.',
    floorLine: (source) => `Sol : ${source}`,
    floorSource: {
      structure: {
        label: 'géométrie reconstruite',
        hint: 'Le sol vient de la structure de la carte portée par l’artefact : la même donnée que celle qui porte les trajectoires, à 8 mm d’écart médian des positions.',
      },
      image: {
        label: 'image calibrée',
        hint: 'Aucune structure dans l’artefact : le fond est une image vue du dessus, posée à la main sur les coordonnées monde de ses coins (mapImages.config.ts). C’est un repère, pas une mesure.',
      },
      grid: {
        label: 'grille',
        hint: 'Ni structure ni image calibrée pour cette carte : le fond est une simple grille métrique, alignée sur l’origine du monde. Elle ne montre aucun mur — elle donne l’échelle, et sert à relever les quatre coordonnées d’une calibration.',
      },
    },
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
