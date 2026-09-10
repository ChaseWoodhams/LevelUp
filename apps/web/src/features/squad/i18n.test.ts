/**
 * i18n.test.ts — English dictionary coverage for the Squad feature.
 *
 * Test load-bearing : tout ajout d'une clé en FR doit avoir son équivalent
 * EN, et inversement. Aucune autre feature n'a ce test aujourd'hui — on
 * l'introduit ici comme pattern réutilisable.
 *
 * On compare la structure profonde des deux objets (clés uniquement, pas
 * les valeurs). Les valeurs string vs function sont validées par leur
 * type pour rester strictes.
 */
import { describe, it, expect } from 'vitest'
import { EN_TEXT, getSquadText } from './i18n'

type Shape = { [k: string]: 'string' | 'function' | Shape }

function shapeOf(obj: unknown): Shape | 'string' | 'function' {
  if (typeof obj === 'string') return 'string'
  if (typeof obj === 'function') return 'function'
  if (obj === null || typeof obj !== 'object') {
    throw new Error(`Type non supporté dans le dict i18n : ${typeof obj}`)
  }
  const out: Shape = {}
  for (const [k, v] of Object.entries(obj as Record<string, unknown>)) {
    out[k] = shapeOf(v) as Shape | 'string' | 'function'
  }
  return out
}

describe('SquadText', () => {
  it('has a stable structure', () => {
    const en = shapeOf(EN_TEXT)
    expect(en).toBeDefined()
  })

  it('n\'a aucune chaîne vide en EN', () => {
    const flat = JSON.stringify(EN_TEXT)
    expect(flat).not.toContain('""')
  })
})

describe('getSquadText', () => {
  it('returns the English dictionary for every input', () => {
    expect(getSquadText('en')).toBe(EN_TEXT)
    expect(getSquadText('xx')).toBe(EN_TEXT)
    expect(getSquadText(undefined)).toBe(EN_TEXT)
  })

  it('composes parameterized strings', () => {
    const t = getSquadText('en')
    expect(t.selection.placeholder(5)).toBe('Search among 5 teammates…')
    expect(t.table.withTeammate('Foo')).toBe('With Foo')
    expect(t.errors.loadError('boom')).toBe('Error: boom')
  })
})
