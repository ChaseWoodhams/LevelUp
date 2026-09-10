/**
 * Tests — skillTiers : invariants des grilles LUSR/CSR + sélection gridForRatingTypes.
 * Garde-fou contre une dérive de config (bornes non contiguës, sous-paliers manquants)
 * et contre une régression de la logique de choix de grille (CSR vs LUSR vs mixte).
 */
import { describe, it, expect } from 'vitest'
import {
  LUSR_TIER_GRID,
  CSR_TIER_GRID,
  gridForRatingTypes,
  subTierPosition,
  localizeTierName,
  localizeTierLabel,
  skillTierSortValue,
  composeTierLabel,
} from './skillTiers'

describe('grilles de paliers', () => {
  for (const [name, grid] of [['LUSR', LUSR_TIER_GRID], ['CSR', CSR_TIER_GRID]] as const) {
    it(`${name} : tiers triés, contigus, sous-paliers ≥ 1`, () => {
      const t = grid.tiers
      for (let i = 0; i < t.length; i++) {
        expect(t[i].max).toBeGreaterThan(t[i].min)
        expect(t[i].subTiers).toBeGreaterThanOrEqual(1)
        if (i + 1 < t.length) expect(t[i].max).toBe(t[i + 1].min) // contiguïté
      }
    })

    it(`${name} : palier sommet ouvert (Onyx, sans sous-palier)`, () => {
      const top = grid.tiers[grid.tiers.length - 1]
      expect(top.max).toBeGreaterThanOrEqual(9000)
      expect(top.subTiers).toBe(1)
      expect(top.en).toBe('Onyx')
    })
  }
})

describe('gridForRatingTypes', () => {
  it('tous CSR → grille CSR', () => {
    expect(gridForRatingTypes(['CSR', 'CSR'])).toBe(CSR_TIER_GRID)
  })
  it('insensible à la casse (csr)', () => {
    expect(gridForRatingTypes(['csr'])).toBe(CSR_TIER_GRID)
  })
  it('tous LUSR → grille LUSR', () => {
    expect(gridForRatingTypes(['LUSR'])).toBe(LUSR_TIER_GRID)
  })
  it('mixte LUSR+CSR → grille LUSR (non-régression, échelle legacy)', () => {
    expect(gridForRatingTypes(['LUSR', 'CSR'])).toBe(LUSR_TIER_GRID)
  })
  it('vide ou null/undefined → grille LUSR (défaut)', () => {
    expect(gridForRatingTypes([])).toBe(LUSR_TIER_GRID)
    expect(gridForRatingTypes([null, undefined])).toBe(LUSR_TIER_GRID)
  })
})

describe('subTierPosition', () => {
  it('CSR : sous-paliers de 50 pts (Diamant)', () => {
    // Diamond CSR [1200,1500], 6 sub-tiers of 50. 1452 → sub-tier [1450,1500].
    const p = subTierPosition(CSR_TIER_GRID, 1452)
    expect(p).not.toBeNull()
    expect(p!.subTierMin).toBe(1450)
    expect(p!.subTierWidth).toBe(50)
    expect(p!.pct).toBeCloseTo(0.04, 5)
  })

  it('LUSR : largeur de sous-palier variable selon le tier (Platine = 100)', () => {
    // Platinum LUSR [1600,1800], 2 sub-tiers of 100. 1770 → [1700,1800].
    const p = subTierPosition(LUSR_TIER_GRID, 1770)
    expect(p).not.toBeNull()
    expect(p!.subTierMin).toBe(1700)
    expect(p!.subTierWidth).toBe(100)
    expect(p!.pct).toBeCloseTo(0.7, 5)
  })

  it('LUSR : Or = sous-paliers de 33.3 pts (≠ 50)', () => {
    // Gold LUSR [1400,1600], 6 sub-tiers ≈ 33.33. 1452 → [1433.3,1466.7].
    const p = subTierPosition(LUSR_TIER_GRID, 1452)
    expect(p).not.toBeNull()
    expect(p!.subTierWidth).toBeCloseTo(33.333, 2)
    expect(p!.subTierMin).toBeCloseTo(1433.333, 2)
  })

  it('palier ouvert (Onyx) → null', () => {
    expect(subTierPosition(CSR_TIER_GRID, 1600)).toBeNull()
    expect(subTierPosition(LUSR_TIER_GRID, 2100)).toBeNull()
  })

  it('hors grille (sous le plancher) → null', () => {
    expect(subTierPosition(LUSR_TIER_GRID, 500)).toBeNull()
  })
})

describe('localizeTierName', () => {
  it('English names stay as they are', () => {
    expect(localizeTierName('Gold', 'en')).toBe('Gold')
    expect(localizeTierName('Platinum', 'en')).toBe('Platinum')
  })
  it('legacy French names resolve to English (Or → Gold, Platine → Platinum)', () => {
    expect(localizeTierName('Or', 'en')).toBe('Gold')
    expect(localizeTierName('Platine', 'en')).toBe('Platinum')
    expect(localizeTierName('Argent', 'en')).toBe('Silver')
    expect(localizeTierName('Diamant', 'en')).toBe('Diamond')
  })
  it('invariants (Bronze, Onyx, Champion)', () => {
    expect(localizeTierName('Bronze', 'en')).toBe('Bronze')
    expect(localizeTierName('Onyx', 'en')).toBe('Onyx')
    expect(localizeTierName('Champion', 'en')).toBe('Champion')
  })
  it('nom inconnu → renvoyé tel quel', () => {
    expect(localizeTierName('Placement', 'en')).toBe('Placement')
    expect(localizeTierName('', 'en')).toBe('')
  })
})

describe('localizeTierLabel', () => {
  it('legacy French baked label → English (« Or IV » → « Gold IV »)', () => {
    expect(localizeTierLabel('Or IV', 'en')).toBe('Gold IV')
    expect(localizeTierLabel('Platine II', 'en')).toBe('Platinum II')
    expect(localizeTierLabel('Diamant III', 'en')).toBe('Diamond III')
  })
  it('English baked label unchanged, Arabic sub-tier kept', () => {
    expect(localizeTierLabel('Platinum', 'en')).toBe('Platinum')
    expect(localizeTierLabel('Platinum 4', 'en')).toBe('Platinum 4')
    expect(localizeTierLabel('Gold 3', 'en')).toBe('Gold 3')
    expect(localizeTierLabel('Gold IV', 'en')).toBe('Gold IV')
  })
  it('Onyx (invariant) + suffixe valeur préservé', () => {
    expect(localizeTierLabel('Onyx', 'en')).toBe('Onyx')
    expect(localizeTierLabel('Onyx 1500', 'en')).toBe('Onyx 1500')
  })
  it('sentinelles / null / vide → inchangés', () => {
    expect(localizeTierLabel('Placement', 'en')).toBe('Placement')
    expect(localizeTierLabel('Placement (2 restants)', 'en')).toBe('Placement (2 restants)')
    expect(localizeTierLabel(null, 'en')).toBeNull()
    expect(localizeTierLabel(undefined, 'en')).toBeUndefined()
    expect(localizeTierLabel('', 'en')).toBe('')
  })
})

describe('skillTierSortValue (tri colonne Rang)', () => {
  it('ordonne les paliers majeurs (Bronze < Or < Onyx < Champion)', () => {
    const bronze = skillTierSortValue('Bronze')!
    const or = skillTierSortValue('Or')!
    const onyx = skillTierSortValue('Onyx')!
    const champion = skillTierSortValue('Champion')!
    expect(bronze).toBeLessThan(or)
    expect(or).toBeLessThan(onyx)
    expect(onyx).toBeLessThan(champion)
  })

  it('départage les sous-paliers (romain et arabe) au sein d’un palier', () => {
    // Roman (LUSR): Diamant III < Diamant VI.
    expect(skillTierSortValue('Diamant III')!).toBeLessThan(skillTierSortValue('Diamant VI')!)
    // Arabic (CSR/H5): Gold 3 < Gold 6.
    expect(skillTierSortValue('Gold 3')!).toBeLessThan(skillTierSortValue('Gold 6')!)
    // A legacy French label and its English form sort to the same place.
    expect(skillTierSortValue('Or IV')).toBe(skillTierSortValue('Gold IV'))
  })

  it('un sous-palier élevé ne dépasse jamais le palier supérieur', () => {
    // Diamant VI stays below Onyx (factor 10000 per major tier).
    expect(skillTierSortValue('Diamant VI')!).toBeLessThan(skillTierSortValue('Onyx')!)
    // Onyx with a raw CSR value stays above bare Onyx, below Champion.
    expect(skillTierSortValue('Onyx 1500')!).toBeGreaterThan(skillTierSortValue('Onyx')!)
    expect(skillTierSortValue('Onyx 1500')!).toBeLessThan(skillTierSortValue('Champion')!)
  })

  it('entrée non reconnue (Placement, brut, null, vide) → undefined (rangé en bas)', () => {
    expect(skillTierSortValue('Placement')).toBeUndefined()
    expect(skillTierSortValue('Placement (2 restants)')).toBeUndefined()
    expect(skillTierSortValue(null)).toBeUndefined()
    expect(skillTierSortValue(undefined)).toBeUndefined()
    expect(skillTierSortValue('')).toBeUndefined()
  })
})

describe('composeTierLabel', () => {
  it('name + Roman sub-tier (Diamond III / Gold IV)', () => {
    expect(composeTierLabel('Diamond', 3, 'en')).toBe('Diamond III')
    expect(composeTierLabel('Gold', 4, 'en')).toBe('Gold IV')
    expect(composeTierLabel('Platinum', 1, 'en')).toBe('Platinum I')
  })
  it('Onyx (palier ouvert) → nom seul, quel que soit le sous-palier', () => {
    expect(composeTierLabel('Onyx', 0, 'en')).toBe('Onyx')
    expect(composeTierLabel('Onyx', 3, 'en')).toBe('Onyx')
  })
  it('sous-palier hors 1..6 (0 ou >6) → nom seul', () => {
    expect(composeTierLabel('Diamond', 0, 'en')).toBe('Diamond')
    expect(composeTierLabel('Gold', 7, 'en')).toBe('Gold')
  })
})
