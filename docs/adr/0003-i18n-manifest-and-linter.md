# ADR 0003 — English-only i18n manifests and linter

**Status** — Accepted, superseding the former two-locale contract.

## Context

User-facing strings had accumulated across feature dictionaries, JSX, API
fixtures, and manifest files. A second locale multiplied the sources and made
it easy for copy, formatting, and generated output to drift.

## Decision

TOML manifests remain the source of truth for shared UI strings. Each key has
one English value, generated TypeScript dictionaries are committed, and the
custom linter continues to reject hardcoded user-facing strings where a
manifest key exists. The runtime locale is fixed to `en`, formatting uses
`en-US`, and consumers do not select a language from URLs, headers, or query
parameters.

Title-specific labels and API metadata follow the same rule: configuration and
public DTOs contain English values and English field names. Runtime reads and
new ingestion use `en`/`en-US` only. Existing asset rows, historical database
translation columns, and migration IDs remain readable for upgrades, but they
are not selected by the active resolver and are not exposed as public
translation fields. Historical non-English seed hooks are retained as no-ops
so an upgrade never creates or mutates new non-English rows. Legacy language
route prefixes redirect to the English route and never select another locale.
Existing public static asset paths, including historical non-English filenames,
remain stable for read compatibility; new references should use canonical
English asset names when available.

## Consequences

- Manifest generation is deterministic and has no locale parity requirement.
- Generated dictionaries and API clients must be regenerated after source
  changes.
- The repository no longer maintains a translated documentation mirror.
- Compatibility code for historical database columns must not reintroduce
  language negotiation or expose translation fields in public responses.
- The OpenAPI source and generated clients describe the English-only public
  contract; regeneration is required after contract changes.
