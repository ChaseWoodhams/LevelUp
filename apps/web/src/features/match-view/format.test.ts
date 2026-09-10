import { describe, it, expect } from 'vitest'
import { buildMatchHeadingStr } from './format'

// Les tests unitaires de normalizeModeLabel vivent désormais avec le helper
// partagé : src/lib/halo/modeLabel.test.ts.

describe('buildMatchHeadingStr', () => {
  it('fr : assemble "Mode sur Carte" quand les deux sont présents', () => {
    expect(buildMatchHeadingStr('Forbidden', 'Capture the Flag')).toBe(
      'Capture the Flag on Forbidden',
    )
  })

  it('en : assemble "Mode on Carte"', () => {
    expect(buildMatchHeadingStr('Forbidden', 'CTF')).toBe('CTF on Forbidden')
  })

  it('normalise le mode avant assemblage (strip "Arena:")', () => {
    expect(buildMatchHeadingStr('Forbidden', 'Arena:CTF on Forbidden')).toBe(
      'CTF on Forbidden',
    )
  })

  it('strip le nom de carte EN collé même si mapUI est FR (régression "Slayer on Forest sur Forêt")', () => {
    expect(buildMatchHeadingStr('Forêt', 'Slayer on Forest')).toBe('Slayer on Forêt')
  })

  it('retourne juste la carte si le mode est null — cas dégradé "Forbidden"', () => {
    expect(buildMatchHeadingStr('Forbidden', null)).toBe('Forbidden')
    expect(buildMatchHeadingStr('Forbidden', undefined)).toBe('Forbidden')
    expect(buildMatchHeadingStr('Forbidden', '')).toBe('Forbidden')
  })

  it('retourne juste le mode si la carte est null', () => {
    expect(buildMatchHeadingStr(null, 'Assassin')).toBe('Assassin')
  })

  it('retourne chaîne vide si les deux sont null', () => {
    expect(buildMatchHeadingStr(null, null)).toBe('')
  })

  it('mode EN non traduit + carte → "CTF sur Forbidden" (EN leak documenté)', () => {
    expect(buildMatchHeadingStr('Forbidden', 'CTF')).toBe('CTF on Forbidden')
  })
})
