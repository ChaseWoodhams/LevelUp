/**
 * COPIED FILE — origin: apps/web/src/features/match-replay/i18n.ts
 * Origin at commit: a1520ad0c — the commit that last changed it, so
 * `git diff 4cc6e4b97 HEAD -- <origin>` is what the origin has learnt since.
 *
 * Byte-identical to the origin below this header, and src/copies.guard.test.ts
 * enforces it. A fix belongs upstream first. See features/replay/README.md.
 */
/**
 * i18n de la feature match-replay (rejeu 2D vue du dessus). Strings UI FR + EN,
 * parité par typage Record<ReplayLocale, ReplayText>.
 */
import type { Locale } from '@/lib/i18n/locale'

/** Alias local du type central : la feature ne redéclare pas l'union des langues. */
export type ReplayLocale = Locale

interface ReplayText {
  title: string
  back: string
  play: string
  pause: string
  restart: string
  loading: string
  empty: string
  note: string
  livesSuffix: string
  aliveSuffix: string
  speed: string
  time: string
  floor: string
  floorAll: string
  floorLow: string
  floorMid: string
  floorHigh: string
  propsSuffix: string
  /** Calques que le lecteur peut éteindre. */
  layers: string
  layerAim: string
  layerShield: string
  layerAimHint: string
  layerShieldHint: string
  /** Fiches joueur : ce qui est lu, et ce qui ne l'est pas. */
  rosterEmpty: string
  teamUnknown: string
  unknownPlayer: string
  shieldUnread: string
  loadoutUnread: string
  loadoutAge: string
  respawnIn: string
  respawnUnknown: string
  respawnUnknownHint: string
  /** Ligne d'inventaire : grenades, capacité, munitions. */
  inventoryAge: string
  grenadeSelected: string
  abilityUnknown: string
  abilityUnknownHint: string
  ammoNone: string
  ammoNoneHint: string
  ammoSlotHint: string
  gaugeLabel: string
  /** Bandeau de couverture : ce que chaque calque a rattaché sur ce qui existait. */
  coverageTitle: string
  coverageHint: string
  layerShots: string
  layerGrenades: string
  causeNoSlot: string
  causeAmbiguous: string
  causeOutOfWindow: string
  causeUnpublished: string
  bridgeRead: string
  bridgeNotRead: string
  coverageLeak: string
}

export const REPLAY_TEXT: Record<ReplayLocale, ReplayText> = {
  en: {
    title: '2D replay',
    back: 'Back to match',
    play: 'Play',
    pause: 'Pause',
    restart: 'Restart',
    loading: 'Loading replay…',
    empty: 'No 2D replay available for this match.',
    note: 'Trajectories decoded from the film (top-down). One trail = one life: a respawning player starts a new trail, with no per-player attribution. Playback at 1x follows the real match time.',
    livesSuffix: 'lives',
    aliveSuffix: 'on the map',
    speed: 'Speed',
    time: 'Match time',
    floor: 'Floor',
    floorAll: 'All',
    floorLow: 'Lower',
    floorMid: 'Mid',
    floorHigh: 'Upper',
    propsSuffix: 'map props',
    layers: 'Layers',
    layerAim: 'Aim',
    layerShield: 'Shield',
    layerAimHint:
      'Look direction, decoded from the same record as the position. The game only retransmits it when it changes: an older reading fades instead of vanishing, and nothing is drawn beyond five seconds.',
    layerShieldHint:
      'Shield read from the same record as the position. An empty bar with a line under it is a BROKEN shield, not a missing measurement.',
    rosterEmpty: 'No life from the film could be attached to a player: the replay stays anonymous.',
    teamUnknown: 'No team',
    unknownPlayer: 'Unknown player',
    shieldUnread: 'shield not transmitted',
    loadoutUnread: 'weapons not read on this life',
    loadoutAge: 'Weapons read',
    respawnIn: 'back in',
    respawnUnknown: 'back ?',
    respawnUnknownHint:
      'The film carries no next life for this player: the delay is not readable, and it is not guessed. This happens at the end of the match, which the film closes with no event.',
    inventoryAge: 'Inventory read',
    grenadeSelected: 'Equipped type: the only one carried, so the one the next throw will use.',
    abilityUnknown: 'unknown ability',
    abilityUnknownHint:
      'The ability table is partial: 4 indices observed for 11 abilities in the game. An index outside the table keeps its number rather than borrowing a neighbouring ability name.',
    ammoNone: 'none',
    ammoNoneHint:
      'The film writes nothing for this slot. For a charge weapon that means FULL: the stream is differential, and full is the default value, so it is never transmitted.',
    gaugeLabel: 'charge left',
    ammoSlotHint:
      'Ammo for the record slot bearing this number. Across the 198 pairings whose reading is unique, 197 agree with the weapon of the same rank above. A reading with several candidates is not reliable and is flagged separately.',
    coverageTitle: 'What is attached',
    coverageHint:
      'The denominator is what THE FILM carries, not what the match counted: the film only records shots that deal damage.',
    layerShots: 'Shots',
    layerGrenades: 'Grenade throws',
    causeNoSlot: 'no known trail for that player at that moment',
    causeAmbiguous: 'several possible trails',
    causeOutOfWindow: 'no position close enough in time',
    causeUnpublished: 'trail too short to be published',
    bridgeRead: 'Attachment read from the film',
    bridgeNotRead: 'Incomplete attachment — some events are not published',
    coverageLeak: 'Inconsistent count: events are unaccounted for',
  },
}
