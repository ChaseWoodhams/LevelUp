package replay

import (
	"encoding/json"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// pos construit une position décodée à ts millisecondes du départ.
func pos(slot uint32, ms int, x, y, z float32) filmdec.BipedPosition {
	// HasWorld : les points sans coordonnée monde ne sont pas publiés (une position n'est
	// une coordonnée que si les bornes de la carte étaient connues au décodage).
	return filmdec.BipedPosition{Slot: slot, TimestampUS: uint64(ms) * 1000, X: x, Y: y, Z: z, HasWorld: true}
}

// TestBuildFromPositions_Timeline valide le mappage du temps RÉEL sur l'axe de frames :
// l'origine est le premier paquet, une frame vaut FrameIntervalMS, et la durée annoncée
// couvre tout le film (le bug historique était un axe = index de record, d'où un rejeu
// beaucoup trop court).
func TestBuildFromPositions_Timeline(t *testing.T) {
	const slot uint32 = 512
	in := []filmdec.BipedPosition{
		pos(slot, 10_000, 1, 1, 0.5),
		pos(slot, 10_100, 2, 2, 0.5),
		pos(slot, 20_000, 3, 3, 0.5), // +10 s => frame 100
	}
	doc := BuildFromPositions("000d5950", "halo_infinite", in, nil, Options{FrameIntervalMS: 100})

	if doc.SchemaVersion != SchemaVersion {
		t.Errorf("SchemaVersion = %d, attendu %d (l'ajout de champs optionnels ne l'incrémente pas)", doc.SchemaVersion, SchemaVersion)
	}
	if doc.FrameIntervalMS != 100 {
		t.Errorf("FrameIntervalMS = %d, attendu 100", doc.FrameIntervalMS)
	}
	if doc.FrameCount != 101 || doc.DurationMS != 10_100 {
		t.Errorf("FrameCount/DurationMS = %d/%d, attendu 101/10100", doc.FrameCount, doc.DurationMS)
	}
	// L'AXE COUVRE TOUT LE FILM — c'est l'objet de ce test — SANS QUE LA TRACK LE COUVRE.
	// Le trou de 10 s dépasse `lifeGapUS` : le même slot qui revient après une absence aussi
	// longue est une RÉAPPARITION, pas la suite du même déplacement. Le troisième échantillon
	// ouvre donc une seconde vie, qui reste sous MinPoints et n'est pas publiée. Ce que ce
	// test verrouille est que la première track ne s'étire PAS jusqu'à la frame 100 : recollée,
	// elle faisait tracer une ligne droite en travers de la mort et tenait le joueur pour
	// vivant tout du long (cf. decimateTracks).
	if len(doc.Tracks) != 1 {
		t.Fatalf("attendu 1 track publiee (la 2e vie est sous MinPoints), obtenu %d", len(doc.Tracks))
	}
	pts := doc.Tracks[0].Points
	want := []Point{{T: 0, X: 1, Y: 1, Z: 0.5}, {T: 1, X: 2, Y: 2, Z: 0.5}}
	if len(pts) != len(want) {
		t.Fatalf("points = %+v, attendu %+v", pts, want)
	}
	for i := range want {
		if pts[i] != want[i] {
			t.Errorf("point %d = %+v, attendu %+v", i, pts[i], want[i])
		}
	}
	if doc.Tracks[0].StartFrame != 0 || doc.Tracks[0].EndFrame != 1 {
		t.Errorf("fenêtre de vie = [%d,%d], attendu [0,1] — une vie ne franchit pas le trou",
			doc.Tracks[0].StartFrame, doc.Tracks[0].EndFrame)
	}
}

// TestBuildFromPositions_SplitsLivesOnGap : un slot qui REVIENT après plus de `lifeGapUS`
// donne DEUX tracks, pas une.
//
// C'est le défaut qui faisait bouger un joueur mort à l'écran : `decimateTracks` accumulait
// par slot, et un slot réattribué après une réapparition recollait les deux séjours en une
// seule trajectoire. Le client, lui, interpole entre deux points consécutifs — il traçait donc
// une ligne droite en travers de la mort, et `isAliveAt` tenait la vie pour continue. Constaté
// sur `0e97be38` : des trous internes de 4,8 s, 7,8 s et 13,1 s dans une même track publiée,
// pour un délai de réapparition mesuré à ~8 s.
func TestBuildFromPositions_SplitsLivesOnGap(t *testing.T) {
	const slot uint32 = 512
	in := []filmdec.BipedPosition{
		pos(slot, 0, 1, 1, 0.5),
		pos(slot, 100, 2, 2, 0.5),
		pos(slot, 10_000, 3, 3, 0.5), // +9,9 s : au-delà de lifeGapUS => nouvelle vie
		pos(slot, 10_100, 4, 4, 0.5),
	}
	doc := BuildFromPositions("000d5950", "halo_infinite", in, nil, Options{FrameIntervalMS: 100})
	if len(doc.Tracks) != 2 {
		t.Fatalf("attendu 2 tracks (deux vies), obtenu %d : %+v", len(doc.Tracks), doc.Tracks)
	}
	if doc.Tracks[0].EndFrame != 1 || doc.Tracks[1].StartFrame != 100 {
		t.Errorf("les deux vies doivent border le trou : [%d..%d] puis [%d..%d]",
			doc.Tracks[0].StartFrame, doc.Tracks[0].EndFrame,
			doc.Tracks[1].StartFrame, doc.Tracks[1].EndFrame)
	}
	// LES DEUX GARDENT LE MÊME SLOT : c'est le même biped, et le pont slot -> joueur les
	// nomme ensemble. Ce qui change est qu'une vie ne franchit plus la mort.
	if doc.Tracks[0].Slot != slot || doc.Tracks[1].Slot != slot {
		t.Errorf("les deux vies doivent porter le slot %d, obtenu %d et %d",
			slot, doc.Tracks[0].Slot, doc.Tracks[1].Slot)
	}
}

// TestBuildFromPositions_Decimation : plusieurs échantillons dans la même frame ne
// produisent qu'un point (le premier), et une track sous MinPoints n'est pas publiée.
func TestBuildFromPositions_Decimation(t *testing.T) {
	in := []filmdec.BipedPosition{
		pos(512, 0, 10, 20, 1),
		pos(512, 30, 11, 21, 1), // même frame (0..99 ms) -> écrasé
		pos(512, 60, 12, 22, 1), // idem
		pos(512, 100, 13, 23, 2),
		pos(600, 50, -5, -5, 0), // 1 seul point -> exclu
	}
	doc := BuildFromPositions("m", "halo_infinite", in, nil, Options{FrameIntervalMS: 100})
	if len(doc.Tracks) != 1 || doc.Tracks[0].Slot != 512 {
		t.Fatalf("tracks = %+v, attendu la seule track 512", doc.Tracks)
	}
	if n := len(doc.Tracks[0].Points); n != 2 {
		t.Errorf("points après décimation = %d, attendu 2", n)
	}
	if doc.Tracks[0].Team != -1 {
		t.Errorf("Team par défaut = %d, attendu -1 (attribution non faite)", doc.Tracks[0].Team)
	}
	want := Bounds{MinX: 10, MinY: 20, MaxX: 13, MaxY: 23, MinZ: 1, MaxZ: 2}
	if doc.Bounds != want {
		t.Errorf("Bounds = %+v, attendu %+v", doc.Bounds, want)
	}
}

// TestBuildFromPositions_Empty : pas de position -> document vide mais bien formé.
func TestBuildFromPositions_Empty(t *testing.T) {
	doc := BuildFromPositions("m", "halo_infinite", nil, nil, Options{})
	if len(doc.Tracks) != 0 || doc.FrameCount != 0 || doc.DurationMS != 0 {
		t.Errorf("document non vide: %+v", doc)
	}
	if doc.FrameIntervalMS != DefaultFrameIntervalMS {
		t.Errorf("FrameIntervalMS = %d, attendu le défaut %d", doc.FrameIntervalMS, DefaultFrameIntervalMS)
	}
}

// TestGeometryBounds : la géométrie porte ses propres bornes (les props débordent de la
// zone parcourue), et l'absence de géométrie laisse le champ nil.
func TestGeometryBounds(t *testing.T) {
	objs := []MapObject{{TypeID: 1, X: -10, Y: 5}, {TypeID: 2, X: 30, Y: -8}}
	doc := BuildFromPositions("m", "halo_infinite", nil, nil, Options{Geometry: objs})
	if doc.GeometryBounds == nil {
		t.Fatal("GeometryBounds nil alors que la géométrie est fournie")
	}
	want := Bounds{MinX: -10, MinY: -8, MaxX: 30, MaxY: 5}
	if *doc.GeometryBounds != want {
		t.Errorf("GeometryBounds = %+v, attendu %+v", *doc.GeometryBounds, want)
	}
	if BuildFromPositions("m", "halo_infinite", nil, nil, Options{}).GeometryBounds != nil {
		t.Error("GeometryBounds devrait être nil sans géométrie")
	}
}

// TestShieldZeroSurvivesSerialization — GARDE-RAIL du piège omitempty. Un bouclier BRISÉ
// (0) est l'information la plus utile du champ ; s'il était déclaré `float32,omitempty` il
// serait omis exactement comme une absence de mesure, et le client afficherait un bouclier
// plein sur un joueur qui vient d'en perdre un. Ce test échoue si quelqu'un « simplifie »
// le pointeur en valeur.
func TestShieldZeroSurvivesSerialization(t *testing.T) {
	zero, full := float32(0), float32(1)
	doc := ReplayDocument{Tracks: []Track{{Slot: 1, Points: []Point{
		{T: 0, X: 1, Y: 2, Sh: &zero},
		{T: 1, X: 1, Y: 2, Sh: &full},
		{T: 2, X: 1, Y: 2}, // pas de mesure
	}}}}
	blob, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(blob), `"sh":0`) {
		t.Fatalf("un bouclier à zéro a été OMIS de la sérialisation — piège omitempty : %s", blob)
	}
	var back ReplayDocument
	if err := json.Unmarshal(blob, &back); err != nil {
		t.Fatal(err)
	}
	p := back.Tracks[0].Points
	if p[0].Sh == nil || *p[0].Sh != 0 {
		t.Fatalf("bouclier nul relu comme %v", p[0].Sh)
	}
	if p[1].Sh == nil || *p[1].Sh != 1 {
		t.Fatalf("bouclier plein relu comme %v", p[1].Sh)
	}
	if p[2].Sh != nil {
		t.Fatalf("absence de mesure relue comme %v", *p[2].Sh)
	}
}
