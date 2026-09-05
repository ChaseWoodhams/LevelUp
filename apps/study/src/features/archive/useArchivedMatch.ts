/**
 * useArchivedMatch.ts — one match, fetched once, with a way to ask again.
 *
 * NO RETRY LOOP, AND THAT IS A REQUIREMENT RATHER THAN AN OMISSION. Two of the failures this
 * screen can meet would loop forever under a naive retry policy: a match the archive has never
 * built (there will never be an artifact) and a capture holding the file (it will pass, but on
 * its own schedule, and hammering it helps nobody). Both answer with a screen that says what
 * happened and a button that asks again — a reader deciding when to retry, rather than a timer
 * deciding for them.
 *
 * ALL THE DECIDING IS IN `loadArchivedMatch`, which is pure of React and tested with fixture
 * payloads. What is left here is the part only a component can do: run it, drop the answer if
 * the reader has moved on, and let them ask again.
 */
import { useCallback, useEffect, useState } from 'react'

import { loadArchivedMatch, type MatchLoad } from './studyApi'

/** The screen's state: still asking, or one of the answers `loadArchivedMatch` names. */
export type MatchScreenState = { kind: 'loading' } | MatchLoad

export interface ArchivedMatch {
  screen: MatchScreenState
  /** Ask again — the same request, on the reader's word. */
  reload: () => void
}

export function useArchivedMatch(matchId: string): ArchivedMatch {
  const [screen, setScreen] = useState<MatchScreenState>({ kind: 'loading' })
  const [attempt, setAttempt] = useState(0)

  useEffect(() => {
    const controller = new AbortController()
    let live = true
    setScreen({ kind: 'loading' })
    loadArchivedMatch(matchId, { signal: controller.signal })
      .then((load) => {
        if (live) setScreen(load)
      })
      .catch((err: unknown) => {
        // `loadArchivedMatch` names every failure it can see, so reaching here means one it
        // could not. It still must not vanish: an unreported rejection would leave the screen
        // spinning with nothing anywhere saying why.
        if (live) setScreen({ kind: 'failed', message: err instanceof Error ? err.message : String(err) })
      })
    return () => {
      live = false
      controller.abort()
    }
  }, [matchId, attempt])

  const reload = useCallback(() => setAttempt((n) => n + 1), [])
  return { screen, reload }
}
