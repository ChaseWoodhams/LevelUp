/**
 * COPIED FILE — origin: apps/web/src/lib/i18n/locale.ts
 * Origin at commit: a1520ad0c — the commit that last changed it, so
 * `git diff 7a79c7961 HEAD -- <origin>` is what the origin has learnt since.
 *
 * Byte-identical to the origin below this header, and src/copies.guard.test.ts
 * enforces it. A fix belongs upstream first. See features/replay/README.md.
 */
/**
 * Canonical application locale type.
 *
 * This leaf module owns the runtime locale list and imports nothing. URL
 * parsing, manifest formatting, and the remaining API compatibility code all
 * depend on it, which keeps the dependency direction acyclic.
 */
export const KNOWN_LOCALES = ['en'] as const

export type Locale = (typeof KNOWN_LOCALES)[number]

/** Type guard for the canonical English route segment. */
export function isKnownLocale(segment: string): segment is Locale {
  return (KNOWN_LOCALES as readonly string[]).includes(segment)
}
