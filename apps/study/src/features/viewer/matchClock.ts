/**
 * matchClock.ts — L'HEURE DU MATCH, à côté de l'heure du document.
 *
 * DEUX HORLOGES, ET ELLES NE PARTAGENT PAS LEUR ZÉRO. L'axe du document commence au PREMIER
 * ÉCHANTILLON DE POSITION du film, pas au début de la partie : à la frame 0, les joueurs sont
 * déjà répliqués et se déplacent. L'artefact publie donc `matchClockZeroMs`, l'endroit de cet
 * axe où le zéro de l'horloge du MATCH tombe — négatif quand ce zéro précède la première image,
 * ce qui est le cas ordinaire (mesuré -5 692 ms sur `36e80b83`).
 *
 * CE QUE CETTE HORLOGE N'EST PAS : le coup d'envoi jouable. Le zéro publié est l'origine
 * FORMELLE de la partie (création/chargement) ; sur le même film, les barrières tombent vers
 * 21 s d'axe, soit ~27 s d'horloge de match. Rien dans le film ne date cet instant-là, et ce
 * module n'en invente pas : il traduit une horloge, il ne marque pas une manche.
 *
 * PUR, et c'est pourquoi il vit hors du composant : un décalage de signe sur une horloge se
 * voit mal à l'écran et se teste très bien.
 */

/** Millisecondes écoulées depuis le zéro de l'horloge du match, pour un instant de l'axe. */
export function matchTimeMs(axisMs: number, matchClockZeroMs: number): number {
  return axisMs - matchClockZeroMs
}

/**
 * formatMatchClock rend `m:ss`, et `-m:ss` AVANT le zéro du match.
 *
 * LE SIGNE EST PORTÉ PLUTÔT QU'ÉCRÊTÉ : un instant antérieur au début de la partie est une
 * information (c'est l'avant-match), et l'afficher comme « 0:00 » le ferait passer pour le
 * coup d'envoi. Les secondes sont TRONQUÉES et non arrondies, pour que l'affichage ne
 * dépasse jamais l'instant réellement atteint.
 */
export function formatMatchClock(ms: number): string {
  const sign = ms < 0 ? '-' : ''
  const total = Math.floor(Math.abs(ms) / 1000)
  const minutes = Math.floor(total / 60)
  const seconds = total % 60
  return `${sign}${minutes}:${String(seconds).padStart(2, '0')}`
}
