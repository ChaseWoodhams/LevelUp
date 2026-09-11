/**
 * i18n du module Prestige (dictionnaire local FR/EN, même pattern que
 * features/ascension/i18n.ts). Les composants Prestige étaient historiquement
 * français-only (strings hardcodées) ; ce module centralise les libellés pour
 * les rendre i18n-compatibles et lever les warnings `no-hardcoded-strings`.
 *
 * Règle : la version FR doit être en français PROPRE (pas de franglais). Les
 * emprunts anglais (baseline, deadline, preset, headshot…) sont traduits.
 */
import type { ManifestLocale } from '@/lib/i18n/format'

export interface PrestigeText {
  // ── Objectif (ObjectiveRow) ──
  objectiveSquadBadge: string
  objectivePPTooltip: string
  objectiveBadgeAlt: string
  // ── Arc (ArcSummary) ──
  arcCompleted: string
  arcPreset: string
  arcTotalPPTooltip: string
  /** Petit tag accolé au bonus de complétion d'arc (ex: "+150 PP bonus"). */
  arcBonusBadge: string
  arcBonusPPTooltip: string
  // ── Formulaire de création (CreateChallengeForm) ──
  formNewChallenge: string
  formCancel: string
  formModeHybrid: string
  formModeAuto: string
  formFieldMetric: string
  formFieldTarget: string
  formFieldWindow: string
  windowSessions: string
  windowRollingDays: string
  windowDeadline: string
  formFieldDate: string
  formFieldValue: string
  formFieldCadence: string
  cadenceFree: string
  cadenceDaily: string
  cadenceWeekly: string
  cadenceMonthly: string
  formFieldLabel: string
  formLabelPlaceholder: string
  formLoadingSuggestions: string
  formNoTemplates: string
  formHybridHint: string
  formFieldAdjustedTarget: string
  formGenerating: string
  formAutoHint: string
  /** Interpolé : `{target}` remplacé par le palier Héroïque. */
  formAcceptHeroic: string
  formCreating: string
  formCreate: string
  // ── Cooldown anti-farming (métrique en repos) ──
  /** Badge sur un modèle indisponible. Interpolé : `{time}` (ex: "3 h", "2 j"). */
  cooldownBadge: string
  cooldownUnitHour: string
  cooldownUnitDay: string
  /** Message affiché si la création est refusée (429 cooldown_active). */
  cooldownErrorMessage: string
  // ── Suppression d'arc (MyArcsSection) ──
  arcDeleteButton: string
  /** Titre du dialogue. Interpolé : `{title}`. */
  arcDeleteTitle: string
  /** Option cascade. Interpolé : `{n}` (nombre d'objectifs). */
  arcDeleteWithObjectives: string
  arcDeleteKeepObjectives: string
  arcDeleteCancel: string
  // ── Formulaire de création d'arc (CreateArcForm) ──
  arcFormNew: string
  arcFormTitle: string
  arcFormTitlePlaceholder: string
  arcFormDescription: string
  arcFormDescriptionPlaceholder: string
  arcFormCreate: string
  // ── MomentCard ──
  momentAchieved: string
  momentBaseline: string
  momentDelta: string
  momentMatches: string
}

const EN: PrestigeText = {
  objectiveSquadBadge: 'Squad',
  objectivePPTooltip: 'Prestige Points earned on completion',
  objectiveBadgeAlt: 'Objective',
  arcCompleted: 'Completed',
  arcPreset: 'Preset',
  arcTotalPPTooltip: "Total Prestige Points from the arc's objectives",
  arcBonusBadge: 'bonus',
  arcBonusPPTooltip:
    "Arc completion bonus, credited once all steps are done — on top of each objective's PP.",
  formNewChallenge: 'New challenge',
  formCancel: 'Cancel',
  formModeHybrid: 'Hybrid',
  formModeAuto: 'Automatic',
  formFieldMetric: 'Metric',
  formFieldTarget: 'Target',
  formFieldWindow: 'Window',
  windowSessions: 'Sessions',
  windowRollingDays: 'Rolling days',
  windowDeadline: 'Deadline',
  formFieldDate: 'Date (YYYY-MM-DD)',
  formFieldValue: 'Value',
  formFieldCadence: 'Cadence',
  cadenceFree: 'Free',
  cadenceDaily: 'Daily',
  cadenceWeekly: 'Weekly',
  cadenceMonthly: 'Monthly',
  formFieldLabel: 'Label (optional)',
  formLabelPlaceholder: 'e.g. Slayer Lv.2',
  formLoadingSuggestions: 'Loading suggestions…',
  formNoTemplates: 'No template available. Switch to free mode to create a custom challenge.',
  formHybridHint: 'Pick a template; adjust the target if you want.',
  formFieldAdjustedTarget: 'Adjusted target',
  formGenerating: 'Generating proposals…',
  formAutoHint: '3 challenges drawn from the catalog, accept directly.',
  formAcceptHeroic: 'Accept (Heroic: {target})',
  formCreating: 'Creating…',
  formCreate: 'Create challenge',
  cooldownBadge: 'Available in {time}',
  cooldownUnitHour: 'h',
  cooldownUnitDay: 'd',
  cooldownErrorMessage: 'Metric on cooldown — try again later.',
  arcDeleteButton: 'Delete',
  arcDeleteTitle: 'Delete the arc “{title}”?',
  arcDeleteWithObjectives: 'Also delete the {n} objectives',
  arcDeleteKeepObjectives: 'Keep the objectives (detach them)',
  arcDeleteCancel: 'Cancel',
  arcFormNew: 'New arc',
  arcFormTitle: 'Title',
  arcFormTitlePlaceholder: 'e.g. Spartan Ascension',
  arcFormDescription: 'Description (optional)',
  arcFormDescriptionPlaceholder: 'e.g. Chain objectives to climb the tiers.',
  arcFormCreate: 'Create arc',
  momentAchieved: 'Achieved',
  momentBaseline: 'Baseline',
  momentDelta: 'Delta',
  momentMatches: 'matches',
}

export function getPrestigeText(locale: ManifestLocale): PrestigeText {
  return locale === 'en' ? EN : EN
}
