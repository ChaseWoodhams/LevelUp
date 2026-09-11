// Auto-genere par scripts/build_i18n_manifests.mjs - NE PAS EDITER A LA MAIN.
// Source : apps/web/src/lib/i18n/manifests/first_blood.toml

export const firstBloodManifest = {
  "first_blood.empty": { en: "No first kill or first death in this scope" },
  "first_blood.label.advance": { en: "{gap} ahead" },
  "first_blood.label.median_prefix": { en: "med." },
  "first_blood.title": { en: "First kill / first death" },
  "first_blood.tooltip.first_death": { en: "{player} · match {match} · first death {time}" },
  "first_blood.tooltip.first_kill": { en: "{player} · match {match} · first kill {time}" },
  "first_blood.tooltip.gap": { en: "median advance window: {gap}" },
  "first_blood.tooltip.median_death": { en: "median first death {time} ({n}/{total, plural, one {# match} other {# matches}})" },
  "first_blood.tooltip.median_kill": { en: "median first kill {time} ({n}/{total, plural, one {# match} other {# matches}})" },
} as const

export type FirstBloodManifestKey = keyof typeof firstBloodManifest
