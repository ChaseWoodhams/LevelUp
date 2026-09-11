/**
 * matchClock.test.ts — le signe, le zéro, et l'avant-match.
 *
 * Les cas sont ceux qui se lisent mal à l'écran : un décalage de signe donne une horloge
 * plausible et fausse, et c'est exactement ce qu'un test attrape mieux qu'un coup d'oeil.
 */
import { describe, expect, it } from 'vitest'

import { formatMatchClock, matchTimeMs } from './matchClock'

describe('matchTimeMs', () => {
  it("décale l'axe du document vers l'horloge du match", () => {
    // Mesure réelle de `36e80b83` : le zéro du match tombe 5 692 ms AVANT la frame 0.
    const zero = -5692
    expect(matchTimeMs(0, zero)).toBe(5692)
    expect(matchTimeMs(21_000, zero)).toBe(26_692) // l'ouverture observée, ~26,7 s de match
  })

  it('rend un instant NÉGATIF avant le zéro du match', () => {
    // Un artefact dont le film commence AVANT la partie : l'avant-match est un temps négatif,
    // pas un zéro.
    expect(matchTimeMs(0, 3_000)).toBe(-3_000)
  })
})

describe('formatMatchClock', () => {
  it('rend m:ss', () => {
    expect(formatMatchClock(0)).toBe('0:00')
    expect(formatMatchClock(5_692)).toBe('0:05')
    expect(formatMatchClock(26_692)).toBe('0:26')
    expect(formatMatchClock(743_400)).toBe('12:23')
  })

  it("porte le signe de l'avant-match plutôt que de l'écrêter", () => {
    // Ecrêter à 0:00 ferait passer l'avant-match pour le coup d'envoi.
    expect(formatMatchClock(-3_000)).toBe('-0:03')
    expect(formatMatchClock(-75_000)).toBe('-1:15')
  })

  it('TRONQUE les secondes, ne les arrondit pas', () => {
    // Arrondir afficherait une seconde que la lecture n'a pas encore atteinte.
    expect(formatMatchClock(1_999)).toBe('0:01')
    expect(formatMatchClock(59_999)).toBe('0:59')
  })
})
