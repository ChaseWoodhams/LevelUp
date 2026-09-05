/**
 * schemaVersion.ts — THE ONE VERSION THIS VIEWER CLAIMS TO UNDERSTAND.
 *
 * WHY A GUARD AT ALL, WHEN THE WEB APP HAS NONE. The app's own replay route reads whatever
 * artifact its own build step just produced: producer and consumer ship together, and its
 * contract test says so in as many words ("le client ne branche sur aucune version ; il lit
 * ce qu'il trouve"). The study archive is the opposite situation — artifacts are captured
 * hourly and kept, so an archive holds documents built by SEVERAL versions of the builder,
 * including versions older than whatever this app was last taught. Reading a v3 document
 * with v2 rules would not crash; it would draw something, and being wrong quietly is the one
 * outcome a study tool must not have.
 *
 * A BREAKING CHANGE IS THE ONLY THING THAT MOVES THIS NUMBER. The producer's own rule
 * (`document.go`) is that adding an OPTIONAL field does not increment it — so this constant
 * does not need bumping every time the artifact grows a layer, and an artifact that predates
 * a new optional field is still readable. `schemaVersion.guard.test.ts` is what keeps the
 * value honest: it reads the constant out of the Go source rather than trusting this comment,
 * which is what the ticket asks for ("read from the current source rather than hardcoded from
 * the spec text").
 *
 * WHAT AN UNRECOGNISED VERSION DOES: nothing is drawn. `parseReplayPayload` refuses to
 * normalise the document at all, so no layer, no roster and no coverage figure can reach the
 * screen from a document this viewer cannot read — the refusal is structural, not a banner
 * over a rendered map.
 */
import type { ReplayDocument } from '@/lib/api/types'

import { normalizeReplayDocument, type ReplayDocumentReady } from '../replay/replayNormalize'

/**
 * The value of `replay.SchemaVersion` in
 * `apps/go-api/internal/analysis/replay/document.go`, read there on 2026-09-04.
 * Pinned to that source by `schemaVersion.guard.test.ts` — do not edit one without the other.
 */
export const SUPPORTED_SCHEMA_VERSION = 2

/**
 * ReplayPayload — what came back from the archive, once its version has been ruled on.
 *
 * A DISCRIMINATED UNION AND NOT A THROW, because "this artifact is too new for me" is a
 * normal answer about a specific match, not a failure of the request: the fetch worked, the
 * server is fine, and the screen has something true to say. Modelling it as an error would
 * put it in the same bucket as a dead server and lose that distinction.
 */
export type ReplayPayload =
  | { kind: 'ready'; doc: ReplayDocumentReady }
  | { kind: 'unsupported'; version: number }

/**
 * parseReplayPayload rules on the version, then normalises — in that order, and the order is
 * the whole point.
 *
 * Normalisation (`normalizeReplayDocument`) is the app's nullability frontier: it fills the
 * absent arrays so nothing downstream carries a `?? []`. Running it on a document of an
 * unknown shape would produce a plausible-looking `ReplayDocumentReady` from fields that may
 * no longer mean what their names say. So the version decides FIRST, and an unrecognised one
 * never reaches the frontier.
 */
export function parseReplayPayload(raw: ReplayDocument): ReplayPayload {
  const version = raw.schemaVersion
  if (version !== SUPPORTED_SCHEMA_VERSION) return { kind: 'unsupported', version }
  return { kind: 'ready', doc: normalizeReplayDocument(raw) }
}
