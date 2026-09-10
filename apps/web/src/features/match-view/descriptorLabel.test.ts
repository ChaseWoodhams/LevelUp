/**
 * Tests unitaires — buildDescriptorLabel.
 *
 * Covers every variant of the `ContextDescriptor` discriminated union, graceful
 * degradation (missing gamertag/startTimeUtc → ''), and the Intl date formatting for
 * `session` / `period`.
 *
 * Note: Intl.DateTimeFormat output depends on the runner's time zone; the assertions
 * use regexes to stay tolerant of it.
 */
import { describe, it, expect } from 'vitest'

import { buildDescriptorLabel } from './descriptorLabel'
import type { ContextDescriptor } from '@/lib/match-nav/navContext'

describe('buildDescriptorLabel', () => {
  it('null/undefined → chaîne vide', () => {
    expect(buildDescriptorLabel(null, 'en')).toBe('')
    expect(buildDescriptorLabel(undefined, 'en')).toBe('')
  })

  describe('kind = recent', () => {
    const d: ContextDescriptor = { kind: 'recent' }
    it('"recent"', () => expect(buildDescriptorLabel(d, 'en')).toBe('recent'))
  })

  describe('kind = favorites', () => {
    const d: ContextDescriptor = { kind: 'favorites' }
    it('"favorites"', () => expect(buildDescriptorLabel(d, 'en')).toBe('favorites'))
  })

  describe('kind = media', () => {
    const d: ContextDescriptor = { kind: 'media' }
    it('"with media"', () => expect(buildDescriptorLabel(d, 'en')).toBe('with media'))
  })

  describe('kind = top_matches', () => {
    const d: ContextDescriptor = { kind: 'top_matches' }
    it('"top performances"', () => expect(buildDescriptorLabel(d, 'en')).toBe('top performances'))
  })

  describe('kind = with_player', () => {
    it('with a gamertag: "with X"', () => {
      expect(
        buildDescriptorLabel({ kind: 'with_player', gamertag: 'CoolMate' }, 'en'),
      ).toBe('with CoolMate')
    })
    it('gamertag vide → chaîne vide (dégradation)', () => {
      expect(buildDescriptorLabel({ kind: 'with_player', gamertag: '' }, 'en')).toBe('')
    })
  })

  describe('kind = session', () => {
    it('with startTimeUtc: "from session of" + date and time', () => {
      const got = buildDescriptorLabel(
        { kind: 'session', startTimeUtc: '2026-05-07T21:30:00Z' },
        'en',
      )
      expect(got).toMatch(/^from session of \d{2}\/\d{2}\/\d{2} at \d{2}:\d{2}\s?(AM|PM)$/)
    })
    it('startTimeUtc absent → chaîne vide', () => {
      expect(buildDescriptorLabel({ kind: 'session', startTimeUtc: '' }, 'en')).toBe('')
    })
    it('startTimeUtc invalide → fallback ISO brut dans le label', () => {
      const got = buildDescriptorLabel(
        { kind: 'session', startTimeUtc: 'pas-une-date' },
        'en',
      )
      // fmtShortDateTime returns the raw ISO string when parsing fails
      expect(got).toBe('from session of pas-une-date')
    })
  })

  describe('kind = period', () => {
    it('from + to: "from period <from> to <to>"', () => {
      const got = buildDescriptorLabel(
        { kind: 'period', from: '2026-04-01T00:00:00Z', to: '2026-05-01T00:00:00Z' },
        'en',
      )
      expect(got).toMatch(/^from period \d{2}\/\d{2}\/\d{2} to \d{2}\/\d{2}\/\d{2}$/)
    })
    it('from only: "since <from>"', () => {
      const got = buildDescriptorLabel({ kind: 'period', from: '2026-04-01T00:00:00Z' }, 'en')
      expect(got).toMatch(/^since \d{2}\/\d{2}\/\d{2}$/)
    })
    it('to only: "until <to>"', () => {
      const got = buildDescriptorLabel({ kind: 'period', to: '2026-05-01T00:00:00Z' }, 'en')
      expect(got).toMatch(/^until \d{2}\/\d{2}\/\d{2}$/)
    })
    it('ni from ni to → chaîne vide', () => {
      expect(buildDescriptorLabel({ kind: 'period' }, 'en')).toBe('')
    })
  })

  describe('kind = playlist', () => {
    it('"in <name>"', () => {
      expect(
        buildDescriptorLabel({ kind: 'playlist', name: 'Ranked Arena' }, 'en'),
      ).toBe('in Ranked Arena')
    })
    it('name vide → chaîne vide', () => {
      expect(buildDescriptorLabel({ kind: 'playlist', name: '' }, 'en')).toBe('')
    })
  })

  describe('kind = mode', () => {
    it('"in <category>"', () => {
      expect(buildDescriptorLabel({ kind: 'mode', category: 'BTB' }, 'en')).toBe('in BTB')
    })
    it('category vide → chaîne vide', () => {
      expect(buildDescriptorLabel({ kind: 'mode', category: '' }, 'en')).toBe('')
    })
  })
})
