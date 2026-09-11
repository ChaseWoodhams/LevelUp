package replay

import (
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

// lives.go — LE PONT SLOT -> JOUEUR, LU AU LIEU D'ÊTRE VOTÉ.
//
// POURQUOI CE FICHIER REMPLACE UN VOTE. Le pont vivait dans owners.go, où il était élu :
// les lancers de grenade votaient pour désigner le propriétaire d'un slot. Un vote a
// besoin d'électeurs, et il y avait 70 lancers pour 99 vies — dont le premier à 73,1 s.
// Aucune vie antérieure ne pouvait donc être nommée, quelle que soit la qualité du
// décodage : **le défaut était dans le choix de la méthode, pas dans les données**.
// Résultat mesuré de ce vote : 26 slots couverts sur 99, et 147 tirs publiés sur 519.
//
// CE QU'ON FAIT À LA PLACE. Chaque vie de biped se termine par une mort, et le film porte
// le fil des morts : une victime, datée, nommée par son XUID. On nomme donc chaque vie par
// LA MORT QUI LA TERMINE. C'est une jointure sur un fait, pas une élection.
//
// MESURES sur 000d5950 (cmd/tmp_deathnaming), toutes avec leur témoin :
//
//	vies nommées                    90 / 105        témoin (morts replacées au hasard) : 10
//	écart d'appariement             médiane 34 ms, maximum 36 ms
//	slots changeant de porteur      0 / 90          — la table slot -> joueur est licite
//	tirs rattachés                  475 / 519 = 91,5 %   contre 398 par le vote supprimé
//	arme du tir dans le loadout     405 / 418 = 96,9 %   témoin (autre slot vivant) : 3,7 %
//
// LE DERNIER CONTRÔLE EST LE PLUS IMPORTANT : il ne partage AUCUNE pièce avec ce fichier.
// L'arme vient des records de dégât du flux de trames, le loadout du balayage des familles
// dans les records de biped des images-clés. Un rapport de 26x entre le rattachement et
// son témoin ne s'obtient pas par construction.
//
// L'INDEX DE JOUEUR N'EST PLUS RÉSOLU, IL EST LU. Il fut un temps calculé par affectation de
// coût minimal sur les 8! permutations ; le film l'écrit, et `player_index.go` le lit (26
// chunks concordants sur 000d5950, table identique à celle que le calcul produisait). Le pont
// n'a donc plus aucune part de choix : deux lectures composées, et rien d'autre.

// deathMatchWindowMS est l'écart maximal accepté entre la fin d'une vie et une mort du
// fil. La médiane mesurée étant de 34 ms et le maximum de 36, cette fenêtre borne le bruit
// d'horloge, pas le signal.
const deathMatchWindowMS = 150

// lifeGapUS : au-delà de ce trou dans un même slot, on ouvre une nouvelle vie. 5 s est
// très au-dessus du pas de réplication (~16 ms) et bien en deçà du temps de réapparition
// mesuré (médiane 8,0 s).
const lifeGapUS = 5_000_000

// Death est une mort du fil, telle que le film la porte : une identité et un instant.
// L'identité est le XUID — jamais un index (cf. la règle « un ordre n'est pas une
// identité », qui a déjà produit une fausse découverte dans ce chantier).
type Death struct {
	// XUID identifie la victime. Stable, global, indépendant de tout tri.
	XUID uint64
	// Gamertag est le nom porté PAR LE FILM lui-même, dans le même enregistrement que le xuid
	// (32 octets UTF-16LE). Il n'est pas obligatoire au rattachement — celui-ci ne travaille
	// que sur le xuid — mais il rend le rejeu lisible SANS base de données, ce qui est la
	// propriété que tout ce pipeline cherche à préserver. Vide si l'enregistrement ne le porte
	// pas ; l'identité reste alors le xuid.
	Gamertag string
	// TimeMS est l'instant de la mort sur l'horloge du MATCH (origine = début du match),
	// qui n'est pas celle du film. Le décalage entre les deux est résolu par mesure.
	TimeMS int64
}

// lifeSpan est une vie de biped : les positions d'un même slot sans trou majeur.
type lifeSpan struct {
	slot     uint32
	from, to int64  // microsecondes, horloge du film
	xuid     uint64 // identité lue dans le fil des morts ; 0 = non nommée
}

// lifeResumeM : a slot silent for more than lifeGapUS that comes back within this distance of
// where it stopped is the SAME life resuming after a replication dropout, not a respawn.
//
// Measured (lives_split_measure_test.go): both same-slot returns on the archived Aquarius films
// resumed 0.0 m from their last position, after 7.2 s and 7.9 s; left as two lives, the second
// half carried the slot's name onto a second track. Every respawn in the six films' dumps landed
// 5.4 m or more from where the previous life ended — a respawn is a spawn point, not the spot.
const lifeResumeM = 1.0

// resumesInPlace reports whether b continues a in place: both carry world coordinates (a quantum
// without map bounds is not a position) and b lies within lifeResumeM of a.
//
// ONE RULE FOR TWO CUTS. buildLifeSpans (naming) and decimateTracks (publication) must cut lives
// at the same places, or the bridge would name lives the artifact does not carry.
func resumesInPlace(a, b filmdec.BipedPosition) bool {
	if !a.HasWorld || !b.HasWorld {
		return false
	}
	dx, dy, dz := float64(b.X-a.X), float64(b.Y-a.Y), float64(b.Z-a.Z)
	return dx*dx+dy*dy+dz*dz <= lifeResumeM*lifeResumeM
}

// buildLifeSpans découpe les trajectoires en vies. Un slot qui disparaît plus de lifeGapUS
// puis revient AILLEURS est une NOUVELLE vie : le slot migre aux réapparitions. S'il revient là
// où il s'était arrêté (resumesInPlace), c'est la même vie après une coupure de réplication.
func buildLifeSpans(tracks map[uint32]slotTrack) []lifeSpan {
	slots := make([]uint32, 0, len(tracks))
	for s := range tracks {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	var out []lifeSpan
	for _, s := range slots {
		pts := tracks[s].pts
		if len(pts) == 0 {
			continue
		}
		start, last, prev := int64(pts[0].TimestampUS), int64(pts[0].TimestampUS), pts[0]
		for _, p := range pts[1:] {
			t := int64(p.TimestampUS)
			if t-last > lifeGapUS && !resumesInPlace(prev, p) {
				out = append(out, lifeSpan{slot: s, from: start, to: last})
				start = t
			}
			last, prev = t, p
		}
		out = append(out, lifeSpan{slot: s, from: start, to: last})
	}
	return out
}

// bestDeathOffset résout le décalage entre l'horloge du fil des morts et celle du film.
//
// LE MAXIMUM EST UN PLATEAU : toute la largeur de la fenêtre d'acceptation donne le même
// compte. On retient son CENTRE — au bord, tous les écarts d'appariement vaudraient la
// demi-fenêtre, ce qui ferait passer un mauvais calage pour bon.
func bestDeathOffset(lives []lifeSpan, deaths []Death) (int64, int) {
	ends := lifeEndsMS(lives)
	if len(ends) == 0 || len(deaths) == 0 {
		return 0, 0
	}
	lo, hi := ends[0], ends[0]
	for _, e := range ends {
		lo, hi = minI64(lo, e), maxI64(hi, e)
	}
	bestN := -1
	var plateau []int64
	// La plage balayée est celle des fins de vie : l'origine du fil des morts est le début
	// du match, qui tombe forcément dedans. La marge amont couvre l'avant-match.
	for off := lo - 60_000; off <= hi; off += 10 {
		if n := countDeathMatches(ends, deaths, off); n > bestN {
			bestN, plateau = n, []int64{off}
		} else if n == bestN {
			plateau = append(plateau, off)
		}
	}
	return plateau[len(plateau)/2], bestN
}

// countDeathMatches compte les morts appariables à une fin de vie, chaque vie servant une
// seule fois.
func countDeathMatches(ends []int64, deaths []Death, off int64) int {
	used := make([]bool, len(ends))
	n := 0
	for _, d := range deaths {
		if i := nearestFreeEnd(ends, used, d.TimeMS+off); i >= 0 {
			used[i] = true
			n++
		}
	}
	return n
}

// nearestFreeEnd rend l'index de la fin de vie libre la plus proche de target, dans la
// fenêtre ; -1 si aucune.
func nearestFreeEnd(ends []int64, used []bool, target int64) int {
	bi, bd := -1, int64(deathMatchWindowMS+1)
	for i, e := range ends {
		if used[i] {
			continue
		}
		if d := absI64(e - target); d < bd {
			bd, bi = d, i
		}
	}
	return bi
}

// nameLivesByDeaths pose l'identité de la victime sur la vie que sa mort termine.
//
// L'APPARIEMENT EST GLOUTON PAR ÉCART CROISSANT, ce qui rend le résultat indépendant de
// l'ordre d'itération — une boucle naïve donnerait un résultat différent selon l'ordre des
// slots, donc non reproductible.
//
// LE PLUS PETIT ÉCART N'EST PAS UNE PREUVE : C'EST LE TOTAL QUI DÉCIDE, ET IL PEUT ÊTRE À
// ÉGALITÉ. Le glouton prend la plus petite distance d'abord, ce qui force tout le reste. Sur un
// ÉCHANGE (deux joueurs qui tombent l'un sur l'autre), les quatre distances mesurées sur un
// match réel étaient :
//
//	vie 512 (fin 8709648)  <- mort A (8709636) : 12 ms      <- mort B (8709602) : 46 ms
//	vie 517 (fin 8709682)  <- mort A (8709636) : 46 ms      <- mort B (8709602) : 80 ms
//
// Le glouton prend le 12, ce qui laisse 80 : total 92. L'appariement CROISÉ coûte 46 + 46 =
// 92 — EXACTEMENT le même total. Les deux lectures sont donc également compatibles avec
// l'horloge, et c'est la mauvaise qui a été retenue : chaque identité s'est posée sur le camp
// d'en face (vérifié sur la géométrie des réapparitions, que ce paquet ne connaît pas et ne
// doit pas connaître — le film ne porte aucune notion d'équipe).
//
// AUCUN ÉCART INDIVIDUEL N'ÉTAIT POURTANT À ÉGALITÉ : 12, 46, 46, 80. Un test d'égalité sur les
// distances ne voit rien ; ce qui est à égalité, c'est la SOMME de l'appariement. On cherche
// donc, avant de valider une paire, s'il existe un ÉCHANGE (d,l)+(d',l') -> (d,l')+(d',l) qui
// ne coûte pas plus cher. S'il en existe un, rien dans l'horloge ne départage les deux
// lectures, et AUCUNE des quatre extrémités n'est nommée — la règle qui a déjà fait retirer le
// vote de owners.go : mieux vaut ne rien afficher que quelque chose de faux.
//
// LE TÉMOIN D'ARME (w, nil = absent) NE CHANGE QUE LES CAS REFUSÉS. Il n'est consulté qu'à
// l'intérieur des deux branches d'ambiguïté ci-dessus, jamais sur une paire que l'horloge
// tranche seule : ce qui était nommé hier l'est encore, à l'identique. Et il ne parle que
// lorsqu'un candidat a des accords d'arme et l'autre AUCUN (cf. lives_witness.go), de sorte
// qu'un film à loadouts uniformes le rend silencieux et rétablit exactement le refus.
func nameLivesByDeaths(
	lives []lifeSpan, deaths []Death, off int64, w *weaponWitness,
) namingReport {
	ends := lifeEndsMS(lives)
	type pair struct {
		di, li int
		d      int64
	}
	var ps []pair
	cost := map[[2]int]int64{}
	byDeath, byLife := map[int][]int{}, map[int][]int{}
	for di, d := range deaths {
		target := d.TimeMS + off
		for li, e := range ends {
			delta := absI64(e - target)
			if delta > deathMatchWindowMS {
				continue
			}
			ps = append(ps, pair{di, li, delta})
			cost[[2]int{di, li}] = delta
			byDeath[di] = append(byDeath[di], li)
			byLife[li] = append(byLife[li], di)
		}
	}
	sort.Slice(ps, func(i, j int) bool {
		if ps[i].d != ps[j].d {
			return ps[i].d < ps[j].d
		}
		return ps[i].li < ps[j].li // départage stable : jamais l'ordre de la map
	})
	usedD, usedL := make([]bool, len(deaths)), make([]bool, len(lives))
	rejectedD, rejectedL := make([]bool, len(deaths)), make([]bool, len(lives))
	free := func(di, li int) bool {
		return !usedD[di] && !usedL[li] && !rejectedD[di] && !rejectedL[li]
	}
	var rep namingReport
	for _, p := range ps {
		if !free(p.di, p.li) {
			continue
		}
		// Un rival À DISTANCE ÉGALE suffit, sans qu'un échange complet existe : UNE mort
		// entre DEUX vies qui finissent au même instant n'a pas de second appariement à
		// comparer, et l'ordre de parcours tranchait seul.
		if rd, rl, tied := tiedRival(p.di, p.li, p.d, cost, byDeath, byLife, free); tied {
			switch w.judgeRival(p.di, p.li, rd, rl, lives, deaths) {
			case verdictFirst:
				rep.tiesResolved++ // la paire proposée est soutenue, le rival ne l'est pas
			case verdictSecond:
				// Le témoin soutient le RIVAL : on laisse simplement passer cette paire.
				// Rien n'est marqué — le rival est encore libre, sa propre paire est plus
				// loin dans `ps` à la même distance, et elle sera jugée à son tour.
				rep.witnessDeferred++
				continue
			default:
				// LE RIVAL SORT AUSSI. Ne rejeter que la paire proposée laisserait le rival
				// libre pour l'appariement suivant, où il serait à son tour tranché par le
				// seul ordre de parcours — le tirage au sort déplacé d'un cran, pas supprimé.
				rejectedD[p.di], rejectedL[p.li] = true, true
				if rd >= 0 {
					rejectedD[rd] = true
				}
				if rl >= 0 {
					rejectedL[rl] = true
				}
				continue
			}
		} else if di2, li2, tied := tiedExchange(p.di, p.li, p.d, cost, byDeath, byLife, free); tied {
			switch w.judgeExchange(p.di, p.li, di2, li2, lives, deaths) {
			case verdictFirst:
				rep.tiesResolved++
			case verdictSecond:
				rep.witnessDeferred++
				continue
			default:
				rejectedD[p.di], rejectedL[p.li] = true, true
				rejectedD[di2], rejectedL[li2] = true, true
				continue
			}
		} else {
			// Paire tranchée par l'horloge seule : c'est le CAS TÉMOIN, celui dont on connaît
			// déjà la réponse. On y interroge l'arme sans jamais s'en servir, uniquement pour
			// publier si ce second témoin dit la même chose que le premier.
			rep.control(w, p.di, p.li, lives, deaths)
		}
		usedD[p.di], usedL[p.li] = true, true
		lives[p.li].xuid = deaths[p.di].XUID
		rep.named++
	}
	for _, r := range rejectedL {
		if r {
			rep.ambiguous++
		}
	}
	return rep
}

// namingReport porte le résultat du nommage ET de quoi juger le second témoin.
//
// LE CONTRÔLE EST PUBLIÉ AVEC L'EFFET, et c'est le point : `tiesResolved` seul dirait
// combien de refus ont été levés, sans dire si on a eu raison de les lever. Les compteurs
// d'accord/désaccord mesurent le témoin là où la réponse est déjà connue — les paires que
// l'horloge tranche sans ambiguïté — et c'est le seul endroit où il est falsifiable.
type namingReport struct {
	named, ambiguous int
	// tiesResolved : ambiguïtés levées par l'arme (le témoin a parlé pour la paire posée).
	tiesResolved int
	// witnessDeferred : ambiguïtés où le témoin a désigné l'autre lecture. Comptées à part
	// parce qu'elles ne nomment rien ICI — la paire soutenue est jugée à son tour.
	witnessDeferred int
	// controlAgree / controlContradict : sur les paires NON ambiguës, l'arme confirme-t-elle
	// l'horloge ? Un désaccord notable disqualifierait le témoin partout ailleurs.
	controlAgree, controlContradict int
	// controlSilent : paires non ambiguës sans aucun tir exploitable. Publié pour que
	// l'accord se lise sur son vrai dénominateur.
	controlSilent int
}

// control interroge le témoin sur une paire que l'horloge a tranchée seule, et ne fait qu'en
// enregistrer l'avis.
func (r *namingReport) control(w *weaponWitness, di, li int, lives []lifeSpan, deaths []Death) {
	if w == nil {
		return
	}
	l := lives[li]
	hits, misses := w.agreement(deaths[di].XUID, l.slot, l.from, l.to)
	switch {
	case hits == 0 && misses == 0:
		r.controlSilent++
	case hits > 0 && misses == 0:
		r.controlAgree++
	case hits == 0 && misses > 0:
		r.controlContradict++
	default:
		// Accords ET désaccords sur la même vie : le loadout a changé en cours de vie
		// (ramassage) ou une image-clé manque. Ni confirmation ni contradiction.
		r.controlSilent++
	}
}

// judgeRival départage une mort entre deux vies, ou une vie entre deux morts.
func (w *weaponWitness) judgeRival(
	di, li, rd, rl int, lives []lifeSpan, deaths []Death,
) verdict {
	if w == nil {
		return verdictSilent
	}
	a, _ := w.agreement(deaths[di].XUID, lives[li].slot, lives[li].from, lives[li].to)
	switch {
	case rl >= 0: // MÊME mort, deux vies : quelle vie portait l'arme de ce joueur ?
		b, _ := w.agreement(deaths[di].XUID, lives[rl].slot, lives[rl].from, lives[rl].to)
		return choose(a, b)
	case rd >= 0: // MÊME vie, deux morts : lequel des deux joueurs portait celle du slot ?
		b, _ := w.agreement(deaths[rd].XUID, lives[li].slot, lives[li].from, lives[li].to)
		return choose(a, b)
	default:
		return verdictSilent
	}
}

// judgeExchange départage les deux lectures d'un échange : {di->li, di2->li2} contre
// {di->li2, di2->li}. On compare les SOMMES d'accords des deux affectations complètes, parce
// que c'est l'affectation entière qui est à égalité de coût, pas une seule de ses branches.
func (w *weaponWitness) judgeExchange(
	di, li, di2, li2 int, lives []lifeSpan, deaths []Death,
) verdict {
	if w == nil {
		return verdictSilent
	}
	straight, _ := w.agreement(deaths[di].XUID, lives[li].slot, lives[li].from, lives[li].to)
	s2, _ := w.agreement(deaths[di2].XUID, lives[li2].slot, lives[li2].from, lives[li2].to)
	crossed, _ := w.agreement(deaths[di].XUID, lives[li2].slot, lives[li2].from, lives[li2].to)
	c2, _ := w.agreement(deaths[di2].XUID, lives[li].slot, lives[li].from, lives[li].to)
	return choose(straight+s2, crossed+c2)
}

// tiedRival dit si la paire proposée a un rival À DISTANCE ÉGALE : une autre vie encore libre
// à la même distance de cette mort, ou une autre mort encore libre à la même distance de cette
// vie. C'est l'ambiguïté qui n'a pas besoin d'un second appariement pour exister — une seule
// mort entre deux vies jumelles — et que `tiedExchange`, qui compare des SOMMES, ne voit pas.
// Le rival est rendu (mort ou vie, l'autre à -1) pour sortir avec la paire : cf. l'appelant.
func tiedRival(
	di, li int, dl int64,
	cost map[[2]int]int64, byDeath, byLife map[int][]int,
	free func(int, int) bool,
) (rivalDeath, rivalLife int, tied bool) {
	for _, li2 := range byDeath[di] {
		if li2 != li && free(di, li2) && cost[[2]int{di, li2}] == dl {
			return -1, li2, true
		}
	}
	for _, di2 := range byLife[li] {
		if di2 != di && free(di2, li) && cost[[2]int{di2, li}] == dl {
			return di2, -1, true
		}
	}
	return -1, -1, false
}

// tiedExchange cherche un échange qui coûterait AUTANT OU MOINS que la paire proposée.
//
// LES QUATRE DISTANCES DOIVENT EXISTER. Un échange dont une branche sort de la fenêtre
// d'acceptation n'est pas une lecture concurrente : il laisserait une extrémité sans mort, ce
// qui est un appariement différent, pas le même à l'envers. On ne compare donc que des
// échanges complets — c'est ce qui garde le refus rare, et attaché au cas qui l'a motivé.
func tiedExchange(
	di, li int, dl int64,
	cost map[[2]int]int64, byDeath, byLife map[int][]int,
	free func(int, int) bool,
) (int, int, bool) {
	for _, li2 := range byDeath[di] {
		if li2 == li || !free(di, li2) {
			continue
		}
		for _, di2 := range byLife[li] {
			if di2 == di || !free(di2, li) || !free(di2, li2) {
				continue
			}
			c22, ok := cost[[2]int{di2, li2}]
			if !ok {
				continue
			}
			// Le glouton s'apprête à poser (di,li) ; l'autre lecture pose (di,li2) et
			// (di2,li). À somme égale, l'horloge ne dit plus laquelle est la bonne.
			if cost[[2]int{di, li2}]+cost[[2]int{di2, li}] <= dl+c22 {
				return di2, li2, true
			}
		}
	}
	return 0, 0, false
}

// lifeEndsMS rend la fin de chaque vie en millisecondes.
func lifeEndsMS(lives []lifeSpan) []int64 {
	out := make([]int64, len(lives))
	for i, l := range lives {
		out[i] = l.to / 1000
	}
	return out
}

// ownersFromLives compose la table slot -> index de joueur à partir des vies nommées et du
// pont index -> XUID.
//
// LA TABLE EST LICITE PARCE QU'UN SLOT NE CHANGE PAS DE PORTEUR : mesuré 0 slot sur 90 sur
// 000d5950. Si un film violait cette propriété, la table serait fausse par construction —
// d'où le compteur de collisions rendu au rapport plutôt que masqué.
// Le second retour donne slot -> XUID, c'est-à-dire l'IDENTITÉ du porteur et non son rang.
// Les deux sortent du même parcours et de la même règle de collision : les séparer ferait
// diverger deux tables censées dire la même chose.
func ownersFromLives(
	lives []lifeSpan, xuidToIndex map[uint64]int,
) (map[uint32]int, map[uint32]uint64, int) {
	out := map[uint32]int{}
	byXUID := map[uint32]uint64{}
	conflicted := map[uint32]bool{}
	collisions := 0
	for _, l := range lives {
		if l.xuid == 0 {
			continue
		}
		idx, ok := xuidToIndex[l.xuid]
		if !ok {
			continue
		}
		if prev, seen := out[l.slot]; seen && prev != idx {
			collisions++
			conflicted[l.slot] = true
			continue
		}
		out[l.slot] = idx
		byXUID[l.slot] = l.xuid
	}
	// LE SLOT EN CONFLIT SORT ENTIÈREMENT, PREMIÈRE LECTURE COMPRISE. Il ne sortait pas :
	// la première identité lue restait publiée et seule la contradictoire était écartée —
	// ce que le commentaire d'à côté décrivait pourtant déjà comme « on ne tranche pas, on
	// ne publie pas ». Garder la première est un départage par ORDRE DE PARCOURS, c'est-à-dire
	// le tirage au sort que tout ce fichier refuse ailleurs ; et `verdictOfBridge` déclare
	// DÉJÀ le pont « non publiable » dès la première collision, si bien que le pont servait
	// une identité que son propre verdict disait irrecevable.
	for slot := range conflicted {
		delete(out, slot)
		delete(byXUID, slot)
	}
	return out, byXUID, collisions
}

func absI64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func minI64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func maxI64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
