# English-only audit inventory — 2026-09-06

## Implementation status

The audit findings were implemented in the working tree on 2026-09-06. The
active web, study, API, configuration, metadata-ingestion, documentation, and
operator surfaces now use English only. The generated OpenAPI contract and
both TypeScript clients were refreshed, and the frontend and affected backend
checks pass.

The remaining French markers are intentional compatibility inventory: legacy
database column names and values, historical migration IDs and filenames,
existing asset paths, and fixtures that prove old data can still be read.
Those paths are read-compatible only. New migrations and ingestion do not add
non-English rows, and the runtime resolver uses `en`/`en-US` exclusively.

Validation performed during implementation includes frontend typechecks,
focused Go packages, the migration suite, the DuckDB repository suite, OpenAPI
regeneration, generated-client regeneration, and live HTTP checks after the
final service restart. Re-run scans must exclude this audit file because its
purpose is to retain the original marker inventory.

## Cadrage

This is an inventory campaign for the current LevelUp repository. The requested
end state is English-only: no French UI, French metadata, French locale
negotiation, French documentation branch, French operator messages, or French
asset naming that is part of the product contract.

The audit covered:

- the web app, study app, API, CLI tools, and operator-facing output;
- title configuration, metadata ingestion, migrations, database-facing
  translation columns, and API contracts;
- documentation, agent notes, issue guidance, generated artifacts, fixtures,
  tests, and CI/E2E configuration;
- static asset names and tracked data artifacts.

The working tree was already dirty before this audit. Existing changes were
preserved. This file remains the inventory artifact for the campaign; the
implementation changes are recorded in the repository files and generated
outputs described above.

The repository currently has two conflicting policies:

- CLAUDE.md requires French responses and French-first UI terminology.
- docs/adr/0003-i18n-manifest-and-linter.md defines French and English as a
  deliberate two-locale contract.

Both policies were replaced by the English-only contract described above. The
findings below preserve the original inventory and required-edit record.

Evidence scans used during the campaign included:

    rg -l --hidden --glob '!.git/**' --glob '!node_modules/**' --glob '!data/**' --glob '!static/**' --glob '!*.bin' --glob '!*.png' --glob '!*.jpg' --glob '!*.jpeg' --glob '!*.webp' --glob '!*.zip' --glob '!*.ekur*' --glob '!*.duckdb' --glob '!*.parquet' "[À-ÿœæ]" .

    rg -l --hidden --glob '!.git/**' --glob '!node_modules/**' --glob '!data/**' --glob '!static/**' --glob '!*.bin' --glob '!*.png' --glob '!*.jpg' --glob '!*.jpeg' --glob '!*.webp' --glob '!*.zip' --glob '!*.ekur*' --glob '!*.duckdb' --glob '!*.parquet' "(?i)français|francaise|francais|traduction|localisation|libellé|libellés|en français|version française|locale FR|FR-first|fr-FR|label_fr|name_fr|description_fr|docs/FR|docs-fr" .

    rg -n "fr-FR|label_fr|name_fr|description_fr|title_fr|condition_fr|LocaleFR|docs/FR|X-LevelUp-Locale" apps config docs scripts README.md CLAUDE.md .ai .github

The first scan found 4,891 files containing accented text. The second found
941 files containing explicit French or localization markers. These are
heuristics rather than a language detector: proper names, copied data, and
identifiers need manual classification. Binary images, archives, DuckDB files,
and Parquet files require separate inspection.

The complete file lists for each edit group can be regenerated with:

    git ls-files docs/FR
    rg -l '^\s*fr\s*=' apps/web/src/lib/i18n/manifests
    rg -l 'label_fr|description_fr|title_fr|condition_fr|name_fr' config apps/go-api
    rg -l 'fr-FR|LocaleFR|X-LevelUp-Locale|docs/FR' apps config docs scripts README.md CLAUDE.md .ai .github
    git ls-files 'static/commendations/halo_5_guardians/*'
    git ls-files data

These commands are intentionally part of the registry: the repository contains
generated and dirty-worktree files, so a future implementation must regenerate
the exact impacted set from the post-contract source rather than rely on a
hand-maintained filename list.

## Findings retained

### E1 — The frontend locale contract is French plus English

Priority: P1. This is a runtime blocker for an English-only product.

Audit rule: active application locale contracts and user-facing dictionaries
must expose English only. Consequence: the current store can select or
reconstruct French UI at runtime. Reproduction: inspect KNOWN_LOCALES, the
store default, and the manifest entries listed below.

Where:

- apps/web/src/lib/i18n/locale.ts declares KNOWN_LOCALES as fr and en.
- apps/study/src/lib/i18n/locale.ts copies the same two-locale contract.
- apps/web/src/stores/appShellStore.ts defaults and hydrates locale as fr.
- apps/study/src/App.tsx renders the locale switch.
- apps/web/src/lib/i18n/manifests/ contains 21 TOML manifests with 2,964
  French entries paired with English entries.
- apps/web/src/lib/i18n/generated/ contains the corresponding generated
  bilingual TypeScript dictionaries.
- Inline dictionaries remain in web feature files and in study i18n files,
  including replay, viewer, shell, settings, squad, review, match view,
  explorer, achievements, and chart review.

Required edits:

1. Make en the only Locale value and the only KNOWN_LOCALES entry in both
   applications.
2. Remove the locale selector and all state, persistence, and hydration logic
   that exists only to select French.
3. Keep the English values in the 21 manifests, delete the French values, and
   regenerate the 21 generated dictionaries using the repository generator.
4. Collapse inline dictionaries to a single English value or a single English
   object.
5. Simplify locale-dependent formatters, skill tier labels, calendar labels,
   page titles, review text, and catalog label selection to English-only code.
6. Replace the French-oriented no-anglicisms guard and locale ratchet with an
   English-only invariant.
7. Update all web and study tests that set locale to fr, exercise French
   fallback behavior, or assert bilingual parity.

Adverse verification: after the edit, a runtime search for locale branches,
French fallback values, and bilingual object shapes must return no active
application paths. Generated freshness checks must still pass.

### E2 — URL routing and API requests carry a selectable language

Priority: P1. Existing links and API requests can select French even if the
visible default is changed.

Audit rule: public routes and requests must not carry a selectable French
variant. Consequence: old links and headers can reintroduce French behavior
after a default change. Reproduction: open the optional language route and
inspect setApiLocale and X-LevelUp-Locale.

Where:

- apps/web/src/routes/{-$lang}/t/$titleSlug.tsx introduces an optional language
  route segment and routeTree.gen.ts contains its generated representation.
- apps/web/src/lib/title-routing/useLangParam.ts and
  initTitleFromLocation.ts read and propagate the language segment.
- apps/web/src/lib/api/client.ts sends X-LevelUp-Locale.
- title middleware resolves that header into request context.
- API handlers accept locale or lang query parameters for home, help, and
  field mappings.
- apps/web/src/app/providers/document-lang-provider.tsx and
  apps/web/src/lib/formatters/intlLocale.ts map the app locale to fr-FR or
  en-US.

Required edits:

1. Choose one stable English URL shape. Remove the optional language segment
   from the route tree, or retain it only as an English compatibility redirect.
2. Remove useLangParam, setApiLocale, the locale request header, and locale
   query handling when they have no remaining English-only purpose.
3. If old /fr links must remain reachable, add an explicit redirect to the
   English route and test that it never renders French.
4. Make document language and Intl formatting permanently en-US.
5. Regenerate routeTree.gen.ts rather than editing generated route output by
   hand.
6. Remove the language-specific route, provider, client, and handler tests;
   replace them with the English route and English request contract.

Adverse verification: crawl old and current routes, inspect outgoing API
requests, and verify that no request or response can select a French variant.

### E3 — The backend domain and HTTP contract contain French fields and locale
negotiation

Priority: P1. This is an API and stored-data compatibility change.

Audit rule: the public contract and serialized domain model must contain only
English fields and defaults. Consequence: clients and responses still expose
French fields even when the UI is English. Reproduction: inspect the OpenAPI
schemas, generated clients, locale middleware, and LocaleFR references.

Where:

- apps/go-api/internal/api/middleware/title.go resolves locale from
  X-LevelUp-Locale.
- Home, help, field-mapping, pattern, and admin data-quality handlers use
  French defaults or locale branches.
- apps/go-api/internal/games/mappings/types.go defines LocaleFR and loaders
  require English and French labels and descriptions.
- Domain responses contain fields such as name_fr, description_fr,
  next_rank_name_fr, max_rank_name_fr, tier_name_fr, label_fr, and
  map_name_fr.
- apps/go-api/api/openapi.yaml and openapi_manual_fragment.yaml document the
  bilingual fields, locale enums, French defaults, and translation endpoints.
- apps/web/src/lib/api/generated.ts and
  apps/study/src/lib/api/generated.ts reproduce those fields and enums.
- OG metadata code defines LocaleFR and defaults to it.
- openapi documentation HTML currently declares a French document language.

Required edits:

1. Replace the locale contract with an English-only constant or remove it
   where no language choice is needed.
2. Remove French response fields from domain structs, wire models, OpenAPI
   schemas, handlers, serializers, and generated clients.
3. Remove locale query parameters and French defaults from handlers.
4. Simplify mappings loaders, weapon labels, medal labels, rank labels, and
   outcome labels to English values.
5. Remove the French OG metadata branch and make generated social metadata
   English.
6. Change generated documentation language to en.
7. Regenerate OpenAPI and both TypeScript API clients from the new contract.
8. Update contract tests, schema semantics tests, replay golden tests, and
   fixtures so they prove the English contract only.

Compatibility decision required during implementation: removing fields or
translation columns can break clients and existing databases. Decide whether
to preserve historical migrations for upgrade compatibility, add a forward
cleanup migration, or perform a coordinated schema/version cut. The chosen
decision must be recorded in an ADR before deleting migration history.

Adverse verification: compare generated clients to the OpenAPI source, query
the affected endpoints with old locale parameters, and inspect serialized
responses for any French field or locale value.

### E4 — Title configuration and metadata ingestion store French content

Priority: P1. French values are part of the data source and ingestion pipeline,
not only presentation.

Audit rule: active title configuration must contain only canonical English
labels and descriptions. Consequence: a clean ingest can recreate French
metadata from configuration. Reproduction: scan title TOML for paired locale
keys and *_fr fields.

Where:

- config/titles/halo_5/mappings/asset_labels_fr.toml is a standalone French
  mapping file.
- Halo 5 and Halo Infinite fields, assets, outcomes, and weapon mappings use
  paired en and fr values.
- Halo Infinite challenge templates use label_fr and description_fr.
- Halo Infinite milestones use title_fr and condition_fr.
- Halo 5 milestones use title_fr.
- Halo Infinite arc presets use title_fr and description_fr.
- Halo Infinite regulation configuration contains French explanatory comments.
- Halo 5 playlist mappings include a raw French playlist key used as an
  override.
- The title-specific mapping loaders and their tests enforce the paired
  locale schema.

Required edits:

1. Delete the French-only asset mapping file after moving any required English
   values into the canonical English mapping.
2. Remove fr entries and *_fr fields from all title TOML configuration.
3. Replace the French playlist override with a stable English key or a
   canonical identifier.
4. Remove paired-locale validation from the mapping loader and require only
   English values.
5. Update mapping fixtures and loader tests to the English schema.
6. Review comments in TOML files for French wording and rewrite them in
   English.
7. Add a configuration validation check that rejects French locale keys in
   active title configuration.

Adverse verification: load every title configuration from a clean process and
verify that no active configuration file contains locale-paired labels,
French-only filenames, or a French fallback key.

### E5 — Database repositories, migrations, and fetch commands implement French
data paths

Priority: P1. Removing the UI branch without changing these paths leaves French
data in new databases and in read resolution.

Audit rule: ingestion and persistence must produce and resolve English data
only. Consequence: French rows and columns survive in new or upgraded
databases. Reproduction: inspect h5-metadata-fetch, resolver preference order,
DuckDB queries, and French migrations.

Where:

- apps/go-api/cmd/h5-metadata-fetch/main.go performs French metadata fetches and
  seeds French weapon, team-color, commendation, and asset translations.
- apps/go-api/internal/domain/asset_langs.go lists fr-FR among target
  languages.
- apps/go-api/internal/assetnames/resolver.go defaults to fr-FR and en-US.
- metadata repository resolution prefers fr-FR before en-US.
- DuckDB repositories and queries use *_fr columns, French-first fallback
  order, and locale-specific translation tables.
- French-specific migrations and tests include:
  mode_playlist_fr.go, playlist_fr_test.go,
  steps_metadata_playlist_fr.go, steps_metadata_playlist_fr_test.go,
  steps_metadata_purge_weapons_name_fr.go, and its test.
- Weapon registry code still contains name_fr schema and seed references
  despite comments describing a later removal.
- Admin data-quality handlers expose translation upsert endpoints and French
  DTO fields.
- CLI commands for rank translations, career rank refresh, metadata repair, and
  asset population accept or produce localized values.

Required edits:

1. Make the asset target-language list and resolver defaults English-only.
2. Remove French-first ordering from every metadata repository and query.
3. Remove French fetch, seed, and fallback passes from h5-metadata-fetch.
4. Remove or migrate *_fr columns and translation rows according to the
   compatibility decision in E3.
5. Remove translation-admin endpoints and DTOs, or convert the feature into
   English-only metadata quality checks.
6. Remove French-specific migration code from the active migration plan only
   after deciding how existing installations upgrade.
7. Update migration tests, repository tests, and integration fixtures to use
   English-only rows.
8. Remove French flags and output from rank, career, repair, and population
   commands.
9. Rebuild or purge existing data artifacts after the schema and ingestion
   changes.

Adverse verification: initialize a clean database, run metadata ingestion,
inspect tables and views for French columns/rows, then upgrade a representative
old database and verify that English data remains readable.

### E6 — The applications contain French user-facing strings outside manifests

Priority: P1. A manifest-only change would leave visible French behavior.

Audit rule: every user-facing state must render English text in normal,
empty, loading, and error paths. Consequence: users can still see French in
feature-specific UI and server refusals. Reproduction: scan inline dictionaries
and exercise their feature tests and E2E states.

Where:

- web inline dictionaries include feature labels, match-view encounter
  explanations, heatmap labels, combat profile labels, filter labels, settings,
  squad presets, review questions, and calendar text.
- study app i18n includes shell, replay, and viewer dictionaries.
- study replay catalog selection chooses label.en or label.fr.
- chart-review contains French review questions and result text.
- study-server filter refusal messages are French.
- web E2E setup and mocks default to French and pin fr-FR.

Required edits:

1. Retain or rewrite the English branch in each inline dictionary and remove
   the French branch.
2. Simplify helpers that choose between English and French values.
3. Translate any business copy that has no English counterpart; do not silently
   substitute identifiers for user-facing text.
4. Change server refusal, validation, loading, empty-state, and error strings
   to English.
5. Update mocks, E2E environment variables, snapshots, and assertions to use
   English defaults.
6. Verify browser document title, accessibility labels, tooltips, chart text,
   and error states in both primary apps.

Adverse verification: exercise every route and major empty/error state in the
web and study apps with a clean English profile and search rendered output for
French words and punctuation patterns.

### E7 — CLI, logs, comments, and operator documentation contain French text

Priority: P2 for comments and notes; P1 for operator-visible output.

Audit rule: repository text and operator output must be English unless it is an
approved external value or proper noun. Consequence: operators and maintainers
still receive French messages and instructions. Reproduction: repeat the
accented-text and explicit-marker scans and inspect CLI output.

Where:

- apps/go-api/cmd/admin/main.go contains French usage, errors, and success
  messages.
- Other Go commands and scripts contain French log messages and comments,
  including metadata, archive, refresh, and repair tooling.
- The repository-wide accented-text scan found 4,891 matching files, including
  large groups under apps/go-api, apps/web, apps/study, docs, and .ai.
- The explicit localization-marker scan found 941 matching files.

Required edits:

1. Translate all operator-visible CLI output, errors, help, and log messages to
   English.
2. Translate comments and docstrings where the repository policy requires all
   repository text to be English.
3. Preserve protocol values, proper nouns, title names, and game terminology
   when they are canonical identifiers rather than French prose.
4. Add a reviewable allowlist for unavoidable external strings instead of
   suppressing the whole scan.

Adverse verification: run the two broad scans again after excluding only
generated/binary artifacts and this audit registry, then manually classify every
remaining match.

### E8 — Documentation, agent guidance, notes, and CI explicitly preserve French

Priority: P2 for historical notes; P1 for active instructions and developer
workflows.

Audit rule: active repository guidance must describe one English-only policy.
Consequence: future changes can restore French through contradictory agent,
documentation, hook, or CI instructions. Reproduction: follow README and docs
links, read CLAUDE.md, and inspect the docs synchronization hook.

Where:

- docs/FR contains 18 tracked French documents, including a French ADR mirror.
- docs/README_FR.md redirects to the French documentation branch.
- README.md links to the French README and French docs.
- docs/CONTRIBUTING.md and docs/SYNC_GUIDE.md describe the French mirror.
- CLAUDE.md contains French response, UI, and docs/FR policy rules.
- .ai/I18N_REFERENCE.md is a French localization guide and describes the
  two-locale product contract.
- .github/copilot-instructions.md contains stale French UI and docstring
  guidance.
- scripts/git-hooks/lefthook/docs-fr-sync.sh synchronizes docs with docs/FR.
- CI and E2E documentation describe French-pinned browser checks.
- .ai notes and archived plans contain French implementation notes and
  localization decisions.
- docs/adr/0003 documents French plus English as the intended architecture.

Required edits:

1. Remove docs/FR and docs/README_FR.md if historical French documentation is
   not required; otherwise translate it and move it into the English docs tree.
2. Remove French links, mirror instructions, and synchronization hooks.
3. Rewrite CLAUDE.md, agent guidance, and active ADR language to state the
   English-only contract.
4. Replace or archive I18N_REFERENCE.md with an English-only localization
   policy that documents the absence of runtime language selection.
5. Translate active notes and Copilot instructions; clearly mark historical
   records if they must remain for provenance.
6. Update CI, E2E, release, and contribution documentation to remove French
   locale setup.

Adverse verification: build documentation links, run repository hooks, and
search active instructions for French locale policy before merging.

### E9 — Static asset names and prebuilt artifacts preserve French identifiers

Priority: P2, with P1 impact where filenames are product lookup keys.

Audit rule: product asset identifiers must be stable and language-neutral or
English. Consequence: localized filenames can break lookups, URLs, or rebuilds.
Reproduction: enumerate the H5 commendation directory and trace each filename
through static handlers and tests.

Where:

- static/commendations/halo_5_guardians contains a distinct French-named subset
  of tracked PNG files, including localized commendation and weapon labels.
- Static asset handlers and tests expose these paths through the application.
- data contains 296 tracked files, mostly cached rank images and prebuilt
  metadata archives or database artifacts.
- Existing generated public assets and the dirty worktree may contain files
  that are not represented by source scans.

Required edits:

1. Decide whether static filenames are external URLs. If they are internal,
   rename them to stable English or identifier-based names and update every
   handler, manifest, fixture, and test reference.
2. If they are public URLs, preserve redirect compatibility while changing
   canonical paths.
3. Inspect archives, DuckDB files, and Parquet artifacts with their native
   tools; remove or regenerate French metadata rather than relying on text
   grep.
4. Rebuild generated public assets and verify that no French filename is
   referenced by source or served by the static handler.

Adverse verification: enumerate static references, open representative
commendation URLs, and scan regenerated data with SQL or archive-aware tools.

### E10 — Generated artifacts, fixtures, and golden tests will otherwise recreate
French

Priority: P1 for build correctness.

Audit rule: generated output and tests must encode the active English-only
source contract. Consequence: regeneration or CI can restore French fields,
routes, and fixtures. Reproduction: run generated freshness guards and inspect
the generated client, manifest, route, and golden-test outputs.

Where:

- OpenAPI clients in both apps contain French fields and locale enums.
- i18n generated dictionaries contain paired fr/en objects.
- routeTree.gen.ts contains the language route.
- replay, medal, career, and analysis golden fixtures contain bilingual labels.
- Go and TypeScript tests explicitly assert French defaults, French fallback
  order, French request parameters, and bilingual parity.
- Generated freshness and ratchet tests encode the old two-locale policy.

Required edits:

1. Change source configuration and schemas first.
2. Regenerate OpenAPI clients, i18n dictionaries, route output, and any
   checked-in derived artifacts using repository generators.
3. Remove French fixtures and rewrite expected output to English.
4. Delete tests that only prove French behavior; replace them with tests for
   English fallback, URL redirects if retained, and schema absence.
5. Run the full relevant Go, web, study, contract, and E2E suites after
   regeneration.

Adverse verification: run every generated-freshness guard and inspect the
generated diff for reintroduced locale fields.

## Findings excluded from the retained set

| Candidate | Why it was excluded |
| --- | --- |
| Halo, Spartan, weapon names, map names, and other game proper nouns | They are canonical product data and are not French solely because they contain accented characters or shared vocabulary. |
| Matches for short text such as fr inside framework, fresh, or from | These are lexical false positives and do not identify a French surface. |
| Binary image, archive, DuckDB, and Parquet contents | The text scan cannot establish their language; they remain an explicit follow-up in E9. |
| Existing dirty-worktree implementation changes | They were pre-existing and were not edited or evaluated as part of this language audit. |
| General code correctness, performance, security, or domain-model quality | Those are separate review axes and outside this campaign. |

## Axes without a retained finding

No separate correctness or performance audit was performed. No claim is made
that the broad text scan found every French string, or that every accented
string is French. The final implementation must combine the marker scans with
manual review of user-facing output and native inspection of stored artifacts.

## Required implementation sequence

1. Record the English-only product and repository contract in active guidance
   and an ADR. Decide the database/API compatibility policy for removing
   French fields and migrations.
2. Convert title configuration, mapping loaders, metadata resolution, and
   ingestion to English-only.
3. Remove frontend locale state, selectors, dictionaries, route language
   segment, API language propagation, and document language branching.
4. Remove backend locale negotiation, French domain fields, translation
   handlers, and French OG/API behavior.
5. Regenerate OpenAPI clients, i18n dictionaries, route output, and other
   derived files.
6. Update database migrations and rebuild or clean prebuilt data artifacts.
7. Rename or redirect French static asset paths and update references.
8. Translate active CLI output, comments, notes, documentation, and CI
   instructions; remove the French docs mirror workflow.
9. Update tests and fixtures, then run the full validation suite.
10. Repeat the marker scans, generated-freshness checks, documentation-link
    checks, and native data scans before declaring the repository English-only.

## Validation gates for the eventual implementation

- No active source or configuration reference to fr-FR, LocaleFR, name_fr,
  description_fr, label_fr, title_fr, condition_fr, or a French default.
- No runtime locale selector or language request header.
- OpenAPI, generated clients, generated i18n, and route output are fresh.
- Clean and upgraded databases contain only English metadata columns and rows.
- Web and study builds, focused tests, contract tests, and E2E checks pass.
- Static handlers serve only canonical English or identifier-based asset paths.
- Active docs, agent instructions, CI, and hooks contain no French policy.
- Remaining accented text is manually classified as a proper noun, external
  data, or an approved exception.

## Suite

This registry is complete for the requested inventory campaign. Implementation
should proceed as a separate change set so the API/data compatibility decision,
generated-file changes, and application behavior can each be reviewed.
