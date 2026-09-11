#!/usr/bin/env node
/**
 * Garde-fou — `apps/web/src/lib/api/generated.ts` est-il À JOUR vis-à-vis de
 * `apps/go-api/api/openapi.yaml` ?
 *
 * Chaînon manquant du verrouillage du contrat (contre-revue V72). Deux maillons
 * existaient déjà :
 *   - openapi.yaml ← code Go        : `make openapi-check` / TestOpenAPIYAMLIsUpToDate
 *   - generated.ts ← surface figée  : `contract-surface.guard.test.ts` (DISPARITIONS only)
 * Rien ne vérifiait que generated.ts DÉRIVE de l'openapi.yaml courant : une évolution
 * de contrat committée sans `make generate-types` laissait le front typé sur l'ANCIEN
 * contrat (schéma ajouté invisible, corps de réponse manquant, enum non resserré) —
 * `tsc` restant vert puisque les types sont simplement périmés, pas incohérents.
 *
 * Méthode : on rejoue EXACTEMENT la commande du script npm `generate-types`
 * (openapi-typescript, même version, mêmes options) vers un fichier temporaire, puis on
 * compare octet à octet. Rejouer le générateur — plutôt que ré-analyser le YAML — rend
 * la comparaison insensible au formatage et fidèle par construction.
 *
 * Usage      : node tools/check-generated-types-fresh.mjs [apps/web|apps/study]
 * Exit code  : 0 = à jour · 1 = drift (ou générateur absent / en échec)
 * Réparation : cd <app> && npm run generate-types
 *
 * Appelé par : `make openapi-check` (gate manuelle du contrat), le garde-rail
 * `apps/web/src/lib/api/generated-types-fresh.guard.test.ts` (CI, job Frontend) et son
 * homologue dans `apps/study` — qui génère ses types depuis LE MÊME contrat, d'où le
 * paramètre plutôt qu'une seconde copie de ce script.
 */

import { execFileSync } from 'node:child_process'
import { existsSync, mkdtempSync, readFileSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const REPO_ROOT = resolve(__dirname, '..')
const OPENAPI_YAML = join(REPO_ROOT, 'apps/go-api/api/openapi.yaml')

// Application vérifiée par défaut. Le paramètre existe depuis que `apps/study` génère
// ses types depuis LE MÊME contrat : une seconde app, pas une seconde implémentation —
// dupliquer ce script aurait laissé les deux vérifications diverger.
const DEFAULT_APP_DIR = 'apps/web'

/**
 * Retourne null si le `generated.ts` de `appDir` est à jour, sinon le message d'erreur
 * expliquant le drift (ou l'impossibilité de conclure). Aucun effet de bord process : le
 * caller décide (exit code côté CLI, assertion côté test).
 *
 * `appDir` est un chemin RELATIF à la racine du dépôt (`apps/web`, `apps/study`) : chaque
 * app a ses propres `node_modules`, donc son propre CLI openapi-typescript, et c'est bien
 * celui-là qu'il faut rejouer — une version différente produirait un diff qui ne dit rien
 * du contrat.
 */
export function checkGeneratedTypesFresh(appDir = DEFAULT_APP_DIR) {
  const APP_DIR = join(REPO_ROOT, appDir)
  const GENERATED_TS = join(APP_DIR, 'src/lib/api/generated.ts')
  // Entrée JS du CLI (et non node_modules/.bin/…) : invocable par `node` sur les trois
  // plateformes, sans dépendre du shim shell/cmd.
  const CLI = join(APP_DIR, 'node_modules/openapi-typescript/bin/cli.js')
  const FIX_HINT = `Réparation : cd ${appDir} && npm run generate-types`

  for (const [label, path] of [
    ['contrat OpenAPI', OPENAPI_YAML],
    ['types générés', GENERATED_TS],
    ['CLI openapi-typescript (npm ci manquant ?)', CLI],
  ]) {
    if (!existsSync(path)) return `${label} introuvable : ${path}`
  }

  const tmp = mkdtempSync(join(tmpdir(), 'levelup-gen-types-'))
  const candidate = join(tmp, 'generated.ts')
  try {
    try {
      execFileSync(process.execPath, [CLI, OPENAPI_YAML, '-o', candidate], {
        cwd: APP_DIR,
        stdio: 'pipe',
      })
    } catch (err) {
      const detail = err?.stderr?.toString().trim() || err?.message || String(err)
      return `échec de la génération de référence :\n${detail}\n  ${FIX_HINT}`
    }

    const expected = readFileSync(candidate, 'utf8')
    const actual = readFileSync(GENERATED_TS, 'utf8')
    if (expected === actual) return null

    // Première ligne divergente : suffit à identifier le schéma/chemin en cause.
    const exp = expected.split('\n')
    const act = actual.split('\n')
    let i = 0
    while (i < exp.length && i < act.length && exp[i] === act[i]) i++
    return (
      `DRIFT — ${appDir}/src/lib/api/generated.ts ne correspond pas à openapi.yaml.\n` +
      `  Première divergence ligne ${i + 1} :\n` +
      `    committé : ${JSON.stringify(act[i] ?? '<fin de fichier>')}\n` +
      `    attendu  : ${JSON.stringify(exp[i] ?? '<fin de fichier>')}\n` +
      `  ${FIX_HINT}`
    )
  } finally {
    rmSync(tmp, { recursive: true, force: true })
  }
}

// Exécution directe (make openapi-check) : import en tant que module = pas d'effet.
// Argument optionnel : le dossier de l'app à vérifier (défaut apps/web).
if (process.argv[1] && resolve(process.argv[1]) === resolve(fileURLToPath(import.meta.url))) {
  const problem = checkGeneratedTypesFresh(process.argv[2] || DEFAULT_APP_DIR)
  if (problem) {
    console.error(`[generated-types] ${problem}`)
    process.exit(1)
  }
  console.log('[generated-types] OK — generated.ts dérive bien de openapi.yaml.')
}
