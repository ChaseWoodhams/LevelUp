/**
 * Wrapper runtime des manifests i18n typés (lib/i18n/generated/*).
 *
 * Resolves a manifest key with ICU MessageFormat (plural, select, and typed
 * interpolation).
 *
 * Component usage:
 *
 *   import { commonManifest } from '@/lib/i18n/generated/common'
 *   import { formatMessage } from '@/lib/i18n/format'
 *
 *   formatMessage(commonManifest, 'common.period.last_1y', 'en')
 *   // -> "Last year"
 *
 *   formatMessage(commonManifest, 'common.kpi.matches_count', 'en', { n: 5 })
 *   // -> "5 matches"
 *
 *   formatMessage(commonManifest, 'common.kpi.matches_count', 'en', { n: 1 })
 *   // -> "1 match"
 */
import { IntlMessageFormat } from 'intl-messageformat'
import type { Locale } from '@/lib/i18n/locale'

/** Manifest locale alias for the canonical `Locale` type. */
export type ManifestLocale = Locale

/**
 * Generic shape of a generated manifest: map<key, {en}>. `Record<string, ...>`
 * keeps this helper compatible with generated `as const` modules while callers
 * retain literal-key autocomplete through their own manifest type.
 */
export type ManifestEntries = Readonly<Record<string, Readonly<Record<ManifestLocale, string>>>>

// Process cache for compiled MessageFormat instances. Compiling an ICU string
// is non-trivial, so memoize it across renders.
const formatterCache = new Map<string, IntlMessageFormat>()

function getFormatter(locale: ManifestLocale, message: string): IntlMessageFormat {
  const cacheKey = `${locale}::${message}`
  let fmt = formatterCache.get(cacheKey)
  if (!fmt) {
    fmt = new IntlMessageFormat(message, locale)
    formatterCache.set(cacheKey, fmt)
  }
  return fmt
}

/**
 * formatMessage resolves a manifest key and applies ICU interpolation. Missing
 * keys return the key itself so the problem remains visible without breaking production.
 *
 * `vars` are passed directly to IntlMessageFormat.format(): plural, select,
 * date, and other ICU expressions are supported.
 */
export function formatMessage<M extends ManifestEntries>(
  manifest: M,
  key: keyof M & string,
  locale: ManifestLocale,
  vars?: Record<string, unknown>,
): string {
  const entry = manifest[key]
  if (!entry) {
    // Unknown key: return it visibly so the missing entry is easy to fix.
    return key
  }
  const message = entry[locale]
  if (typeof message !== 'string' || message === '') {
    return key
  }
  if (!vars) {
    // Skip MessageFormat when no interpolation is present.
    if (!message.includes('{')) {
      return message
    }
  }
  try {
    const fmt = getFormatter(locale, message)
    const out = fmt.format(vars)
    return Array.isArray(out) ? out.join('') : String(out)
  } catch {
    // Keep the page usable when a TOML entry has malformed ICU syntax.
    return message
  }
}

/**
 * resetFormatterCache clears the MessageFormat cache for tests.
 */
export function resetFormatterCache(): void {
  formatterCache.clear()
}
