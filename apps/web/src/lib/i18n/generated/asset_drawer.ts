// Auto-genere par scripts/build_i18n_manifests.mjs - NE PAS EDITER A LA MAIN.
// Source : apps/web/src/lib/i18n/manifests/asset_drawer.toml

export const assetDrawerManifest = {
  "asset_drawer.empty.error": { en: "Loading error." },
  "asset_drawer.empty.loading": { en: "Loading…" },
  "asset_drawer.empty.maps": { en: "No map found." },
  "asset_drawer.empty.medals": { en: "No medal found." },
  "asset_drawer.empty.weapons": { en: "No weapon found." },
  "asset_drawer.mini_tab": { en: "References" },
  "asset_drawer.search.placeholder": { en: "Search…" },
  "asset_drawer.tab.maps": { en: "Maps" },
  "asset_drawer.tab.medals": { en: "Medals" },
  "asset_drawer.tab.weapons": { en: "Weapons" },
  "asset_drawer.toggle.close": { en: "Close visual reference" },
  "asset_drawer.toggle.open": { en: "Open visual reference" },
} as const

export type AssetDrawerManifestKey = keyof typeof assetDrawerManifest
