/**
 * fade.test.ts — a colour dimmed is still the same colour.
 */
import { describe, expect, it } from 'vitest'

import { fadeColor } from './fade'

/** The palette writes six-digit hex, which is what `resolveToken` hands back. */
const TEAM = '#818CF8'

describe('fadeColor', () => {
  it('keeps the channels and lowers the opacity', () => {
    expect(fadeColor(TEAM, 0.2)).toBe('rgba(129, 140, 248, 0.2)')
  })

  it('reads the short hex form as the long one', () => {
    expect(fadeColor('#8Cf', 0.5)).toBe('rgba(136, 204, 255, 0.5)')
  })

  it('multiplies an alpha the colour already carried instead of overwriting it', () => {
    // Overwriting would make a dimmed marker MORE opaque than the one it was cut from.
    expect(fadeColor('#8888887F', 0.5)).toBe('rgba(136, 136, 136, 0.249)')
    expect(fadeColor('rgba(10, 20, 30, 0.4)', 0.5)).toBe('rgba(10, 20, 30, 0.2)')
  })

  it('reads the functional notation in either syntax', () => {
    expect(fadeColor('rgb(1, 2, 3)', 1)).toBe('rgba(1, 2, 3, 1)')
    expect(fadeColor('rgb(1 2 3 / 0.5)', 1)).toBe('rgba(1, 2, 3, 0.5)')
  })

  it('clamps an alpha outside 0..1', () => {
    expect(fadeColor(TEAM, 4)).toBe('rgba(129, 140, 248, 1)')
    expect(fadeColor(TEAM, -1)).toBe('rgba(129, 140, 248, 0)')
  })

  it('hands back untouched what it cannot read, rather than guessing', () => {
    // A percentage channel read as a number would silently produce a colour nobody chose.
    expect(fadeColor('rgb(50% 0% 0%)', 0.5)).toBe('rgb(50% 0% 0%)')
    expect(fadeColor('oklch(0.7 0.1 250)', 0.5)).toBe('oklch(0.7 0.1 250)')
    expect(fadeColor('', 0.5)).toBe('')
  })
})
