package replay

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// lives_test.go — le repli « nommer la vie par la mort qui la termine ».
//
// Ces tests portent sur les PROPRIÉTÉS qui fondent le repli, pas sur des valeurs de sortie
// figées : le découpage en vies, le calage d'horloge, l'appariement, et surtout le REFUS de
// trancher quand la donnée ne le permet pas. Un test qui ne vérifierait que le cas nominal
// laisserait passer exactement les régressions qui coûtent cher ici.

// tracksOf construit l'index par slot attendu par buildLifeSpans. Le helper posAt vit
// dans shots_test.go : une seconde copie divergerait (regle du depot sur les doublons).
func tracksOf(pts ...filmdec.BipedPosition) map[uint32]slotTrack {
	return indexBySlot(pts)
}

func TestBuildLifeSpansSplitsOnGap(t *testing.T) {
	// Un même slot, deux séjours séparés par plus de lifeGapUS, le second AILLEURS (une
	// réapparition) : deux vies.
	tr := tracksOf(
		posAt(512, 1_000_000, 0, 0, 0), posAt(512, 2_000_000, 0, 0, 0),
		posAt(512, 20_000_000, 20, 0, 0), posAt(512, 21_000_000, 20, 0, 0),
	)
	lives := buildLifeSpans(tr)
	if len(lives) != 2 {
		t.Fatalf("attendu 2 vies, obtenu %d : %+v", len(lives), lives)
	}
	if lives[0].to != 2_000_000 || lives[1].from != 20_000_000 {
		t.Errorf("bornes de vie inattendues : %+v", lives)
	}
}

// A slot silent past lifeGapUS that comes back where it stopped is one life resuming after a
// dropout (measured 0.0 m on both Aquarius cases), and publication cuts it at the same place.
func TestLifeResumingInPlaceIsOneLifeInSpansAndTracks(t *testing.T) {
	pts := []filmdec.BipedPosition{
		posAt(512, 1_000_000, 5, 5, 0), posAt(512, 2_000_000, 5, 5, 0),
		posAt(512, 9_500_000, 5.3, 5, 0), posAt(512, 10_500_000, 6, 5, 0), // 7.5 s later, 0.3 m away
	}
	if lives := buildLifeSpans(tracksOf(pts...)); len(lives) != 1 {
		t.Errorf("spans = %d, want 1 life resuming in place", len(lives))
	}
	if tracks := decimateTracks(pts, 0, 100_000, 1); len(tracks) != 1 {
		t.Errorf("published tracks = %d, want 1 - the two cuts must agree", len(tracks))
	}

	moved := append([]filmdec.BipedPosition(nil), pts...)
	moved[2], moved[3] = posAt(512, 9_500_000, 25, 5, 0), posAt(512, 10_500_000, 26, 5, 0)
	if lives := buildLifeSpans(tracksOf(moved...)); len(lives) != 2 {
		t.Errorf("spans = %d, want 2 when the slot comes back 20 m away", len(lives))
	}
	if tracks := decimateTracks(moved, 0, 100_000, 1); len(tracks) != 2 {
		t.Errorf("published tracks = %d, want 2 when the slot comes back 20 m away", len(tracks))
	}
}

func TestBuildLifeSpansKeepsContinuousTrackWhole(t *testing.T) {
	// Des échantillons rapprochés ne doivent JAMAIS être coupés : un découpage trop
	// agressif fabriquerait des vies sans mort, donc des vies jamais nommées.
	var pts []filmdec.BipedPosition
	for i := 0; i < 50; i++ {
		pts = append(pts, posAt(512, uint64(i)*16_000, 0, 0, 0))
	}
	if lives := buildLifeSpans(tracksOf(pts...)); len(lives) != 1 {
		t.Fatalf("attendu 1 vie continue, obtenu %d", len(lives))
	}
}

func TestNameLivesByDeathsJoinsOnEnd(t *testing.T) {
	// Deux vies qui se terminent à 2 s et 21 s ; deux morts aux mêmes instants, décalées
	// d'une origine. L'appariement doit rendre chaque identité à sa vie.
	tr := tracksOf(
		posAt(512, 1_000_000, 0, 0, 0), posAt(512, 2_000_000, 0, 0, 0),
		posAt(513, 20_000_000, 0, 0, 0), posAt(513, 21_000_000, 0, 0, 0),
	)
	lives := buildLifeSpans(tr)
	deaths := []Death{{XUID: 111, TimeMS: 2_000 - 500}, {XUID: 222, TimeMS: 21_000 - 500}}
	off, n := bestDeathOffset(lives, deaths)
	if n != 2 {
		t.Fatalf("attendu 2 morts appariables, obtenu %d (decalage %d)", n, off)
	}
	if r := nameLivesByDeaths(lives, deaths, off, nil); r.named != 2 || r.ambiguous != 0 {
		t.Fatalf("attendu 2 vies nommees et 0 ambigue, obtenu %d et %d", r.named, r.ambiguous)
	}
	byslot := map[uint32]uint64{}
	for _, l := range lives {
		byslot[l.slot] = l.xuid
	}
	if byslot[512] != 111 || byslot[513] != 222 {
		t.Errorf("identites mal posees : %+v", byslot)
	}
}

func TestNameLivesByDeathsRefusesTiedExchange(t *testing.T) {
	// LE CAS REEL, AUX DISTANCES MESUREES. Deux vies (512 finit a 8709648 ms, 517 a 8709682)
	// et deux morts (A a 8709636, B a 8709602). Aucune distance n est a egalite : 12, 46, 46,
	// 80. Le glouton prend le 12, ce qui force le 80 — total 92 ; l appariement croise coute
	// 46 + 46 = 92, le MEME total. C est la geometrie des reapparitions (que ce paquet ne
	// connait pas) qui dit que le croise etait le bon : chaque identite s etait posee sur le
	// camp d en face. A somme egale, aucune des deux ne doit etre nommee.
	base := int64(8_709_000)
	tr := tracksOf(
		posAt(512, uint64(base-5_000)*1000, 0, 0, 0), posAt(512, uint64(base+648)*1000, 0, 0, 0),
		posAt(517, uint64(base-5_000)*1000, 0, 0, 0), posAt(517, uint64(base+682)*1000, 0, 0, 0),
	)
	lives := buildLifeSpans(tr)
	deaths := []Death{{XUID: 111, TimeMS: base + 636}, {XUID: 222, TimeMS: base + 602}}
	r := nameLivesByDeaths(lives, deaths, 0, nil)
	named, ambiguous := r.named, r.ambiguous
	if named != 0 || ambiguous != 2 {
		t.Fatalf("un echange a somme egale ne doit rien nommer, obtenu %d nommee(s) et %d ambigue(s)",
			named, ambiguous)
	}
	for _, l := range lives {
		if l.xuid != 0 {
			t.Errorf("la vie du slot %d porte une identite alors que l horloge ne la designe pas", l.slot)
		}
	}
}

func TestNameLivesByDeathsRefusesGenuineTies(t *testing.T) {
	// Deux vies qui finissent au MÊME instant, envers deux morts elles-mêmes au même
	// instant : chaque vie a, envers chaque mort, un delta identique. Rien dans l'horloge
	// ne dit laquelle des deux une mort nomme — trancher par un départage arbitraire (l'un
	// des deux camps observé sur un match réel) poserait une identité sur le mauvais
	// joueur. Aucune des deux ne doit être nommée, et cela de façon reproductible : un
	// refus qui dépendrait de l'ordre d'itération d'une map serait lui-même un pari.
	build := func() (uint64, uint64, int) {
		tr := tracksOf(
			posAt(512, 1_000_000, 0, 0, 0), posAt(512, 5_000_000, 0, 0, 0),
			posAt(513, 1_000_000, 0, 0, 0), posAt(513, 5_000_000, 0, 0, 0),
		)
		lives := buildLifeSpans(tr)
		deaths := []Death{{XUID: 111, TimeMS: 5_000}, {XUID: 222, TimeMS: 5_000}}
		ambiguous := nameLivesByDeaths(lives, deaths, 0, nil).ambiguous
		m := map[uint32]uint64{}
		for _, l := range lives {
			m[l.slot] = l.xuid
		}
		return m[512], m[513], ambiguous
	}
	a1, b1, amb1 := build()
	if a1 != 0 || b1 != 0 {
		t.Fatalf("un ecart a egalite parfaite ne doit nommer ni l'une ni l'autre vie, obtenu (%d,%d)", a1, b1)
	}
	if amb1 != 2 {
		t.Fatalf("les deux vies doivent etre comptees ambigues, obtenu %d", amb1)
	}
	for i := 0; i < 20; i++ {
		if a2, b2, amb2 := build(); a2 != a1 || b2 != b1 || amb2 != amb1 {
			t.Fatalf("refus non deterministe : (%d,%d,%d) puis (%d,%d,%d)", a1, b1, amb1, a2, b2, amb2)
		}
	}
}

func TestNameLivesByDeathsTieDoesNotBlockUnrelatedLife(t *testing.T) {
	// Le refus d'une paire ambigue ne doit pas coûter une vie qui n'a rien à voir avec
	// elle : deux vies à égalité parfaite (512, 513, mort à 5 s) plus une troisième, sans
	// lien de temps avec les deux premières (514, mort à 20 s). Cette dernière doit rester
	// nommée : le refus est scopé au palier de delta où l'ambiguïté existe, pas au match
	// entier.
	tr := tracksOf(
		posAt(512, 1_000_000, 0, 0, 0), posAt(512, 5_000_000, 0, 0, 0),
		posAt(513, 1_000_000, 0, 0, 0), posAt(513, 5_000_000, 0, 0, 0),
		posAt(514, 1_000_000, 0, 0, 0), posAt(514, 20_000_000, 0, 0, 0),
	)
	lives := buildLifeSpans(tr)
	deaths := []Death{{XUID: 111, TimeMS: 5_000}, {XUID: 222, TimeMS: 5_000}, {XUID: 333, TimeMS: 20_000}}
	r := nameLivesByDeaths(lives, deaths, 0, nil)
	named, ambiguous := r.named, r.ambiguous
	if named != 1 || ambiguous != 2 {
		t.Fatalf("attendu 1 vie nommee et 2 ambigues, obtenu %d et %d", named, ambiguous)
	}
	byslot := map[uint32]uint64{}
	for _, l := range lives {
		byslot[l.slot] = l.xuid
	}
	if byslot[514] != 333 {
		t.Errorf("la vie sans ambiguite doit rester nommee, obtenu %+v", byslot)
	}
	if byslot[512] != 0 || byslot[513] != 0 {
		t.Errorf("les vies ambigues ne doivent porter aucune identite, obtenu %+v", byslot)
	}
}

func TestOwnersFromLivesRefusesToPickOnCollision(t *testing.T) {
	// Un slot dont deux vies portent des identités différentes est une contradiction : la
	// table slot -> joueur ne peut pas la représenter. On exige qu'elle soit COMPTÉE et que
	// le slot ne soit PAS publié DU TOUT — garder la premiere lecture serait un departage par
	// ordre de parcours, et `verdictOfBridge` declare deja le pont non publiable des la
	// premiere collision : publier quand meme une identite contredirait ce verdict.
	lives := []lifeSpan{
		{slot: 512, from: 0, to: 1_000_000, xuid: 111},
		{slot: 512, from: 10_000_000, to: 11_000_000, xuid: 222},
	}
	owners, byXUID, collisions := ownersFromLives(lives, map[uint64]int{111: 0, 222: 1})
	if collisions != 1 {
		t.Errorf("attendu 1 collision comptee, obtenu %d", collisions)
	}
	if _, published := owners[512]; published {
		t.Errorf("un slot contradictoire ne doit publier AUCUNE identite, obtenu index %d", owners[512])
	}
	// Les deux tables sortent du meme parcours : elles doivent se taire ENSEMBLE, sans quoi un
	// client nommerait une trace que le rattachement de ses evenements ne connait pas.
	if _, published := byXUID[512]; published {
		t.Errorf("la table d'identites doit se taire avec celle des index, obtenu xuid %d", byXUID[512])
	}
}

func TestNameTracksLeavesUnbridgedLivesAnonymous(t *testing.T) {
	// Une vie que le fil des morts n'a pas nommee reste SANS identite. La remplir d'un
	// « inconnu » ou du porteur d'un slot voisin serait exactement la faute qui a fait
	// supprimer le vote : mieux vaut ne rien afficher que quelque chose de faux.
	tracks := []Track{{Slot: 512}, {Slot: 513}}
	nameTracks(tracks, map[uint32]uint64{512: 2533274800000001})
	if tracks[0].XUID != "2533274800000001" {
		t.Errorf("la trace pontee doit porter son xuid en decimal, obtenu %q", tracks[0].XUID)
	}
	if tracks[1].XUID != "" {
		t.Errorf("la trace non pontee doit rester anonyme, obtenu %q", tracks[1].XUID)
	}
}

func TestBuildRosterIsSortedAndStable(t *testing.T) {
	// L'ordre d'iteration d'une map Go est aleatoire : sans tri, l'artefact changerait
	// d'octets a chaque build sans changer de contenu, et deviendrait indiffable.
	idx := PlayerIndexTable{ByXUID: map[uint64]int{2533274800000003: 2, 2533274800000001: 0,
		2533274800000002: 1}}
	first := buildRoster(idx, nil)
	if len(first) != 3 || first[0].FilmIndex != 0 || first[2].FilmIndex != 2 {
		t.Fatalf("roster mal trie : %+v", first)
	}
	if first[0].XUID != "2533274800000001" {
		t.Errorf("xuid attendu en decimal, obtenu %q", first[0].XUID)
	}
	for i := 0; i < 20; i++ {
		if got := buildRoster(idx, nil); got[0].XUID != first[0].XUID || got[2].XUID != first[2].XUID {
			t.Fatalf("roster non reproductible entre deux appels : %+v puis %+v", first, got)
		}
	}
	if buildRoster(PlayerIndexTable{}, nil) != nil {
		t.Errorf("sans table d'index, pas de roster invente")
	}
}

func TestBuildOwnersPublishesNothingWithoutDeaths(t *testing.T) {
	// SANS FIL DES MORTS, LE PONT EST VIDE — et c'est le comportement voulu depuis le retrait
	// du repli voté. Ce test est le garde-fou de cette décision : si un jour une seconde
	// source réapparaît « pour améliorer la couverture », il tombera. Un rejeu muet se voit ;
	// un rejeu qui pose des tirs sur le mauvais joueur ne se voit pas.
	// Les deux vies finissent à des instants DIFFERENTS (2 s et 10 s) : un slot qui finirait
	// au même instant que 512 introduirait une ambiguïté que ce test n'a pas pour objet — cf.
	// TestNameLivesByDeathsRefusesGenuineTies pour celle-là.
	tr := tracksOf(posAt(512, 1_000_000, 0, 0, 90), posAt(512, 2_000_000, 0, 0, 90),
		posAt(513, 1_000_000, 5, 5, 270), posAt(513, 10_000_000, 5, 5, 270))
	rep := buildOwners(tr, nil, PlayerIndexTable{ByXUID: map[uint64]int{111: 4}, Readings: 26}, nil, nil)
	if len(rep.Owner) != 0 {
		t.Errorf("sans morts, AUCUN slot ne doit etre attribue : %+v", rep.Owner)
	}
	if rep.FromDeaths != 0 {
		t.Errorf("sans morts, la lecture ne produit rien : %+v", rep)
	}

	// SANS TABLE D'INDEX non plus, rien n'est publié : le pont a DEUX maillons lus, et il lui
	// faut les deux.
	if rep3 := buildOwners(tr, []Death{{XUID: 111, TimeMS: 2_000}}, PlayerIndexTable{}, nil, nil); len(rep3.Owner) != 0 {
		t.Errorf("sans table d'index, AUCUN slot ne doit etre attribue : %+v", rep3.Owner)
	}

	// Avec les deux maillons, la lecture nomme le slot.
	deaths := []Death{{XUID: 111, TimeMS: 2_000}}
	rep2 := buildOwners(tr, deaths, PlayerIndexTable{ByXUID: map[uint64]int{111: 4}, Readings: 26}, nil, nil)
	if rep2.DeathsNamed == 0 {
		t.Fatalf("attendu au moins une vie nommee, obtenu %d", rep2.DeathsNamed)
	}
	// TOUT le pont vient de la lecture : c'est l'invariant que le verdict controle aussi.
	if rep2.FromDeaths != len(rep2.Owner) {
		t.Errorf("le pont doit venir ENTIEREMENT de la lecture : %+v", rep2)
	}
}
