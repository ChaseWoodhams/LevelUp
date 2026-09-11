/**
 * COPIED FILE — origin: apps/web/src/lib/accessibility/resolveToken.ts
 * Origin at commit: 677574859 — the commit that last changed it, so
 * `git diff 8d2558d6b HEAD -- <origin>` is what the origin has learnt since.
 *
 * Byte-identical to the origin below this header, and src/copies.guard.test.ts
 * enforces it. A fix belongs upstream first. See features/replay/README.md.
 */
/**
 * resolveToken.ts — Résout un SemanticToken vers sa couleur hex active.
 *
 * Lit la CSS custom property `--ac-<token>` sur :root.
 * Usage réservé aux contextes non-CSS : layouts Plotly, canvas, SVG inline.
 *
 * Pour les composants React, préférer `tokenCssVar(token)` directement dans
 * `style={{ color: tokenCssVar('outcome-win') }}` — plus performant, réactif
 * automatiquement au changement de palette sans re-render.
 */
import { log } from './_logger'
import { tokenVar, type SemanticToken } from './semantic-tokens'

/**
 * Retourne la couleur hex résolue pour un token dans la palette active.
 * Doit être appelé après que `applyPalette()` ait été exécuté.
 */
export function resolveToken(token: SemanticToken): string {
  if (typeof document === 'undefined') {
    log.warn(`ssr:${token}`, `resolveToken called server-side for "${token}" — returning an empty string`)
    return ''
  }

  const value = getComputedStyle(document.documentElement)
    .getPropertyValue(tokenVar(token))
    .trim()

  if (!value) {
    log.error(
      `unresolved:${token}`,
      `Token "${token}" unresolved (CSS var missing). Was applyPalette() called?`,
    )
    return ''
  }

  return value
}
