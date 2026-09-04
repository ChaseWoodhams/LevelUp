/**
 * COPIED FILE — origin: apps/web/src/lib/accessibility/plotlyColorscale.ts
 * Origin at commit: 483c593e0 — the commit that last changed it, so
 * `git diff 483c593e0 HEAD -- <origin>` is what the origin has learnt since.
 *
 * Byte-identical to the origin below this header, and src/copies.guard.test.ts
 * enforces it. A fix belongs upstream first. See features/replay/README.md.
 */
/**
 * plotlyColorscale — helper série-colors pour ECharts/canvas.
 *
 * Usage dans les charts :
 *   color: getSeriesColors(n, ['perf-tier-1','perf-tier-2','perf-tier-3'])
 */
import type { SemanticToken } from './semantic-tokens'
import { resolveToken } from './resolveToken'

/** Retourne N couleurs en cyclant sur la liste de tokens. */
export function getSeriesColors(n: number, tokens: SemanticToken[]): string[] {
  return Array.from({ length: n }, (_, i) => resolveToken(tokens[i % tokens.length]))
}
