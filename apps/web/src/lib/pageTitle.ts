import type { Locale } from '@/lib/i18n/locale'
import { playerRelativePath } from '@/lib/title-routing'

/** Titre localisé — parité FR/EN obligatoire par typage (CLAUDE.md règle 1). */
interface LocalizedTitle {
  en: string
}

interface RouteTitleRule {
  pattern: string
  title: LocalizedTitle
}

// Overrides de titre par SUFFIXE relatif au joueur : le pathname title-scoped a la
// forme `/{-lang}/t/{slug}/players/{playerSlug}{suffix}` et seul le suffixe identifie
// la page (cf. playerRelativePath). Patterns ancrés (^…$) → l'ordre n'est indicatif
// que de l'intention (le plus spécifique déclaré avant le plus générique). Aucun
// littéral `/players/` (garde-rail D-10).
//
// Table EXHAUSTIVE (I18, 2026-07-24) : couvre CHAQUE route réelle sous le scope joueur
// (garde-rail `pageTitle.test.ts` — un nouveau fichier de route sans entrée ici fait
// échouer le test). Remplace l'ancienne dérivation depuis
// `shellNavigation.{PLAYER_PRIMARY,PLAYER_SECONDARY}_NAV_ITEMS` : ces deux exports
// n'avaient plus AUCUN autre consommateur (la nav réelle vit dans `navL1Sections.tsx` /
// `NavL2.tsx`, locale-aware via `commonManifest`), portaient des libellés FR figés sans
// variante EN, et sont supprimés du même coup avec `shellNavigation.ts` (règle CLAUDE.md
// n°7, 0 code mort). Les libellés EN ci-dessous reprennent VERBATIM les traductions
// canoniques déjà établies ailleurs (`lib/i18n/generated/common.ts` `common.nav.*`,
// `features/citations/i18n` via `citationsManifest`, `features/compare/i18n.ts`,
// `features/squad/i18n.ts`) pour rester cohérentes avec la barre d'onglets réellement
// affichée.
const PLAYER_SUFFIX_OVERRIDES: RouteTitleRule[] = [
  // Accueil
  { pattern: '', title: { en: 'Home' } }, // racine joueur nue
  { pattern: '/home', title: { en: 'Home' } },
  // Solo
  { pattern: '/stats/timeseries', title: { en: 'Time series' } },
  { pattern: '/stats/sessions', title: { en: 'Sessions' } },
  { pattern: '/stats/synthesis', title: { en: 'Summary' } },
  { pattern: '/stats', title: { en: 'Solo' } },
  // Escouade
  { pattern: '/squad/synergies', title: { en: 'Synergies' } },
  { pattern: '/squad/contributions', title: { en: 'Contributions' } },
  { pattern: '/squad/dynamique', title: { en: 'Dynamics' } },
  { pattern: '/squad', title: { en: 'Squad' } },
  // Carrière — nuance Citations/Commendations (I18) : la source est fixée par la ROUTE
  // (/career/citations = moteur dérivé Infinite, /career/commendations = totaux natifs
  // H5), jamais par une donnée runtime — même distinction que l'ex-effet local de
  // `UnifiedCitationsPage` (titleKey), désormais supprimé au profit de cette table
  // (source unique). FR identique dans les deux cas (« Citations » est le terme
  // officiel Halo FR pour les deux titres, cf. `common.nav.tab_citations`).
  { pattern: '/career/citations', title: { en: 'Citations' } },
  { pattern: '/career/commendations', title: { en: 'Commendations' } },
  { pattern: '/career/medals', title: { en: 'Medals' } },
  { pattern: '/career/season-pass', title: { en: 'Season pass' } },
  { pattern: '/career', title: { en: 'Career' } },
  // Ascension (refonte 4 onglets 2026-07 : Profil + Objectifs + Entraînement + Réalisations)
  {
    pattern: '/ascension/objectifs',
    title: { en: 'Ascension — Objectives' },
  },
  {
    pattern: '/ascension/coaching',
    title: { en: 'Ascension — Coaching' },
  },
  {
    pattern: '/ascension/realisations',
    title: { en: 'Ascension — Achievements' },
  },
  { pattern: '/ascension', title: { en: 'Ascension' } },
  // Route historique /objectifs redirect → /ascension/objectifs (préservée pour bookmarks).
  { pattern: '/objectifs', title: { en: 'Ascension' } },
  // Communauté / Palmarès
  { pattern: '/community/compare', title: { en: 'Head-to-head' } },
  { pattern: '/community/relations', title: { en: 'Relations' } },
  { pattern: '/community/prestige', title: { en: 'Leaderboard PP' } },
  { pattern: '/community', title: { en: 'Community' } },
  // Médias / Explorer
  { pattern: '/media', title: { en: 'Media' } },
  { pattern: '/explorer', title: { en: 'Explorer' } },
  // Divers
  { pattern: '/matches/$matchId/replay', title: { en: 'Replay' } },
  { pattern: '/matches/$matchId', title: { en: 'Match' } },
  { pattern: '/notifications', title: { en: 'Notifications' } },
]

// Titres des pages agnostiques (hors scope joueur). Table EXHAUSTIVE (I18) — couvre
// chaque route STATIQUE réelle, dont les 6 sous-onglets Administration (DC-8) qui
// n'avaient PAS de titre propre : le pattern `/admin` est ANCRÉ (`^\/admin\/?$`) et ne
// matchait jamais `/admin/xxx`, donc `/admin/management`, `/admin/data`,
// `/admin/detections`, `/admin/sync` et `/admin/system` retombaient tous sur le
// fallback 'LevelUp'. Remplace aussi la dérivation depuis
// `shellNavigation.GLOBAL_SHELL_LINKS` (même sort que PLAYER_*_NAV_ITEMS ci-dessus —
// supprimé, 0 autre consommateur).
const STATIC_ROUTE_TITLES: RouteTitleRule[] = [
  { pattern: '/', title: { en: 'Home' } },
  {
    pattern: '/admin/detections',
    title: { en: 'Administration — Detections' },
  },
  {
    pattern: '/admin/data',
    title: { en: 'Administration — Data' },
  },
  {
    pattern: '/admin/sync',
    title: { en: 'Administration — Sync' },
  },
  {
    pattern: '/admin/system',
    title: { en: 'Administration — System' },
  },
  {
    pattern: '/admin/management',
    title: { en: 'Administration — Management' },
  },
  { pattern: '/admin', title: { en: 'Administration' } },
  { pattern: '/changelog', title: { en: 'Changelog' } },
  { pattern: '/groups', title: { en: 'My groups' } },
  { pattern: '/help', title: { en: 'Help' } },
  { pattern: '/join', title: { en: 'Join a group' } },
  // Sandbox dev interne, jamais lié depuis la nav prod (cf. ChartsShowcasePage) —
  // conservé pour que l'onglet ne reste pas nu si on y accède en direct.
  { pattern: '/lab/charts', title: { en: 'Charts gallery' } },
  { pattern: '/login', title: { en: 'Sign in' } },
  { pattern: '/onboarding/openspartan', title: { en: 'Welcome' } },
  { pattern: '/register', title: { en: 'Register' } },
  { pattern: '/settings', title: { en: 'Settings' } },
  { pattern: '/setup', title: { en: 'Setup' } },
]

function escapeRegex(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function compileRoutePattern(pattern: string): RegExp {
  if (pattern === '/') return /^\/$/
  if (pattern === '') return /^$/

  const escaped = escapeRegex(pattern)
  const withParams = escaped.replace(/\\\$[A-Za-z][A-Za-z0-9]*/g, '[^/]+')
  return new RegExp(`^${withParams}/?$`)
}

const PLAYER_RULES = PLAYER_SUFFIX_OVERRIDES.map((rule) => ({
  ...rule,
  regex: compileRoutePattern(rule.pattern),
}))

const STATIC_RULES = STATIC_ROUTE_TITLES.map((rule) => ({
  ...rule,
  regex: compileRoutePattern(rule.pattern),
}))

/**
 * Titre d'onglet navigateur pour un pathname donné, dans la locale active.
 *
 * MÉCANISME UNIQUE (I18, 2026-07-24) : consommé exclusivement par l'effet
 * `[pathname, locale]` de `__root.tsx`. Les anciens effets locaux dupliqués
 * (`MedalsPage`, `ComparePage`, `UnifiedCitationsPage`) sont supprimés — ils étaient de
 * toute façon systématiquement écrasés par CE résolveur, rejoué à chaque navigation
 * après le montage des effets enfants (cause du symptôme « onglet figé/nu »).
 */
export function resolvePageTitle(pathname: string, locale: Locale): string {
  // Sous un scope joueur : on matche le SUFFIXE ; sinon (page agnostique) le pathname.
  const suffix = playerRelativePath(pathname)
  const rules = suffix !== null ? PLAYER_RULES : STATIC_RULES
  const target = suffix !== null ? suffix : pathname
  const matchingRule = rules.find((rule) => rule.regex.test(target))
  return matchingRule ? `LevelUp - ${matchingRule.title[locale]}` : 'LevelUp'
}
