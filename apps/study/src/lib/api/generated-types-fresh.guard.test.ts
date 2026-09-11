/// <reference types="node" />
// @vitest-environment node
/**
 * generated-types-fresh.guard.test.ts — `generated.ts` DERIVES from the contract.
 *
 * WHY THIS APP NEEDS ITS OWN. `apps/web` already has this guard, and it is pinned to
 * `apps/web` in the shared script. This app generates its types from the SAME
 * `apps/go-api/api/openapi.yaml`, so it inherits the same failure mode: a contract that
 * moves without anyone re-running `generate-types` leaves the viewer typed against the
 * OLD replay document, with `tsc` perfectly green — stale types are not incoherent
 * types, they are just wrong about the world.
 *
 * That failure would be worse here than in the web app: `replayContract.test.ts` proves
 * the nullability frontier is complete AGAINST THE CONTRACT, and a stale `generated.ts`
 * would turn that proof into a proof about last month's contract.
 *
 * The logic stays in `tools/check-generated-types-fresh.mjs`, called with this app's
 * directory. One implementation, two apps — the script took a parameter rather than a
 * second copy.
 *
 * Fixing a failure:  cd apps/study && npm run generate-types
 * (preceded by `make openapi-gen` if the Go contract is what moved).
 */
import { describe, expect, it } from 'vitest'
// @ts-expect-error — Node script outside this app's tsc program (ESM/TS boundary).
import { checkGeneratedTypesFresh } from '../../../../../tools/check-generated-types-fresh.mjs'

describe('generated contract', () => {
  it(
    're-running openapi-typescript produces NO diff',
    () => {
      const problem = checkGeneratedTypesFresh('apps/study') as string | null
      expect(problem, problem ?? '').toBeNull()
    },
    30_000, // the generator re-reads the whole contract (~1 s locally, margin for CI)
  )
})
