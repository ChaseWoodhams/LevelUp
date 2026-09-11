/**
 * floorScope.ts — WHICH STRUCTURE SURFACES BELONG TO THE PLAYED FLOOR.
 *
 * `mapFloor.buildFloorGrid` rasterises whatever surfaces it is handed onto a grid sized
 * to `bounds` — the area the PLAYERS covered, deliberately smaller than the whole map
 * (its own header explains why: a map's structure can span hundreds of metres of skybox
 * and distant decor while the play area is a enaction of that). What it does NOT do is
 * check that a surface actually falls inside that grid: `cellIndex` CLAMPS an
 * out-of-range coordinate to the nearest edge cell rather than dropping it, so a surface
 * far outside the frame is not excluded — it is squeezed onto whichever border cell its
 * clamped coordinate lands on.
 *
 * MEASURED, NOT ASSUMED. On this app's first real capture (0e97be38, study epic #2),
 * 3,293 of the match's 10,908 structure surfaces (30%) sit outside the played bounds,
 * and the result on screen was exactly what that predicts: a floor of disconnected
 * blocks piled along one edge of the canvas instead of the rooms the match actually
 * played out in — every one of those surfaces clamped into the same handful of border
 * cells. The fixture that exercised this code before never had the problem: its 8
 * hand-placed slabs all sit well inside its bounds by construction.
 *
 * FILTERED HERE, BEFORE mapFloor EVER SEES THE LIST, rather than inside it: mapFloor.ts
 * is a byte-identical copy of `apps/web`'s and a fix belongs upstream first (its own
 * README) — study-side filtering repairs the picture without touching the copy.
 */
import type { ReplayBounds } from '@/lib/api/types'

import type { ReplaySurfaceReady } from '../replay/replayNormalize'

/**
 * withinFloorBounds keeps a surface whose axis-aligned box overlaps the played area AT
 * ALL — not "is centred inside it". A room the match's bounds only clip the doorway of
 * is still part of the floor a player stood on; requiring the surface's CENTRE to fall
 * inside would drop it for no better reason than where its midpoint happens to land.
 */
export function withinFloorBounds(s: ReplaySurfaceReady, bounds: ReplayBounds): boolean {
  return s.x1 >= bounds.minX && s.x0 <= bounds.maxX && s.y1 >= bounds.minY && s.y0 <= bounds.maxY
}

/** floorSurfacesOf filters a document's structure down to what its floor grid can use. */
export function floorSurfacesOf(
  structure: readonly ReplaySurfaceReady[],
  bounds: ReplayBounds,
): ReplaySurfaceReady[] {
  return structure.filter((s) => withinFloorBounds(s, bounds))
}
