/**
 * mapImages.config.ts — THE HAND-EDITED CALIBRATION FILE.
 *
 * This is data, not logic: one entry per map, added by hand, once. Everything that READS it is
 * `mapCalibration.ts`.
 *
 * WHY A TYPED MODULE AND NOT A .json. The file has to carry the instructions for filling it in
 * — calibrating a map is a manual measurement, and an operator opening a bare `{}` has nothing
 * to go on — and JSON cannot hold a comment. It also has to fail LOUDLY on a typo: a mistyped
 * key in JSON produces a calibration that is silently ignored, and "this map has no image" is
 * indistinguishable from "this map has an image and I misspelled maxY". Typed, `maxY` is the
 * build.
 *
 * HOW TO CALIBRATE A MAP — the whole procedure:
 *
 *   1. Put a top-down image of the map under `public/maps/`, e.g. `public/maps/olympus.png`.
 *      It is served same-origin, which is what keeps a floor somebody measures distances on
 *      out of a third party's hands.
 *   2. Open a match on that map with NO structure geometry. The grid is drawn every
 *      `GRID_STEP_M` metres from the world origin, and its labels are world coordinates: that
 *      is the ruler.
 *   3. Read off the world x/y of the image's left, right, bottom and top edges — the edges of
 *      the IMAGE, not of the playable area, since the image is stretched onto that rectangle.
 *   4. Add the entry below, keyed by the match's `map_module` (the archive's own key — it is
 *      shown in the browser's row and is stable across renames of the display name).
 *   5. Reload. The floor line under the map says which fallback is in use; it should now say
 *      the image.
 *
 * AN ENTRY THAT IS WRONG IS WORSE THAN NO ENTRY, which is why the default is empty: a floor
 * lined up by eye is a picture of a map, and a reader will measure engagement distances on it.
 * Take the four numbers off the grid; do not fit the image to the play area by dragging it
 * until it looks right.
 */
import type { MapImageConfig } from './mapCalibration'

/**
 * No map is calibrated yet, and the file ships that way ON PURPOSE: every calibration is a
 * measurement somebody made, and there is no honest default for one. Uncalibrated maps fall
 * through to the grid, which says nothing it cannot support.
 *
 * The shape of an entry, for the copy-paste:
 *
 *   olympus: { image: '/maps/olympus.png', world: { minX: -60, minY: -40, maxX: 60, maxY: 40 } },
 */
export const MAP_IMAGES: MapImageConfig = {}
