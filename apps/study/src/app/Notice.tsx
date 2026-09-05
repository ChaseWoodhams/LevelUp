/**
 * Notice — a screen that is not a replay: what happened, why, and what to do next.
 *
 * ONE COMPONENT FOR ALL OF THEM because the five ways a match can fail to open differ in their
 * WORDS, not in their shape, and giving each its own markup is how they drift into looking
 * like five unrelated failures. The variation that matters is carried by the tone token: an
 * archive held by a capture is not a fault and must not be painted like one.
 */
import type { ReactNode } from 'react'

import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'

/**
 * The tone of a notice, as a semantic token.
 *
 * `info` for a state that is nobody's fault and will pass (a capture holding the archive);
 * `warning` for a match that has nothing to show yet; `destructive` for something that is
 * actually broken. No fourth: a notice that cannot be placed in one of the three is a notice
 * whose meaning has not been decided.
 */
type Tone = 'info' | 'warning' | 'destructive'

interface NoticeProps {
  tone: Tone
  title: string
  hint: string
  /** A way out, when there is one — a retry, a link back. */
  action?: ReactNode
}

export function Notice({ tone, title, hint, action }: NoticeProps) {
  return (
    <section className="flex flex-col items-start gap-2 rounded-lg border border-border bg-card p-4">
      <h2 className="text-sm font-medium" style={{ color: tokenCssVar(tone) }}>
        {title}
      </h2>
      <p className="max-w-[70ch] text-xs text-muted-foreground">{hint}</p>
      {action}
    </section>
  )
}
