# Skill : db-schema — Schéma DuckDB LevelUp

## Structure des chemins — multi-titres

Tous les chemins passent par `PathResolver` (`internal/domain/title/registry.go`).
**Ne jamais construire de chemin manuellement** avec `filepath.Join(repoRoot, "data", ...)`.

```go
// Correct
paths.SharedDBPath(titleSlug)         // data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb
paths.MetadataDBPath(titleSlug)       // data/titles/halo_infinite/warehouse/metadata.duckdb
paths.PlayerDBPath(titleSlug, gt)     // data/titles/halo_infinite/players/Chocoboflor/stats.duckdb
paths.GlobalXuidAliasesDBPath()       // data/global/xbox_aliases.duckdb (P5, ADR 0008 — global Microsoft)

// Interdit
filepath.Join(repoRoot, "data", "warehouse", "shared_matches_v2.duckdb")
```

## Quelle DB contient quoi ?

| DB | Chemin résolu | Contenu |
|---|---|---|
| `shared_matches_v2.duckdb` | `data/titles/{slug}/warehouse/` | Stats matchs de TOUS les joueurs |
| `metadata.duckdb` | `data/titles/{slug}/warehouse/` | Référentiels (modes, armes, rangs, médailles) |
| `shared_pve.duckdb` | `data/titles/{slug}/warehouse/` | Stats Firefight |
| `shared_social.duckdb` | `data/titles/{slug}/warehouse/` | Données sociales (followers, activité) |
| `stats.duckdb` | `data/titles/{slug}/players/{gamertag}/` | Enrichissements individuels uniquement |
| `xbox_aliases.duckdb` | `data/global/` | **Global** — mapping xuid→gamertag Xbox Services (P5, ADR 0008) |
| `archive.duckdb` | `data/study/` | **Outil d'étude** — matchs archivés par `cmd/study-archiver` (ticket #6). Hors périmètre app : voir ci-dessous |

## archive.duckdb — la base de l'outil d'étude (hors app)

Base **locale, mono-writer**, écrite UNIQUEMENT par `cmd/study-archiver` (job batch
sériel). Elle n'est pas dans le circuit `BatchBuilder`/`persist` (ADR 0019/0030) et le
serveur ne l'ouvre jamais en écriture. Trois tables :

| Table | Contenu |
|---|---|
| `matches` | 1 ligne par match archivé : `match_id` (PK), `short_id`, `played_at`, `map_name`, `map_module`, `mode`, `playlist`, `duration_ms`, `source_gamertag`, `film_state`, `skip_reason`, `artifact_path`, `built_at`, `decoder_rev`, compteurs décodés (`tracks`, `points`, `shots`, `named_lives`, `total_lives`), `recorded_at` |
| `participants` | 1 ligne par (match, joueur) : `xuid`, `gamertag`, `team` (0 Eagle / 1 Cobra), `outcome` (1 nul / 2 victoire / 3 défaite / 4 abandon), `kills`, `deaths`, `assists` — **source : match stats, jamais le film** (le film ne porte aucune information d'équipe) |
| `watchlist` | `gamertag` (PK), `xuid`, `added_at`, `last_checked` — alimentée par `watch` (#8). `xuid` : résolu UNE fois via l'endpoint profil Xbox Live puis relu de la base (aucun appel ultérieur). `added_at` n'est écrit qu'à l'insertion ; `last_checked` est estampillé à chaque passe réussie — c'est ce qui distingue « rien de neuf » de « le job ne tourne plus ». Correspondance gamertag **insensible à la casse** (les gamertags Xbox le sont). |

`film_state` : `pending` \| `downloaded` \| `expired` \| `failed`. **Politique de reprise
(#7, `cmd/study-archiver/filmstate.go`)** : `expired` est le SEUL état terminal — le film
CDN est perdu, aucun run ultérieur ne retente le match. `failed` (décodeur en erreur ou
zéro trajectoire) et `downloaded` sans artefact (carte absente du catalogue de bornes)
restent repris à chaque passe : les chunks sont sur disque, un correctif décodeur ou une
mise à jour du catalogue les récupère. Un échec TRANSITOIRE (5xx, timeout, disque) n'écrit
aucune ligne — écrire `expired` sur un incident réseau enterrerait le match pour toujours.

`skip_reason` : `film_absent` (→ `expired`), `no_tracks_decoded` et `build_failed`
(→ `failed`), `unsupported_map` et `no_map_in_stats` (→ `downloaded`, repris plus tard).

**Écritures** : SELECT-then-UPDATE-or-INSERT ligne à ligne, JAMAIS `ON CONFLICT DO UPDATE`
ni delete-then-reinsert. Mono-writer n'est PAS un argument de sûreté vis-à-vis d'ART
(#23046 a crashé malgré mono-writer + PK BIGINT, cf. `no_art_patterns_test.go`), et les
deux clés d'ici sont VARCHAR. **Lectures** (`cmd/study-server` #12, `status` #9) :
`OpenReadForQuery`, jamais `OpenReadOnly` forcé — DuckDB refuse un handle read-only sur un
fichier déjà tenu en RW dans le même process, donc un `OpenReadOnly` forcé casserait
précisément le cas que la lecture seule sert (consulter pendant une passe `watch`).

**Couverture d'un match archivé** : `named_lives / total_lives`, fraction de 1 (ADR 0006) —
il n'y a PAS de colonne `coverage`. `total_lives = 0` signifie « inconnue », pas « nulle » :
l'artefact n'a rapporté aucune vie. Les deux lecteurs le distinguent (`coverageRatio`,
`cmd/study-server/filter.go` : un `CASE` qui rend NULL, donc jamais retenu par un plancher
de couverture).

**`artifact_path` est un chemin ABSOLU de la machine qui a construit l'artefact** : c'est
une trace, pas une adresse. Un lecteur résout le fichier par
`PathResolver.ReplayArtifactPath` (même appel que l'écrivain), et n'utilise la colonne que
comme drapeau « construit / pas construit ».

**Piège — `recorded()` est un lecteur PARTIEL** (`archive.go`) : il ne SELECT que ce dont
le contrôle d'idempotence a besoin, donc `mode`, `playlist`, `played_at`, `source_gamertag`,
`built_at` et `decoder_rev` reviennent à ZÉRO quelle que soit la ligne. Le repasser à
`recordMatch` EFFACERAIT ces colonnes. C'est pourquoi `rebuild` (#10) écrit via
`updateBuild()`, qui ne touche que ce que la construction a produit (artefact, compteurs,
état, révision) et laisse le roster intact — une reconstruction n'a rien de neuf à dire sur
qui a joué.

## shared_matches_v2.duckdb

### match_registry — 1 ligne par match unique
Colonnes clés : `match_id`, `start_time`, `end_time`, `map_id`, `pair_name`, `playlist_id`, `team_game`

### match_participants — stats de tous les joueurs (31 colonnes)
Colonnes clés : `match_id`, `xuid`, `gamertag`, `outcome` (1=Tie,2=Win,3=Loss,4=DNF), `kills`, `deaths`, `assists`, `shots_fired`, `shots_hit`, `damage_dealt`, `damage_taken`, `mmr`, `team_id`, `rank`

### medals_earned
Colonnes : `match_id`, `xuid`, `medal_id`, `count`, `total_personal_score`

### highlight_events
Colonnes : `match_id`, `xuid`, `event_type`, `timestamp`, `details_json`

### killer_victim_pairs
Colonnes : `match_id`, `killer_xuid`, `victim_xuid`, `count`, `weapon_id`

### xuid_aliases
Colonnes : `xuid`, `gamertag`, `last_seen`

## metadata.duckdb

| Table | Clé | Description |
|---|---|---|
| `career_ranks` | `rank_id` | Paliers et noms des rangs Halo |
| `citation_mappings` | `medal_id` | Mapping médaille→citation |
| `mode_name_tr` | `raw_name`, `lang` | Traductions des noms de modes EN→FR |
| `mode_pair_overrides` | `pair_name` | Surcharges manuelles de paires map/mode |
| `mode_prefix_names` | `prefix` | Préfixes canoniques de modes |
| `weapon_labels` | `weapon_id` (UBIGINT) | Labels EN/FR par weapon_id filmshell |

## shared_pve.duckdb

### pve_match_stats
Stats par joueur par match Firefight : `match_id`, `xuid`, `waves`, `boss_kills`,
`grunt_kills`, `elite_kills`, `jackal_kills`, `brute_kills`, `hunter_kills`,
`skimmer_kills`, `crawler_kills`, `soldier_kills`, `knight_kills`, `warden_kills`

## stats.duckdb (par joueur — data/titles/{slug}/players/{gamertag}/)

**Enrichissements uniquement** — les stats de matchs sont dans shared.

| Table | Description |
|---|---|
| `player_match_enrichment` | `performance_score`, `session_id`, `is_with_friends` |
| `personal_score_awards` | Awards objectifs (PersonalScores API) |
| `match_citations` | Citations calculées par match |
| `match_skill_rank` | Rating LUSR ou CSR par match — **append-only : lire via `match_skill_rank_latest`** |
| `career_progression` | Historique rangs |
| `sessions` | Sessions groupées |
| `media_files` | Fichiers médias indexés |
| `media_match_associations` | Associations médias↔matchs |
| `mv_player_matches` | Vue matérialisée matchs joueur |
| `mv_map_stats` | Vue matérialisée stats par map |

## Requête type — stats coéquipier

```sql
-- Stats d'un coéquipier sur matchs communs (depuis shared, pas sa DB)
SELECT mp.*
FROM shared.match_participants mp
WHERE mp.xuid = '{coequipier_xuid}'
  AND mp.match_id IN (
    SELECT match_id FROM shared.match_participants WHERE xuid = '{mon_xuid}'
  )
```

## Tables append-only + vues `_latest` (ADR 0026 — règle critique)

`match_skill_rank`, `match_csrs`, `player_csr_snapshots`, `pve_match_stats` sont
append-only (PK technique `id` + `written_at`). **Toute lecture applicative passe par la
vue `<table>_latest`** — une lecture de la table brute peut servir plusieurs versions
d'une même ligne (rating non déterministe). Écriture = INSERT pur via la couche
`internal/persist/` (jamais d'UPSERT).

Piège associé : `start_time` NULL dans `_latest` — joindre `match_registry` avec le
COALESCE timezone canonique si un tri temporel est nécessaire.

## Règle connexions (Go — modèle mono-process, ADR 0013/0016)

- Lecture shared : via `SharedProvider` / `SharedReader` (B-swap RO↔RW) — jamais
  `sql.Open` direct sur le fichier.
- Lecture d'une DB potentiellement tenue RW par le process : `OpenReadForQuery`
  (jamais `OpenReadOnly` forcé — erreurs « different configuration »).
- Écriture player : sous lease `AcquirePlayerWriterTimeout` (dblease).
- Requête ad hoc en dev : CLI `duckdb` (READ_ONLY) ou `go run apps/go-api/cmd/inspect_bp/main.go`
  — jamais en RW pendant que le serveur tourne.
