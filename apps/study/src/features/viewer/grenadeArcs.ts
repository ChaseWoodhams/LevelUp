/**
 * grenadeArcs.ts — WHOSE ARC IS THIS.
 *
 * WHAT THE ARTIFACT GIVES US, AND WHAT IT WITHHOLDS. A grenade THROW carries the thrower's
 * slot, the frame, and the point it left from — but no flight. A PROJECTILE carries a flight,
 * replicated from launch to rest — but no owner: the archetype the decoder reads
 * (`internal/analysis/replay/projectiles.go`) has no player field at all. So the arc that is
 * already on screen is drawn in a neutral ink while the throw that produced it sits under it
 * in somebody's team colour, and the reader has to guess that the two are the same event.
 *
 * THE PAIRING IS THE UPSTREAM DECODER'S OWN WITNESS, NOT A GUESS INVENTED HERE. That file
 * records the measurement it was built on: 65 of 70 known grenade throws see a trajectory born
 * within 200 ms, against 11 to 13 for the same throws shifted as a block. Coincidence of time
 * AND place is therefore the relation between the two layers, and it is the relation this file
 * computes — with the same honesty rules the rest of the viewer follows:
 *
 *   - a flight that matches NO throw keeps the neutral ink. It happened; whose it was is not
 *     recorded, and a nearby team colour would be an attribution;
 *   - a flight that matches SEVERAL throws also keeps the neutral ink. Two players throwing
 *     from the same spot within a fifth of a second is rare and real, and picking the closer
 *     one would be inventing a tie-break the data does not contain;
 *   - nothing here decides where an arc ENDS. The last replicated point is not an impact — the
 *     film carries no detonation event — and this file does not touch it.
 *
 * PURE, AND NO CANVAS: `paintReplay.ts` draws, this decides.
 */
import type { ReplayGrenade, ReplayProjectile } from '@/lib/api/types'

/**
 * How far apart a throw and the birth of its flight may be on the ground, in metres.
 *
 * A projectile's first replicated position is where the object exists, not where the player
 * stands: it leaves the hand already moving, and one grid step of the replay is a tenth of a
 * second of flight. Four metres covers that gap and stays far below the distance between two
 * players fighting over the same corner.
 */
export const ARC_ORIGIN_RADIUS_M = 4

/**
 * How far apart they may be in time. The decoder's own window is 200 ms; this is that window
 * with one grid step of slack, since a throw and a birth rounded onto a 10 Hz grid can fall on
 * either side of the same instant.
 */
export const ARC_ORIGIN_WINDOW_MS = 300

/**
 * arcThrowers says, for each projectile, which slot threw it — or null when the film does not
 * say so unambiguously.
 *
 * Aligned with `projectiles` by index, which is how the draw layer addresses them.
 */
export function arcThrowers(
  grenades: readonly ReplayGrenade[],
  projectiles: readonly ReplayProjectile[],
  windowFrames: number,
): (number | null)[] {
  return projectiles.map((pr) => throwerOf(grenades, pr, windowFrames))
}

/** throwerOf resolves one flight, or refuses to. */
function throwerOf(
  grenades: readonly ReplayGrenade[],
  projectile: ReplayProjectile,
  windowFrames: number,
): number | null {
  const birth = birthOf(projectile)
  if (birth === null) return null
  let found: number | null = null
  for (const g of grenades) {
    if (Math.abs(g.t - birth.t) > windowFrames) continue
    if (Math.hypot(g.x - birth.x, g.y - birth.y) > ARC_ORIGIN_RADIUS_M) continue
    // A SECOND CANDIDATE ENDS THE SEARCH RATHER THAN REFINING IT: ambiguity is an answer.
    if (found !== null && found !== g.slot) return null
    found = g.slot
  }
  return found
}

/** The frame and place a flight was first replicated at, or null if it carries no point. */
function birthOf(projectile: ReplayProjectile): { t: number; x: number; y: number } | null {
  const first = projectile.p?.[0]
  if (!first || first.length < 3) return null
  const [dt, x, y] = first as number[]
  return { t: projectile.t0 + dt, x, y }
}
