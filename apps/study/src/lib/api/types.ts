/**
 * EXTRACT — origin: apps/web/src/lib/api/types.ts @ cecca7491
 *
 * Not a whole-file copy: the origin is 2 500 lines of API surface, and the replay
 * modules copied into this app read one corner of it. The declarations below are
 * reproduced verbatim from that corner — the `Replay*` aliases and the scoreboard row
 * `rosterLogic.ts` joins on. Nothing is rewritten or trimmed field by field: a partial
 * hand-edit would be the second truth this file exists to avoid.
 *
 * `./generated` is regenerated from the SAME contract as the web app's
 * (`apps/go-api/api/openapi.yaml`, via `npm run generate-types`, kept honest by
 * `generated-types-fresh.guard.test.ts`), so the aliases resolve to the same shapes
 * and `replayContract.test.ts` still checks the nullability frontier against the
 * CONTRACT rather than against a mirror of it.
 *
 * ONE DECLARATION HERE IS NOT AN ALIAS, AND IT IS DELIBERATE. `MatchScoreboardRow` is
 * hand-written in the origin even though the contract publishes a schema of that name,
 * and the two disagree: the generated one types the counters `number | undefined`
 * where the frontend type says `number | null`, and it has no `average_life` at all.
 * The copied `rosterLogic.ts` and its fixtures are written against the frontend shape,
 * so switching to the generated one would break them — and the study server publishes
 * participants in the frontend shape precisely to fit them. So this stays the origin's
 * hand-written interface, copied as-is. It is the one place in this file where a change
 * to the Go contract will NOT arrive on its own.
 */
import type { components } from './generated'

export type PlayerWeaponKillRow = components['schemas']['PlayerWeaponKillRow']

// Champs sprite (médailles Halo 5) — shim manuel comme MatchMedal / MedalDigestItem.
// Sans eux, le drawer scoreboard affichait les médailles H5 vides (GH-5a) faute de PNG.
export type PlayerMedalRow = components['schemas']['PlayerMedalRow'] & {
  sprite_sheet?: string
  sprite_left?: number
  sprite_top?: number
  sprite_width?: number
  sprite_height?: number
}

export interface MatchScoreboardRow {
  xuid: string
  gamertag: string
  team_side: string | null
  /** Libellé d'équipe localisé fourni par le backend (Halo 5 : « Rouge »/« Red »
   *  depuis team_colors). Absent/vide pour les titres sans référentiel d'équipes
   *  (Halo Infinite) → le front retombe sur resolveTeamName (Eagle/Cobra). */
  team_name?: string | null
  /** Couleur d'identité d'équipe (#RRGGBB) fournie par le backend (Halo 5 : depuis
   *  team_colors). Absente pour Halo Infinite → le front retombe sur la map
   *  TEAM_COLORS_HALO_INFINITE (par team_id), puis sur le token ally/enemy. */
  team_color?: string | null
  is_me: boolean
  /** True si participant détecté comme bot (xuid au format "bid(N.0)"). */
  is_bot?: boolean
  rank: number | null
  score: number | null
  kills: number | null
  deaths: number | null
  assists: number | null
  kda?: number | null
  shots_fired: number | null
  shots_hit: number | null
  accuracy: number | null
  damage_dealt: number | null
  damage_taken: number | null
  average_life: string | null
  avg_life_seconds?: number | null
  headshot_kills: number | null
  max_killing_spree: number | null
  perfect_kills: number | null
  power_weapon_kills: number | null
  melee_kills: number | null
  grenade_kills?: number | null
  // Mécaniques de kill natives Halo 5 (assassinats + compétences spartiate) — null hors h5.
  assassination_kills?: number | null
  ground_pound_kills?: number | null
  shoulder_bash_kills?: number | null
  outcome_label: string
  /** V7 — combat yield */
  top_weapon_id?: number | null
  top_weapon_label?: string | null
  offensive_conversion?: number | null
  defensive_resistance?: number | null
  damage_per_kill?: number | null
  damage_per_death?: number | null
  expected_kills?: number | null
  expected_deaths?: number | null
  expected_assists?: number | null
  /** True si les expected K/D viennent du modèle local (Halo 5), pas de l'API skill. */
  locally_estimated?: boolean
  weapon_kills?: PlayerWeaponKillRow[]
  /** Médailles gagnées par CE joueur dans ce match (expander scoreboard). */
  medals?: PlayerMedalRow[]
  /** Performance score 0..100 — uniquement pour les joueurs trackés (main + amis). */
  performance_score?: number | null
  /** True si bot dans l'équipe du joueur — uniquement pour les joueurs trackés. */
  had_bot_teammate?: boolean
  /** Skill rank (CSR/LUSR) pour ce match — uniquement pour les joueurs trackés. */
  skill_rank?: MatchScoreboardSkillRank | null
  /** Stats objectifs (CTF/Zones/Oddball) — null hors mode à objectif ou titre non
   *  supporté (capability objective_stats). Seuls les champs du mode joué sont non-nil. */
  objective?: MatchScoreboardObjective | null
}

export type MatchScoreboardSkillRank = components['schemas']['MatchScoreboardSkillRank']

export type MatchScoreboardObjective = components['schemas']['MatchScoreboardObjective']

// ---------------------------------------------------------------------------
// Rejeu 2D — document d'artefact
// ---------------------------------------------------------------------------

// CES TYPES NE SONT PLUS ÉCRITS À LA MAIN. Depuis que l'endpoint est déclaré en Huma, le
// document de rejeu a un schéma dans `api/openapi.yaml`, donc une définition générée dans
// `generated.ts`. En garder une seconde copie manuscrite, c'est se donner deux vérités qui
// divergeront au premier champ ajouté côté Go — exactement ce que le ratchet
// `tools/lint-contract-ratchet.mjs` interdit.
//
// Les alias ci-dessous existent quand même, et ce n'est pas de la cosmétique : les noms du
// contrat sont génériques (`Track`, `Shot`, `Bounds`…) parce qu'ils vivent dans un espace de
// noms plat partagé par toute l'API. Le préfixe `Replay` dit de quel document ils sont les
// pièces, et évite qu'un `Point` du rejeu soit confondu avec un point de série temporelle.
//
// Artefact pré-construit hors ligne (`cmd/replay-build`). Positions dans le repère monde
// PARTAGÉ ; le client auto-ajuste via `bounds` (échelle absolue non garantie).
// `points[].t` = index de pas de temps ∈ [0, frameCount).
export type ReplayPoint = components['schemas']['Point']
export type ReplayTrack = components['schemas']['Track']
export type ReplayBounds = components['schemas']['Bounds']
export type ReplayMapObject = components['schemas']['MapObject']
export type ReplaySurface = components['schemas']['Surface']
export type ReplayShot = components['schemas']['Shot']
export type ReplayGrenade = components['schemas']['Grenade']
export type ReplayProjectile = components['schemas']['Projectile']
export type ReplayLoadout = components['schemas']['Loadout']
export type ReplayAmmoSlot = components['schemas']['AmmoSlot']
export type ReplayInventory = components['schemas']['Inventory']
export type ReplayLayerCoverage = components['schemas']['LayerCoverage']
export type ReplayBridgeHealth = components['schemas']['BridgeHealth']
export type ReplayCoverage = components['schemas']['Coverage']
export type ReplayDocument = components['schemas']['ReplayDocument']

// La table d'appariement du film : xuid ET index de slot.
//
// LES DEUX CHAMPS NE SONT PAS INTERCHANGEABLES : le xuid IDENTIFIE, l'index ORDONNE et n'a de
// sens qu'à l'intérieur de ce film. Les événements du film désignent leur auteur par index ;
// c'est cette table qui permet de le traduire en identité sans jamais confondre les deux.
// `name` est le gamertag TEL QUE LE FILM L'ÉCRIT — ce n'est pas une résolution, rien n'est
// allé le chercher ailleurs, donc rien ne peut l'avoir mal apparié.
export type ReplayRosterEntry = components['schemas']['RosterEntry']
