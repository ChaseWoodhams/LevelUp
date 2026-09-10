/**
 * i18n.ts — Dictionnaire FR/EN des strings UI de la feature Escouade.
 *
 * Frontière stricte avec les mappings TOML multi-titres :
 *  - Les libellés métier (FieldKey, assets, outcomes) restent dans les TOML
 *    et passent par useFieldLabel / useAssetLabel / useOutcomeLabel.
 *  - Ce fichier ne contient que des strings UI non-titre-bound : titres de
 *    cartes, navigation, empty states, boutons, descriptions, unités.
 *
 * Pattern aligné avec features/compare/i18n.ts et features/home/*.i18n.ts.
 */

import type { Locale } from '@/lib/i18n/locale'

export interface SquadText {
  intlLocale: string
  title: {
    teammates: string
    allTeammates: string
  }
  nav: {
    synergies: string
    contributions: string
    dynamique: string
  }
  selection: {
    placeholder: (count: number) => string
    prompt: string
  }
  filter: {
    experience: string
    playlist: string
    allExperiences: string
    allPlaylists: string
    analyse: string
    /** Option « composition exacte » (désactivée par défaut) + son explication. */
    exactComposition: string
    exactCompositionTitle: string
  }
  /** Bandeau de données partielles (chargements dégradés remontés par l'API). */
  dataIssues: {
    title: string
    teammateMatches: (detail: string) => string
    heatmapTeammate: (detail: string) => string
    mainTeamParticipants: string
    mapStats: string
    unknown: (code: string) => string
  }
  session: {
    label: string
    squad: string
    prev: string
    next: string
    all: string
    reset: string
  }
  table: {
    gamertag: string
    matches: string
    wins: string
    winPct: string
    kd: string
    lastSeen: string
    actions: string
    openCompare: string
    withTeammate: (gamertag: string) => string
  }
  empty: {
    noSelectionTitle: string
    noSelectionDescription: string
    invalidSelectionTitle: string
    invalidSelectionDescription: string
    noChartTitle: string
    noChartDescription: string
    noDataTitle: string
    noDataDescription: string
    /** Message court pour un bloc non-graphe vide (tape, table, scoreboard). */
    noBlockData: string
  }
  synergies: {
    description: string
  }
  contributions: {
    description: string
  }
  charts: {
    hsPkTitle: string
    timelineTitle: string
    timelinePerfName: string
    timelineWinRateName: string
    timelinePerfAxis: string
    timelineWinRateAxis: string
    heatmapTitle: string
    heatmapWinAxis: string
    heatmapMatchesLabel: string
    winRateVsHistorySession: string
    winRateVsHistoryHistory: string
    winRateVsHistoryBulletTitle: string
    winRateVsHistoryBulletParity: string
    winRateVsHistoryBulletZero: string
    winRateVsHistoryBulletCounts: (session: number, history?: number) => string
    /** Tooltip d'aide : explique le suffixe « (n) » sur les libellés d'axe Y (I9). */
    winRateVsHistoryBulletMapCountTooltip: string
    mapPerfVsHistoryTitle: string
    mapPerfVsHistorySession: string
    mapPerfVsHistoryHistory: string
    outcomeSequenceTitle: string
  }
  history: {
    title: string
    description: string
    date: string
    map: string
    playlist: string
    mode: string
    outcome: string
    score: string
    winRateHist: string
    winProb: string
    kda: string
    accuracy: string
    perf: string
    duration: string
    teamMmr: string
    enemyMmr: string
    deltaMMR: string
    session: string
    outcomeLabel: { win: string; loss: string; draw: string; dnf: string }
    prev: string
    next: string
    pageOf: (cur: number, total: number) => string
    totalRows: (n: number) => string
    /** aria-label/tooltip du lien « Ouvrir sur Halo Waypoint » (I19). */
    waypointAriaLabel: string
    /** aria-label du bouton de tri d'un en-tête « Trier par {col} » (I16). */
    sortByAriaLabel: (col: string) => string
    /** Tooltips d'en-tête de colonne (V72-04, icône ⓘ). */
    winRateHistTooltip: string
    winProbTooltip: string
    teamMmrTooltip: string
    enemyMmrTooltip: string
    deltaMmrTooltip: string
  }
  timeline: {
    title: string
    perf: string
    winRate: string
    teamMmr: string
    perfAxis: string
    mmrAxis: string
  }
  heatmap: {
    title: string
    pieceTier1: string
    pieceTier2: string
    pieceTier3: string
    pieceTier4: string
    pieceTier5: string
    noScore: string
  }
  impact: {
    title: string
    colPlayer: string
    colScore: string
    colBadge: string
    /** Tooltips d'en-tête de colonne (V72-04, icône ⓘ). */
    colScoreTooltip: string
    colBadgeTooltip: string
    badgeChampion: string
    badgeChampionShort: string
    badgeWeakLink: string
    badgeWeakLinkShort: string
    badgeStowaway: string
    badgeStowawayShort: string
    badgeNames: Record<string, string>
    badgeDescriptions: Record<string, string>
  }
  perMinute: {
    title: string
    description: string
    frags: string
    deaths: string
    assists: string
    suffix: string
  }
  synergyRadar: {
    title: string
    description: string
    axes: { combat: string; survival: string; support: string; score: string; objective: string; impact: string }
    tooltip: { impact: string; combat: string; survival: string; support: string; score: string; objective: string; glossaryLink: string }
    /** Préfixe de la valeur BRUTE affichée au survol, à côté de la valeur normalisée. */
    rawLabel: string
  }
  intensity: {
    title: string
    /** Sous-titre de la carte Intensité (profil médian + enveloppe). */
    subtitle: string
    /** Tooltip d'aide : courbe / zone d'irrégularité / repère / courbe d'équipe. */
    tooltip: string
    /** Libellé tooltip du trait médian. */
    medianLabel: string
    /** Libellé tooltip de la fourchette interquartile. */
    envelopeLabel: string
    /** Libellé du repère 10 % (activité uniforme). */
    refLabel: string
    /** Libellé de la courbe agrégée d'équipe (superposée à partir de 3 joueurs). */
    teamLabel: string
  }
  efficiencySeries: {
    /** Titre COURT de la carte Rendement (la définition vit dans l'aide ⓘ). */
    rendementCardTitle: string
    /** Titre COURT de la carte Résistance (la définition vit dans l'aide ⓘ). */
    resistanceCardTitle: string
    /** Aide ⓘ des deux cartes : formule des indicateurs + pivot « une vie ». */
    help: string
    /** Nom de l'indicateur offensif au survol (« Rendement »). */
    offensiveMetric: string
    /** Nom de l'indicateur défensif au survol (« Résistance »). */
    defensiveMetric: string
    /** Libellé du repère 100 % (« 1 vie »), sur le trait et au survol. */
    oneLife: string
    /** Libellé des dégâts bruts offensifs au survol. */
    damageDealt: string
    /** Libellé des dégâts bruts défensifs au survol. */
    damageTaken: string
    /** Unité du ratio offensif au survol (« / frag effectif »). */
    perFrag: string
    /** Unité du ratio défensif au survol (« / mort »). */
    perDeath: string
    noData: string
  }
  performanceCharts: {
    title: string
    description: string
    killsDeathsTitle: string
    killsLabel: string
    deathsLabel: string
    /** Libellé de la série « Bonus » (assistances converties) du chart Frags / Morts. */
    bonusLabel: string
    /** Aide ⓘ de la série Bonus : ce que vaut une assistance dans le FDA (ADR 0006). */
    bonusInfo: string
    assistsTitle: string
    kdaTitle: string
    accuracyTitle: string
    avgLifeTitle: string
    performanceTitle: string
    maxSpreeTitle: string
    hsPerfectTitle: string
    hsLabel: string
    perfectLabel: string
    rankTitle: string
    mmrLabel: string
    fragBreakdownTitle: string
  }
  /** « Écart cumulé au FDA attendu » (D3/D7 — différentiel FDA réel vs attendu par joueur). */
  fdaGap: {
    title: string
    /** Caption de la rangée de pastilles KPI (écart moyen par match). */
    averageCaption: string
  }
  /** « Balance des dégâts cumulée » (P3 — dégâts nets ÷ PV-pour-tuer, cumulé par joueur). */
  netLives: {
    title: string
    /** Tooltip explicatif de la formule (jeton {{HP}} = barème du titre). */
    tooltip: string
  }
  /** « Écart d'engagement cumulé » (P4 — résidu pace_observed − team_expected × durée, par joueur). */
  engagementGap: {
    title: string
    /** Tooltip explicatif (unité = événements en excès/déficit). */
    tooltip: string
  }
  weaponKills: {
    title: string
    description: string
    /** Ligne agrégée des armes gun au-delà du top-N dans « Outils de destruction ». */
    otherWeapons: string
  }
  /** Comparatif « Précision par rôle » multi-joueurs (Halo 5) : barres groupées horizontales (1 barre/joueur/rôle, longueur = précision %). */
  weaponAccuracy: {
    title: string
    /** Libellé « Tirs » (contexte tooltip). */
    shotsLabel: string
  }
  killMechanics: {
    title: string
    labels: { assassination: string; ground_pound: string; shoulder_bash: string }
  }
  units: {
    perGame: string
  }
  medals: {
    title: string
    dominantCategory: string
    topMedals: string
    statsDistinct: string
    statsAvg: string
    statsPeak: string
    expandLabel: string
    collapseLabel: string
    noMedals: string
    categoryLabels: {
      multikill: string
      spree: string
      skill: string
      style: string
      mode: string
      proficiency: string
      other: string
    }
  }
  errors: {
    loadError: (message: string) => string
  }
  /** KPI objectifs cumulés de l'escouade (CTF/Zones/Oddball) — V72-03. */
  objectives: {
    title: string
    flagCaptures: string
    flagReturns: string
    flagSteals: string
    flagCarrierTime: string
    zoneCaptures: string
    zoneSecures: string
    zoneTime: string
    skullGrabs: string
    skullCarrierTime: string
  }
}

const EN_TEXT: SquadText = {
  intlLocale: 'en-US',
  title: {
    teammates: 'Teammates',
    allTeammates: 'All teammates',
  },
  nav: {
    synergies: 'Synergies',
    contributions: 'Contributions',
    dynamique: 'Dynamics',
  },
  selection: {
    placeholder: (count) => `Search among ${count} teammates…`,
    prompt: 'Pick up to 3 teammates to analyze your synergies.',
  },
  filter: {
    experience: 'Experience',
    playlist: 'Playlist',
    allExperiences: 'All experiences',
    allPlaylists: 'All playlists',
    analyse: 'Analyse',
    exactComposition: 'Strict line-up',
    exactCompositionTitle:
      'By default every match started together is counted, even if another known player was with you. Tick to keep only matches played with exactly this line-up.',
  },
  dataIssues: {
    title: 'Partial data: some numbers are incomplete.',
    teammateMatches: (detail) => `Could not load ${detail}'s matches: they are missing from the shared population.`,
    heatmapTeammate: (detail) => `Could not load ${detail}'s maps: their grid row is empty.`,
    mainTeamParticipants: 'Teams could not be loaded: the strict line-up option was not applied.',
    mapStats: 'Per-map history could not be loaded: historical references are missing.',
    unknown: (code) => `Incomplete load (${code}).`,
  },
  session: {
    label: 'Session',
    squad: 'Squad',
    prev: 'Previous session',
    next: 'Next session',
    all: '(all)',
    reset: '✕ Reset',
  },
  table: {
    gamertag: 'Gamertag',
    matches: 'Matches',
    wins: 'Wins',
    winPct: 'Win%',
    kd: 'K/D',
    lastSeen: 'Last seen',
    actions: 'Actions',
    openCompare: 'Head-to-head',
    withTeammate: (gamertag) => `With ${gamertag}`,
  },
  empty: {
    noSelectionTitle: 'Synergy analysis',
    noSelectionDescription: 'Pick 1 to 3 teammates to analyze the synergies of the squad.',
    invalidSelectionTitle: 'No shared data',
    invalidSelectionDescription:
      'The selected teammates have no shared matches in the filtered period.',
    noChartTitle: 'Chart unavailable',
    noChartDescription: 'The chart could not be built with the current data.',
    noDataTitle: 'Squad data unavailable',
    noDataDescription:
      'No usable response was returned for this page. Check filters or shared matches availability.',
    noBlockData: 'No data for this selection.',
  },
  synergies: {
    description: 'Comparison of the player\'s stats with each teammate on shared matches.',
  },
  contributions: {
    description: 'Normalized contribution profile for each selected teammate.',
  },
  charts: {
    hsPkTitle: 'Headshot & Perfect kills per game',
    timelineTitle: 'Squad performance over time',
    timelinePerfName: 'Avg. performance',
    timelineWinRateName: 'Win rate',
    timelinePerfAxis: 'Perf. score',
    timelineWinRateAxis: 'Win rate',
    heatmapTitle: 'Win rate by map (squad)',
    heatmapWinAxis: 'Win rate (%)',
    heatmapMatchesLabel: 'Matches',
    winRateVsHistorySession: 'Session',
    winRateVsHistoryHistory: 'All time',
    winRateVsHistoryBulletTitle: 'Session winrate vs history',
    winRateVsHistoryBulletParity: '50% parity',
    winRateVsHistoryBulletZero: '0% (all losses)',
    winRateVsHistoryBulletCounts: (session, history) =>
      `Session: ${session} ${session <= 1 ? 'game' : 'games'} · History: ${
        history === undefined ? '—' : `${history} ${history <= 1 ? 'game' : 'games'}`
      }`,
    winRateVsHistoryBulletMapCountTooltip:
      'The number in parentheses is the number of games played on this map during the session.',
    mapPerfVsHistoryTitle: 'Performance per map — Session vs History',
    mapPerfVsHistorySession: 'Current session',
    mapPerfVsHistoryHistory: 'History',
    outcomeSequenceTitle: 'Match sequence',
  },
  history: {
    title: 'Match history with teammates',
    description: 'All shared matches matching the active filters.',
    date: 'Date',
    map: 'Map',
    playlist: 'Playlist',
    mode: 'Mode',
    outcome: 'Result',
    score: 'Score',
    winRateHist: 'Hist. win%',
    winProb: 'Win prob.',
    kda: 'K/D/A',
    accuracy: 'Accuracy',
    perf: 'Perf.',
    duration: 'Duration',
    teamMmr: 'Team MMR',
    enemyMmr: 'Enemy MMR',
    deltaMMR: 'MMR Gap',
    session: 'Session',
    outcomeLabel: { win: 'Win', loss: 'Loss', draw: 'Tie', dnf: 'DNF' },
    prev: '← Previous',
    next: 'Next →',
    pageOf: (cur, total) => `Page ${cur} / ${total}`,
    totalRows: (n) => `${n} match${n > 1 ? 'es' : ''}`,
    waypointAriaLabel: 'Open on Halo Waypoint',
    sortByAriaLabel: (col) => `Sort by ${col}`,
    winRateHistTooltip: 'Win rate for this squad across all their shared matches.',
    winProbTooltip: 'Win probability estimated before the match, from both teams\' MMR.',
    teamMmrTooltip: 'Average estimated skill level (MMR) of the team.',
    enemyMmrTooltip: 'Average estimated skill level (MMR) of the enemy team.',
    deltaMmrTooltip: 'MMR gap between the team and the enemy team.',
  },
  timeline: {
    title: 'Squad performance by session',
    perf: 'Squad perf',
    winRate: 'Win rate',
    teamMmr: 'Team MMR',
    perfAxis: 'Perf / Win %',
    mmrAxis: 'MMR',
  },
  heatmap: {
    title: 'Performance per player × map',
    pieceTier1: 'Excellent',
    pieceTier2: 'Good',
    pieceTier3: 'Average',
    pieceTier4: 'Below average',
    pieceTier5: 'Poor',
    noScore: 'No score',
  },
  impact: {
    title: 'Teammates impact',
    colPlayer: 'Player',
    colScore: 'Score',
    colBadge: 'Rank',
    colScoreTooltip: 'Cumulative impact score: each positive pill adds, each negative one subtracts.',
    colBadgeTooltip: 'Role in the squad: Champion (1st), Stowaway or Weak link (last).',
    badgeChampion: 'Champion (rank #1)',
    badgeChampionShort: 'Champion',
    badgeWeakLink: 'Weak link (last rank, negative score)',
    badgeWeakLinkShort: 'Weak link',
    badgeStowaway: 'Stowaway (last rank, non-negative score)',
    badgeStowawayShort: 'Stowaway',
    badgeNames: {
      first_blood: 'First blood',
      clutch_finisher: 'Clutch',
      last_casualty: 'Last casualty',
      last_group_kill: 'Late starter',
      first_group_death: 'First down',
      silent_hero: 'Silent hero',
      false_brother: 'False brother',
      top_killer: 'Top killer',
      top_gun: 'Top Gun',
      kamikaze: 'Kamikaze',
    },
    badgeDescriptions: {
      first_blood: 'First kill of the match, across all teams',
      clutch_finisher: 'Last kill dealt by a player from the winning team',
      last_casualty: 'Last death suffered by a player from the losing team',
      last_group_kill: 'Squad member whose first kill came latest in the match',
      first_group_death: 'First death suffered by a squad member',
      silent_hero: 'Winner (excl. top killer) with most assists and fewest deaths',
      false_brother: 'Loser (excl. top killer) with most deaths and fewest assists',
      top_killer: 'Player with the highest kill count in the match',
      top_gun: 'First squad member to reach 10 kills',
      kamikaze: 'Player killed within 1.5 s after one of their own frags (most frequent in the match)',
    },
  },
  perMinute: {
    title: 'Per-minute stats — Frags / Deaths / Assists',
    description: 'Per-player cadence over the filtered scope. Deaths render below the axis (muted player color).',
    frags: 'Frags/min',
    deaths: 'Deaths/min',
    assists: 'Assists/min',
    suffix: ' /min',
  },
  synergyRadar: {
    title: 'Synergy radar',
    description: 'Participation profile computed on matches where all selected teammates were present. Lines only (no fill), max 4 overlaid profiles.',
    axes: {
      combat: 'Combat',
      survival: 'Survival',
      support: 'Support',
      score: 'Score',
      objective: 'Objective',
      impact: 'Impact',
    },
    tooltip: {
      impact: 'Offensive conversion — 225 × (kills + ass/3) / damage. P80 = 0.83.',
      combat: 'Kills + headshots + perfect kills, weighted by accuracy.',
      survival: 'Defensive resistance — damage / (225 × deaths). P80 = 1.59.',
      support: 'Assists × 50.',
      score: 'Personal score per minute played. P80 ≈ 195/min.',
      objective: 'Objective participation per opportunity — weighted actions (captures, grabs, returns…) plus time on the objective, computed on objective matches only and calibrated per mode (a P80 player in their mode scores 80). Axis hidden when no objective match.',
      glossaryLink: '→ Glossary',
    },
    rawLabel: 'raw',
  },
  intensity: {
    title: 'Intensity',
    subtitle: 'Frag distribution across match phases',
    tooltip: 'Each match is split into 10 equal slices. The solid line shows when the player\'s kills happen: start of the match on the left, end on the right. The shaded band around it shows how much this changes from match to match — wide means the player plays very differently depending on the game, narrow means they play much the same way every time. The horizontal dashed line is the level of a match where kills would be spread evenly from start to finish. From 3 players on, the overlaid "Team" dashed curve shows the same profile for the whole squad: above it, the player is more active than the group on that slice.',
    medianLabel: 'Median',
    envelopeLabel: 'P25–P75 envelope',
    refLabel: '10%',
    teamLabel: 'Team',
  },
  efficiencySeries: {
    rendementCardTitle: 'Efficiency',
    resistanceCardTitle: 'Resistance',
    help: 'One curve per player, match by match, on a scale where 100% is exactly one Spartan life. Efficiency = what one life worth of damage dealt buys in effective kills (kills + assists / 3): 100% means one effective kill per life spent, 130% means a third better. Resistance = what is absorbed before each death, measured against one life: 100% is exactly one life, 150% is half again as much. Above the "1 life" marker (green background) performance is better, below it (red background) it falls short — in both cards. The 50–200% scale never changes, so two sessions compare directly. Hovering a match shows the raw values.',
    offensiveMetric: 'Efficiency',
    defensiveMetric: 'Resistance',
    oneLife: '1 life',
    damageDealt: 'Damage dealt',
    damageTaken: 'Damage taken',
    perFrag: '/ effective kill',
    perDeath: '/ death',
    noData: 'No efficiency data available.',
  },
  performanceCharts: {
    title: 'Performance',
    description: 'Time-series aligned on matches where every teammate was present. One line per player, colors mirror the active-player pill and multiselect.',
    killsDeathsTitle: 'Frags / Deaths',
    killsLabel: 'Frags',
    deathsLabel: 'Deaths',
    bonusLabel: 'Bonus',
    bonusInfo:
      'Bonus = assists ÷ 3: in KDA, 3 assists count as 1 kill (KDA = (kills + assists/3) − deaths). The series stacks that bonus on top of the match kills; it is hidden by default, click "Bonus" to show it.',
    assistsTitle: 'Assists',
    kdaTitle: 'KDA',
    accuracyTitle: 'Accuracy',
    avgLifeTitle: 'Avg lifespan',
    performanceTitle: 'Performance',
    maxSpreeTitle: 'Max killing spree',
    hsPerfectTitle: 'Headshots & Perfect kills',
    hsLabel: 'Headshots',
    perfectLabel: 'Perfect kills',
    rankTitle: 'Rank & Team MMR',
    mmrLabel: 'Team MMR',
    fragBreakdownTitle: 'Kill type distribution',
  },
  fdaGap: {
    title: 'Cumulative KDA gap to expected',
    averageCaption: 'Average gap per match',
  },
  netLives: {
    title: 'Cumulative damage balance',
    tooltip: 'Damage balance = (damage dealt − damage taken) ÷ {{HP}} HP, expressed in lives. Positive = the balance carries the team; negative = it costs more than it brings.',
  },
  engagementGap: {
    title: 'Cumulative engagement gap',
    tooltip: 'Engagement gap = (observed pace − expected pace) × match duration, accumulated. Expressed in events in surplus (positive) or deficit (negative) vs expected.',
  },
  weaponKills: {
    title: 'Tools of destruction',
    description: 'Cumulative kills per weapon over shared matches. Sorted ASC: rare weapons on top, primaries at the bottom.',
    otherWeapons: 'Other weapons',
  },
  weaponAccuracy: {
    title: 'Accuracy by role',
    shotsLabel: 'Shots',
  },
  killMechanics: {
    title: 'Kill mechanics',
    labels: { assassination: 'Assassinations', ground_pound: 'Ground Pounds', shoulder_bash: 'Shoulder Bashes' },
  },
  units: {
    perGame: '/game',
  },
  medals: {
    title: 'Medals — Squad summary',
    dominantCategory: 'Dominant category',
    topMedals: 'Top medals',
    statsDistinct: 'Distinct types',
    statsAvg: 'Avg/match',
    statsPeak: 'Peak',
    expandLabel: 'Show all medals',
    collapseLabel: 'Collapse',
    noMedals: 'No medals available for this selection.',
    categoryLabels: {
      multikill: 'Multi-kills',
      spree: 'Spree',
      skill: 'Skill',
      style: 'Style',
      mode: 'Mode',
      proficiency: 'Proficiency',
      other: 'Other',
    },
  },
  errors: {
    loadError: (message) => `Error: ${message}`,
  },
  objectives: {
    title: 'Squad objectives',
    flagCaptures: 'Flag captures',
    flagReturns: 'Flag returns',
    flagSteals: 'Flag steals',
    flagCarrierTime: 'Flag carrier time',
    zoneCaptures: 'Zones captured',
    zoneSecures: 'Zones secured',
    zoneTime: 'Time in zones',
    skullGrabs: 'Skull grabs',
    skullCarrierTime: 'Skull carrier time',
  },
}

const DICTS: Record<Locale, SquadText> = {
  en: EN_TEXT,
}

/** Retourne le dictionnaire pour la locale demandée (fallback fr). */
export function getSquadText(locale: Locale | string | undefined): SquadText {
  if (locale === 'en') return DICTS.en
  return DICTS.en
}

export { EN_TEXT }
