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
