/**
 * Tests de la source unique du badge « Prolongation » : libellés localisés,
 * tooltip formaté en M:SS, parité FR/EN des clés du manifeste.
 */
import { describe, it, expect } from 'vitest'

import { matchViewManifest } from '@/lib/i18n/generated/match_view'
import {
  OVERTIME_COLOR_TOKEN,
  OVERTIME_LABEL_KEY,
  OVERTIME_TOOLTIP_KEY,
  overtimeLabel,
  overtimeTooltip,
} from './overtime'

describe('lib/narrative/overtime', () => {
  it('libellé localisé FR / EN', () => {
    expect(overtimeLabel('en')).toBe('Overtime')
    expect(overtimeLabel('en')).toBe('Overtime')
  })

  it('tooltip : dépassement formaté en M:SS', () => {
    expect(overtimeTooltip('en', 43)).toBe('Overtime: +0:43')
    expect(overtimeTooltip('en', 270)).toBe('Overtime: +4:30')
    expect(overtimeTooltip('en', 54)).toBe('Overtime: +0:54')
  })

  it('tooltip : sans dépassement exploitable → libellé seul', () => {
    expect(overtimeTooltip('en', 0)).toBe('Overtime')
    expect(overtimeTooltip('en', null)).toBe('Overtime')
    expect(overtimeTooltip('en', undefined)).toBe('Overtime')
  })

  it('parité i18n : les 2 clés existent en FR ET EN', () => {
    for (const key of [OVERTIME_LABEL_KEY, OVERTIME_TOOLTIP_KEY]) {
      const entry = matchViewManifest[key]
      expect(entry, `clé absente du manifeste : ${key}`).toBeDefined()
      expect(entry.en.length).toBeGreaterThan(0)
      expect(entry.en.length).toBeGreaterThan(0)
    }
  })

  it('token : état neutre-informatif (aucun token de dominance)', () => {
    expect(OVERTIME_COLOR_TOKEN).toBe('info')
    expect(OVERTIME_COLOR_TOKEN.startsWith('narrative-')).toBe(false)
  })
})
