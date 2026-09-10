# Instructions for GitHub Copilot and AI assistants

LevelUp is a Go backend and React/TypeScript frontend for Halo statistics.
The active product contract is English-only: `en` is the only supported locale,
all user-facing strings are English, and API DTOs use English field names.

Use the repository instructions in `AGENTS.md`, `CLAUDE.md`, and the relevant
files under `docs/agents/`. Preserve the existing architecture and domain
vocabulary. Prefer the established Go, TypeScript, and PowerShell tooling.

Before editing, inspect the surrounding code and existing tests. After editing,
run the narrowest relevant checks, then the app typecheck or Go package tests.
Regenerate checked-in OpenAPI, route, or i18n output with its repository
generator instead of editing generated files by hand.

Do not add locale negotiation, language-specific routes, or new localization
branches. Preserve historical database columns or migrations only when needed
for upgrade compatibility; new writes and API responses must use English fields
and values.
