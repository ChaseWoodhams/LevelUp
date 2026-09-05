/// <reference types="node" />
// @vitest-environment node
/**
 * schemaVersion.guard.test.ts — THE SUPPORTED VERSION IS THE PRODUCER'S, NOT A NUMBER
 * SOMEBODY TYPED.
 *
 * The viewer refuses to draw an artifact whose `schemaVersion` it does not recognise. That
 * refusal is only worth anything while the recognised value is the one the BUILDER actually
 * writes — and the builder is a Go constant in another language, in another app, with no
 * compiler between it and this file. A number copied out of a spec would be right on the day
 * it was copied and silently wrong afterwards, which is exactly the failure the guard exists
 * to prevent: this viewer would then refuse every artifact, or worse, accept one it cannot
 * read.
 *
 * So the value is READ from the Go source. When the builder bumps its version, this test goes
 * red and names the file to look at — the change then has to be a decision (does the viewer
 * handle the new shape?), not a silent drift.
 */
import { readFileSync } from 'node:fs'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

import { SUPPORTED_SCHEMA_VERSION } from './schemaVersion'

const REPO_ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '../../../../..')
const DOCUMENT_GO = join(REPO_ROOT, 'apps/go-api/internal/analysis/replay/document.go')

describe('the supported replay schema version', () => {
  it('is the one the artifact builder writes', () => {
    const source = readFileSync(DOCUMENT_GO, 'utf8')
    const declared = /^const SchemaVersion = (\d+)$/m.exec(source)
    expect(declared, `no 'const SchemaVersion' in ${DOCUMENT_GO}`).not.toBeNull()
    expect(Number(declared![1])).toBe(SUPPORTED_SCHEMA_VERSION)
  })
})
