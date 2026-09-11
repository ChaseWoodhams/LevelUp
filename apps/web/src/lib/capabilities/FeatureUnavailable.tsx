import { EmptyStateCard } from '@/components/ui/empty-state'
import type { TitleCapability } from './capabilities'

/**
 * English labels by capability for the placeholder "feature unavailable for
 * this title". NO-OP for `halo_infinite` (declares
 * toutes les capabilities → ce placeholder n'est jamais rendu en mono-titre).
 */
const FEATURE_LABEL: Record<TitleCapability, string> = {
  matchmaking: 'matches',
  firefight: 'Firefight (PvE)',
  forge: 'Forge',
  media: 'media',
  ranked: 'ranked (CSR)',
  career: 'career',
  season_pass: 'the season pass',
  'asset.images': 'asset images',
  achievements: 'achievements',
  engagement: 'intra-match engagement',
  lusr: 'the LUSR rating',
  'world.leaderboard': 'world leaderboards',
  native_kill_mechanics: 'assassinations and Spartan abilities',
  team_mmr: 'per-match MMR',
  damage_taken: 'damage taken',
  weapon_accuracy: 'accuracy by weapon',
  spartan_customizer: 'Spartan customization',
  expected_stats: 'expected KDA stats',
  waypoint_match_url: 'Halo Waypoint links',
  objective_stats: 'objective stats (CTF/Zones/Oddball)',
}

interface FeatureUnavailableProps {
  capability: TitleCapability
  className?: string
}

/**
 * FeatureUnavailable — placeholder gracieux rendu à la place d'une page entière
 * quand le titre courant ne déclare pas `capability` (cf. {@link RouteCapabilityGate}).
 * Évite une page morte / vide : on annonce explicitement que la fonctionnalité
 * n'existe pas pour ce titre, au lieu d'afficher un squelette sans données.
 */
export function FeatureUnavailable({ capability, className }: FeatureUnavailableProps) {
  const label = FEATURE_LABEL[capability]
  const title = 'Not available for this title'
  const description = `This title does not provide ${label}.`

  return (
    <div className={className ?? 'p-6'}>
      <EmptyStateCard title={title} description={description} />
    </div>
  )
}
