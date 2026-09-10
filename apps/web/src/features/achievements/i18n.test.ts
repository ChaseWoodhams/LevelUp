/**
 * Tests i18n.ts — pickLocalized and formatUnlockedDate.
 *
 * The UI is English-only: pickLocalized returns the English field, and an empty string
 * when the Xbox API left it empty or missing.
 */
import { describe, expect, it } from 'vitest'
import { formatUnlockedDate, pickLocalized } from './i18n'

describe('pickLocalized', () => {
  it('returns the English value', () => {
    expect(pickLocalized('Hello')).toBe('Hello')
  })

  it('empty or undefined → empty string', () => {
    expect(pickLocalized('')).toBe('')
    expect(pickLocalized(undefined)).toBe('')
  })
})

describe('formatUnlockedDate', () => {
  it('retourne null si iso absent', () => {
    expect(formatUnlockedDate(undefined)).toBeNull()
  })

  it('retourne null si iso invalide', () => {
    expect(formatUnlockedDate('not-a-date')).toBeNull()
  })

  it('formats a short date', () => {
    const out = formatUnlockedDate('2026-04-15T10:00:00Z')
    expect(out).toBeTruthy()
    expect(out).toMatch(/2026/)
  })
})
