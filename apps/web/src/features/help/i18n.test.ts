import { describe, it, expect } from 'vitest'

import { DEFAULT_EFFECTIVE_HP_TO_KILL, getHelpText } from './i18n'

/** Concatène tout le texte combat d'un glossaire pour des assertions globales. */
function flatten(hp?: number): string {
  const text = hp === undefined ? getHelpText() : getHelpText(hp)
  return text.glossary.sections
    .flatMap((s) => s.entries)
    .flatMap((e) => [e.term, e.definition, e.formula ?? '', e.example ?? ''])
    .join('\n')
}

describe('getHelpText — copy combat title-aware', () => {
  it('never leaks the {{HP}} token (default and 115)', () => {
    expect(flatten()).not.toContain('{{HP}}')
    expect(flatten(115)).not.toContain('{{HP}}')
  })

  it('applies the Halo Infinite baseline (225) by default', () => {
    expect(DEFAULT_EFFECTIVE_HP_TO_KILL).toBe(225)
    const text = flatten()
    expect(text).toContain('Offensive conversion = 225 × (kills + assists/3) / damage_dealt')
    expect(text).toContain('Defensive resistance = damage_taken / (225 × deaths)')
  })

  it('injects the current title baseline into conversion, resistance, impact and survival (115)', () => {
    const text = flatten(115)
    expect(text).toContain('Offensive conversion = 115 × (kills + assists/3) / damage_dealt')
    expect(text).toContain('Defensive resistance = damage_taken / (115 × deaths)')
    expect(text).toContain('Impact = 115 × (kills + assists/3) / damage dealt')
    expect(text).toContain('Survival = damage taken / (115 × deaths)')
  })

  it('keeps the Halo Infinite calibrated elite references whatever the baseline', () => {
    // The gauge references (0.90 / 1.65) are not tokenised: per-title recalibration deferred.
    expect(flatten(115)).toContain('Elite reference (gauge): 0.90')
    expect(flatten(115)).toContain('Elite reference (gauge): 1.65')
  })
})
