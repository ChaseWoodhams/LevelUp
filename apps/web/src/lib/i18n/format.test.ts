import { beforeEach, describe, expect, it } from 'vitest'

import { commonManifest } from './generated/common'
import { formatMessage, resetFormatterCache } from './format'

describe('formatMessage', () => {
  beforeEach(() => {
    resetFormatterCache()
  })

  it('resolves a simple English message', () => {
    expect(formatMessage(commonManifest, 'common.period.last_1y', 'en')).toBe('Last year')
  })

  it('applies ICU pluralization', () => {
    expect(formatMessage(commonManifest, 'common.kpi.matches_count', 'en', { n: 1 })).toBe('1 match')
    expect(formatMessage(commonManifest, 'common.kpi.matches_count', 'en', { n: 12 })).toBe('12 matches')
  })

  it('returns the key for a missing manifest entry', () => {
    // @ts-expect-error: intentionally exercise an unknown key.
    expect(formatMessage(commonManifest, 'unknown.key', 'en')).toBe('unknown.key')
  })

  it('memoizes formatters', () => {
    const start = performance.now()
    for (let i = 0; i < 1000; i++) {
      formatMessage(commonManifest, 'common.kpi.matches_count', 'en', { n: i })
    }
    expect(performance.now() - start).toBeLessThan(500)
  })

  it('uses the fast path for messages without variables', () => {
    expect(formatMessage(commonManifest, 'common.outcome.win', 'en')).toBe('Win')
  })

  it('returns the key when the English value is empty', () => {
    const partialManifest = {
      'test.partial': { en: '' },
    } as const
    expect(formatMessage(partialManifest, 'test.partial', 'en')).toBe('test.partial')
  })
})

describe('commonManifest integrity', () => {
  it('has a non-empty English value for every key', () => {
    for (const [key, entry] of Object.entries(commonManifest)) {
      expect(entry.en, `key "${key}" has no English value`).toBeTruthy()
    }
  })

  it('contains at least one message', () => {
    expect(Object.keys(commonManifest).length).toBeGreaterThan(0)
  })
})
