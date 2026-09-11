/**
 * chart-review — manifeste de la tournée de revue visuelle.
 *
 * Une entrée = un graphe identifié par une clé stable + le type de verdict
 * attendu de l'utilisateur + une note courte qui pose la question à trancher.
 *
 * Cycle de vie d'une entrée :
 *   1. le chantier qui touche/ajoute/suspecte un graphe l'inscrit ici ;
 *   2. l'utilisateur balaye les pages et statue à l'écran ;
 *   3. l'entrée est RETIRÉE (commit de clôture de la tournée).
 *
 * MÉCANISME INERTE : `CHART_REVIEW` vide (ou clé absente) → `chartReview()`
 * renvoie `undefined` → `ReviewBadge` rend `null` → AUCUN impact visuel, aucun
 * nœud DOM ajouté. L'outillage peut donc rester en place entre deux tournées
 * sans polluer l'interface (DEC-8).
 *
 * Les clés sont des identifiants de GRAPHE (surface + graphe), pas des libellés :
 * elles ne sont jamais affichées.
 */
import type { Locale } from '@/lib/i18n/locale'

/**
 * Verdict attendu de l'utilisateur :
 *   - `verify`  : graphe corrigé/suspect → « est-ce juste et lisible ? »
 *   - `new`     : graphe (ou marqueur) ajouté → « à garder ? »
 *   - `removal` : candidat à la suppression → « on le retire ? »
 */
export type ChartReviewStatus = 'verify' | 'new' | 'removal'

export interface ChartReview {
  status: ChartReviewStatus
  /** Note courte affichée au survol. */
  note: Record<Locale, string>
}

/**
 * Entrées de la tournée en cours. Vider ce dictionnaire suffit à désarmer
 * complètement l'outillage.
 *
 * Tournée « revue analytique Timeseries & Escouade » (2026-07-25).
 */
export const CHART_REVIEW: Record<string, ChartReview> = {
  // ── B1 — durée de vie réelle (fin de l'estimation temps_joué/(morts+1)) ────
  'timeseries.avg_life_trend': {
    status: 'verify',
    note: {
      en: 'This curve now uses the real average life from the API (no longer the time played / (deaths + 1) estimate). Do the values look right?',
    },
  },
  'timeseries.life_histogram': {
    status: 'verify',
    note: {
      en: 'Histogram rebased on the real average life from the API. Older matches without the value fall back to the old estimate: is the distribution still coherent?',
    },
  },

  // ── B5 — axes séparés taux de victoire / MMR ──────────────────────────────
  'timeseries.session_performance': {
    status: 'verify',
    note: {
      en: 'Win rate now has its own percentage axis (0-100) and MMR its own: are both lines finally readable?',
    },
  },

  // ── B3(3) — radar de synergie : valeur brute au survol ────────────────────
  'squad.synergy_radar': {
    status: 'verify',
    note: {
      en: 'The tooltip now shows both the normalized and the raw value of each axis. Does the Score axis look correctly calibrated?',
    },
  },

  // ── F1 — marqueur de dominance sur la bande de résultats ──────────────────
  'timeseries.outcome_tape': {
    status: 'new',
    note: {
      en: 'Diamond = standout match moment (domination, comeback…), same colors as the Explorer Dominance column. Useful here?',
    },
  },
  'squad.outcome_tape': {
    status: 'new',
    note: {
      en: 'Diamond = standout match moment (domination, comeback…), same colors as the Explorer Dominance column. Useful here?',
    },
  },

  // ── Redondances soupçonnées (aucun code changé — arbitrage utilisateur) ───
  'timeseries.fda_distribution': {
    status: 'verify',
    note: {
      en: 'Two readings of KDA on the same tab: this distribution and the "KDA (value)" trend. Does each of them earn its place?',
    },
  },
  'timeseries.fda_value_trend': {
    status: 'verify',
    note: {
      en: 'Two readings of KDA on the same tab: this trend and the distribution. Does each of them earn its place?',
    },
  },
  'timeseries.skill_progression': {
    status: 'verify',
    note: {
      en: 'Two readings of skill rank on the same tab: this long curve and "Skill rank + Performance". Does each of them earn its place?',
    },
  },
  'timeseries.skill_rank_perf': {
    status: 'verify',
    note: {
      en: 'Two readings of skill rank on the same tab: this chart and the progression curve. Does each of them earn its place?',
    },
  },

  // ── Profils d'intensité (ont remplacé les heatmaps le 2026-07-24) ─────────
  'timeseries.intensity_profile': {
    status: 'verify',
    note: {
      en: 'Median profile + P25-P75 envelope: this replaced the matches × phases heatmap. Does it read better?',
    },
  },
  'squad.intensity_profile': {
    status: 'verify',
    note: {
      en: 'Median profile + P25-P75 envelope per player: this replaced the matches × phases heatmap. Does it read better?',
    },
  },
}

/**
 * chartReview — entrée de revue d'un graphe, ou `undefined` si le graphe n'est
 * pas dans la tournée en cours (cas nominal hors revue).
 */
export function chartReview(key: string | undefined): ChartReview | undefined {
  if (!key) return undefined
  return CHART_REVIEW[key]
}
