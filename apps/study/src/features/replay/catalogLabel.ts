/**
 * COPIED FILE — origin: apps/web/src/features/match-replay/catalogLabel.ts
 * Origin at commit: 677574859 — the commit that last changed it, so
 * `git diff 0afd83f7e HEAD -- <origin>` is what the origin has learnt since.
 *
 * Byte-identical to the origin below this header, and src/copies.guard.test.ts
 * enforces it. A fix belongs upstream first. See features/replay/README.md.
 */
/** Read an English label from the replay artifact. */

/** The replay label shape mirrored from replay.Label on the Go side. */
export interface CatalogLabel {
  en?: string
}

export function catalogText(label: CatalogLabel | undefined): string | undefined {
  return label?.en || undefined
}
