/**
 * i18n strings — feature match-view (header refonte 2026-05-05, mock C).
 *
 * Strings UI dédiées au header (nav + actions + labels). Les libellés de
 * stats Halo (modes, maps, playlists) viennent du pipeline backend FR
 * (asset_translations / mode_name_tr) — voir buildMatchHeader Go.
 */

import type { Locale } from '@/lib/i18n/locale'

/** Alias de compat du type central `Locale` (lib/i18n/locale) : conservé car consommé dans plus de 5 fichiers du header match-view. */
export type MatchViewLocale = Locale

export interface MatchViewText {
  prevMatch: string
  nextMatch: string
  matchCounter: (n: number, total: number) => string
  copyMatchId: string
  copied: string
  copyShort: string
  copyTooltip: string
  replayShort: string
  replayTooltip: string
  markIrrelevant: string
  reactivate: string
  excludeShort: string
  excludeTooltip: string
  reactivateTooltip: string
  // Confirmation dialog d'exclusion / réactivation
  excludeConfirmTitle: string
  excludeConfirmBody: string
  reactivateConfirmTitle: string
  reactivateConfirmBody: string
  confirmAction: string
  cancelAction: string
  excludeRankedDenied: string
  excludeErrorRanked: string
  excludeErrorGeneric: string
  performance: string
  rank: string
  /** Libellé localisé de la sentinelle de tier « Placement » (phase de placement). */
  rankPlacement: string
  addFavorite: string
  removeFavorite: string
  mapUnknown: string
  pageErrorTitle: string
  pageRetry: string
  pagePartialLoad: string
  // État dédié 404 match_not_found — match absent du substrat local (pas encore
  // synchronisé, ou identifiant invalide). Remplace l'écran d'erreur générique
  // (retiré le 2026-07-25 avec le fallback LIVE du Match view, cf. BACKLOG).
  notSyncedTitle: string
  notSyncedDescription: string
  noRank: string
  exitContext: string
  outcomeWin: string
  outcomeLoss: string
  outcomeDraw: string
  outcomeDnf: string
  fromDate: string
  toDate: string
  // Charts résumé
  chartKdaTitle: string
  chartSpreeTitle: string
  seriesActual: string
  seriesExpected: string
  seriesHistAvg: string
  labelKills: string
  labelDeaths: string
  labelAssists: string
  labelSpree: string
  labelHeadshots: string
  labelPerfectKills: string
  noHistData: string
  duration: string
  // Radar synergie (joueur actif)
  chartSynergyRadarTitle: string
  radarAxisCombat: string
  radarAxisSurvival: string
  radarAxisSupport: string
  radarAxisScore: string
  radarAxisObjective: string
  radarAxisImpact: string
  radarTooltipImpact: string
  radarTooltipCombat: string
  radarTooltipSurvival: string
  radarTooltipSupport: string
  radarTooltipScore: string
  radarTooltipObjective: string
  radarTooltipGlossaryLink: string
  // Labels de mécaniques natives Halo 5 — colonnes du scoreboard (MatchScoreboard).
  labelGrenade: string
  labelAssassination: string
  labelGroundPound: string
  labelShoulderBash: string
  weaponUnknownPrefix: string
  // Section médias (dans onglet Résumé)
  sectionMedia: string
  mediaNoCaptures: string
  mediaNoCapturesDesc: string
  // Résumé — médailles & citations
  sectionMedals: string
  sectionCitations: string
  newlyMastered: string
  noMedals: string
  noCitations: string
  // Commendations NATIVES (Halo 5) — affichées à la place des citations dérivées
  sectionNativeCommendations: string
  noNativeCommendations: string
  // Onglet Combat — charts en haut (mock match_view.09 / .10 / .11 / .12)
  combatHighlights: string
  combatKdCumulTitle: string
  combatTugOfWarTitle: string
  combatCadenceTitle: string
  combatKillsLabel: string
  combatDeathsLabel: string
  combatTeamLabel: string
  combatEnemyLabel: string
  // Histogramme momentum (carte Dominance) — libellés de tooltip.
  combatMomentumDelta: string
  combatMomentumCumul: string
  combatNemesisTitle: string
  combatBullyTitle: string
  combatNoNemesis: string
  combatKilledMeFmt: (n: number) => string
  combatIKilledFmt: (n: number) => string
  combatNoData: string
  // Overlay capture CTF (charts combat — câblé couche 2)
  combatCtfCaptureLabel: string
  combatCtfCaptureTooltip: (player: string, time: string) => string
  fragDiffNoData: string
  antagonistNoData: string
  impactBadgesNoData: string
  // Libellés des badges d'impact (Match flow), keyés par BadgeKey backend. Le
  // moteur analysis ne produit qu'un libellé FR (BadgeFR) → sous UI EN les cartes
  // restaient en FR (GH-7). Résolution front bilingue par clé, fallback = libellé
  // serveur pour une clé inconnue.
  impactBadgeNames: Record<string, string>
  // Breadcrumb retour (MatchBreadcrumb)
  back: string
  // Onglets de la page (GH2-B2)
  tabGeneral: string
  tabDetails: string
  // Titre du chart Antagonistes (GH2-B2)
  antagonistTitle: string
  // Sections de l'onglet Détails (titres type-1 du catalogue d'harmonisation)
  sectionFlow: string
  sectionDuels: string
  sectionEncounters: string
  // Scoreboard team header (Eagle / Cobra avec couleur team-ally/enemy)
  scoreboardTitle: string
  scoreboardNoData: string
  teamLabelFmt: (name: string) => string
  teamUnknown: string
  teamNumberedFmt: (n: number) => string
  teamMine: string
  teamEnemy: string
  // Scoreboard expander (port de match_view_scoreboard_detail.py)
  sbDetailWeapons: string
  sbDetailMedalsAndCitations: string
  sbDetailMedalsOnly: string
  sbDetailExpected: string
  sbDetailExpectedKills: string
  sbDetailExpectedDeaths: string
  sbDetailExpectedAssists: string
  sbDetailLocallyEstimated: string
  sbDetailLocallyEstimatedHint: string
  sbDetailAntagonist: string
  sbDetailNemesis: string
  sbDetailBully: string
  sbDetailLocal: string
  sbDetailLusr: string
  sbDetailCsr: string
  sbDetailBotNoteLabel: string
  sbDetailBotNoteValue: string
  sbDetailPlayerDb: string
  sbDetailSharedOnly: string
  sbDetailExplorePlayerFmt: (player: string) => string
  // Libellés colonnes scoreboard (utilisés par buildHighlightCols)
  sbColKda: string
  sbColMeleeKills: string
  sbColDamageDealt: string
  sbColDamageTaken: string
  sbColShotsHit: string
  sbColAccuracy: string
  sbColCsr: string
  sbColRank: string
  sbColScore: string
  sbColAssists: string
  sbColMaxSpree: string
  sbColHeadshots: string
  sbColPerfectKills: string
  sbColShotsFired: string
  sbColPowerWeapons: string
  sbColAvgLife: string
  sbColPlayer: string
  sbColTopWeapon: string
  // Tooltips d'en-tête de colonne (V72-04, icône ⓘ) — colonnes non évidentes.
  sbColCsrTooltip: string
  sbColLusrTooltip: string
  sbColRankTooltip: string
  sbColKdaTooltip: string
  sbColAccuracyTooltip: string
  sbColMaxSpreeTooltip: string
  sbColPerfectKillsTooltip: string
  sbColPowerWeaponsTooltip: string
  sbColMeleeKillsTooltip: string
  sbColAvgLifeTooltip: string
  sbColTopWeaponTooltip: string
  sbColOffensiveTooltip: string
  sbColDefensiveTooltip: string
  sbViewHistoryFmt: (gamertag: string) => string
  /** Format du score (séparateurs locale-sensitive : "12 345" FR / "12,345" EN). */
  sbFormatScore: (v: number) => string
  // Nav contextuelle — Phase 2c (descriptor → label compact)
  ctxRecent: string
  ctxFavorites: string
  ctxMedia: string
  ctxTopMatches: string
  ctxWithPlayerFmt: (gamertag: string) => string
  ctxSessionFmt: (date: string) => string
  ctxPeriodFromToFmt: (from: string, to: string) => string
  ctxPeriodFromFmt: (from: string) => string
  ctxPeriodToFmt: (to: string) => string
  ctxPlaylistFmt: (name: string) => string
  ctxModeFmt: (category: string) => string
  /** Compteur intégré : "Matchs récents 12/47" / "Recent matches 12/47". */
  matchCounterCtxFmt: (label: string, n: number, total: number) => string
  /** Section « Objectifs » du scoreboard (CTF/Zones/Oddball) — V72-03. `cols` :
   *  libellé + tooltip d'en-tête par clé de colonne objectif. */
  objectives: {
    title: string
    teamTotal: string
    cols: Record<string, { label: string; tooltip: string }>
  }
}

export const MATCH_VIEW_TEXT: Record<MatchViewLocale, MatchViewText> = {
  en: {
    prevMatch: 'Previous match',
    nextMatch: 'Next match',
    matchCounter: (n, total) => `Match ${n}/${total}`,
    copyMatchId: 'Copy match ID',
    copied: 'Copied',
    copyShort: 'Copy ID',
    copyTooltip: "Copy this match's unique identifier to clipboard",
    replayShort: '2D replay',
    replayTooltip: 'Watch the 2D replay of this match (top-down view)',
    markIrrelevant: 'Mark as irrelevant',
    reactivate: 'Reactivate',
    excludeShort: 'Exclude',
    excludeTooltip: 'Exclude this match from stats and analyses',
    reactivateTooltip: 'Re-include this match in stats and analyses',
    excludeConfirmTitle: 'Exclude this match?',
    excludeConfirmBody:
      'This match will be marked irrelevant and removed from stats. The performance score and LUSR of subsequent matches will be recomputed (a few seconds).',
    reactivateConfirmTitle: 'Reactivate this match?',
    reactivateConfirmBody:
      'This match will be re-included in stats. The performance score and LUSR of subsequent matches will be recomputed (a few seconds).',
    confirmAction: 'Confirm',
    cancelAction: 'Cancel',
    excludeRankedDenied: 'Ranked matches cannot be excluded (official CSR)',
    excludeErrorRanked: 'Ranked matches cannot be excluded.',
    excludeErrorGeneric: 'Could not update exclusion. Try again later.',
    performance: 'Performance',
    rank: 'Rank',
    rankPlacement: 'In placement',
    addFavorite: 'Add to favorites',
    removeFavorite: 'Remove from favorites',
    mapUnknown: 'Unknown map',
    pageErrorTitle: 'Match not found or load error.',
    pageRetry: 'Retry',
    pagePartialLoad: 'This match could not be fully loaded.',
    notSyncedTitle: 'Match not synced yet',
    notSyncedDescription:
      "This match isn't in the local database yet. If it was just played, it will show up here after the next sync — check back in a few minutes. Also double-check that the match link is correct.",
    noRank: 'No rank',
    exitContext: 'Exit context',
    outcomeWin: 'Wins',
    outcomeLoss: 'Losses',
    outcomeDraw: 'Draws',
    outcomeDnf: 'DNF',
    fromDate: 'From',
    toDate: 'To',
    chartKdaTitle: 'K/D/A: Actual vs Expected vs Hist. Avg.',
    chartSpreeTitle: 'Spree · Headshots · Perfect kills',
    seriesActual: 'Actual',
    seriesExpected: 'Expected',
    seriesHistAvg: 'Hist. Avg.',
    labelKills: 'K',
    labelDeaths: 'D',
    labelAssists: 'A',
    labelSpree: 'Killing Spree',
    labelHeadshots: 'Headshots',
    labelPerfectKills: 'Perfect kills',
    noHistData: 'No historical data available',
    duration: 'Duration',
    chartSynergyRadarTitle: 'Synergy radar',
    radarAxisCombat: 'Combat',
    radarAxisSurvival: 'Survival',
    radarAxisSupport: 'Support',
    radarAxisScore: 'Score',
    radarAxisObjective: 'Objective',
    radarAxisImpact: 'Impact',
    radarTooltipImpact: 'Offensive conversion — 225 × (kills + ass/3) / damage. P80 = 0.83.',
    radarTooltipCombat: 'Kills + headshots + perfect kills, weighted by accuracy.',
    radarTooltipSurvival: 'Defensive resistance — damage / (225 × deaths). P80 = 1.59.',
    radarTooltipSupport: 'Assists × 50.',
    radarTooltipScore: 'Residual score after kills (×100) and assists (×50): medals and streaks.',
    radarTooltipObjective: 'Objective participation in this match — weighted actions (captures, grabs, returns…) plus time on the objective, calibrated per mode (mode P80 = 80). Hidden on non-objective matches.',
    radarTooltipGlossaryLink: '→ Glossary',
    labelGrenade: 'Grenade',
    labelAssassination: 'Assassination',
    labelGroundPound: 'Ground Pound',
    labelShoulderBash: 'Shoulder Bash',
    weaponUnknownPrefix: 'Unknown weapon',
    sectionMedia: 'Media',
    mediaNoCaptures: 'No captures',
    mediaNoCapturesDesc: 'Screenshots and clips associated with this match will appear here.',
    sectionMedals: 'Medals',
    sectionCitations: 'Commendations',
    newlyMastered: 'Mastered!',
    noMedals: 'No medals',
    noCitations: 'No commendations',
    sectionNativeCommendations: 'Commendations',
    noNativeCommendations: 'No commendations',
    combatHighlights: 'Highlights',
    combatKdCumulTitle: 'Cumulative frags',
    combatTugOfWarTitle: 'Dominance',
    combatCadenceTitle: 'Kill cadence',
    combatKillsLabel: 'Kills',
    combatDeathsLabel: 'Deaths',
    combatTeamLabel: 'My team',
    combatEnemyLabel: 'Opponents',
    combatMomentumDelta: 'Delta',
    combatMomentumCumul: 'Cumulative',
    combatNemesisTitle: 'Nemesis',
    combatBullyTitle: 'Bully target',
    combatNoNemesis: '—',
    combatKilledMeFmt: (n) => `Martyred you ${n} times`,
    combatIKilledFmt: (n) => `You victimized them ${n} times`,
    combatNoData: 'No data available',
    combatCtfCaptureLabel: 'Capture',
    combatCtfCaptureTooltip: (player, time) => `${player} — captured at ${time}`,
    fragDiffNoData: 'No combat events recorded for this match.',
    antagonistNoData: 'No duel data available for this match.',
    impactBadgesNoData: 'No impact badges for this match.',
    impactBadgeNames: {
      first_blood: 'First blood',
      first_group_death: 'First down',
      clutch_finisher: 'Finisher',
      last_casualty: 'Last casualty',
      last_group_kill: 'Latecomer',
      top_killer: 'Top killer',
      silent_hero: 'Silent hero',
      false_brother: 'False brother',
      top_gun: 'Top Gun',
      kamikaze: 'Kamikaze',
    },
    back: 'Back',
    tabGeneral: 'General',
    tabDetails: 'Details',
    antagonistTitle: 'Antagonists',
    sectionFlow: 'Match flow',
    sectionDuels: 'Duels & head-to-head',
    sectionEncounters: 'Encounter history',
    scoreboardTitle: 'Scoreboard',
    scoreboardNoData: 'No scoreboard data available for this match.',
    teamLabelFmt: (name) => `Team ${name}`,
    teamUnknown: 'Unknown team',
    teamNumberedFmt: (n) => `Team ${n}`,
    teamMine: 'My team',
    teamEnemy: 'Enemy team',
    sbDetailWeapons: 'Weapons',
    sbDetailMedalsAndCitations: 'Medals & commendations',
    sbDetailMedalsOnly: 'Medals',
    sbDetailExpected: 'Expected vs actual',
    sbDetailLocallyEstimated: 'Locally estimated',
    sbDetailLocallyEstimatedHint: 'No skill API for this title: expected kills and deaths from a local model (volume scales with match length), assists from a local model.',
    sbDetailExpectedKills: 'Kills',
    sbDetailExpectedDeaths: 'Deaths',
    sbDetailExpectedAssists: 'Assists',
    sbDetailAntagonist: 'Antagonist',
    sbDetailNemesis: 'Nemesis',
    sbDetailBully: 'Bully target',
    sbDetailLocal: 'Local data',
    sbDetailLusr: 'LUSR',
    sbDetailCsr: 'CSR',
    sbDetailBotNoteLabel: 'Bot teammate',
    sbDetailBotNoteValue: 'At least one bot on the team — stats to be taken with a grain of salt.',
    sbDetailPlayerDb: 'Tracked player (local DB)',
    sbDetailSharedOnly: 'Untracked player (shared DB only)',
    sbDetailExplorePlayerFmt: (player) => `Explore ${player}`,
    sbColKda: 'KDA',
    sbColMeleeKills: 'Melee',
    sbColDamageDealt: 'Damage dealt',
    sbColDamageTaken: 'Damage taken',
    sbColShotsHit: 'Shots hit',
    sbColAccuracy: 'Accuracy',
    sbColCsr: 'CSR',
    sbColRank: 'Rank',
    sbColScore: 'Score',
    sbColAssists: 'Assists',
    sbColMaxSpree: 'Killing spree',
    sbColHeadshots: 'Headshots',
    sbColPerfectKills: 'Perfect kills',
    sbColShotsFired: 'Shots',
    sbColPowerWeapons: 'Power weapons',
    sbColAvgLife: 'Avg. life',
    sbColPlayer: 'Player',
    sbColTopWeapon: 'Top weapon',
    sbColCsrTooltip: 'In-game competitive rank (CSR) reached this match.',
    sbColLusrTooltip: 'In-house rating (LUSR) estimated for this match.',
    sbColRankTooltip: 'Player\'s placement in the match, by score.',
    sbColKdaTooltip: 'KDA = (Kills + Assists/3) − Deaths; rewards impact, not kills/deaths.',
    sbColAccuracyTooltip: 'Accuracy: share of shots that hit the target, as a percentage.',
    sbColMaxSpreeTooltip: 'Longest run of kills without dying.',
    sbColPerfectKillsTooltip: 'Perfect kills: shields broken then a headshot with no missed shot.',
    sbColPowerWeaponsTooltip: 'Kills with power weapons picked up on the map.',
    sbColMeleeKillsTooltip: 'Kills scored in melee.',
    sbColAvgLifeTooltip: 'Average time alive between deaths.',
    sbColTopWeaponTooltip: 'Weapon with the most kills this match.',
    sbColOffensiveTooltip: 'Offensive yield: kills and assists per damage dealt.',
    sbColDefensiveTooltip: 'Resistance: damage absorbed before each death.',
    sbViewHistoryFmt: (gamertag) => `View history with ${gamertag}`,
    sbFormatScore: (v) => new Intl.NumberFormat('en-US').format(v),
    ctxRecent: 'recent',
    ctxFavorites: 'favorites',
    ctxMedia: 'with media',
    ctxTopMatches: 'top performances',
    ctxWithPlayerFmt: (gamertag) => `with ${gamertag}`,
    ctxSessionFmt: (date) => `from session of ${date}`,
    ctxPeriodFromToFmt: (from, to) => `from period ${from} to ${to}`,
    ctxPeriodFromFmt: (from) => `since ${from}`,
    ctxPeriodToFmt: (to) => `until ${to}`,
    ctxPlaylistFmt: (name) => `in ${name}`,
    ctxModeFmt: (category) => `in ${category}`,
    matchCounterCtxFmt: (label, n, total) => `${capitalize(label)} matches ${n}/${total}`,
    objectives: {
      title: 'Objectives',
      teamTotal: 'Team total',
      cols: {
        flag_captures: { label: 'Captures', tooltip: 'Flag captures' },
        flag_returns: { label: 'Returns', tooltip: 'Flag returns' },
        flag_steals: { label: 'Steals', tooltip: 'Flag steals' },
        time_as_flag_carrier_seconds: { label: 'Carrier time', tooltip: 'Time as flag carrier' },
        zone_captures: { label: 'Captures', tooltip: 'Zones captured' },
        zone_secures: { label: 'Secured', tooltip: 'Zones secured' },
        time_in_zones_seconds: { label: 'Zone time', tooltip: 'Time spent in zones' },
        skull_grabs: { label: 'Grabs', tooltip: 'Skull grabs' },
        time_as_skull_carrier_seconds: { label: 'Carrier time', tooltip: 'Time as skull carrier' },
        longest_time_as_skull_carrier_seconds: {
          label: 'Longest',
          tooltip: 'Longest skull possession',
        },
        power_seeds_deposited: { label: 'Deposited', tooltip: 'Power seeds deposited at the base' },
        power_seeds_stolen: {
          label: 'Stolen',
          tooltip: 'Power seeds taken from the enemy base',
        },
        power_seed_carriers_killed: {
          label: 'Carriers killed',
          tooltip: 'Enemy power seed carriers killed',
        },
        time_as_power_seed_carrier_seconds: {
          label: 'Carrier time',
          tooltip: 'Time as power seed carrier',
        },
        successful_extractions: { label: 'Extractions', tooltip: 'Successful extractions' },
        extraction_initiations_completed: {
          label: 'Initiations',
          tooltip: 'Extraction initiations completed',
        },
        extraction_conversions_completed: {
          label: 'Conversions',
          tooltip: 'Enemy beacons converted',
        },
        extraction_conversions_denied: {
          label: 'Conversions denied',
          tooltip: 'Enemy conversions denied',
        },
        vip_kills: { label: 'VIPs killed', tooltip: 'Enemy VIPs killed' },
        times_selected_as_vip: { label: 'Times VIP', tooltip: 'Times selected as VIP' },
        kills_as_vip: { label: 'Kills as VIP', tooltip: 'Kills while being the VIP' },
        time_as_vip_seconds: { label: 'VIP time', tooltip: 'Time spent as VIP' },
        longest_time_as_vip_seconds: {
          label: 'Longest',
          tooltip: 'Longest survival as VIP',
        },
      },
    },
  },
}

/** Capitalise la première lettre — utilisé par le builder EN. */
function capitalize(s: string): string {
  return s.length > 0 ? s.charAt(0).toUpperCase() + s.slice(1) : s
}

/**
 * buildContextLabel — produit un label localisé depuis un MatchFilterSpec.
 *
 * Phase 2b : utilisé quand la cascade tombe sur l'API avec spec URL (pas de
 * filtersLabel pré-localisé dans le matchNavContext). Format compact :
 *   "Classée Arena · Victoires · Depuis 01/04/2026"
 */
import type { MatchFilterSpec } from '@/lib/match-nav/navContext'

export function buildContextLabel(
  spec: MatchFilterSpec | null | undefined,
  locale: MatchViewLocale,
): string {
  if (!spec) return ''
  const t = MATCH_VIEW_TEXT[locale]
  const parts: string[] = []
  if (spec.playlist_names?.length) parts.push(spec.playlist_names.join(', '))
  if (spec.mode_categories?.length) parts.push(spec.mode_categories.join(', '))
  if (spec.outcome) {
    const map: Record<string, string> = {
      win: t.outcomeWin,
      loss: t.outcomeLoss,
      draw: t.outcomeDraw,
      dnf: t.outcomeDnf,
    }
    const lbl = map[spec.outcome]
    if (lbl) parts.push(lbl)
  }
  if (spec.date_from || spec.date_to) {
    const intlLocale = 'en-US'
    const fmt = (iso: string) => {
      const d = new Date(iso)
      if (isNaN(d.getTime())) return iso
      return new Intl.DateTimeFormat(intlLocale, { day: '2-digit', month: '2-digit', year: 'numeric' }).format(d)
    }
    if (spec.date_from && spec.date_to) {
      parts.push(`${fmt(spec.date_from)} → ${fmt(spec.date_to)}`)
    } else if (spec.date_from) {
      parts.push(`${t.fromDate} ${fmt(spec.date_from)}`)
    } else if (spec.date_to) {
      parts.push(`${t.toDate} ${fmt(spec.date_to)}`)
    }
  }
  if (spec.session_id) parts.push(`#${spec.session_id}`)
  return parts.join(' · ')
}

// Note Phase 2c (2026-05-07) : `buildDescriptorLabel` extrait dans
// `./descriptorLabel.ts` pour respecter la limite de 500 lignes/fichier
// (CLAUDE.md §5). `buildContextLabel` (au-dessus, fallback filterSpec)
// reste ici car il dépend uniquement de MATCH_VIEW_TEXT et n'est pas
// amené à grossir.
export { buildDescriptorLabel } from './descriptorLabel'
