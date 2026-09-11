import { describe, it, expect } from 'vitest'
import { queryKeys } from './keys'

describe('English-only query keys', () => {
  it('scopes home and season pass by player and title', () => {
    expect(queryKeys.home('player-1', 'halo_infinite')).toEqual(['home', 'player-1', 'halo_infinite'])
    expect(queryKeys.seasonPass('player-1', 'halo_infinite')).toEqual([
      'palmares', 'player-1', 'halo_infinite', 'season-pass',
    ])
  })

  it('does not add a locale segment to label queries', () => {
    expect(queryKeys.medals('player-1', 'halo_infinite')).not.toContain('fr')
    expect(queryKeys.citations('player-1', 'halo_infinite', 'hash')).not.toContain('fr')
    expect(queryKeys.leaderboardCatalog('player-1', 'halo_infinite')).not.toContain('fr')
  })
})
