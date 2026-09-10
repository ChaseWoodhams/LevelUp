package replay

import (
	"math"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

// grenades.go — LANCERS DE GRENADE.
//
// LE LANCER PORTE DÉJÀ SON AUTEUR. Comme l'événement de tir, il écrit le `FilmIndex` du
// lanceur : celui-ci n'est ni deviné ni voté. Ce fichier n'a donc jamais eu à trouver QUI a
// lancé.
//
// CE QUI ÉTAIT MAL CONÇU, ET QUI EST CORRIGÉ ICI. Pour dessiner le lancer, on prenait la
// position du BIPED du lanceur — ce qui exigeait de connaître son biped, donc de passer par le
// pont, donc d'échouer quand la vie n'était pas nommée. **Sept lancers sur soixante-dix étaient
// perdus pour cette seule raison.**
//
// Or le lancer fait naître un PROJECTILE, dont la position est décodée, et **dont le premier
// point est la main du lanceur** : la position cherchée est déjà là, sans le pont. Mesuré :
// la naissance est à 0,77 unité du biped auteur (médiane), contre 6,4 pour un instant permuté
// et 33,9 pour un biped tiré au hasard.
//
// LA HIÉRARCHIE DES SOURCES, dans cet ordre et pour cette raison :
//
//	1. la naissance du PROJECTILE     position de l'objet lancé, décodée — aucun pont
//	2. la position du BIPED du lanceur si le pont le connaît, et si aucun projectile n'apparie
//
// La seconde n'est pas un repli voté : c'est la même grandeur lue ailleurs. Le champ `Src` dit
// laquelle a servi, pour que l'écran puisse les distinguer s'il le veut.

// grenadeBirthWindowUS est la fenêtre dans laquelle un projectile doit naître après un lancer
// pour qu'on les tienne pour le même événement. Mesuré : 65 des 70 lancers apparient à
// ±200 ms, contre 11 à 13 pour les mêmes lancers décalés en bloc dans le temps.
const grenadeBirthWindowUS = 200_000

// Sources d'une position de lancer, publiées telles quelles dans l'artefact.
const (
	// GrenadeSrcProjectile : position du projectile à sa naissance — la main du lanceur.
	GrenadeSrcProjectile = "projectile"
	// GrenadeSrcBiped : position du biped du lanceur, quand aucun projectile n'apparie.
	GrenadeSrcBiped = "biped"
)

// Grenade est un lancer de grenade, situé dans le temps et l'espace.
type Grenade struct {
	// T est l'index de frame, sur le même axe que Point.T.
	T int `json:"t"`
	// Slot est le biped lanceur quand il est connu (0 sinon). Il sert à relier le lancer à une
	// trajectoire ; il n'est PAS nécessaire pour situer le lancer.
	Slot uint32 `json:"slot"`
	// Idx est l'index de joueur ÉCRIT dans le film. C'est lui l'auteur, toujours renseigné.
	Idx int `json:"i"`
	// X, Y sont la position du lancer.
	X float32 `json:"x"`
	Y float32 `json:"y"`
	// Rank est le RANG du type de grenade : un index dans ReplayDocument.GrenadeLabels,
	// la seule table qui les nomme.
	//
	// C'ÉTAIT UN NOM JUSQU'AU 2026-08-02, et c'est ce qui a produit la contradiction du
	// lot 3.1 : le lancer disait « Shock » là où le compteur porté du MÊME type disait
	// « Dynamo », sur la même fiche. Un index ne peut pas diverger de sa table.
	// Pas d'omitempty : le rang 0 (fragmentation) est une valeur, pas une absence.
	Rank int `json:"rank"`
	// Src dit d'où vient la position : GrenadeSrcProjectile ou GrenadeSrcBiped.
	Src string `json:"s"`
}

// buildGrenades situe les lancers et rend la couverture.
func buildGrenades(pos []filmdec.BipedPosition, throws []filmdec.GrenadeThrow,
	origin, step uint64, owner map[uint32]int, proj []filmdec.ProjectileTrack) ([]Grenade, LayerCoverage) {
	cov := LayerCoverage{Available: len(throws)}
	if len(throws) == 0 {
		return nil, cov
	}
	births := projectileBirths(proj)
	tracks := indexBySlot(pos)
	var out []Grenade
	for _, g := range throws {
		rank, known := g.Rank()
		if !known {
			// Le décodeur ne rend que des tags de sa liste blanche ; un tag hors rangs
			// serait un lancer qu'aucune table ne peut nommer. On ne le publie pas
			// plutôt que de le poser sur le rang 0 (fragmentation).
			cov.count(reasonNoSlot)
			continue
		}
		gr, ok := locateThrow(g, births, tracks, owner)
		if !ok {
			cov.count(reasonNoSlot)
			continue
		}
		cov.count(reasonAttached)
		gr.T = int((g.TimestampUS - origin) / step)
		gr.Idx = g.FilmIndex
		gr.Rank = rank
		out = append(out, gr)
	}
	// Tri TOTAL : deux lancers tombent souvent sur la même frame de la grille (10 Hz), et un
	// départage arbitraire suffit à changer l'artefact d'un octet à l'autre.
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		switch {
		case a.T != b.T:
			return a.T < b.T
		case a.Idx != b.Idx:
			return a.Idx < b.Idx
		case a.Slot != b.Slot:
			return a.Slot < b.Slot
		case a.X != b.X:
			return a.X < b.X
		default:
			return a.Y < b.Y
		}
	})
	return out, cov
}

// locateThrow situe un lancer : d'abord par la naissance de son projectile, sinon par le biped
// de son auteur quand le pont le connaît.
//
// L'AUTEUR EST RÉSOLU EN PREMIER, ET C'EST UN CORRECTIF. La naissance était choisie sur le
// TEMPS SEUL : `birthNear` prenait la naissance la plus proche dans une fenêtre de 200 ms,
// et le départage des naissances simultanées venait du tri (donc de X). Quand deux joueurs
// lancent dans la même fenêtre — banal —, le lancer recevait la position du projectile de
// l'AUTRE. Mesuré sur `36e80b83` avant correctif : 171 lancers publiés sur 247 n'avaient
// AUCUN joueur à moins de 4 m, distance médiane au joueur le plus proche 7,95 m, pire cas
// 24,68 m. À comparer à la mesure qui fondait cette source : 0,77 unité entre la naissance et
// le biped de son auteur (6,4 pour un instant permuté) — 7,95 m est le régime du contrôle
// négatif, pas celui du signal.
//
// LE BIPED DE L'AUTEUR EST DONC LE JUGE, quand le pont le donne : parmi les naissances de la
// fenêtre, on retient celle qui est à portée de sa main, et aucune si elle n'y est pas. Sans
// pont, la source reste utilisable — c'était sa raison d'être — mais une fenêtre qui porte
// PLUSIEURS naissances n'est plus tranchée au hasard : elle n'est pas publiée.
func locateThrow(g filmdec.GrenadeThrow, births []filmdec.ProjectileSample,
	tracks map[uint32]slotTrack, owner map[uint32]int) (Grenade, bool) {
	slot, author := authorBiped(g, tracks, owner)
	if b, ok := birthForThrow(births, g.TimestampUS, author); ok {
		// LE SLOT EST PORTÉ MÊME ICI, et il ne l'était pas : la branche projectile rendait un
		// Grenade sans Slot, donc à zéro. 201 des 247 lancers de `36e80b83` sortaient ainsi, et
		// `grenadeArcs.ts` — qui colore l'arc par le slot de son lanceur — ne pouvait plus
		// nommer personne. Pire, zéro RESSEMBLE à un slot : le garde d'ambiguïté du client
		// (« deux candidats de slots différents annulent ») ne se déclenchait jamais entre deux
		// zéros. La position ne dépend toujours pas du pont ; le slot, lui, est publié dès que
		// le pont le connaît.
		return Grenade{Slot: slot, X: round2(b.X), Y: round2(b.Y), Src: GrenadeSrcProjectile}, true
	}
	if author == nil {
		return Grenade{}, false
	}
	return Grenade{Slot: slot, X: round2(author.X), Y: round2(author.Y), Src: GrenadeSrcBiped}, true
}

// authorBiped rend le slot du lanceur et sa position répliquée, quand le pont et le film les
// donnent. Le slot peut être connu sans que la position le soit (réplication trop lointaine).
func authorBiped(g filmdec.GrenadeThrow, tracks map[uint32]slotTrack,
	owner map[uint32]int) (uint32, *filmdec.BipedPosition) {
	slot, reason := slotFor(tracks, owner, g.FilmIndex, g.TimestampUS)
	if reason != reasonAttached {
		return 0, nil
	}
	p, d := tracks[slot].at(g.TimestampUS)
	if d > shotPosToleranceUS || !p.HasWorld {
		return slot, nil
	}
	return slot, &p
}

// grenadeAuthorRadiusM est la distance maximale acceptée entre la naissance d'un projectile et
// le biped de son lanceur.
//
// MÊME NOMBRE QUE `ARC_ORIGIN_RADIUS_M` CÔTÉ CLIENT, et pour la même raison : les deux
// répondent à « cette naissance est-elle celle de CE lanceur ». La mesure qui fonde la source
// donne 0,77 unité de médiane entre une naissance et son auteur ; 4 m laisse largement passer
// le signal tout en écartant le projectile d'un joueur voisin.
const grenadeAuthorRadiusM = 4

// birthForThrow choisit la naissance qui appartient à CE lancer.
func birthForThrow(births []filmdec.ProjectileSample, at uint64,
	author *filmdec.BipedPosition) (filmdec.ProjectileSample, bool) {
	cands := birthsInWindow(births, at)
	if len(cands) == 0 {
		return filmdec.ProjectileSample{}, false
	}
	if author == nil {
		// Rien ne départage deux naissances simultanées sans le biped de l'auteur : une seule
		// candidate est une lecture, plusieurs sont un tirage au sort.
		if len(cands) == 1 {
			return cands[0], true
		}
		return filmdec.ProjectileSample{}, false
	}
	best, bestD := filmdec.ProjectileSample{}, math.MaxFloat64
	for _, c := range cands {
		if d := math.Hypot(float64(c.X-author.X), float64(c.Y-author.Y)); d < bestD {
			bestD, best = d, c
		}
	}
	if bestD > grenadeAuthorRadiusM {
		return filmdec.ProjectileSample{}, false
	}
	return best, true
}

// birthsInWindow rend TOUTES les naissances de la fenêtre, et pas seulement la plus proche
// dans le temps : c'est le fait qu'il y en ait plusieurs qui porte l'ambiguïté.
func birthsInWindow(births []filmdec.ProjectileSample, at uint64) []filmdec.ProjectileSample {
	var lo uint64
	if at > grenadeBirthWindowUS {
		lo = at - grenadeBirthWindowUS
	}
	hi := at + grenadeBirthWindowUS
	i := sort.Search(len(births), func(k int) bool { return births[k].TimestampUS >= lo })
	var out []filmdec.ProjectileSample
	for ; i < len(births) && births[i].TimestampUS <= hi; i++ {
		out = append(out, births[i])
	}
	return out
}

// projectileBirths rend le premier point de chaque projectile, trié par instant.
//
// LE TRI EST TOTAL, ET C'EST LA CONDITION DE REPRODUCTIBILITÉ DE L'ARTEFACT. Plusieurs
// projectiles naissent au MÊME instant de réplication, et l'ordre d'arrivée vient d'une
// itération de map : sans départage stable, deux constructions du même film ne rendent pas la
// même tranche (mesuré : 12,72 / −187,11 contre 11,41 / 17,99 sur le lancer t=1580 de
// `01e1f945`). Départager par la position rend l'ordre indépendant de l'amont.
//
// CE TRI NE CHOISIT PLUS LA POSITION PUBLIÉE, et c'est le correctif de `locateThrow` : le
// choix parmi les naissances d'une même fenêtre revient au biped de l'auteur, pas au rang
// dans la tranche. L'ordre reste requis — il rend `birthsInWindow` reproductible — mais il
// n'arbitre plus rien.
func projectileBirths(proj []filmdec.ProjectileTrack) []filmdec.ProjectileSample {
	out := make([]filmdec.ProjectileSample, 0, len(proj))
	for _, p := range proj {
		if len(p.Pts) > 0 {
			out = append(out, p.Pts[0])
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		switch {
		case a.TimestampUS != b.TimestampUS:
			return a.TimestampUS < b.TimestampUS
		case a.X != b.X:
			return a.X < b.X
		case a.Y != b.Y:
			return a.Y < b.Y
		default:
			return a.Z < b.Z
		}
	})
	return out
}

// keepGrenadesOfPublishedTracks écarte les lancers rattachés à un biped SANS trajectoire
// publiée.
//
// UN LANCER SITUÉ PAR SON PROJECTILE N'EST PAS CONCERNÉ : il ne dépend d'aucune trajectoire,
// sa position est celle de l'objet lancé. Le filtrer reviendrait à jeter une donnée complète
// parce qu'une donnée voisine manque.
func keepGrenadesOfPublishedTracks(gren []Grenade, tracks []Track) []Grenade {
	return keepOfPublishedTracks(gren, tracks, func(g Grenade, published map[uint32]bool) bool {
		return g.Src == GrenadeSrcProjectile || published[g.Slot]
	})
}
