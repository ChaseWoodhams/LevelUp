/**
 * replayFixture.ts — an artifact built by hand, so the viewer has something real to
 * draw before there is a server to fetch one from.
 *
 * IT IS SHAPED LIKE THE REAL THING, DELIBERATELY. Same contract type
 * (`ReplayDocument`), same schema version, same nullable arrays, same world frame —
 * so it crosses `normalizeReplayDocument` exactly as an archived artifact will, and
 * the boundary is exercised rather than bypassed. Swapping this constant for a fetch
 * is then a one-line change (issue #13), not a rewrite.
 *
 * IT IS NOT RECORDED DATA. Every position, shot and throw below was written to put a
 * layer on screen: a reconstructed floor with steps, two teams, lives that end and
 * respawn, one player who dies for good, shots with and without a readable heading,
 * grenade throws, a projectile, keyframe loadouts and inventories, and a coverage
 * block whose rejects add up. No number here says anything about a real match.
 */
import type {
  MatchScoreboardRow,
  ReplayDocument,
  ReplayPoint,
  ReplaySurface,
  ReplayTrack,
} from '@/lib/api/types'

/** Frame axis: 10 Hz for 60 s, the order of magnitude the builder writes. */
const FRAME_INTERVAL_MS = 100
const FRAME_COUNT = 600
/** One position sample every half second, as the film replicates them. */
const SAMPLE_STEP = 5

/** Arena, in metres of the shared world frame. */
const BOUNDS = { minX: 0, minY: 0, maxX: 40, maxY: 30, minZ: 0, maxZ: 6 }

/** slab is one BSP footprint: an axis-aligned floor face at altitude `z`. */
function slab(x0: number, y0: number, x1: number, y1: number, z: number): ReplaySurface {
  return { x0, y0, x1, y1, z, zb: z - 0.5 }
}

/**
 * The floor: four quadrants at ground level joined by a corridor, a centre platform
 * 2.5 m up and two landings halfway. The altitude steps are what makes `mapFloor`
 * draw edges — a single flat slab would rasterise to one featureless wash.
 */
const STRUCTURE: ReplaySurface[] = [
  slab(4, 4, 20, 13, 0),
  slab(20, 4, 36, 13, 0),
  slab(4, 17, 20, 26, 0),
  slab(20, 17, 36, 26, 0),
  slab(4, 13, 36, 17, 0),
  slab(12, 11, 16, 19, 1.2),
  slab(24, 11, 28, 19, 1.2),
  slab(16, 11, 24, 19, 2.5),
]

/** A player's identity, fixed for the whole match. */
interface FixturePlayer {
  xuid: string
  gamertag: string
  side: string | null
}

const PLAYERS: FixturePlayer[] = [
  { xuid: '2533274800000001', gamertag: 'Aigle-01', side: 'Eagle' },
  { xuid: '2533274800000002', gamertag: 'Aigle-02', side: 'Eagle' },
  { xuid: '2533274800000003', gamertag: 'Cobra-01', side: 'Cobra' },
  { xuid: '2533274800000004', gamertag: 'Cobra-02', side: 'Cobra' },
  // The film names him; the archive has no participants row for him. He is drawn and
  // grouped on his own, never folded into a team. That rule is why he is here.
  { xuid: '2533274800000005', gamertag: 'Sans-equipe', side: null },
]

/** A leg of a walk: where the player is, and where they are looking. */
interface Waypoint {
  x: number
  y: number
  z: number
  /** Aim heading in degrees, atan2 convention: 0 = +X, 90 = +Y. */
  h: number
}

/** A life: one biped slot, alive over a frame window, walking a path. */
interface Life {
  xuid: string
  slot: number
  start: number
  end: number
  path: Waypoint[]
  /** Shield readings, as `[frame, fraction]` — the film only replicates changes. */
  shield: [number, number][]
}

const LIVES: Life[] = [
  {
    xuid: '2533274800000001',
    slot: 0,
    start: 0,
    end: 260,
    path: [
      { x: 7, y: 7, z: 0, h: 45 },
      { x: 14, y: 12, z: 0, h: 30 },
      { x: 20, y: 15, z: 2.5, h: 0 },
      { x: 28, y: 15, z: 1.2, h: 350 },
    ],
    shield: [[0, 1], [140, 0.62], [210, 0.18], [255, 0]],
  },
  {
    xuid: '2533274800000001',
    slot: 4,
    start: 340,
    end: 599,
    path: [
      { x: 8, y: 22, z: 0, h: 300 },
      { x: 16, y: 19, z: 1.2, h: 340 },
      { x: 24, y: 21, z: 0, h: 10 },
      { x: 33, y: 23, z: 0, h: 25 },
    ],
    shield: [[340, 1], [470, 0.75]],
  },
  {
    xuid: '2533274800000002',
    slot: 1,
    start: 0,
    end: 420,
    path: [
      { x: 6, y: 24, z: 0, h: 315 },
      { x: 12, y: 18, z: 0, h: 330 },
      { x: 18, y: 15, z: 2.5, h: 0 },
      { x: 12, y: 9, z: 0, h: 200 },
      { x: 7, y: 6, z: 0, h: 225 },
    ],
    shield: [[0, 1], [230, 0.44], [400, 0.05]],
  },
  {
    // Dies at 560 and never comes back: after that his card must show the respawn as a
    // gap, never a guessed delay. That branch has no other way to be seen on screen.
    xuid: '2533274800000002',
    slot: 5,
    start: 500,
    end: 560,
    path: [
      { x: 10, y: 8, z: 0, h: 60 },
      { x: 15, y: 13, z: 0, h: 40 },
    ],
    shield: [[500, 1], [548, 0.2]],
  },
  {
    xuid: '2533274800000003',
    slot: 2,
    start: 0,
    end: 300,
    path: [
      { x: 34, y: 8, z: 0, h: 180 },
      { x: 27, y: 12, z: 1.2, h: 160 },
      { x: 21, y: 15, z: 2.5, h: 180 },
      { x: 14, y: 17, z: 1.2, h: 195 },
    ],
    shield: [[0, 1], [120, 0.8], [280, 0.1]],
  },
  {
    xuid: '2533274800000003',
    slot: 6,
    start: 380,
    end: 599,
    path: [
      { x: 34, y: 22, z: 0, h: 200 },
      { x: 26, y: 20, z: 0, h: 190 },
      { x: 20, y: 16, z: 2.5, h: 170 },
    ],
    shield: [[380, 1], [520, 0.9]],
  },
  {
    // Survives the whole replay: nothing ever writes a death marker for this slot.
    xuid: '2533274800000004',
    slot: 3,
    start: 0,
    end: 599,
    path: [
      { x: 33, y: 24, z: 0, h: 240 },
      { x: 26, y: 18, z: 0, h: 215 },
      { x: 22, y: 14, z: 2.5, h: 190 },
      { x: 28, y: 9, z: 0, h: 300 },
      { x: 34, y: 6, z: 0, h: 340 },
      { x: 30, y: 14, z: 0, h: 100 },
    ],
    shield: [[0, 1], [180, 0.55], [300, 1], [500, 0.35]],
  },
  {
    // In the film's roster, absent from the scoreboard: an ungrouped bucket of one.
    xuid: '2533274800000005',
    slot: 7,
    start: 60,
    end: 480,
    path: [
      { x: 18, y: 5, z: 0, h: 90 },
      { x: 19, y: 12, z: 0, h: 90 },
      { x: 19, y: 20, z: 0, h: 95 },
      { x: 17, y: 25, z: 0, h: 120 },
    ],
    shield: [[60, 1], [320, 0.5]],
  },
]

/**
 * samplePath walks the polyline at a fixed cadence and writes one point per sample.
 *
 * Heading and shield are attached the way the film replicates them — sparsely. The
 * heading rides the leg being walked; the shield is written only on the frames the
 * life declares, so the fade on a stale reading has something to fade from.
 */
function samplePath(life: Life): ReplayPoint[] {
  const legs = life.path.length - 1
  const span = life.end - life.start
  const points: ReplayPoint[] = []
  for (let t = life.start; t <= life.end; t += SAMPLE_STEP) {
    const u = span === 0 ? 0 : ((t - life.start) / span) * legs
    const leg = Math.min(Math.floor(u), legs - 1)
    const f = u - leg
    const a = life.path[leg]
    const b = life.path[leg + 1]
    const point: ReplayPoint = {
      t,
      x: a.x + (b.x - a.x) * f,
      y: a.y + (b.y - a.y) * f,
      z: a.z + (b.z - a.z) * f,
    }
    // Aim is on roughly two points in five, as the record replicates it. The 0 -> 360
    // rewrite is the producer's own omitempty guard, kept here so the fixture cannot
    // publish a heading that reads back as "not measured".
    if (t % (SAMPLE_STEP * 2) === 0) point.h = a.h === 0 ? 360 : a.h
    const reading = life.shield.find(([frame]) => frame === t)
    if (reading) point.sh = reading[1]
    points.push(point)
  }
  return points
}

function track(life: Life): ReplayTrack {
  const player = PLAYERS.find((p) => p.xuid === life.xuid)
  return {
    slot: life.slot,
    // The artifact writes -1 when the film carries no team for a life; nothing here
    // invents one from the scoreboard.
    team: -1,
    xuid: life.xuid,
    name: player?.gamertag,
    startFrame: life.start,
    endFrame: life.end,
    points: samplePath(life),
  }
}

/** Weapon families, keyed by the 64-bit weapon id the film writes in hexadecimal. */
const WEAPON_LABELS: NonNullable<ReplayDocument['weaponLabels']> = {
  '0000000100000000': { en: 'Assault Rifle', fr: 'Fusil d’assaut', fx: 'ballistic' },
  '0000000200000000': { en: 'Battle Rifle', fr: 'Fusil de combat', fx: 'ballistic' },
  '0000000300000000': { en: 'Plasma Pistol', fr: 'Pistolet plasma', fx: 'plasma' },
  '0000000400000000': { en: 'Needler', fr: 'Needler', fx: 'needles' },
  '0000000500000000': { en: 'Rocket Launcher', fr: 'Lance-roquettes', fx: 'explosive' },
  // Deliberately outside the drawn families: an unknown effect must fall back to the
  // neutral mark, never borrow a neighbouring shape.
  '0000000600000000': { en: 'Prototype', fr: 'Prototype', fx: 'unknown-family' },
}

/** Grenade types, in the rank order the inventory counters use. */
const GRENADE_LABELS: NonNullable<ReplayDocument['grenadeLabels']> = [
  { en: 'Frag', fr: 'Fragmentation' },
  { en: 'Plasma', fr: 'Plasma' },
  { en: 'Spike', fr: 'Spike' },
]

/** Armour abilities, keyed by the ability index the inventory reads. */
const ABILITY_LABELS: NonNullable<ReplayDocument['abilityLabels']> = {
  '0': { en: 'Grappleshot', fr: 'Grappin' },
  '1': { en: 'Repulsor', fr: 'Repulseur' },
}

/**
 * Shots. The film only records a shot that DEALT damage, so every one of these is a
 * hit; two carry no heading, which is how ~80 % of real records arrive.
 */
const SHOTS: NonNullable<ReplayDocument['shots']> = [
  { t: 40, slot: 0, x: 10.5, y: 9.4, h: 35, w: '0000000200000000' },
  { t: 44, slot: 2, x: 31.2, y: 9.8, h: 185, w: '0000000100000000' },
  { t: 120, slot: 1, x: 11.4, y: 17.2, h: 330, w: '0000000400000000' },
  { t: 126, slot: 7, x: 18.6, y: 10.1, w: '0000000300000000' },
  { t: 205, slot: 3, x: 24.8, y: 15.6, h: 200, w: '0000000500000000' },
  { t: 208, slot: 0, x: 22.1, y: 15.0, h: 20, w: '0000000200000000' },
  { t: 250, slot: 2, x: 17.4, y: 16.3, w: '0000000600000000' },
  { t: 390, slot: 6, x: 32.0, y: 21.4, h: 195, w: '0000000100000000' },
  { t: 460, slot: 4, x: 20.2, y: 20.4, h: 15, w: '0000000200000000' },
  { t: 540, slot: 5, x: 13.1, y: 11.6, h: 45, w: '0000000400000000' },
]

/** Grenade throws. `s` says where the position came from, not what was thrown. */
const GRENADES: NonNullable<ReplayDocument['grenades']> = [
  { t: 90, slot: 1, i: 1, x: 12.8, y: 16.4, rank: 0, s: 'projectile' },
  { t: 230, slot: 3, i: 3, x: 23.4, y: 14.2, rank: 1, s: 'projectile' },
  { t: 430, slot: 6, i: 2, x: 27.6, y: 20.1, rank: 2, s: 'biped' },
]

/** One projectile track: [dt, x, y] steps from its birth frame. */
const PROJECTILES: NonNullable<ReplayDocument['projectiles']> = [
  {
    t0: 230,
    p: [
      [0, 23.4, 14.2],
      [4, 22.0, 13.4],
      [8, 20.6, 12.8],
      [12, 19.4, 12.5],
    ],
    rest: true,
  },
]

/** Loadouts, read at keyframes — roughly one every 20 s, as the film writes them. */
const LOADOUTS: NonNullable<ReplayDocument['loadouts']> = [
  { t: 0, slot: 0, w: ['0000000200000000', '0000000100000000'] },
  { t: 0, slot: 1, w: ['0000000100000000', '0000000400000000'] },
  { t: 0, slot: 2, w: ['0000000200000000', '0000000300000000'] },
  { t: 0, slot: 3, w: ['0000000100000000', '0000000500000000'] },
  { t: 60, slot: 7, w: ['0000000300000000'] },
  { t: 200, slot: 0, w: ['0000000200000000', '0000000500000000'] },
  { t: 200, slot: 3, w: ['0000000100000000', '0000000600000000'] },
  { t: 340, slot: 4, w: ['0000000200000000'] },
  { t: 380, slot: 6, w: ['0000000100000000', '0000000400000000'] },
  { t: 500, slot: 5, w: ['0000000400000000'] },
]

/**
 * Inventories, read at the same keyframes. `am` follows the order of `w`, and the
 * gauge counts what has been CONSUMED — the card draws its complement.
 */
const INVENTORY: NonNullable<ReplayDocument['inventory']> = [
  { t: 0, slot: 0, g: [2, 0, 0], a: 0, d: 2, am: [{ mag: 36, res: 108 }, { mag: 32, res: 96 }], cand: 1 },
  { t: 0, slot: 1, g: [1, 1, 0], a: 1, d: 0, am: [{ mag: 32, res: 96 }, { mag: 20, res: 40 }], cand: 1 },
  { t: 0, slot: 2, g: [0, 2, 0], a: 0, d: 0, am: [{ mag: 36, res: 72 }, { gauge: 0 }], cand: 1 },
  { t: 0, slot: 3, g: [2, 0, 1], a: 1, d: 0, am: [{ mag: 32, res: 96 }, { mag: 2, res: 0 }], cand: 2 },
  { t: 60, slot: 7, g: [0, 0, 0], d: 2, am: [{ gauge: 0.1 }], cand: 1 },
  { t: 200, slot: 0, g: [1, 0, 0], a: 0, d: 0, am: [{ mag: 21, res: 72 }, { mag: 2, res: 0 }], cand: 1 },
  // Ability index 7 is not in ABILITY_LABELS: the card must keep the number and say
  // it cannot read it, rather than borrowing the name of a neighbour.
  { t: 200, slot: 3, g: [0, 0, 1], a: 7, d: 1, am: [{ mag: 14, res: 48 }, { gauge: 0.6 }], cand: 3 },
  { t: 340, slot: 4, g: [2, 1, 0], a: 0, d: 0, am: [{ mag: 36, res: 108 }], cand: 1 },
  { t: 380, slot: 6, g: [1, 0, 0], a: 1, d: 0, am: [{ mag: 30, res: 60 }, { mag: 18, res: 36 }], cand: 1 },
  { t: 500, slot: 5, g: [0, 1, 0], a: 1, d: 0, am: [{ mag: 12, res: 20 }], cand: 1 },
]

/**
 * Coverage. The layer sums balance — attached + every reject cause = available — so
 * the banner reports a real shortfall rather than the leak it is there to catch.
 */
const COVERAGE: NonNullable<ReplayDocument['coverage']> = {
  shots: { attached: 10, available: 13, noSlot: 2, ambiguous: 1, outOfWindow: 0, unpublished: 0 },
  grenades: { attached: 3, available: 3, noSlot: 0, ambiguous: 0, outOfWindow: 0, unpublished: 0 },
  objectives: { attached: 0, available: 0, noSlot: 0, ambiguous: 0, outOfWindow: 0, unpublished: 0 },
  bridge: {
    slots: 8,
    livesTotal: LIVES.length,
    livesNamed: LIVES.length,
    fromReading: 8,
    indexReadings: 8,
    indexDisagreements: 0,
    slotCollisions: 0,
  },
  verdict: { shots: 'partial', grenades: 'nominal', objectives: 'nominal' },
}

/**
 * FIXTURE_REPLAY_DOCUMENT — the artifact, exactly as the transport would hand it
 * over: raw, unnormalised, `schemaVersion` at the value `replay.SchemaVersion` holds
 * in `apps/go-api/internal/analysis/replay/document.go`.
 */
export const FIXTURE_REPLAY_DOCUMENT: ReplayDocument = {
  schemaVersion: 2,
  matchId: '00000000-0000-4000-8000-0000000f1x7e',
  titleSlug: 'halo_infinite',
  frameCount: FRAME_COUNT,
  frameIntervalMs: FRAME_INTERVAL_MS,
  durationMs: FRAME_COUNT * FRAME_INTERVAL_MS,
  bounds: BOUNDS,
  structureBounds: BOUNDS,
  structure: STRUCTURE,
  tracks: LIVES.map(track),
  // The film's roster names everyone it saw, including the player the archive has no
  // participants row for — that asymmetry is the point of keeping the fifth player.
  roster: PLAYERS.map((p, i) => ({ filmIndex: i, xuid: p.xuid, name: p.gamertag })),
  shots: SHOTS,
  grenades: GRENADES,
  projectiles: PROJECTILES,
  loadouts: LOADOUTS,
  inventory: INVENTORY,
  weaponLabels: WEAPON_LABELS,
  grenadeLabels: GRENADE_LABELS,
  abilityLabels: ABILITY_LABELS,
  coverage: COVERAGE,
}

/**
 * FIXTURE_SCOREBOARD — the rows the archive will serve for this match.
 *
 * Shaped like `MatchScoreboardRow` because that is what `rosterLogic` joins on, and
 * because the study server publishes participants in that shape (issue #13). The
 * fifth player of the film has no row on purpose: the join must leave him ungrouped.
 */
export const FIXTURE_SCOREBOARD: MatchScoreboardRow[] = PLAYERS.filter(
  (p) => p.side !== null,
).map((p, i) => ({
  xuid: p.xuid,
  gamertag: p.gamertag,
  team_side: p.side,
  is_me: false,
  rank: i + 1,
  score: 1200 - i * 130,
  kills: 14 - i * 2,
  deaths: 8 + i,
  assists: 5 - i,
  shots_fired: null,
  shots_hit: null,
  accuracy: null,
  damage_dealt: null,
  damage_taken: null,
  average_life: null,
  headshot_kills: null,
  max_killing_spree: null,
  perfect_kills: null,
  power_weapon_kills: null,
  melee_kills: null,
  outcome_label: p.side === 'Eagle' ? 'win' : 'loss',
}))
