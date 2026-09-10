/**
 * i18n des modals média (MediaMatchPicker, CoverFlowModal).
 * Séparé du i18n.ts principal pour respecter SRP.
 */

import type { Locale } from '@/lib/i18n/locale'

export interface MatchPickerText {
  title: string
  /** Variante quand le média n'a PAS encore de match associé (hasCurrentMatch=false). */
  titleAssociate: string
  capturePrefix: string
  closeAriaLabel: string
  windowLabel: string
  matchesFound: (count: number) => string
  loading: string
  error: string
  noMatchesFound: string
  lobbyUnavailable: string
  currentBadge: string
  confirmTitle: string
  /** Variante « Confirmer l'association ? » (hasCurrentMatch=false). */
  confirmTitleAssociate: string
  cancel: string
  confirm: string
  applying: string
  deltaUnder1Min: string
  deltaMinFormat: (m: number) => string
  spectators: string
  /** Libellé d'équipe — reçoit le nom officiel Halo (Eagle…) ou le numéro. */
  teamLabel: (n: string | number) => string
}

export interface CoverFlowText {
  reassociateButton: string
  reassociateTitle: string
  associateButton: string
  associateTitle: string
  viewMatchButton: string
  viewMatchTitle: string
  chainButton: string
  enableChaining: string
  disableChaining: string
  closeAriaLabel: string
  audioGame: string
  audioVoice: string
  audioGroupLabel: string
  enterFullscreen: string
  exitFullscreen: string
  /** Suppression définitive d'un média (item 3.1) — propriétaire ou admin. */
  deleteButton: string
  deleteTitle: string
  deleteConfirmTitle: string
  /** Corps de la confirmation : dit explicitement que c'est irréversible. */
  deleteConfirmBody: string
  deleteConfirmLabel: string
  deleteCancelLabel: string
  deleteSuccess: string
  deleteError: string
}

export interface MediaModalsText {
  matchPicker: MatchPickerText
  coverFlow: CoverFlowText
}

const EN: MediaModalsText = {
  matchPicker: {
    title: 'Reassociate this media',
    titleAssociate: 'Associate this media',
    capturePrefix: 'Capture:',
    closeAriaLabel: 'Close',
    windowLabel: 'Window:',
    matchesFound: (count) => `${count} match${count > 1 ? 'es' : ''} found`,
    loading: 'Loading…',
    error: 'Loading error',
    noMatchesFound: 'No match found in this window. Broaden the search.',
    lobbyUnavailable: 'Lobby unavailable',
    currentBadge: 'current',
    confirmTitle: 'Confirm reassociation?',
    confirmTitleAssociate: 'Confirm association?',
    cancel: 'Cancel',
    confirm: 'Confirm',
    applying: 'Applying…',
    deltaUnder1Min: '< 1 min',
    deltaMinFormat: (m) => `±${m} min`,
    spectators: 'Spectators',
    teamLabel: (n) => `Team ${n}`,
  },
  coverFlow: {
    reassociateButton: 'Reassociate',
    reassociateTitle: 'Reassociate this media to another match',
    associateButton: 'Associate',
    associateTitle: 'Associate this media to a match',
    viewMatchButton: 'View match →',
    viewMatchTitle: 'Open the associated match page',
    chainButton: 'Chain',
    enableChaining: 'Enable chaining',
    disableChaining: 'Disable chaining',
    closeAriaLabel: 'Close',
    audioGame: 'Game',
    audioVoice: 'Voice',
    audioGroupLabel: 'Audio tracks',
    enterFullscreen: 'Fullscreen',
    exitFullscreen: 'Exit fullscreen',
    deleteButton: 'Delete',
    deleteTitle: 'Permanently delete this media',
    deleteConfirmTitle: 'Delete this media?',
    deleteConfirmBody:
      'The file will be erased from the server and cannot be recovered. '
      + 'The media will disappear from the gallery and from match pages.',
    deleteConfirmLabel: 'Delete permanently',
    deleteCancelLabel: 'Cancel',
    deleteSuccess: 'Media deleted.',
    deleteError: 'Could not delete this media.',
  },
}

const TEXT: Record<Locale, MediaModalsText> = { en: EN }

export function getMediaModalsText(_locale?: string | null): MediaModalsText {
  return TEXT['en']
}
