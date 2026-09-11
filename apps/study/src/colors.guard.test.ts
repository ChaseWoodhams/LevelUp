/// <reference types="node" />
// @vitest-environment node
/**
 * colors.guard.test.ts — NO COLOUR LITERAL IN THIS APP'S OWN CODE.
 *
 * The repository's rule is that a colour which MEANS something — a team, a shot, a
 * verdict — passes through a semantic token, never a hex value and never a Tailwind
 * colour class. `tools/lint-no-hardcoded-colors.mjs` enforces it, and it enforces it
 * over `apps/web/src`: that path is hardcoded in the script, so this app is outside its
 * reach. Nothing else stands between the rule and this folder.
 *
 * Rather than restate the rule in a README and hope, it is asserted here, over the same
 * two shapes the shared linter looks for.
 *
 * THREE EXEMPTIONS, and each one is a file where a hex value is the SUBJECT rather than a
 * design decision:
 *   - `styles/tokens.css` — the palette. Hex values are its entire reason to exist, and
 *     `copies.guard.test.ts` checks each against the palette it was extracted from.
 *   - `lib/api/generated.ts` — generated from the OpenAPI contract, not written here.
 *   - `features/viewer/fade.test.ts` — the test of the colour-notation parser. It asserts what
 *     `#818CF8` dimmed comes back as; the literals ARE the cases, and none of them reaches a
 *     pixel. `fade.ts` itself carries no literal and is not exempt.
 * Copied files are NOT exempt: none of them carries a literal today, and if one arrives
 * from upstream it should be seen here rather than waved through.
 */
import { readdirSync, readFileSync, statSync } from 'node:fs'
import { dirname, join, relative, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const STUDY_SRC = resolve(dirname(fileURLToPath(import.meta.url)))

const EXEMPT = ['styles/tokens.css', 'lib/api/generated.ts', 'features/viewer/fade.test.ts']

/** `#abc`, `#aabbcc`, `#aabbccdd` — the shapes a colour literal actually takes. */
const HEX_RE = /#[0-9a-fA-F]{3}(?:[0-9a-fA-F]{3}(?:[0-9a-fA-F]{2})?)?\b/g

/** `bg-red-500`, `text-slate-200`, `border-emerald-600`… — Tailwind's colour scales. */
const TAILWIND_RE = new RegExp(
  String.raw`\b(?:bg|text|border|ring|fill|stroke|from|via|to|decoration|outline|shadow|accent|caret|divide|placeholder)-` +
    '(?:slate|gray|grey|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose)' +
    String.raw`-\d{2,3}\b`,
  'g',
)

function filesUnder(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const path = join(dir, name)
    if (statSync(path).isDirectory()) return filesUnder(path)
    return /\.(ts|tsx|css)$/.test(name) ? [path] : []
  })
}

const scanned = filesUnder(STUDY_SRC)
  .map((path) => ({ path, rel: relative(STUDY_SRC, path).replace(/\\/g, '/') }))
  .filter((f) => !EXEMPT.includes(f.rel))
  .map((f) => ({ ...f, text: readFileSync(f.path, 'utf8') }))

/**
 * hits reports offending lines, minus comment lines: the copied modules DISCUSS the rule
 * in prose (`canvasInk.ts` quotes a hex to explain why it refuses one), and flagging a
 * sentence about a colour would train the next reader to silence the guard.
 */
function hits(text: string, pattern: RegExp): string[] {
  return text
    .split('\n')
    .filter((line) => !/^\s*(\/\/|\/?\*)/.test(line))
    .filter((line) => new RegExp(pattern.source, pattern.flags).test(line))
    .map((line) => line.trim())
}

describe('colour literals', () => {
  it('scans something — an empty sweep would pass for the wrong reason', () => {
    expect(scanned.length).toBeGreaterThan(20)
  })

  it('no hex value outside the palette', () => {
    const offenders = scanned.flatMap((f) => hits(f.text, HEX_RE).map((l) => `${f.rel}: ${l}`))
    expect(offenders, 'use a semantic token (--ac-*), not a hex value').toEqual([])
  })

  it('no Tailwind colour class', () => {
    const offenders = scanned.flatMap((f) => hits(f.text, TAILWIND_RE).map((l) => `${f.rel}: ${l}`))
    expect(offenders, 'use a design-system class (bg-card, text-muted-foreground…)').toEqual([])
  })
})
