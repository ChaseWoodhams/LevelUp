package replay

import (
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/weaponv3"
)

// lives_witness.go — UN SECOND TÉMOIN POUR DÉPARTAGER LES VIES QUE L'HORLOGE NE SÉPARE PAS.
//
// LE PROBLÈME QU'IL ADRESSE. `nameLivesByDeaths` apparie sur UNE seule grandeur : l'écart
// entre la fin d'une vie et l'instant d'une mort. Deux joueurs qui meurent dans la même
// milliseconde — une grenade, un doublé — produisent deux morts et deux vies à distance
// EXACTEMENT égale, et l'horloge n'a plus rien à dire. Les deux paires sont alors refusées
// (cf. tiedRival / tiedExchange), ce qui laisse deux vies anonymes et, avec elles, tous les
// tirs qu'elles portaient.
//
// CE QUI NE PEUT PAS SERVIR DE SECOND TÉMOIN, vérifié avant d'écrire ce fichier :
//
//	la position de la mort   le fil des morts porte {XUID, gamertag, type, instant} et
//	                         RIEN d'autre — ni lieu, ni tueur.
//	l'équipe                 Track.Team vaut -1 : le film ne la porte pas (cf. document.go).
//	l'instant des tirs       FireEvent porte son tireur mais NI slot NI position. « Ce joueur
//	                         a-t-il tiré pendant cette vie » ne discrimine rien : une vie dure
//	                         des secondes, et sur cette fenêtre tout le monde tire.
//
// CE QUI SERT. L'arme. `FireEvent` porte l'arme du TIREUR (par index de film) et
// `KeyframeLoadout` porte les familles portées par un SLOT, sur la même horloge et par deux
// décodeurs indépendants. Croiser les deux dit si le joueur X portait, à cet instant, ce que
// le slot S portait. C'est la mesure que `loadouts.go` documente déjà comme témoin croisé :
// 98,3 % d'accord sur le bon slot contre 7,2-7,7 % sur un autre slot vivant (Cliffhanger).
//
// LA LIMITE EST CONNUE ET ELLE EST RESPECTÉE ICI. Le même fichier mesure Catalyst (Slayer
// standard) : positif 99,0 %, mais négatifs 76-81 %, parce que 120 loadouts sur 168 y sont
// IDENTIQUES (AR + Sidekick de départ). Quand tout le monde porte la même chose, l'arme ne
// distingue personne. D'où LA RÈGLE STRICTE ci-dessous : le témoin ne tranche QUE lorsqu'un
// candidat a des accords et que l'autre n'en a AUCUN. Sur un film à loadouts uniformes les
// deux candidats marquent, le témoin se tait, et le refus reste celui d'aujourd'hui. Un
// témoin faible devient ainsi SILENCIEUX, jamais faux — c'est la propriété qui autorise à
// l'ajouter sans rouvrir la question tranchée le 2026-07-28.

// weaponWitnessToleranceUS borne l'âge du keyframe consulté pour un tir. Les keyframes
// arrivent toutes les ~18-20 s (cf. loadouts.go) et le loadout ne change qu'au ramassage ;
// au-delà de cette borne on ne conclut pas, plutôt que de comparer à une arme périmée.
//
// LES 4 DÉSACCORDS EXAMINÉS DE CLIFFHANGER étaient tous des tirs survenus 3 à 17 s après la
// dernière image-clé de leur vie. Cette borne les exclut du témoignage au lieu de les
// compter comme des contradictions.
const weaponWitnessToleranceUS = 25_000_000

// weaponShot est un tir réduit à ce que le témoin utilise : quand, et quelle famille d'arme.
type weaponShot struct {
	tUS    uint64
	family uint32
}

// slotFamilies est le loadout d'un slot à un instant : les familles portées.
type slotFamilies struct {
	tUS      uint64
	families map[uint32]bool
}

// weaponWitness répond à une seule question : les armes que CE JOUEUR tirait apparaissent-
// elles dans les armes que CE SLOT portait, aux mêmes instants ?
//
// Les deux côtés viennent de décodeurs différents (records type 105 pour les tirs, records
// biped des keyframes pour les loadouts) : c'est ce qui fait du croisement un témoignage et
// non une tautologie.
type weaponWitness struct {
	shotsByIndex map[int][]weaponShot    // index de film -> tirs, triés par instant
	loadBySlot   map[uint32][]slotFamilies // slot -> loadouts, triés par instant
	indexOf      map[uint64]int          // xuid -> index de film
}

// newWeaponWitness indexe les deux lectures. Rend nil si l'une manque : un témoin sans
// preuve doit être ABSENT, pas silencieusement vide — l'appelant teste nil et se comporte
// alors exactement comme avant ce fichier.
func newWeaponWitness(
	fire []filmdec.FireEvent, loads []filmdec.KeyframeLoadout, xuidToIndex map[uint64]int,
) *weaponWitness {
	if len(fire) == 0 || len(loads) == 0 || len(xuidToIndex) == 0 {
		return nil
	}
	w := &weaponWitness{
		shotsByIndex: map[int][]weaponShot{},
		loadBySlot:   map[uint32][]slotFamilies{},
		indexOf:      xuidToIndex,
	}
	known := weaponv3.KnownWeaponHigh32
	for _, e := range fire {
		if e.WeaponID == 0 {
			continue
		}
		fam := uint32(e.WeaponID >> 32)
		// Une famille hors catalogue ne témoigne de rien : le loadout n'est balayé QUE sur
		// le catalogue, donc elle serait absente des deux côtés et compterait comme un
		// désaccord systématique.
		if _, ok := known[fam]; !ok {
			continue
		}
		w.shotsByIndex[e.FilmIndex] = append(w.shotsByIndex[e.FilmIndex], weaponShot{e.TimestampUS, fam})
	}
	for _, l := range loads {
		if len(l.Families) == 0 {
			continue
		}
		set := make(map[uint32]bool, len(l.Families))
		for _, f := range l.Families {
			set[f] = true
		}
		w.loadBySlot[l.Slot] = append(w.loadBySlot[l.Slot], slotFamilies{l.TimestampUS, set})
	}
	for k := range w.shotsByIndex {
		s := w.shotsByIndex[k]
		sort.Slice(s, func(i, j int) bool { return s[i].tUS < s[j].tUS })
	}
	for k := range w.loadBySlot {
		s := w.loadBySlot[k]
		sort.Slice(s, func(i, j int) bool { return s[i].tUS < s[j].tUS })
	}
	return w
}

// agreement compte, sur la fenêtre [fromUS, toUS], les tirs du joueur xuid dont l'arme est
// présente (hits) ou absente (misses) du loadout du slot au même instant.
//
// LE LOADOUT CONSULTÉ EST LE DERNIER <= T, comme dans loadouts.go : un loadout vaut jusqu'au
// keyframe suivant. Un tir sans keyframe antérieur assez proche ne témoigne pas du tout.
func (w *weaponWitness) agreement(xuid uint64, slot uint32, fromUS, toUS int64) (hits, misses int) {
	if w == nil {
		return 0, 0
	}
	idx, ok := w.indexOf[xuid]
	if !ok {
		return 0, 0
	}
	shots := w.shotsByIndex[idx]
	loads := w.loadBySlot[slot]
	if len(shots) == 0 || len(loads) == 0 {
		return 0, 0
	}
	lo := sort.Search(len(shots), func(i int) bool { return int64(shots[i].tUS) >= fromUS })
	for i := lo; i < len(shots) && int64(shots[i].tUS) <= toUS; i++ {
		fams, ok := w.loadoutAt(loads, shots[i].tUS)
		if !ok {
			continue
		}
		if fams[shots[i].family] {
			hits++
		} else {
			misses++
		}
	}
	return hits, misses
}

// loadoutAt rend les familles portées par le slot au dernier keyframe <= tUS, si celui-ci
// est assez récent pour témoigner.
func (w *weaponWitness) loadoutAt(loads []slotFamilies, tUS uint64) (map[uint32]bool, bool) {
	i := sort.Search(len(loads), func(i int) bool { return loads[i].tUS > tUS }) - 1
	if i < 0 || tUS-loads[i].tUS > weaponWitnessToleranceUS {
		return nil, false
	}
	return loads[i].families, true
}

// verdict est ce que le témoin dit d'un choix entre deux options.
type verdict int

const (
	// verdictSilent : le témoin ne tranche pas. C'est le cas NOMINAL sur un film à loadouts
	// uniformes, et il laisse le refus d'ambiguïté intact.
	verdictSilent verdict = iota
	// verdictFirst / verdictSecond : une option a des accords, l'autre AUCUN.
	verdictFirst
	verdictSecond
)

// choose applique la règle stricte : trancher seulement si un côté a des accords et que
// l'autre n'en a aucun.
//
// POURQUOI PAS « LE PLUS D'ACCORDS ». Comparer 12 contre 9 serait un vote sur une grandeur
// bruitée, et c'est exactement ce que ce chantier a retiré d'owners.go. Zéro contre non-zéro
// n'est pas une préférence : c'est l'absence totale de preuve d'un côté.
func choose(hitsA, hitsB int) verdict {
	switch {
	case hitsA > 0 && hitsB == 0:
		return verdictFirst
	case hitsB > 0 && hitsA == 0:
		return verdictSecond
	default:
		return verdictSilent
	}
}
