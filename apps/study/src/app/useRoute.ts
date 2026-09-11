/**
 * useRoute.ts — the four lines of React that subscribe to the URL.
 *
 * All the reading is in `route.ts`, which is pure and tested; this file exists only to turn
 * `hashchange` into a re-render. `useSyncExternalStore` rather than an effect over `useState`
 * because the hash is exactly what it is for: a value that lives outside React and changes
 * without React being told.
 */
import { useSyncExternalStore } from 'react'

import { parseRoute, type Route } from './route'

function subscribe(onChange: () => void): () => void {
  window.addEventListener('hashchange', onChange)
  return () => window.removeEventListener('hashchange', onChange)
}

function currentHash(): string {
  return window.location.hash
}

/** The hash as it is on the server: there is none, so every render starts at home. */
function serverHash(): string {
  return ''
}

/** useRoute gives the screen the URL names, and re-renders when it changes. */
export function useRoute(): Route {
  const hash = useSyncExternalStore(subscribe, currentHash, serverHash)
  return parseRoute(hash)
}
