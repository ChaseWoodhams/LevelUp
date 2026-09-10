/**
 * i18n du réglage des pistes audio des médias (bouton engrenage + modale).
 * Hand-written FR/EN (Record<Locale, T>) comme i18n-modals.ts — pas de manifest TOML.
 */

import type { Locale } from '@/lib/i18n/locale'

export interface MediaAudioConfigText {
  gearAriaLabel: string
  title: string
  intro: string
  modeAuto: string
  modeAutoHint: string
  modeManual: string
  modeManualHint: string
  tracksLabel: string
  tracksHint: string
  trackLabel: (n: number) => string
  roleGame: string
  roleVoice: string
  roleOther: string
  addTrack: string
  removeTrack: string
  emptyManual: string
  cancel: string
  save: string
  saving: string
  saveError: string
  closeAriaLabel: string
}

const EN: MediaAudioConfigText = {
  gearAriaLabel: 'Audio tracks settings',
  title: 'Media audio tracks',
  intro:
    'Declare the role of your audio tracks for upcoming transcodes. Your recording track order is stable, so this setting applies to all your future media.',
  modeAuto: 'Automatic',
  modeAutoHint: 'Analysis detects game and voices on its own.',
  modeManual: 'Manual',
  modeManualHint: 'The role of each track is declared manually, in order.',
  tracksLabel: 'Tracks',
  tracksHint: 'Track 1 = first audio track of the file.',
  trackLabel: (n) => `Track ${n}`,
  roleGame: 'Game',
  roleVoice: 'Voice',
  roleOther: 'Other tracks',
  addTrack: 'Add a track',
  removeTrack: 'Remove',
  emptyManual: 'Add at least one track.',
  cancel: 'Cancel',
  save: 'Save',
  saving: 'Saving…',
  saveError: 'Save failed',
  closeAriaLabel: 'Close',
}

const TEXT: Record<Locale, MediaAudioConfigText> = { en: EN }

export function getMediaAudioConfigText(): MediaAudioConfigText {
  return TEXT['en']
}
