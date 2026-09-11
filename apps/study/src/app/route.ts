/**
 * route.ts — WHERE THE READER IS, read from the URL and nothing else.
 *
 * HASH ROUTING, DELIBERATELY, AND NOT THE WEB APP'S ROUTER. `apps/web` uses TanStack Router
 * because it has dozens of routes, typed search params, loaders and a file-based tree. This
 * app has two screens and one parameter. A hash also buys the property that matters most for
 * a tool served off a plain static build: every URL resolves without a server rewrite rule, so
 * `npm run preview`, a `file://` open and the dev server all behave the same.
 *
 * THE PARSING IS PURE AND LIVES HERE, not in the component, so the one case that has ever
 * gone wrong in a hand-rolled router — an identifier that needs escaping — is a unit test
 * rather than a thing to remember. `useRoute.ts` is the four lines of React that subscribe to
 * it.
 */

/** A screen of the study tool. */
export type Route =
  /** The landing screen: how to open a match, and the sample. */
  | { kind: 'home' }
  /** One archived match, by the identifier the archive filed it under. */
  | { kind: 'match'; matchId: string }
  /**
   * The hand-written artifact, drawn through the same viewer.
   *
   * IT IS A ROUTE AND NOT A TEST FIXTURE ONLY, because the viewer has to be reviewable
   * without an archive: the sample is the one document that puts every layer on screen —
   * lives that end and respawn, shots with and without a heading, a player the roster has no
   * row for — and it does it with no server running and no film captured.
   */
  | { kind: 'sample' }

const MATCH_PREFIX = '#/match/'
const SAMPLE_HASH = '#/sample'

/**
 * parseRoute reads a `location.hash`.
 *
 * ANYTHING UNRECOGNISED IS HOME, never an error screen: a hash is user-editable text, and a
 * typo in it is not a fault to report — it is a reader who has landed nowhere and needs the
 * way in. An empty identifier (`#/match/`) is the same case.
 */
export function parseRoute(hash: string): Route {
  if (hash === SAMPLE_HASH) return { kind: 'sample' }
  if (hash.startsWith(MATCH_PREFIX)) {
    const matchId = safeDecode(hash.slice(MATCH_PREFIX.length))
    if (matchId !== '') return { kind: 'match', matchId }
  }
  return { kind: 'home' }
}

/** matchHref is the link to one archived match, escaped for the hash. */
export function matchHref(matchId: string): string {
  return `${MATCH_PREFIX}${encodeURIComponent(matchId)}`
}

/** sampleHref is the link to the hand-written artifact. */
export function sampleHref(): string {
  return SAMPLE_HASH
}

/** homeHref is the link back to the landing screen. */
export function homeHref(): string {
  return '#/'
}

/**
 * safeDecode undoes the escaping, and never throws.
 *
 * `decodeURIComponent` rejects a lone `%` — which a hand-edited URL produces without trying.
 * Reading such a hash as its literal text is right: the identifier will simply not be found,
 * which is the honest outcome, where an exception here would take down the whole screen.
 */
function safeDecode(segment: string): string {
  try {
    return decodeURIComponent(segment)
  } catch {
    return segment
  }
}
