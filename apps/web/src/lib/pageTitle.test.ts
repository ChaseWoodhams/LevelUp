/// <reference types="node" />
// @vitest-environment node
/**
 * Tests — resolvePageTitle (I18, 2026-07-24: browser tab titles).
 *
 * Two blocks:
 *  1. Unit cases (behaviour per pathname, the Citations/Commendations nuance, the kept
 *     fallback).
 *  2. Ratchet guard-rail — scans `src/routes/**` (same convention as
 *     `lib/title-routing/no-title-literals.ratchet.test.ts`): EVERY real route
 *     (declares `component:`, excluding pure layouts and transitional redirects — see
 *     the documented exclusions) must resolve a NON-fallback title. A new route without
 *     an entry in `pageTitle.ts` fails this test.
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'
import { resolvePageTitle } from './pageTitle'
import { routeTemplateSuffix } from './title-routing'

describe('resolvePageTitle (URLs title-scoped)', () => {
  it('override par suffixe joueur (forme courte /t/)', () => {
    expect(resolvePageTitle('/t/halo_infinite/players/jgtm/stats/timeseries', 'en')).toBe(
      'LevelUp - Time series',
    )
  })

  it('override par suffixe joueur avec segment langue', () => {
    expect(resolvePageTitle('/en/t/halo_5/players/x/career/citations', 'en')).toBe('LevelUp - Citations')
  })

  it('titre dérivé d’un item de nav (accueil)', () => {
    expect(resolvePageTitle('/t/halo_infinite/players/x/home', 'en')).toBe('LevelUp - Home')
  })

  it('page de match', () => {
    expect(resolvePageTitle('/t/halo_infinite/players/x/matches/abc-123', 'en')).toBe('LevelUp - Match')
  })

  it('racine joueur nue → Accueil', () => {
    expect(resolvePageTitle('/t/halo_infinite/players/x', 'en')).toBe('LevelUp - Home')
  })

  it('page statique (agnostique)', () => {
    expect(resolvePageTitle('/settings', 'en')).toBe('LevelUp - Settings')
    expect(resolvePageTitle('/', 'en')).toBe('LevelUp - Home')
  })

  it('suffixe/pathname inconnu → LevelUp (fallback conservé)', () => {
    expect(resolvePageTitle('/t/halo_infinite/players/x/zzz-inconnu', 'en')).toBe('LevelUp')
    expect(resolvePageTitle('/zzz-static-inconnu', 'en')).toBe('LevelUp')
  })

  it('nuance Citations (moteur dérivé Infinite) vs Commendations (natif H5)', () => {
    // The title follows the ROUTE (/career/citations vs /career/commendations), never the
    // title actually active.
    expect(resolvePageTitle('/t/halo_infinite/players/x/career/citations', 'en')).toBe('LevelUp - Citations')
    expect(resolvePageTitle('/t/halo_5/players/x/career/commendations', 'en')).toBe('LevelUp - Commendations')
  })

  it('trous comblés (I18) : /career/medals et /squad/dynamique', () => {
    expect(resolvePageTitle('/t/halo_infinite/players/x/career/medals', 'en')).toBe('LevelUp - Medals')
    expect(resolvePageTitle('/t/halo_infinite/players/x/squad/dynamique', 'en')).toBe('LevelUp - Dynamics')
  })

  it('trous comblés (I18) : sous-onglets Administration (ex-pattern `/admin` ancré, ne matchait aucun enfant)', () => {
    expect(resolvePageTitle('/admin/management', 'en')).toBe('LevelUp - Administration — Management')
    expect(resolvePageTitle('/admin/system', 'en')).toBe('LevelUp - Administration — System')
  })
})

// ─── Garde-rail : toutes les routes RÉELLES résolvent un titre non-fallback ───────────

const routesRoot = resolve(process.cwd(), 'src', 'routes')

/**
 * Fichiers exclus du balayage — layouts purs : ils rendent un `<Outlet/>`/écran de
 * gate, jamais un contenu de page propre. Leur route ENFANT (index) partage la MÊME
 * URL normalisée (ex. `admin.tsx` id `/admin` == `admin/index.tsx` id `/admin/`) et
 * porte l'exigence de titre à leur place — les lister ici évite un doublon inutile.
 */
const LAYOUT_ONLY_RELATIVE_PATHS = new Set([
  'admin.tsx',
  join('{-$lang}', 't', '$titleSlug.tsx'),
  join('{-$lang}', 't', '$titleSlug', 'players', '$playerSlug.tsx'),
  join('{-$lang}', 't', '$titleSlug', 'players', '$playerSlug', 'ascension.tsx'),
  join('{-$lang}', 't', '$titleSlug', 'players', '$playerSlug', 'squad.tsx'),
])

// Splat de redirection legacy (Phase 3, `routes/players/$.tsx`) : composant déclaratif
// qui ne rend jamais un titre stable (redirige avant peinture) — hors périmètre titre.
const SPLAT_EXCLUDED = join('players', '$.tsx')

function walkRouteFiles(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      out.push(...walkRouteFiles(full))
    } else if (/\.tsx$/.test(entry.name) && !/\.test\.tsx$/.test(entry.name)) {
      out.push(full)
    }
  }
  return out
}

const PLAYER_MARKER = '/players/$playerSlug'
const DUMMY_PARAM = 'dummy'

/**
 * Convertit l'id de route brut (littéral passé à `createFileRoute(...)`) en un
 * pathname de test concret :
 *  - convention TanStack Router du underscore final de segment (`career_` → `career`,
 *    URL identique à sans le underscore, cf. routeTree.gen.ts `id: '/career_', path:
 *    '/career'`) ;
 *  - suffixe relatif au joueur (routeTemplateSuffix) reconstruit en pathname `/t/…` ;
 *  - params résiduels (`$matchId`, …) remplacés par une valeur fixe.
 */
function toTestPathname(routeId: string): string {
  const deUnderscored = routeId.replace(/_(?=\/|$)/, '')
  if (deUnderscored.includes(PLAYER_MARKER)) {
    const rawSuffix = routeTemplateSuffix(deUnderscored)
    const suffix = rawSuffix.replace(/\$[A-Za-z][A-Za-z0-9]*/g, DUMMY_PARAM)
    return `/t/halo_infinite/players/x${suffix}`
  }
  return deUnderscored.replace(/\$[A-Za-z][A-Za-z0-9]*/g, DUMMY_PARAM)
}

interface RouteFixture {
  file: string
  routeId: string
}

function collectRealRouteFixtures(): RouteFixture[] {
  const fixtures: RouteFixture[] = []
  for (const file of walkRouteFiles(routesRoot)) {
    const rel = file.slice(routesRoot.length + 1)
    if (LAYOUT_ONLY_RELATIVE_PATHS.has(rel) || rel === SPLAT_EXCLUDED) continue
    const content = readFileSync(file, 'utf8')
    // Redirection pure (`beforeLoad` sans `component`) : /citations, /commendations,
    // /compare, /synthesis, /objectifs/, /palmares/*, quelques /admin/* legacy — ne
    // rendent jamais de page stable, donc jamais de titre stable non plus.
    if (!/\bcomponent\s*:/.test(content)) continue
    const match = /createFileRoute\(\s*['"]([^'"]+)['"]/.exec(content)
    if (!match) continue // __root.tsx (createRootRouteWithContext) et fichiers non-route
    fixtures.push({ file: rel, routeId: match[1] })
  }
  return fixtures
}

describe('garde-rail : toutes les routes réelles ont un titre (anti-régression I18)', () => {
  const fixtures = collectRealRouteFixtures()

  it('le balayage a trouvé un nombre plausible de routes (non vide, ne masque pas un chemin cassé)', () => {
    expect(fixtures.length).toBeGreaterThan(30)
  })

  it.each(fixtures)('$file → non-fallback title', ({ routeId }) => {
    const pathname = toTestPathname(routeId)
    const title = resolvePageTitle(pathname, 'en')
    expect(title, `pathname=${pathname} routeId=${routeId}`).not.toBe('LevelUp')
  })
})
