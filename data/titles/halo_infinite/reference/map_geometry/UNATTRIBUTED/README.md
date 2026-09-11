# Props de carte NON ATTRIBUÉS

`map_objects.csv` vivait un cran plus haut, dans `map_geometry/`, et `MapGeometryDir` ne
prenait que le titre : ce fichier était donc servi comme décor de CHAQUE match, quelle que
soit la carte. Mesuré sur six matchs de trois cartes : 382 props identiques partout.

**On ne sait pas de quelle carte il vient.** Le CSV ne porte aucune colonne de carte, et ses
coordonnées (x de -8 à ..., y de 10 à ...) tiennent dans les bornes de plusieurs cartes.

Il est donc mis de côté plutôt que supprimé : la donnée est peut-être juste, c'est son
ATTRIBUTION qui manque. Pour la remettre en service, l'identifier puis la déplacer dans
`map_geometry/<module>/map_objects.csv` (p. ex. `map_geometry/sgh_streets/`). Le chargeur
lit désormais un répertoire par module ; l'absence n'est pas fatale, elle est journalisée et
le rejeu se dessine sans props.
