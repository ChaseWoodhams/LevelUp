/**
 * controls.tsx — THE SMALL CONTROL, defined once.
 *
 * WHY THIS FILE EXISTS. `h-7 px-2 text-xs` is the size of every control in this app's dense
 * rows — layer toggles, speeds, frame steps, jumps, the focus digits, the language switch — and
 * it had reached eight hand-written copies. The repository's rule is explicit about what
 * happens next: at the third copy, centralise AND add a guard-rail, because a factorisation
 * without one re-diverges (its own worked example is a predicate that went from 8 copies to 36
 * *after* being centralised). `controls.guard.test.ts` is that guard-rail.
 *
 * TWO COMPONENTS, BECAUSE THERE ARE TWO KINDS OF CONTROL, and the difference is not cosmetic:
 * a toggle is a state the reader can see and un-set, and it owes `aria-pressed`; an action
 * happens once and owes nothing. Collapsing them into one component with a nullable `active`
 * would let a plain action ship without the attribute, or with it and lying.
 *
 * This file is NOT a copy from `apps/web` — the web app has no equivalent — so it carries no
 * origin header and no drift guard.
 */
import type { ComponentProps, ReactNode } from 'react'

import { Button } from './button'

/**
 * The one size. Kept as a constant rather than inlined into the two components below because
 * the guard-rail has to be able to say "this literal belongs here and nowhere else", and a
 * named constant is what makes that sentence checkable.
 */
export const COMPACT_CONTROL = 'h-7 px-2 text-xs'

/** Digits, glyphs and clock-like labels line up only in a monospaced face. */
const COMPACT_MONO = 'h-7 px-2 font-mono text-xs'

type ButtonVariant = ComponentProps<typeof Button>['variant']

interface CommonProps {
  onClick: () => void
  /** Tooltip. On a control whose label is a glyph or a digit, this is where the meaning lives. */
  title?: string
  /** Accessible name, for a control whose visible label is a glyph or a digit. */
  label?: string
  /** Render the label in a monospaced face: digits and glyphs, not words. */
  mono?: boolean
  children: ReactNode
}

/**
 * Toggle — a control that is on or off, and says so.
 *
 * `aria-pressed` is not optional here and that is the point of the component: a toggle that
 * only changes colour is invisible to anyone not looking at it.
 */
export function Toggle({ on, ...props }: CommonProps & { on: boolean }) {
  return (
    <Button
      variant={on ? 'default' : 'ghost'}
      size="sm"
      onClick={props.onClick}
      className={props.mono ? COMPACT_MONO : COMPACT_CONTROL}
      title={props.title}
      aria-label={props.label}
      aria-pressed={on}
    >
      {props.children}
    </Button>
  )
}

/**
 * CompactAction — a control that does something once.
 *
 * `enabled` rather than `disabled` because every caller has a positive reason to hand ("there
 * IS a next death"), and negating it at the call site is where the double negatives start.
 */
export function CompactAction({
  enabled = true,
  variant = 'ghost',
  ...props
}: CommonProps & { enabled?: boolean; variant?: ButtonVariant }) {
  return (
    <Button
      variant={variant}
      size="sm"
      onClick={props.onClick}
      disabled={!enabled}
      className={props.mono ? COMPACT_MONO : COMPACT_CONTROL}
      title={props.title}
      aria-label={props.label}
    >
      {props.children}
    </Button>
  )
}
