/**
 * Queries qualité données — compteurs d'inconnus + listes détaillées.
 * Pas de polling continu : staleTime 60 s + refetch au focus + invalidation
 * après chaque action de résolution.
 */
import { keepPreviousData, useQuery } from '@tanstack/react-query'

import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type {
  AdminDataQualityCounts,
  AdminDataQualityIssues,
  DataQualityIssueKind,
} from '@/lib/api/types'

// English is the only metadata language used by the data quality dashboard.
export const DATA_QUALITY_LOCALE = 'en'

export function useDataQualityCounts() {
  return useQuery({
    queryKey: queryKeys.adminDataQuality,
    queryFn: () =>
      api.get<AdminDataQualityCounts>('/admin/monitoring/data-quality'),
    staleTime: 60_000,
    retry: false,
  })
}

export function useDataQualityIssues(kind: DataQualityIssueKind, limit = 50, offset = 0) {
  return useQuery({
    queryKey: queryKeys.adminDataQualityIssues(kind, limit, offset),
    queryFn: () =>
      api.get<AdminDataQualityIssues>(
        `/admin/monitoring/data-quality/issues?kind=${kind}&limit=${limit}&offset=${offset}`,
      ),
    staleTime: 60_000,
    retry: false,
    // Pagination serveur : garder la page précédente affichée le temps du fetch
    // de la suivante (pas de flash de table vide entre deux pages).
    placeholderData: keepPreviousData,
  })
}
