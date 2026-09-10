/**
 * i18n.test.ts — buildContextLabel : produit un label localisé depuis un
 * MatchFilterSpec pour la barre de navigation contextuelle (Phase 2b).
 */
import { describe, it, expect } from 'vitest'

import { buildContextLabel } from './i18n'
import type { MatchFilterSpec } from '@/lib/match-nav/navContext'

describe('buildContextLabel', () => {
  it('spec null : chaîne vide', () => {
    expect(buildContextLabel(null, 'en')).toBe('')
    expect(buildContextLabel(undefined, 'en')).toBe('')
  })

  it('playlist seule', () => {
    expect(
      buildContextLabel({ playlist_names: ['Classée Arena'] }, 'en'),
    ).toBe('Classée Arena')
  })

  it('playlist + mode', () => {
    expect(
      buildContextLabel(
        { playlist_names: ['Classée Arena'], mode_categories: ['Ranked'] },
        'en',
      ),
    ).toBe('Classée Arena · Ranked')
  })

  it('multi-playlists : jointes par virgule dans le label (Phase 3)', () => {
    expect(
      buildContextLabel({ playlist_names: ['Classée Arena', 'Grande bataille'] }, 'en'),
    ).toBe('Classée Arena, Grande bataille')
  })

  it('outcome FR vs EN', () => {
    const spec: MatchFilterSpec = { outcome: 'win' }
    expect(buildContextLabel(spec, 'en')).toBe('Wins')
    expect(buildContextLabel(spec, 'en')).toBe('Wins')
  })

  it('range complet de dates', () => {
    const spec: MatchFilterSpec = {
      date_from: '2026-04-01T00:00:00Z',
      date_to: '2026-05-01T00:00:00Z',
    }
    const got = buildContextLabel(spec, 'en')
    expect(got).toContain('→')
    expect(got).toContain('04')
    expect(got).toContain('05')
  })

  it('seulement date_from : "Depuis JJ/MM/YYYY"', () => {
    const spec: MatchFilterSpec = { date_from: '2026-04-01T00:00:00Z' }
    expect(buildContextLabel(spec, 'en')).toMatch(/^Depuis /)
    expect(buildContextLabel(spec, 'en')).toMatch(/^From /)
  })

  it('seulement date_to : "Jusqu\'au JJ/MM/YYYY"', () => {
    const spec: MatchFilterSpec = { date_to: '2026-05-01T00:00:00Z' }
    expect(buildContextLabel(spec, 'en')).toMatch(/^Jusqu'au /)
    expect(buildContextLabel(spec, 'en')).toMatch(/^To /)
  })

  it('combinaison complète FR', () => {
    const spec: MatchFilterSpec = {
      playlist_names: ['Classée'],
      mode_categories: ['BTB'],
      outcome: 'loss',
      date_from: '2026-04-01T00:00:00Z',
    }
    const got = buildContextLabel(spec, 'en')
    expect(got).toContain('Classée')
    expect(got).toContain('BTB')
    expect(got).toContain('Losses')
    expect(got).toContain('Depuis')
    expect(got.split(' · ')).toHaveLength(4)
  })

  it('session_id : préfixé par #', () => {
    expect(
      buildContextLabel({ session_id: 'sess-2026-04-30' }, 'en'),
    ).toBe('#sess-2026-04-30')
  })
})
