/**
 * i18n of the study tool's own chrome — everything that is NOT the replay view.
 *
 * The replay modules copied under `features/replay/` bring their own table
 * (`REPLAY_TEXT`) and the viewer adds its transport strings (`VIEWER_TEXT`); this one
 * covers the frame around them: the landing screen, and what each way of failing to open
 * a match says. Same rule as the rest of the fork: every UI string exists in French AND
 * English, and the parity is held by the type, not by discipline.
 */
import type { Locale } from '@/lib/i18n/locale'

interface ShellText {
  /** Titre du document — suit la langue choisie, comme tout le reste de l'écran. */
  documentTitle: string
  appName: string
  localeLabel: string
  /** Écran d'accueil. */
  homeIntro: string
  openLabel: string
  openPlaceholder: string
  openSubmit: string
  openHint: string
  sampleLink: string
  sampleBadge: string
  sampleHint: string
  back: string
  /** États d'ouverture d'un match. */
  loading: string
  notFound: string
  notFoundHint: string
  noArtifact: string
  noArtifactHint: string
  busy: string
  busyHint: string
  failed: string
  failedHint: string
  retry: string
  /**
   * Refus de rendu : la version de schéma de l'artefact n'est pas celle que ce lecteur sait
   * lire. Prend les deux versions — dire laquelle a été trouvée est ce qui rend le message
   * actionnable.
   */
  unsupported: (found: number, supported: number) => string
  unsupportedHint: string
}

export const SHELL_TEXT: Record<Locale, ShellText> = {
  fr: {
    documentTitle: 'LevelUp — Étude',
    appName: 'Outil d’étude',
    localeLabel: 'Langue',
    homeIntro:
      'Ouvrir un match archivé pour le regarder se dérouler vue du dessus : trajectoires, tirs, lancers, et les fiches joueur à côté de la carte.',
    openLabel: 'Identifiant du match',
    openPlaceholder: '000d5950',
    openSubmit: 'Ouvrir',
    openHint:
      'Forme courte ou complète, les deux fonctionnent. Le serveur d’étude (study-server) doit tourner : `go run ./apps/go-api/cmd/study-server`.',
    sampleLink: 'Voir l’artefact de démonstration',
    sampleBadge: 'Artefact de démonstration',
    sampleHint:
      'Aucune donnée réelle : cet artefact est écrit à la main pour montrer chaque calque du rejeu, sans archive ni serveur.',
    back: 'Retour',
    loading: 'Lecture de l’archive…',
    notFound: 'Aucun match archivé sous cet identifiant',
    notFoundHint:
      'L’archive ne connaît ce match ni sous sa forme courte ni sous sa forme complète. Vérifier l’identifiant, ou capturer le film avec `study-archiver fetch-one`.',
    noArtifact: 'Ce match est enregistré, mais son artefact de rejeu est absent',
    noArtifactHint:
      'Rien à dessiner : le match a été relevé sans que son film soit assemblé, ou l’artefact a disparu du cache depuis. `study-archiver rebuild <match>` le reconstruit hors ligne.',
    busy: 'L’archive est momentanément tenue par une capture',
    busyHint:
      'Une seule écriture à la fois sur la base : la capture horaire la tient. Rien n’est cassé — réessayer dans quelques secondes.',
    failed: 'Le serveur d’étude n’a pas répondu',
    failedHint:
      'Le serveur est-il démarré ? `go run ./apps/go-api/cmd/study-server` écoute sur 127.0.0.1:8100, et le serveur de développement le relaie.',
    retry: 'Réessayer',
    unsupported: (found, supported) =>
      `Artefact en version de schéma ${found} ; ce lecteur lit la version ${supported}.`,
    unsupportedHint:
      'Rien n’est dessiné, volontairement : lire un document d’une autre version avec ces règles-ci produirait une carte plausible et fausse. Reconstruire l’artefact (`study-archiver rebuild <match>`) le réécrit dans la version courante.',
  },
  en: {
    documentTitle: 'LevelUp — Study',
    appName: 'Study tool',
    localeLabel: 'Language',
    homeIntro:
      'Open an archived match and watch it play out top-down: trails, shots, throws, and the player cards beside the map.',
    openLabel: 'Match identifier',
    openPlaceholder: '000d5950',
    openSubmit: 'Open',
    openHint:
      'Short or full form, either works. The study server has to be running: `go run ./apps/go-api/cmd/study-server`.',
    sampleLink: 'See the sample artifact',
    sampleBadge: 'Sample artifact',
    sampleHint:
      'No real data: this artifact is written by hand to put every replay layer on screen, with no archive and no server.',
    back: 'Back',
    loading: 'Reading the archive…',
    notFound: 'No archived match under that identifier',
    notFoundHint:
      'The archive knows this match under neither its short nor its full form. Check the identifier, or capture the film with `study-archiver fetch-one`.',
    noArtifact: 'This match is recorded, but its replay artifact is missing',
    noArtifactHint:
      'Nothing to draw: the match was recorded without its film being assembled, or the artifact has since left the cache. `study-archiver rebuild <match>` rebuilds it offline.',
    busy: 'A capture is holding the archive',
    busyHint:
      'One writer at a time on the database, and the hourly capture has it. Nothing is broken — try again in a few seconds.',
    failed: 'The study server did not answer',
    failedHint:
      'Is it running? `go run ./apps/go-api/cmd/study-server` listens on 127.0.0.1:8100, and the dev server proxies to it.',
    retry: 'Try again',
    unsupported: (found, supported) =>
      `Artifact in schema version ${found}; this viewer reads version ${supported}.`,
    unsupportedHint:
      'Nothing is drawn, deliberately: reading a document of another version with these rules would produce a plausible map that is wrong. Rebuilding the artifact (`study-archiver rebuild <match>`) writes it out in the current version.',
  },
}
