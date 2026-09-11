/**
 * canvasFraming.ts — HOW TALL THE CANVAS IS, WHEN THE SCENE ITSELF SAYS SO.
 *
 * The origin's ReplayCanvas fixes the height and derives the width from the scene's
 * aspect ratio (`replayLogic.fitWidth`) — right for the maps it was tuned against, every
 * one of them wider than it is tall. A PORTRAIT scene (bounds taller than wide) inverts
 * that: a fixed height forces a narrow width, and the played area renders as a thin,
 * cramped column with wasted space either side — measured on this app's first real
 * capture (0e97be38: a ~28 x 41 m play area, rendering ~350 px wide inside a container
 * several times that, at the fixed 480 px height).
 *
 * THE SWITCH IS ON THE SCENE'S OWN SHAPE, NOT ON A RATIO AGAINST THE CONTAINER — and
 * that distinction is the one bug worth naming, because the more obvious formula gets it
 * wrong. "Full-width height, clamped to a minimum" sounds like it protects every
 * landscape scene, but it does not: at a 1000 px container and a 480 px minimum, the
 * reference ratio the clamp actually protects is ~2.08:1, so an ordinary 40 x 30 scene
 * (1.33:1 — this app's own fixture) would ALSO grow past 480 px under it, on nothing
 * more than a wide enough container. A wide container is the ordinary case, not an edge
 * one, so that formula would have changed height for most matches, not just portrait
 * ones. Branching on the scene's own aspect ratio (bh vs bw) is what keeps the two cases
 * separate: every scene at least as wide as it is tall renders at exactly
 * MIN_CANVAS_HEIGHT regardless of container width, byte-for-byte what it was before this
 * file existed. Only a scene TALLER than it is wide takes the second branch, and there
 * the full-width height is clamped to [MIN_CANVAS_HEIGHT, MAX_CANVAS_HEIGHT] — the
 * second bound so an extreme aspect ratio goes back to being width-constrained rather
 * than pushing the page arbitrarily tall.
 */
import type { ReplayBounds } from '@/lib/api/types'

/** Every landscape scene's height, unchanged from before this file existed. */
export const MIN_CANVAS_HEIGHT = 480

/** How tall a portrait scene may push the canvas before it goes width-constrained again. */
export const MAX_CANVAS_HEIGHT = 760

/**
 * sceneCanvasHeight picks the canvas height for one scene at one container width.
 *
 * A scene at least as wide as it is tall answers the minimum UNCONDITIONALLY — not
 * "usually", regardless of `availableWidth` — which is what makes this additive rather
 * than a rewrite: nothing about an existing landscape match's framing can change no
 * matter how wide its container gets. `availableWidth` of 0 (container not yet measured)
 * answers the minimum for the same reason a portrait scene would get it anyway: there is
 * nothing to fit a width against yet.
 */
export function sceneCanvasHeight(
  bounds: ReplayBounds,
  availableWidth: number,
  pad: number,
): number {
  const bw = Math.max(bounds.maxX - bounds.minX, 1e-6)
  const bh = Math.max(bounds.maxY - bounds.minY, 1e-6)
  if (bh <= bw || availableWidth <= 0) return MIN_CANVAS_HEIGHT
  const fullWidthHeight = (availableWidth - 2 * pad) * (bh / bw) + 2 * pad
  return Math.min(Math.max(fullWidthHeight, MIN_CANVAS_HEIGHT), MAX_CANVAS_HEIGHT)
}
