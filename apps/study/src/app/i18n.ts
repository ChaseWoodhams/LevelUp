/**
 * i18n of the study tool's own chrome — everything that is NOT the replay view.
 *
 * The replay modules copied under `features/replay/` bring their own table
 * (`REPLAY_TEXT`); this one covers the frame around them. Same rule as the rest of
 * the fork: every UI string exists in French AND English, and the parity is held by
 * the type, not by discipline.
 */
import type { Locale } from '@/lib/i18n/locale'

interface ShellText {
  /** Titre du document — suit la langue choisie, comme tout le reste de l'écran. */
  documentTitle: string
  appName: string
  fixtureBadge: string
  fixtureHint: string
  localeLabel: string
  rosterTitle: string
}

export const SHELL_TEXT: Record<Locale, ShellText> = {
  fr: {
    documentTitle: 'LevelUp — Étude',
    appName: 'Outil d’étude',
    fixtureBadge: 'Artefact de démonstration',
    fixtureHint:
      'Aucune donnée réelle : cet artefact est écrit à la main pour montrer chaque calque du rejeu.',
    localeLabel: 'Langue',
    rosterTitle: 'Fiches joueur',
  },
  en: {
    documentTitle: 'LevelUp — Study',
    appName: 'Study tool',
    fixtureBadge: 'Sample artifact',
    fixtureHint:
      'No real data: this artifact is written by hand to put every replay layer on screen.',
    localeLabel: 'Language',
    rosterTitle: 'Player cards',
  },
}
