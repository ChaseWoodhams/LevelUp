/**
 * Libellés FR/EN du sélecteur d'escouades (presets + gestion).
 *
 * Dict local par feature (suffixe `.i18n.ts`) : source de vérité des chaînes UI
 * de useSquadPresets, hors scan FieldKey (lint-no-hardcoded-fields).
 */
export const SQUAD_PRESETS_STRINGS = {
  en: {
    squadsHeader: 'My squads',
    groupsHeader: 'My groups',
    save: 'Save lineup',
    saving: 'Saving…',
    saved: 'Lineup already saved',
    saveSuccess: 'Squad saved',
    saveError: 'Save failed',
    manage: 'Manage',
    done: 'Done',
    rename: 'Rename',
    ok: 'OK',
    del: 'Delete',
    confirmDelete: 'Confirm?',
    usualPrefix: 'mostly',
  },
}

export type SquadPresetsStrings = (typeof SQUAD_PRESETS_STRINGS)['en']
