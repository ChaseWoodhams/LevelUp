/**
 * route.test.ts — the two screens, and the identifiers that reach them intact.
 */
import { describe, expect, it } from 'vitest'

import { homeHref, matchHref, parseRoute, sampleHref } from './route'

describe('parseRoute', () => {
  it('opens a match by the identifier in the hash', () => {
    expect(parseRoute('#/match/000d5950')).toEqual({ kind: 'match', matchId: '000d5950' })
  })

  it('accepts the full identifier as well as the short one — the server resolves either', () => {
    const full = '00000000-0000-4000-8000-0000000f1x7e'
    expect(parseRoute(`#/match/${full}`)).toEqual({ kind: 'match', matchId: full })
  })

  it('unescapes what matchHref escaped — the two are one round trip', () => {
    const id = 'a b/c'
    expect(parseRoute(matchHref(id))).toEqual({ kind: 'match', matchId: id })
  })

  it('reads a malformed escape as literal text instead of throwing', () => {
    // A hand-edited URL produces this without trying, and an exception here would take the
    // whole screen down over a typo.
    expect(parseRoute('#/match/100%')).toEqual({ kind: 'match', matchId: '100%' })
  })

  it('lands on home for an empty identifier, an unknown hash, or no hash at all', () => {
    expect(parseRoute('#/match/')).toEqual({ kind: 'home' })
    expect(parseRoute('#/nowhere')).toEqual({ kind: 'home' })
    expect(parseRoute('')).toEqual({ kind: 'home' })
    expect(parseRoute(homeHref())).toEqual({ kind: 'home' })
  })

  it('opens the sample artifact on its own hash', () => {
    expect(parseRoute(sampleHref())).toEqual({ kind: 'sample' })
  })
})
