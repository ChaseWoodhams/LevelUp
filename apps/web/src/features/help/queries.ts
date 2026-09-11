import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'

interface ReleaseNotesResponse {
  content: string
}

export function useReleaseNotes() {
  return useQuery({
    queryKey: queryKeys.releaseNotes(),
    queryFn: () => api.get<ReleaseNotesResponse>('/help/release-notes'),
    staleTime: 10 * 60 * 1000, // 10 min
  })
}
