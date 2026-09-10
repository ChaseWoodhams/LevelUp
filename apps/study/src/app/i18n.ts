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
  archiveChanged: string
  archiveChangedHint: string
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
  /** Navigateur d'archive : filtres, colonnes, états vides. */
  browserTitle: string
  filterMap: string
  filterMode: string
  filterPlayer: string
  filterFrom: string
  filterTo: string
  filterCoverage: string
  filterAny: string
  filterClear: string
  colDate: string
  colMap: string
  colMode: string
  colPlayers: string
  colScore: string
  colCoverage: string
  /** Scores finaux, dans l'ordre Eagle / Cobra. */
  scoreHint: string
  coverageHint: string
  coverageUnknown: string
  coverageUnknownHint: string
  unknownValue: string
  /** Combien de lignes sont affichées, et sur combien. */
  shownOf: (shown: number, total: number) => string
  truncated: (loaded: number, total: number) => string
  emptyArchive: string
  emptyArchiveHint: string
  emptyFiltered: string
  emptyFilteredHint: string
  openByIdTitle: string
}

export const SHELL_TEXT: Record<Locale, ShellText> = {
  en: {
    archiveChanged: 'The archive changed while loading',
    archiveChangedHint: 'Reload the archive to display the complete match list.',
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
    browserTitle: 'Archived matches',
    filterMap: 'Map',
    filterMode: 'Mode',
    filterPlayer: 'Player',
    filterFrom: 'From',
    filterTo: 'To',
    filterCoverage: 'Minimum coverage',
    filterAny: 'Any',
    filterClear: 'Clear the filters',
    colDate: 'Date',
    colMap: 'Map',
    colMode: 'Mode',
    colPlayers: 'Players',
    colScore: 'Score',
    colCoverage: 'Coverage',
    scoreHint:
      'Final score, Eagle / Cobra. Scores missing from match stats or older archives remain unknown.',
    coverageHint:
      'The share of lives the decoder could NAME — how much of the match the artifact can attribute to a player. It is what separates a match worth studying from one whose data is sparse, which is why it is read before opening one.',
    coverageUnknown: 'unknown',
    coverageUnknownHint:
      'The artifact reported no lives at all: there was nothing to attach anything to. "Unknown" is not zero, and no floor admits it.',
    unknownValue: 'unknown',
    shownOf: (shown, total) => `${shown} of ${total} matches`,
    truncated: (loaded, total) =>
      `The archive holds ${total} matches; this table loaded the ${loaded} most recent, and the filters apply to those.`,
    emptyArchive: 'Nothing archived yet',
    emptyArchiveHint:
      'The archive is empty: capture a film with `study-archiver fetch-one --xuid <xuid> <matchId>`, or run a pass over the watchlist with `study-archiver watch`.',
    emptyFiltered: 'No match answers these filters',
    emptyFilteredHint:
      'The archive is not empty — the filtering is what lets nothing through. Widen the dates, lower the minimum coverage, or clear everything.',
    openByIdTitle: 'Open by identifier',
  },
}
