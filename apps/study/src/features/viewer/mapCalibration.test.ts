/**
 * mapCalibration.test.ts — the lookup and the fallback order, with no canvas anywhere near.
 *
 * That separation is the ticket's own requirement, and it is what makes these cases cheap to
 * state: a calibration is four numbers and a path, and every case below is a way of getting
 * the wrong floor under a match.
 */
import { describe, expect, it } from 'vitest'

import { calibrationFor, floorSourceOf, isUsable, type MapImageConfig } from './mapCalibration'
import { MAP_IMAGES } from './mapImages.config'

const OLYMPUS = { image: '/maps/olympus.png', world: { minX: -60, minY: -40, maxX: 60, maxY: 40 } }

const CONFIG: MapImageConfig = {
  olympus: OLYMPUS,
  ' Streets ': { image: '/maps/streets.png', world: { minX: 0, minY: 0, maxX: 100, maxY: 80 } },
  flat: { image: '/maps/flat.png', world: { minX: 10, minY: 0, maxX: 10, maxY: 80 } },
  nameless: { image: '  ', world: { minX: 0, minY: 0, maxX: 1, maxY: 1 } },
}

describe('calibrationFor', () => {
  it('finds a map by its module key', () => {
    expect(calibrationFor('olympus', CONFIG)).toBe(OLYMPUS)
  })

  it('matches whatever case and spacing either side was typed in', () => {
    expect(calibrationFor('OLYMPUS', CONFIG)).toBe(OLYMPUS)
    expect(calibrationFor('  olympus ', CONFIG)).toBe(OLYMPUS)
    expect(calibrationFor('streets', CONFIG)?.image).toBe('/maps/streets.png')
  })

  it('answers null for a map nobody has calibrated', () => {
    expect(calibrationFor('aquarius', CONFIG)).toBeNull()
  })

  it('answers null when the match does not name a map at all', () => {
    expect(calibrationFor(null, CONFIG)).toBeNull()
    expect(calibrationFor(undefined, CONFIG)).toBeNull()
    expect(calibrationFor('   ', CONFIG)).toBeNull()
  })

  it('refuses corners that enclose no area rather than dividing by zero later', () => {
    expect(calibrationFor('flat', CONFIG)).toBeNull()
  })

  it('refuses an entry with no image', () => {
    expect(calibrationFor('nameless', CONFIG)).toBeNull()
  })

  it('answers null against the empty file this app ships with', () => {
    expect(calibrationFor('olympus', MAP_IMAGES)).toBeNull()
  })
})

describe('isUsable', () => {
  it('rejects a rectangle that is inverted or not a number', () => {
    expect(isUsable({ image: 'a.png', world: { minX: 10, minY: 0, maxX: 0, maxY: 10 } })).toBe(false)
    expect(isUsable({ image: 'a.png', world: { minX: 0, minY: 10, maxX: 10, maxY: 0 } })).toBe(false)
    expect(isUsable({ image: 'a.png', world: { minX: NaN, minY: 0, maxX: 10, maxY: 10 } })).toBe(false)
    expect(isUsable(null)).toBe(false)
  })

  it('accepts a rectangle with an area', () => {
    expect(isUsable(OLYMPUS)).toBe(true)
  })
})

describe('floorSourceOf', () => {
  it('prefers real geometry over a picture somebody lined up by hand, by default', () => {
    expect(floorSourceOf(true, true)).toBe('structure')
    expect(floorSourceOf(true, false)).toBe('structure')
  })

  it('falls to the calibrated image when there is no structure', () => {
    expect(floorSourceOf(false, true)).toBe('image')
  })

  it('falls to the grid when there is neither', () => {
    expect(floorSourceOf(false, false)).toBe('grid')
  })

  describe('preferImage — the one opt-in exception', () => {
    it('draws the image over structure when the map has asked for it', () => {
      expect(floorSourceOf(true, true, true)).toBe('image')
    })

    it('still refuses to draw an image that has not actually loaded', () => {
      // The preference is not permission to lie: no image, no floor to show for one.
      expect(floorSourceOf(true, false, true)).toBe('structure')
    })

    it('changes nothing for a map that has not opted in', () => {
      expect(floorSourceOf(true, true, false)).toBe('structure')
      expect(floorSourceOf(true, true, undefined)).toBe('structure')
    })
  })
})
