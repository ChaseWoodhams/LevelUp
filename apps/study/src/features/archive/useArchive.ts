/**
 * useArchive.ts — the archive's table, fetched once, with a way to ask again.
 *
 * The same shape as `useArchivedMatch`, and for the same reason: no retry loop. The failure a
 * browser meets most often is a capture holding the database, which passes on its own schedule
 * and is made no better by being hammered; the screen says so and offers a button.
 *
 * The complete archive is loaded in pages once per visit or reload. Filtering and sorting are
 * local, so narrowing the table asks nothing of the server. Each page releases the database
 * handle; a capture taking it between requests yields the existing busy state.
 */
import { useCallback, useEffect, useState } from 'react'

import { listArchivedMatches, type ArchiveLoad } from './studyApi'

/** The screen's state: still asking, or one of the answers `listArchivedMatches` names. */
export type ArchiveScreenState = { kind: 'loading' } | ArchiveLoad

export interface Archive {
  screen: ArchiveScreenState
  /** Ask again — the same request, on the reader's word. */
  reload: () => void
}

export function useArchive(): Archive {
  const [screen, setScreen] = useState<ArchiveScreenState>({ kind: 'loading' })
  const [attempt, setAttempt] = useState(0)

  useEffect(() => {
    const controller = new AbortController()
    let live = true
    setScreen({ kind: 'loading' })
    listArchivedMatches({ signal: controller.signal })
      .then((load) => {
        if (live) setScreen(load)
      })
      .catch((err: unknown) => {
        // `listArchivedMatches` names every failure it can see, so reaching here means one it
        // could not. It still must not vanish: an unreported rejection would leave the screen
        // spinning with nothing anywhere saying why.
        if (live) setScreen({ kind: 'failed', message: err instanceof Error ? err.message : String(err) })
      })
    return () => {
      live = false
      controller.abort()
    }
  }, [attempt])

  const reload = useCallback(() => setAttempt((n) => n + 1), [])
  return { screen, reload }
}
