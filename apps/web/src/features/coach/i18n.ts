/**
 * Strings i18n du module coach (ADR 0020 Phase 10).
 * Convention identique aux autres features : FR + EN inline.
 */

export interface CoachStrings {
  proposalsTitle: string
  proposalsEmpty: string
  proposalsOptInHint: string
  proposalsLoadError: string

  // Actions
  accept: string
  dismiss: string
  accepting: string
  dismissing: string

  // Détails carte
  origin: string
  originCatalog: string
  originSynthesized: string
  strength: string
  signal: string
  kindChallenge: string
  kindArc: string
  suggestedTier: string

  // Toast / feedback
  acceptedSuccess: string
  dismissedSuccess: string
  acceptError: string
  dismissError: string
}

/* The product has one runtime language. */

export const coachStringsEN: CoachStrings = {
  proposalsTitle: 'Coach suggestions',
  proposalsEmpty: 'No suggestions right now. Come back after a few matches.',
  proposalsOptInHint:
    "Turn on 'Proactive coach' in settings to receive suggestions for objectives and Prestige arcs calibrated on recent trends.",
  proposalsLoadError: 'Failed to load suggestions.',

  accept: 'Accept',
  dismiss: 'Dismiss',
  accepting: 'Creating...',
  dismissing: 'Removing...',

  origin: 'Origin',
  originCatalog: 'Catalog',
  originSynthesized: 'Synthesized',
  strength: 'Signal strength',
  signal: 'Signal',
  kindChallenge: 'Challenge',
  kindArc: 'Arc',
  suggestedTier: 'Suggested tier',

  acceptedSuccess: 'Suggestion accepted — challenge created in Prestige.',
  dismissedSuccess: 'Suggestion dismissed.',
  acceptError: 'Error while accepting.',
  dismissError: 'Error while dismissing.',
}

export function getCoachStrings(): CoachStrings {
  return coachStringsEN
}
