/**
 * Tests unitaires — buildAscensionTips (tips de jeu, source coachingTipsManifest).
 */
import { describe, it, expect } from 'vitest'
import { buildAscensionTips } from './tips'

const CATEGORIES = new Set(['Combat', 'Impact', 'Objective', 'Score', 'Support', 'Survival'])

describe('buildAscensionTips', () => {
  it('returns game tips bounded by MAX_TIPS', () => {
    const tips = buildAscensionTips()
    expect(tips.length).toBeGreaterThan(0)
    expect(tips.length).toBeLessThanOrEqual(14)
  })

  it('uses a coaching category as the term', () => {
    for (const tip of buildAscensionTips()) {
      expect(CATEGORIES.has(tip.term)).toBe(true)
    }
  })

  it('every tip carries non-empty advice and a coaching id, no glossary link', () => {
    for (const tip of buildAscensionTips()) {
      expect(tip.id).toMatch(/^coaching_tips\./)
      expect(tip.shortDef.length).toBeGreaterThan(0)
      expect(tip.href).toBeUndefined()
    }
  })

  it('collapses internal whitespace and newlines in the advice', () => {
    for (const tip of buildAscensionTips()) {
      expect(tip.shortDef).not.toMatch(/\s{2,}/)
      expect(tip.shortDef).not.toMatch(/\n/)
    }
  })

  it('excludes category meta-keys (title, related_signals)', () => {
    for (const tip of buildAscensionTips()) {
      expect(tip.id).not.toMatch(/\.title$/)
      expect(tip.id).not.toMatch(/\.related_signals$/)
    }
  })

  it('does not produce duplicate ids', () => {
    const tips = buildAscensionTips()
    const ids = new Set(tips.map((t) => t.id))
    expect(ids.size).toBe(tips.length)
  })
})
