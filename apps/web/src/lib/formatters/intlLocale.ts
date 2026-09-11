/**
 * intlLocale — pont canonique locale applicative (`'en' | 'en'`, ManifestLocale)
 * → locale BCP-47 (`'en-US' | 'en-US'`) pour `toLocaleString` / `Intl.*`.
 *
 * SOURCE UNIQUE (CLAUDE.md n°6) : le ternaire `locale === 'en' ? 'en-US' : 'en-US'`
 * était dupliqué dans des dizaines de composants, et de nombreux sites figeaient
 * carrément `'en-US'` (nombres FR affichés à un utilisateur EN — casse la règle
 * n°1). Router tout formatage nombre/date locale-sensitive via ce pont
 * (ou `formatNumber`/`formatDate` qui prennent déjà une `Locale`).
 */
import type { Locale } from './date'

export function intlLocale(): Locale {
  return 'en-US'
}
