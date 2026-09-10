/**
 * i18n.ts — traductions FR + EN du composant SessionBriefing.
 *
 * Les libellés outcomes (Victoire / Défaite / Égalité / Abandon) ne sont PAS
 * dans ce fichier — ils proviennent de outcomes.toml via useFieldMappings()
 * pour rester title-aware (multi-titres).
 */


export interface BriefingTexts {
  /** Libellé "Résultats" + helper pluralisation outcomes — utilisés dans la
   *  Results bar de SquadVerdict (squad mode uniquement). */
  rail: {
    resultsLabel: string
  }
  verdict: {
    teamScore: string
    deltaBonusPositive: (delta: number) => string
    deltaBonusNegative: (delta: number) => string
  }
  drill: {
    activeView: (gamertag: string) => string
    resetButton: string
  }
  grid: {
    titleSelf: string
    trendHint: string
    matchesPlayed: string
    totalDuration: string
    fragsPerMatch: string
    deathsPerMatch: string
    assistsPerMatch: string
    accuracy: string
    lifespan: string
    perMin: string
    perMatch: string
    rankDeltaCSR: string
    rankDeltaLUSR: string
    rendement: string
    resistance: string
    /** Libellé combiné de la card composite Rendement/Résistance (aligné sur la home). */
    offDef: string
    refBaseline: string
  }
  /** Format pluriel pour les libellés outcomes — utilisé dans la Results bar */
  pluralize: (count: number, singular: string) => string
}

const EN: BriefingTexts = {
  rail: {
    resultsLabel: 'Results',
  },
  verdict: {
    teamScore: 'Team score',
    deltaBonusPositive: (d) => `Team bonus +${d}`,
    deltaBonusNegative: (d) => `Team penalty ${d}`,
  },
  drill: {
    activeView: (gt) => `Viewing: ${gt}`,
    resetButton: '✕ back to my stats',
  },
  grid: {
    titleSelf: 'My stats this session',
    trendHint: '▲/▼ vs team average on this session',
    matchesPlayed: 'Matches played',
    totalDuration: 'Total duration',
    fragsPerMatch: 'Frags per match',
    deathsPerMatch: 'Deaths per match',
    assistsPerMatch: 'Assists per match',
    accuracy: 'Avg accuracy',
    lifespan: 'Avg lifespan',
    perMin: '/min',
    perMatch: '/match',
    rankDeltaCSR: 'CSR change',
    rankDeltaLUSR: 'LUSR change',
    rendement: 'Rendement',
    resistance: 'Resistance',
    offDef: 'Off. / Def.',
    refBaseline: 'ref. 100%',
  },
  // EN : pluralisation par "s" couvre nos labels outcomes (Win → Wins, Loss → Losses
  // est géré séparément si needed — ici on assume singulier sans -s).
  pluralize: (count, singular) => {
    if (count <= 1) return singular
    if (singular.endsWith('s') || singular.endsWith('x')) return `${singular}es`
    return `${singular}s`
  },
}

export function getBriefingTexts(): BriefingTexts {
  return EN
}
