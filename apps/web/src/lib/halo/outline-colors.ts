/**
 * outline-colors.ts — Palette des couleurs d'outline Halo Infinite.
 *
 * Official English names and official Halo Infinite hex codes.
 * Utilisé dans les settings d'accessibilité pour mapper les couleurs
 * que l'utilisateur a choisies in-game sur les tokens team-ally / team-enemy.
 *
 * ⚠️  Ce fichier est une exception justifiée à la règle "zéro magic hex" —
 *     il centralise les définitions de couleurs Halo, pas les composants.
 */

export interface HaloOutlineColor {
  id: string
  nameEn: string
  hex: string
}

export const HALO_OUTLINE_COLORS: readonly HaloOutlineColor[] = [
  { id: 'grass',        nameEn: 'Grass',        hex: '#BEFC77' },
  { id: 'citron',       nameEn: 'Citron',       hex: '#A8FF3A' },
  { id: 'jade',         nameEn: 'Jade',         hex: '#96FFC4' },
  { id: 'mint',         nameEn: 'Mint',         hex: '#43FF93' },
  { id: 'sky-blue',     nameEn: 'Sky Blue',     hex: '#5DD4FF' },
  { id: 'cerulean',     nameEn: 'Cerulean',     hex: '#4DC0FF' },
  { id: 'sunshine',     nameEn: 'Sunshine',     hex: '#FFFB6C' },
  { id: 'pineapple',    nameEn: 'Pineapple',    hex: '#FFFA2E' },
  { id: 'carrot',       nameEn: 'Carrot',       hex: '#FF734D' },
  { id: 'tangelo',      nameEn: 'Tangelo',      hex: '#FF4D0A' },
  { id: 'salmon',       nameEn: 'Salmon',       hex: '#FF4C4C' },
  { id: 'vermilion',    nameEn: 'Vermilion',    hex: '#FF5756' },
  { id: 'cotton-candy', nameEn: 'Cotton Candy', hex: '#FFB0FF' },
  { id: 'cherry',       nameEn: 'Cherry',       hex: '#FC4DDD' },
  { id: 'lavender',     nameEn: 'Lavender',     hex: '#B986DA' },
  { id: 'aubergine',    nameEn: 'Aubergine',    hex: '#B64EFB' },
]

export function findOutlineColor(id: string | null): HaloOutlineColor | null {
  if (!id) return null
  return HALO_OUTLINE_COLORS.find((c) => c.id === id) ?? null
}
