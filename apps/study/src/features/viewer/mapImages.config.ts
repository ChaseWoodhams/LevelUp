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
export const MAP_IMAGES: MapImageConfig = {
  /**
   * sgh_streets ("Streets") — NOT CALIBRATED, AND THAT IS THE POINT. The four numbers are not a
   * measurement anybody made; they are the map's own sbsp bounding box, and the image is
   * rendered onto exactly that rectangle. Nothing here was fitted, so there is nothing here to
   * have fitted wrong.
   *
   * WHERE THE IMAGE COMES FROM. The map's geometry is extracted from the game's own module
   * files with `ekur` and rendered top-down in Blender by `tools/map-render/` (whose README
   * carries the whole pipeline). The camera is orthographic and framed on
   * X[-24.32224, 27.407486], Y[-23.018236, 29.866623] — the sbsp AABB, which is the same frame
   * the replay's coordinates live in. So a world point maps to a pixel by linear interpolation
   * of the rectangle below, which is what this file's contract already asks for.
   *
   * THE CHECK THAT MATTERS: 175,484 player positions from four archived matches, every one of
   * them (100.00%) landing on drawn geometry, with no calibration step in between. The previous
   * asset was a piece of callout art registered against two Forge-extracted flag positions —
   * a two-point fit, which cannot distinguish a rotation from its mirror, and did in fact get
   * the rotation backwards until a player in one of the archived matches said which room they
   * spawned in. None of that fragility survives here: there is no fit.
   *
   * ROOFS ARE CUT AWAY, per region. A plain top-down render hides every street under its
   * canopy, and no per-object rule removes them — a building arrives as ONE mesh spanning floor
   * to roof, so its bounding box starts at ground level and every "is this a roof" test says
   * no. Only the camera's near plane cuts per pixel, and one cut height cannot serve a map that
   * stacks walkways over streets (reachable floors here span 0 to 5 m). The asset is therefore
   * composited from a stack of plan cuts, each pixel taking the lowest cut that still clears
   * the local reachable floor — derived from where players actually stood — by 2 m.
   *
   * `preferOverStructure` STAYS SET, for the reason it was set originally: this map's
   * reconstructed floor is hundreds of raw BSP rectangles with no rooms or corridors drawn in,
   * and reads as a jumble even where every one is placed correctly.
   */
  sgh_streets: {
    image: '/maps/sgh_streets.png',
    world: { minX: -24.32224, minY: -23.018236, maxX: 27.407486, maxY: 29.866623 },
    preferOverStructure: true,
  },
}
