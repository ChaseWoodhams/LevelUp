/**
 * Queries TanStack Query — Settings (Sprint 51 : split depuis setup/queries.ts).
 *
 * Contient uniquement les hooks liés à la configuration applicative.
 * Ne pas importer depuis features/setup/.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'
import type {
  AsyncJobStatus,
  BackfillStartRequest,
  BackupRunResult,
  BackupStatusResponse,
  PlayersListResponse,
  SettingsResponse,
  UpdateSettingsRequest,
} from '@/lib/api/types'

export function useSettings() {
  return useQuery({
    queryKey: queryKeys.settings,
    queryFn: () => api.get<SettingsResponse>('/settings'),
    staleTime: 5 * 60 * 1000,
    select: (data: SettingsResponse): SettingsResponse => ({
      ...data,
      lang: 'en',
      discord_lang: 'en',
    }),
  })
}

export function useUpdateSettings() {
  const qc = useQueryClient()
  const demoMode = useAppShellStore((s) => s.demoMode)
  return useMutation({
    mutationFn: async (req: UpdateSettingsRequest) => {
      // Demo settings are read-only; language fields are always English.
      if (demoMode) {
        const current =
          qc.getQueryData<SettingsResponse>(queryKeys.settings) ?? ({} as SettingsResponse)
        return { ...current, lang: 'en', discord_lang: 'en' }
      }
      const settings: UpdateSettingsRequest = { ...req }
      delete settings.lang
      delete settings.discord_lang
      return api.patch<SettingsResponse>('/settings', settings)
    },
    onSuccess: (data) => {
      qc.setQueryData(queryKeys.settings, { ...data, lang: 'en', discord_lang: 'en' })
    },
  })
}

export function useScanMedia() {
  return useMutation({
    mutationFn: () => api.post<unknown>('/settings/media/scan', {}),
  })
}

export function useRecalculateSessions() {
  return useMutation({
    mutationFn: () => api.post<{ job_id: string }>('/settings/sessions/recalculate', {}),
  })
}

export function useStartBackfill() {
  return useMutation({
    mutationFn: (req: BackfillStartRequest) =>
      api.post<AsyncJobStatus>('/backfill/start', req),
  })
}

export function useBackupStatus() {
  return useQuery({
    queryKey: [...queryKeys.settings, 'backup-status'],
    queryFn: () => api.get<BackupStatusResponse>('/settings/backup/status'),
    staleTime: 30 * 1000,
  })
}

export function useRunBackup() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => api.post<BackupRunResult>('/settings/backup/run', {}),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: [...queryKeys.settings, 'backup-status'] })
    },
  })
}

// ─── Sélection par titre (onglet « Jeux ») ───────────────────────────────────

export interface PlayerTitleStatus {
  slug: string
  name: string
  status: 'active' | 'coming_soon' | 'archived'
  enrolled: boolean
  syncEnabled: boolean
}

interface TitleSyncResult {
  gamertag: string
  title_slug: string
  sync_enabled: boolean
}

interface TitlePurgeResult {
  gamertag: string
  title_slug: string
  data_removed: boolean
}

/**
 * usePlayerTitles agrège, pour le joueur courant, le statut sync par titre.
 * Source : GET /players par titre (header X-LevelUp-Title override). GET /players
 * filtre par titre courant, donc on itère les titres disponibles (Option B, N
 * petit). Chaque appel est résilient : un titre en erreur → enrolled=false.
 */
export function usePlayerTitles() {
  const availableTitles = useAppShellStore((s) => s.availableTitles)
  const playerSlug = useAppShellStore((s) => s.currentPlayer?.player_slug ?? '')
  return useQuery({
    queryKey: queryKeys.playerTitles(playerSlug),
    enabled: playerSlug !== '',
    queryFn: async (): Promise<PlayerTitleStatus[]> => {
      const titles = availableTitles.filter((t) => t.status !== 'archived')
      return Promise.all(
        titles.map(async (t): Promise<PlayerTitleStatus> => {
          try {
            const resp = await api.get<PlayersListResponse>('/players', { 'X-LevelUp-Title': t.slug })
            const entry = (resp.items ?? []).find((p) => p.player_slug === playerSlug)
            return {
              slug: t.slug,
              name: t.name,
              status: t.status,
              enrolled: entry != null,
              syncEnabled: entry?.sync_enabled ?? false,
            }
          } catch {
            return { slug: t.slug, name: t.name, status: t.status, enrolled: false, syncEnabled: false }
          }
        }),
      )
    },
  })
}

/** useSetTitleSync bascule actif/pause d'un titre du joueur courant. */
export function useSetTitleSync() {
  const qc = useQueryClient()
  const playerSlug = useAppShellStore((s) => s.currentPlayer?.player_slug ?? '')
  return useMutation({
    mutationFn: ({ slug, enabled }: { slug: string; enabled: boolean }) =>
      api.patch<TitleSyncResult>(
        `/profiles/${encodeURIComponent(playerSlug)}/titles/${encodeURIComponent(slug)}/sync`,
        { enabled },
      ),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.playerTitles(playerSlug) })
      void qc.invalidateQueries({ queryKey: queryKeys.bootstrap })
    },
  })
}

/** usePurgeTitleData supprime le profil + les données d'un titre du joueur courant. */
export function usePurgeTitleData() {
  const qc = useQueryClient()
  const playerSlug = useAppShellStore((s) => s.currentPlayer?.player_slug ?? '')
  return useMutation({
    mutationFn: ({ slug }: { slug: string }) =>
      api.delete<TitlePurgeResult>(
        `/profiles/${encodeURIComponent(playerSlug)}/titles/${encodeURIComponent(slug)}/data`,
      ),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.playerTitles(playerSlug) })
      void qc.invalidateQueries({ queryKey: queryKeys.bootstrap })
    },
  })
}
