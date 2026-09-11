/**
 * format.ts — the two numbers this tool writes out, formatted in the reader's language.
 *
 * A SHARE IS A FRACTION OF 1, EVERYWHERE (ADR 0006), and it is turned into a percentage HERE and
 * nowhere else. `Intl` is what knows that French puts a space before the sign and English does
 * not, and a second call site formatting one by hand is how a table and its own filter come to
 * disagree about what 85 % looks like.
 */
import type { Locale } from '@/lib/i18n/locale'

/** formatShare renders a 0..1 fraction as a percentage, with no decimals. */
export function formatShare(value: number, locale: Locale): string {
  return new Intl.NumberFormat(locale, { style: 'percent', maximumFractionDigits: 0 }).format(value)
}

/**
 * formatInstant renders an ISO timestamp in the reader's language, or null when there is none
 * to render.
 *
 * NULL RATHER THAN A FALLBACK STRING: what to show for a match whose stats named no start time
 * is a decision for the screen — it has the word for "unknown" in the right language — not for
 * a formatter.
 */
export function formatInstant(iso: string | undefined, locale: Locale): string | null {
  if (!iso) return null
  const at = new Date(iso)
  if (Number.isNaN(at.getTime())) return null
  return at.toLocaleString(locale, { dateStyle: 'short', timeStyle: 'short' })
}
