/**
 * floorScope.test.ts — a surface only reaches mapFloor when it can actually land inside
 * the grid mapFloor rasterises onto.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayBounds } from '@/lib/api/types'
import type { ReplaySurfaceReady } from '../replay/replayNormalize'

import { floorSurfacesOf, withinFloorBounds } from './floorScope'

const BOUNDS: ReplayBounds = { minX: 0, minY: 0, maxX: 40, maxY: 30 }

function slab(x0: number, y0: number, x1: number, y1: number): ReplaySurfaceReady {
  return { x0, y0, x1, y1, z: 0, zb: -0.5, poly: [] }
}

describe('withinFloorBounds', () => {
  it('keeps a surface fully inside the played area', () => {
    expect(withinFloorBounds(slab(5, 5, 15, 15), BOUNDS)).toBe(true)
  })

  it('keeps a surface that only partially overlaps — a doorway the bounds clip', () => {
    expect(withinFloorBounds(slab(-5, 5, 5, 15), BOUNDS)).toBe(true)
    expect(withinFloorBounds(slab(35, 5, 45, 15), BOUNDS)).toBe(true)
  })

  it('keeps a surface exactly touching an edge', () => {
    expect(withinFloorBounds(slab(-10, -10, 0, 0), BOUNDS)).toBe(true)
  })

  it('drops a surface entirely outside the played area', () => {
    expect(withinFloorBounds(slab(100, 100, 110, 110), BOUNDS)).toBe(false)
    expect(withinFloorBounds(slab(-50, 5, -20, 15), BOUNDS)).toBe(false)
  })

  it('drops a surface outside on one axis even when the other overlaps', () => {
    // Shares Y with the play area but sits far off in X — the case that used to get
    // clamped onto the grid's edge column instead of excluded.
    expect(withinFloorBounds(slab(500, 5, 510, 15), BOUNDS)).toBe(false)
  })
})

describe('floorSurfacesOf', () => {
  it('keeps only the surfaces the grid can actually place', () => {
    const inside = slab(5, 5, 15, 15)
    const outside = slab(500, 500, 510, 510)
    expect(floorSurfacesOf([inside, outside], BOUNDS)).toEqual([inside])
  })

  it('measured against the real match this exists for: filtering removes surfaces, never adds', () => {
    const structure = [slab(5, 5, 15, 15), slab(-1000, -1000, -990, -990), slab(20, 20, 25, 25)]
    const filtered = floorSurfacesOf(structure, BOUNDS)
    expect(filtered.length).toBeLessThan(structure.length)
    expect(filtered.every((s) => structure.includes(s))).toBe(true)
  })
})
