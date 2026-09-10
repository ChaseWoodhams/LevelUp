import { describe, it, expect } from 'vitest'
import { offensiveLabel, defensiveLabel, activityLabel } from './combatProfileLabels'

describe('combatProfileLabels — locale-aware', () => {
  it('offensiveLabel rend FR et EN', () => {
    expect(offensiveLabel('disperse', 'en')).toBe('Scattered')
    expect(offensiveLabel('disperse', 'en')).toBe('Scattered')
    expect(offensiveLabel('chirurgical', 'en')).toBe('Chirurgical')
    expect(offensiveLabel('chirurgical', 'en')).toBe('Surgical')
  })

  it('defensiveLabel rend FR et EN', () => {
    expect(defensiveLabel('inebranlable', 'en')).toBe('Unshakable')
    expect(defensiveLabel('inebranlable', 'en')).toBe('Unshakable')
    expect(defensiveLabel('fragile', 'en')).toBe('Fragile')
  })

  it('activityLabel rend FR et EN', () => {
    expect(activityLabel('agressif', 'en')).toBe('Agressif')
    expect(activityLabel('agressif', 'en')).toBe('Aggressive')
    expect(activityLabel('passif', 'en')).toBe('Passive')
  })

  it('null/undefined → null', () => {
    expect(offensiveLabel(null, 'en')).toBeNull()
    expect(defensiveLabel(undefined, 'en')).toBeNull()
    expect(activityLabel(null, 'en')).toBeNull()
  })
})
