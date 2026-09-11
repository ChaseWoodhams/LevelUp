/// <reference types="node" />
// @vitest-environment node
/**
 * controls.guard.test.ts — THE SMALL CONTROL IS DEFINED IN ONE PLACE.
 *
 * `h-7 px-2 text-xs` reached eight hand-written copies across this app's dense control rows
 * before it was pulled into `controls.tsx`. The repository's rule does not stop at "centralise":
 * it asks for a guard-rail in the same breath, and it says why in the plainest terms available —
 * its own worked example is a predicate that went from 8 copies to 36 *after* being centralised,
 * because nothing stopped the ninth from being written by hand.
 *
 * So this is the thing that stops the ninth. The literal may appear in `controls.tsx` and
 * nowhere else in this app's own code; `features/replay/` is exempt because it is copied
 * verbatim from `apps/web` and cannot be edited from here (`copies.guard.test.ts` is what holds
 * those files).
 *
 * A NEW SIZE IS NOT A VIOLATION — it is a new control. Add it to `controls.tsx` and let this
 * guard cover it too.
 */
import { readdirSync, readFileSync, statSync } from 'node:fs'
import { dirname, join, relative, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const HERE = dirname(fileURLToPath(import.meta.url))
const STUDY_SRC = resolve(HERE, '../..')
const OWNER = 'components/ui/controls.tsx'
/** This file names the literal in order to look for it; it is not a copy of it. */
const SELF = 'components/ui/controls.guard.test.ts'

/** The size prefix every compact control shares. Matching on it catches the mono variant too. */
const COMPACT = 'h-7 px-2'

function sourceFiles(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const path = join(dir, name)
    if (statSync(path).isDirectory()) return name === 'replay' ? [] : sourceFiles(path)
    return /\.tsx?$/.test(name) ? [path] : []
  })
}

describe('the compact control size', () => {
  const files = sourceFiles(STUDY_SRC).map((path) => ({
    rel: relative(STUDY_SRC, path).replace(/\\/g, '/'),
    text: readFileSync(path, 'utf8'),
  }))

  it('scans something — an empty sweep would pass for the wrong reason', () => {
    expect(files.length).toBeGreaterThan(10)
  })

  it('is declared by the shared control, so there is something to point at', () => {
    const owner = files.find((f) => f.rel === OWNER)
    expect(owner, `${OWNER} is missing`).toBeDefined()
    expect(owner!.text).toContain(COMPACT)
  })

  it('is written by hand nowhere else', () => {
    const offenders = files
      .filter((f) => f.rel !== OWNER && f.rel !== SELF && f.text.includes(COMPACT))
      .map((f) => f.rel)
    expect(offenders, `use <Toggle> or <CompactAction> from ${OWNER}`).toEqual([])
  })
})
