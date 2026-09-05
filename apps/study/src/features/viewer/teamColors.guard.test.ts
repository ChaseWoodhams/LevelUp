/// <reference types="node" />
// @vitest-environment node
/**
 * teamColors.guard.test.ts — THE MAP AND THE PANEL READ THE SAME LIST OF TOKENS.
 *
 * The team colours exist twice in this app, and they have to: the roster panel's list is a
 * private function inside `ReplayTeams.tsx`, a file copied verbatim from `apps/web` and
 * therefore not editable from here, so there is no shared constant to point both at today.
 * Two copies is what the repository allows; what it asks for at the second is that they cannot
 * drift apart in silence, because the failure would be soundless — the map painting a team one
 * colour and its cards another, with nothing red anywhere.
 *
 * So the list is READ back out of the copied component and compared. If the origin ever adds a
 * fourth comparison token, this goes red and names the file. A THIRD copy owes a shared helper
 * instead of a third guard.
 */
import { readFileSync } from 'node:fs'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

import { TEAM_TOKENS } from './teamColors'

const HERE = dirname(fileURLToPath(import.meta.url))
const REPLAY_TEAMS = resolve(HERE, '../replay/ReplayTeams.tsx')

/** The panel's own list, as its `teamColor()` writes it: `const tokens = [...] as const`. */
const TOKENS_RE = /const tokens = \[([^\]]+)\] as const/

describe('the team colour tokens', () => {
  it('are the ones the roster panel paints its headings with', () => {
    const source = readFileSync(REPLAY_TEAMS, 'utf8')
    const declared = TOKENS_RE.exec(source)
    expect(declared, `no token list found in ${join(REPLAY_TEAMS)}`).not.toBeNull()

    const panelTokens = declared![1]
      .split(',')
      .map((t) => t.trim().replace(/^'|'$/g, ''))
      .filter((t) => t !== '')

    expect(panelTokens).toEqual([...TEAM_TOKENS])
  })
})
