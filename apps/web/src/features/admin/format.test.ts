import { describe, expect, it } from 'vitest'

import { adminRelativeTime, formatDurationMs, formatIntervalMinutes } from './format'

describe('adminRelativeTime', () => {
  const now = new Date('2026-06-11T12:00:00Z').getTime()

  it('retourne un tiret pour les ISO vides ou invalides', () => {
    expect(adminRelativeTime(undefined, 'en', now)).toBe('—')
    expect(adminRelativeTime('', 'en', now)).toBe('—')
    expect(adminRelativeTime('pas-une-date', 'en', now)).toBe('—')
  })

  it('couvre les bornes minute/heure/jour en FR', () => {
    expect(adminRelativeTime('2026-06-11T11:59:50Z', 'en', now)).toBe("just now")
    expect(adminRelativeTime('2026-06-11T11:45:00Z', 'en', now)).toBe('15 min ago')
    expect(adminRelativeTime('2026-06-11T09:00:00Z', 'en', now)).toBe('3 h ago')
    expect(adminRelativeTime('2026-06-09T12:00:00Z', 'en', now)).toBe('2 d ago')
  })

  it('couvre les bornes en EN', () => {
    expect(adminRelativeTime('2026-06-11T11:45:00Z', 'en', now)).toBe('15 min ago')
    expect(adminRelativeTime('2026-06-11T09:00:00Z', 'en', now)).toBe('3 h ago')
  })

  it('bascule sur la date locale au-delà de 7 jours', () => {
    const out = adminRelativeTime('2026-05-01T12:00:00Z', 'en', now)
    expect(out).toContain('2026')
    expect(out).not.toContain('il y a')
  })
})

describe('formatDurationMs', () => {
  it('retourne un tiret pour les valeurs absentes ou négatives', () => {
    expect(formatDurationMs(undefined)).toBe('—')
    expect(formatDurationMs(-5)).toBe('—')
    expect(formatDurationMs(Number.NaN)).toBe('—')
  })

  it('formate ms, secondes (décimale localisée), minutes et heures', () => {
    expect(formatDurationMs(850)).toBe('850 ms')
    expect(formatDurationMs(2400)).toBe('2.4 s')
    expect(formatDurationMs(2400)).toBe('2.4 s')
    expect(formatDurationMs(65_000)).toBe('1 min 05 s')
    expect(formatDurationMs(120_000)).toBe('2 min')
    expect(formatDurationMs(4_320_000)).toBe('1 h 12 min')
    expect(formatDurationMs(7_200_000)).toBe('2 h')
  })
})

describe('formatIntervalMinutes', () => {
  it('gère minutes, heures rondes et mixte', () => {
    expect(formatIntervalMinutes(undefined)).toBe('—')
    expect(formatIntervalMinutes(0)).toBe('—')
    expect(formatIntervalMinutes(15)).toBe('15 min')
    expect(formatIntervalMinutes(360)).toBe('6 h')
    expect(formatIntervalMinutes(90)).toBe('1 h 30 min')
  })
})
