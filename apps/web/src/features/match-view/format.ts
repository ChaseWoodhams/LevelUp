import { normalizeModeLabel } from '@/lib/halo/modeLabel'

export function buildMatchHeadingStr(
  mapUI: string | null | undefined,
  modeUI: string | null | undefined,
): string {
  const normalizedMode = normalizeModeLabel(modeUI, mapUI)
  const connector = 'on'
  if (normalizedMode && mapUI) {
    return `${normalizedMode} ${connector} ${mapUI}`
  }
  return normalizedMode ?? mapUI ?? ''
}
