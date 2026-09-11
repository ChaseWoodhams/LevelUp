/// <reference types="node" />
// @vitest-environment node
/**
 * copies.guard.test.ts — THE COPIES HAVE NOT DRIFTED.
 *
 * WHY A GUARD AND NOT A CONVENTION. Most of this app's replay view is copied from
 * `apps/web` (there is no workspace wiring cross-app imports, and `apps/web/**` is out
 * of bounds from here). A copy is honest only for as long as somebody keeps it so, and
 * "keep it so" written in a README is exactly the shape of debt this repository has
 * learnt to distrust: a convention without a guard-rail re-diverges. So the property
 * is asserted, not asked for.
 *
 * TWO PROPERTIES, AND A THIRD FOR THE TOKENS:
 *   1. every file carrying a `COPIED FILE — origin:` header is byte-identical to that
 *      origin below the header;
 *   2. the SHA in the header really is the origin's last-modifying commit — so
 *      `git diff <sha> HEAD -- <origin>` says what the origin has learnt since;
 *   3. the `--ac-*` values in `styles/tokens.css`, which are an EXTRACT and therefore
 *      escape (1), still match the palette they were extracted from.
 *
 * WHAT A FAILURE MEANS. Not "this app is broken" — "the origin moved". Re-copy the
 * file, bump its SHA, and check whether the change was a fix that belongs here too.
 */
import { execFileSync } from 'node:child_process'
import { readdirSync, readFileSync, statSync } from 'node:fs'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const STUDY_SRC = resolve(dirname(fileURLToPath(import.meta.url)))
const REPO_ROOT = resolve(STUDY_SRC, '../../..')

/** First line of the header, carrying the origin path. */
const ORIGIN_RE = /^ \* COPIED FILE — origin: (.+)$/m
/** Second line, carrying the commit the origin was at. */
const SHA_RE = /^ \* Origin at commit: ([0-9a-f]{7,40}) /m
/** End of the header block: everything after it must match the origin byte for byte. */
const HEADER_END = '\n */\n'

function sourceFiles(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const path = join(dir, name)
    if (statSync(path).isDirectory()) return sourceFiles(path)
    return /\.(ts|tsx)$/.test(name) ? [path] : []
  })
}

/** Every file in this app that declares itself a copy, with its origin and SHA. */
const copies = sourceFiles(STUDY_SRC).flatMap((path) => {
  const text = readFileSync(path, 'utf8')
  const origin = ORIGIN_RE.exec(text)?.[1]
  if (!origin) return []
  return [{ path, text, origin, sha: SHA_RE.exec(text)?.[1] }]
})

/** lastCommitOf gives the short SHA of the commit that last changed `repoPath`. */
function lastCommitOf(repoPath: string): string {
  return execFileSync('git', ['log', '-1', '--format=%h', '--', repoPath], {
    cwd: REPO_ROOT,
    encoding: 'utf8',
  }).trim()
}

describe('copied files', () => {
  it('are declared at all — a silent zero here would make every case below vacuous', () => {
    expect(copies.length).toBeGreaterThan(20)
  })

  it.each(copies.map((c) => [c.origin, c] as const))('%s is unchanged in the copy', (_o, copy) => {
    const cut = copy.text.indexOf(HEADER_END)
    expect(cut, `no header block in ${copy.path}`).toBeGreaterThan(-1)
    const body = copy.text.slice(cut + HEADER_END.length)
    expect(body).toBe(readFileSync(join(REPO_ROOT, copy.origin), 'utf8'))
  })

  it.each(copies.map((c) => [c.origin, c] as const))('%s carries its real SHA', (_o, copy) => {
    expect(copy.sha).toBeDefined()
    expect(copy.sha).toBe(lastCommitOf(copy.origin))
  })
})

/**
 * The semantic tokens are the one copy that cannot be byte-compared: `tokens.css` holds
 * the handful of `--ac-*` values the copied components read, out of a palette of sixty.
 * A wrong value there would not fail anything — it would just paint the wrong colour —
 * so the values are checked against the palette they came from.
 */
describe('semantic colour tokens', () => {
  const PALETTE = join(REPO_ROOT, 'apps/web/src/lib/accessibility/palettes/default.ts')

  it('hold the same values as the palette they were extracted from', () => {
    const palette = new Map<string, string>()
    for (const [, token, hex] of readFileSync(PALETTE, 'utf8').matchAll(
      /'([a-z0-9-]+)':\s*'(#[0-9A-Fa-f]{3,8})'/g,
    )) {
      palette.set(token, hex.toUpperCase())
    }
    expect(palette.size).toBeGreaterThan(0)

    const css = readFileSync(join(STUDY_SRC, 'styles/tokens.css'), 'utf8')
    const declared = [...css.matchAll(/--ac-([a-z0-9-]+):\s*(#[0-9A-Fa-f]{3,8});/g)]
    expect(declared.length).toBeGreaterThan(0)

    for (const [, token, hex] of declared) {
      expect(palette.get(token), `--ac-${token} is not a token of the palette`).toBeDefined()
      expect(hex.toUpperCase(), `--ac-${token}`).toBe(palette.get(token))
    }
  })
})
