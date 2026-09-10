/**
 * Tests — initTitleFromLocation (module title-routing, D-9 câblage boot).
 *
 * Le helper est PUR/testable (prend le pathname en paramètre) : il affirme titre
 * et langue sur le client API depuis le segment d'URL, quand présents. Aucun test
 * sur la VALEUR du slug (D-2, verbatim).
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'

const setApiTitleSlug = vi.fn()

vi.mock('@/lib/api/client', () => ({
  setApiTitleSlug: (slug: string | null) => setApiTitleSlug(slug),
}))

import { initTitleFromLocation } from './initTitleFromLocation'

describe('initTitleFromLocation', () => {
  beforeEach(() => {
    setApiTitleSlug.mockClear()
  })

  it('segment titre présent → setApiTitleSlug(slug)', () => {
    initTitleFromLocation('/t/halo_5/players/x/home')
    expect(setApiTitleSlug).toHaveBeenCalledWith('halo_5')
  })

  it('segments langue + titre → setApiTitleSlug(slug)', () => {
    initTitleFromLocation('/en/t/halo_infinite/players/x/home')
    expect(setApiTitleSlug).toHaveBeenCalledWith('halo_infinite')
  })

  it('page agnostique (aucun segment) → aucun setter', () => {
    initTitleFromLocation('/settings')
    expect(setApiTitleSlug).not.toHaveBeenCalled()
  })

  it('page joueur legacy sans segment → aucun setter (no-op Phase 1)', () => {
    initTitleFromLocation('/players/x/home')
    expect(setApiTitleSlug).not.toHaveBeenCalled()
  })
})
