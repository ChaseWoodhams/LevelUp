/** Read an English label from the replay artifact. */
import type { ReplayLocale } from './i18n'

/** The replay label shape mirrored from replay.Label on the Go side. */
export interface CatalogLabel {
  en?: string
}

export function catalogText(label: CatalogLabel | undefined, _locale: ReplayLocale): string | undefined {
  return label?.en || undefined
}
