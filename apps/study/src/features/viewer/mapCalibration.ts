/**
 * mapCalibration.ts — WHICH FLOOR A MATCH GETS, AND WHY.
 *
 * WHAT THE FALLBACK ORDER ACTUALLY WAS BEFORE THIS FILE (checked, not assumed — the ticket
 * asks for exactly that). `mapFloor.ts` has no chain in it at all: it rasterises BSP surfaces
 * into an altitude grid and knows nothing about what to do when there are none. The chain was
 * spread across two other places — `useReplayPainter` built a grid only when
 * `doc.structure.length > 0`, and `paintReplay.paintGround` drew the Forge props when there was
 * no pre-painted floor and nothing at all when there were no props either. So the real order
 * was: reconstructed floor -> Forge props -> an empty background. A match on a map with neither
 * was watched over a blank rectangle.
 *
 * WHAT IT IS NOW: reconstructed floor -> calibrated top-down image -> plain grid. The props
 * layer keeps its place ON TOP of the last two, where it was already the only landmark; what it
 * is no longer is the last line of defence.
 *
 * CALIBRATION IS MANUAL, PER MAP, AND ONCE. An image plus the world coordinates of its corners
 * (`mapImages.config.ts`). Nothing here derives a scale from the match: a replay's bounds are
 * the area the PLAYERS covered, which is smaller than the map and different in every match, so
 * fitting an image to them would stretch the same map differently every time — a floor that
 * lies about distances, on every match, silently. An uncalibrated map therefore gets the grid,
 * which claims nothing.
 *
 * PURE. The lookup and the decision are numbers and strings; the drawing is `studyDraw.ts` and
 * the loading is `useReplayPainter.ts`.
 */

/** The world rectangle an image's corners sit on, in the replay's own coordinate frame. */
export interface WorldRect {
  minX: number
  minY: number
  maxX: number
  maxY: number
}

/** One calibrated map: a top-down image, and where its corners are in the world. */
export interface MapImageCalibration {
  /**
   * URL of the image as this app serves it — a path under `public/`, so it is fetched
   * same-origin and needs no CORS. A remote URL would put a third party between the reader and
   * a floor they are measuring distances on.
   */
  image: string
  /** World coordinates of the image's edges. `minY` is its BOTTOM: the world's +Y is up. */
  world: WorldRect
}

/** The calibration file's shape: one entry per map MODULE, which is the archive's stable key. */
export type MapImageConfig = Readonly<Record<string, MapImageCalibration>>

/**
 * Which floor is actually under the match.
 *
 * Three values and no fourth: this is what the reader is told, so it names what is DRAWN. A
 * calibrated map whose image has not arrived yet is on the grid, and says grid, until it is.
 */
export type FloorSource = 'structure' | 'image' | 'grid'

/**
 * calibrationFor finds the entry for a map module, or null.
 *
 * THE KEY IS FOLDED AND TRIMMED because it is hand-typed on both sides — into the config by
 * whoever calibrates the map, and into the archive by the stats payload — and a lookup that
 * missed on a capital letter would present as "this map has no calibration", which is the one
 * answer that looks correct while being wrong.
 *
 * A DEGENERATE RECTANGLE IS NOT A CALIBRATION. Corners that do not enclose an area cannot map
 * an image onto the world; treating such an entry as configured would divide by zero or paint
 * a line. It is refused here rather than at the point of drawing, so the fallback below it can
 * take over honestly.
 */
export function calibrationFor(
  mapModule: string | null | undefined,
  config: MapImageConfig,
): MapImageCalibration | null {
  const key = (mapModule ?? '').trim().toLowerCase()
  if (key === '') return null
  for (const [name, entry] of Object.entries(config)) {
    if (name.trim().toLowerCase() !== key) continue
    return isUsable(entry) ? entry : null
  }
  return null
}

/** isUsable rejects an entry that cannot be drawn: no image, or corners enclosing no area. */
export function isUsable(entry: MapImageCalibration | null | undefined): boolean {
  if (!entry || entry.image.trim() === '') return false
  const w = entry.world
  return (
    Number.isFinite(w.minX) &&
    Number.isFinite(w.minY) &&
    Number.isFinite(w.maxX) &&
    Number.isFinite(w.maxY) &&
    w.maxX > w.minX &&
    w.maxY > w.minY
  )
}

/**
 * floorSourceOf picks the floor, in the one order the whole app agrees on.
 *
 * REAL GEOMETRY WINS, ALWAYS. A reconstructed floor is the same data that carries the
 * trajectories — measured against them, median error 8 mm — while a calibrated image is a
 * picture somebody lined up by hand. Where both exist, the one that was measured is the one
 * drawn.
 */
export function floorSourceOf(hasStructure: boolean, imageReady: boolean): FloorSource {
  if (hasStructure) return 'structure'
  return imageReady ? 'image' : 'grid'
}
