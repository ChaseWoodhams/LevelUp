/**
 * trailLogic.ts — THE SHAPE OF A TRAIL, AND HOW OLD EACH PART OF IT IS.
 *
 * WHY THIS EXISTS BESIDE A COPIED MODULE THAT ALREADY DRAWS TRAILS. The copied player layer
 * (`features/replay/replayMarkers.ts`) strokes one flat polyline per life at a single opacity:
 * the reader can see WHERE somebody has been, never in which ORDER. On a map where two players
 * cross, a flat trail says the two paths exist and refuses to say which way either of them ran.
 * That module is a byte-for-byte copy of `apps/web` and cannot be edited from here, so the
 * study viewer draws its own trail and turns the copied one off (cf. `paintReplay.ts`).
 *
 * THE AGE TRAVELS WITH THE POINT, which is the whole reason this file is not just `trailAt`.
 * `replayLogic.trailAt` returns positions; a fade needs to know how far back each of them was
 * sampled. Deriving that afterwards from an index would be wrong the moment the film skips a
 * sample — and the film skips samples constantly, since it only transmits what changed.
 *
 * PURE, AND NO CANVAS. The drawing is `trailLayer.ts`; everything decided here is decided
 * against numbers and is tested against numbers.
 */
import type { ReplayPoint } from '@/lib/api/types'

import { positionAt } from '../replay/replayLogic'

/**
 * TRAIL_WINDOW_MS — the trailing windows the reader can choose between, in real time.
 *
 * SIX SECONDS BY DEFAULT, and the default is a shorter window than the copied layer's seven:
 * this trail fades, so its far end is already faint, and the useful length of a fading trail
 * is shorter than that of a flat one. Six seconds is also about one engagement — long enough
 * to show how somebody arrived at a fight, short enough that eight of them do not fill the map.
 *
 * `null` is the FULL LIFE PATH: everything the film sampled between the spawn and the current
 * frame. It is a window like the others rather than a separate mode, so the same code path
 * draws both and there is no second way for a trail to be built.
 */
export const TRAIL_WINDOW_MS: readonly (number | null)[] = [3_000, 6_000, null]

/** The window a viewer opens on. Cf. TRAIL_WINDOW_MS for why it is the middle one. */
export const DEFAULT_TRAIL_WINDOW_MS = 6_000

/** One vertex of a trail: where the player was, and how long ago that was sampled. */
export interface TrailVertex {
  x: number
  y: number
  /** Frames between this sample and the frame being drawn. 0 = right now. */
  age: number
}

/**
 * trailPath returns a life's trail at `frame`, oldest vertex first, head last.
 *
 * `windowFrames` of `Infinity` means the whole life so far — the full-path mode. A window of
 * zero or less yields nothing rather than a degenerate one-vertex path: there is no line to
 * draw through a single point, and returning one would make every caller check the length.
 *
 * THE HEAD IS INTERPOLATED, the rest are raw samples. That is what makes the trail end exactly
 * under the player's marker at every frame instead of snapping forward once per sample.
 */
export function trailPath(points: ReplayPoint[], frame: number, windowFrames: number): TrailVertex[] {
  if (!(windowFrames > 0) || points.length === 0) return []
  const oldest = frame - windowFrames
  const out: TrailVertex[] = []
  for (const p of points) {
    if (p.t < oldest) continue
    if (p.t > frame) break
    out.push({ x: p.x, y: p.y, age: frame - p.t })
  }
  const head = positionAt(points, frame)
  if (head) {
    const tail = out[out.length - 1]
    if (!tail || tail.x !== head.x || tail.y !== head.y) out.push({ ...head, age: 0 })
  }
  return out.length < 2 ? [] : out
}

/**
 * trailFadeSpan gives the age over which a trail's opacity is graded.
 *
 * WITH A FIXED WINDOW IT IS THE WINDOW, so "faint" means the same age on every player and at
 * every moment — a trail that faded over its own length would read as a scale that changes
 * per player. In FULL-PATH mode there is no window to grade against, so the oldest vertex is
 * the far end by definition; without that the whole path would come out at one opacity, which
 * is the flat trail this file exists to replace.
 */
export function trailFadeSpan(path: TrailVertex[], windowFrames: number): number {
  if (Number.isFinite(windowFrames)) return Math.max(windowFrames, 1)
  // The path is ordered oldest first, so its first vertex IS the far end.
  return Math.max(path[0]?.age ?? 1, 1)
}

/**
 * trailFade converts an age into the share of the trail's opacity a segment keeps.
 *
 * It never reaches zero: the far end of a trail has to stay visible, or the reader loses the
 * only thing that says where a player came from. `TRAIL_OLDEST_SHARE` is that floor.
 */
export const TRAIL_OLDEST_SHARE = 0.15

export function trailFade(age: number, span: number): number {
  if (!(span > 0)) return 1
  const r = Math.min(Math.max(age / span, 0), 1)
  return 1 - (1 - TRAIL_OLDEST_SHARE) * r
}
