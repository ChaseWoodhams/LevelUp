/// <reference types="node" />
// @vitest-environment node
/**
 * normalized.guard.test.ts — THE FRONTIER IS CROSSED ONCE, AND THE VIEWER DOES NOT RE-CHECK.
 *
 * `normalizeReplayDocument` exists so that "no trail" and "the field is null" stop being two
 * different things everywhere downstream. Its own header names the failure it was built to end:
 * without a single crossing, the difference is paid for in `?.` and `?? []` scattered through
 * every caller, "and one is always missing".
 *
 * A `?.` on an array the frontier has already filled is worse than redundant. It is a claim —
 * that this array might be null here — and the claim is false. The next reader either believes
 * it and adds another, or checks and wastes the trip. Both outcomes erode the guarantee the
 * frontier was built to give, which is exactly how the guards this repository has learnt to
 * write get earned: the property was asserted once in a ticket, and then asserted here so it
 * cannot quietly stop being true.
 *
 * WHAT THIS DOES NOT FORBID: optional chaining on fields the frontier LEAVES optional —
 * `weaponLabels`, `abilityLabels`, `geometryBounds` and the rest. Those really can be absent,
 * and the list below is read from the normaliser itself rather than typed out here, so a field
 * that joins or leaves the frontier moves this guard with it.
 */
import { readFileSync } from 'node:fs'
import { dirname, join, relative, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { readdirSync, statSync } from 'node:fs'

import { describe, expect, it } from 'vitest'

const HERE = dirname(fileURLToPath(import.meta.url))
const STUDY_SRC = resolve(HERE, '../..')
const NORMALIZE = join(STUDY_SRC, 'features/replay/replayNormalize.ts')

/**
 * The fields `normalizeReplayDocument` fills, read out of its return statement — the lines of
 * the form `    geometry: raw.geometry ?? [],`.
 */
function filledFields(): string[] {
  const source = readFileSync(NORMALIZE, 'utf8')
  const body = source.slice(source.indexOf('export function normalizeReplayDocument'))
  return [...new Set([...body.matchAll(/^\s{4}(\w+): \(?raw\./gm)].map((m) => m[1]))]
}

/**
 * The code this guard covers: the app's OWN modules. `features/replay/` is excluded because it
 * is copied verbatim from `apps/web` and cannot be edited from here — `copies.guard.test.ts`
 * is what holds those files, and a second guard contradicting it would only be noise.
 */
function ownFiles(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const path = join(dir, name)
    if (statSync(path).isDirectory()) return name === 'replay' ? [] : ownFiles(path)
    return /\.tsx?$/.test(name) && !/\.test\.tsx?$/.test(name) ? [path] : []
  })
}

describe('the normalised document', () => {
  const fields = filledFields()

  it('has fields to guard at all — an empty list would make the sweep vacuous', () => {
    expect(fields.length).toBeGreaterThan(5)
    expect(fields).toContain('tracks')
    expect(fields).toContain('shots')
  })

  it('is never re-checked for null in this app own code', () => {
    const offenders: string[] = []
    for (const path of ownFiles(STUDY_SRC)) {
      const text = readFileSync(path, 'utf8')
      text.split('\n').forEach((line, i) => {
        if (/^\s*(\/\/|\/?\*)/.test(line)) return
        for (const field of fields) {
          if (line.includes(`.${field}?.`) || line.includes(`.${field} ??`)) {
            offenders.push(`${relative(STUDY_SRC, path).replace(/\\/g, '/')}:${i + 1} — ${line.trim()}`)
          }
        }
      })
    }
    expect(
      offenders,
      'normalizeReplayDocument already filled these arrays; a null check here is a false claim',
    ).toEqual([])
  })
})
