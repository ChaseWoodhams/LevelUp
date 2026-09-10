/** Read an English label from the replay artifact. */

/** The replay label shape mirrored from replay.Label on the Go side. */
export interface CatalogLabel {
  en?: string
}

export function catalogText(label: CatalogLabel | undefined): string | undefined {
  return label?.en || undefined
}
