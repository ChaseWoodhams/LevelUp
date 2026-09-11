package filmdec

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"testing"
)

// default_state_sweep_test.go — CALIBRER LE PRÉFIXE D'UN ENREGISTREMENT KEYFRAME.
//
// CE QUE « PRÉFIXE » DÉSIGNE ICI : tout ce qui précède le masque de présence dans un
// enregistrement de la table keyframe — en-tête du record PLUS default-state de l'archétype.
// On ne cherche pas à séparer les deux : c'est leur somme qui place le curseur.
//
// TROIS ERREURS DE MÉTHODE, TOUTES PAYÉES, TOUTES CONSERVÉES ICI :
//
//  1. « 0 désync » ne prouve rien. La sonde de présence donnait 0 désync sur ti=5, 6, 9 et 13
//     alors qu'ils décodaient ZÉRO composant : un masque lu à zéro fait sortir la boucle
//     immédiatement, et l'absence d'erreur ne dit alors que « rien n'a été tenté ».
//
//  2. « ça décode sans désync » ne suffit pas non plus. Cinq préfixes faisaient décoder 100 %
//     des enregistrements ti=9. Sur un archétype à champs courts, un masque mal aligné produit
//     facilement une lecture plausible.
//
//  3. LE BALAYAGE PARTAIT DE +64 BITS. Les 64 bits d'en-tête viennent d'un utilitaire écrit
//     pour le BIPED (probe_export.go) et n'ont AUCUN appui dans le binaire — vérifié au
//     désassemblage. Partir de +64 rendait invisible tout préfixe plus court, donc peut-être
//     le bon. En repartant du début du record, deux candidats (24 et 60) lisent enfin.
//
// LE CRITÈRE QUI TRANCHE EST PAR MATCH, PAS SUR LE TAS. ti=9 i0 est le désignateur d'équipe,
// et la vérité est connue hors du film : une arène classée, c'est 8 joueurs, DEUX équipes,
// réparties 4-4, et personne ne change d'équipe. Agréger les six films masque exactement cette
// structure — c'est pourquoi ce banc rapporte film par film.
//
// LANCEMENT :
//
//	REPLAY_FILMS=<...> SWEEP_MAX=300 go test ./internal/analysis/filmdec/ \
//	  -run TestTeamDesignatorWidthSweep -v -timeout 60m

// filmTeams : pour un préfixe et un film, slot -> ensemble des équipes vues.
type filmTeams map[string]map[int]map[uint8]bool

func TestTeamDesignatorWidthSweep(t *testing.T) {
	root := os.Getenv("REPLAY_FILMS")
	if root == "" {
		t.Skip("REPLAY_FILMS non défini : banc hors CI")
	}
	maxW := envInt("SWEEP_MAX", 300)
	const targetTI = 9

	seen := make([]filmTeams, maxW+1)
	hits := make([]int, maxW+1)
	for i := range seen {
		seen[i] = filmTeams{}
	}
	records := 0

	forEachKeyframeRecordFilm(t, root, func(film string, pay []byte, r KeyframeRec, reg *Registry) {
		if r.TI != targetTI {
			return
		}
		arch, ok := reg.Archetype(r.TI)
		if !ok {
			return
		}
		records++
		for w := 0; w <= maxW; w++ {
			var team uint8
			var got bool
			compProbeHook = func(_ uint32, name string, payload any, _ bool) {
				if name != compManagedPlayerTeamDesignator {
					return
				}
				if td, ok := payload.(TeamDesignator); ok {
					team, got = td.Team, true
				}
			}
			br := NewBitReader(pay)
			br.SetBitPos(r.Bit) // debut REEL du record : cf. erreur de methode n°3
			br.Skip(w)
			br.ReadBit()
			tr := decodeDeltaWithArch(br, arch, uint32(r.TI))
			compProbeHook = nil
			if tr.DesyncAt >= 0 || !got {
				continue
			}
			hits[w]++
			if seen[w][film] == nil {
				seen[w][film] = map[int]map[uint8]bool{}
			}
			if seen[w][film][r.Slot] == nil {
				seen[w][film][r.Slot] = map[uint8]bool{}
			}
			seen[w][film][r.Slot][team] = true
		}
	})
	if records == 0 {
		t.Fatalf("aucun enregistrement ti=%d", targetTI)
	}
	t.Logf("ti=%d : %d enregistrements, prefixes 0..%d", targetTI, records, maxW)

	type cand struct {
		w, hits, goodFilms, films int
		detail                    string
	}
	var cands []cand
	for w := 0; w <= maxW; w++ {
		if hits[w] == 0 {
			continue
		}
		good, det := 0, ""
		films := make([]string, 0, len(seen[w]))
		for f := range seen[w] {
			films = append(films, f)
		}
		sort.Strings(films)
		for _, f := range films {
			bySlot := seen[w][f]
			counts := map[uint8]int{}
			stable := 0
			for _, teams := range bySlot {
				if len(teams) != 1 {
					continue
				}
				stable++
				for v := range teams {
					counts[v]++
				}
			}
			// Signature attendue d'une arene classee : 8 slots stables, 2 equipes, 4-4.
			ok := stable == 8 && len(counts) == 2
			if ok {
				for _, n := range counts {
					if n != 4 {
						ok = false
					}
				}
			}
			if ok {
				good++
			}
			det += fmt.Sprintf(" %s[slots=%d/stables=%d/%s]", f[:4], len(bySlot), stable, splitOf(counts))
		}
		cands = append(cands, cand{w, hits[w], good, len(films), det})
	}
	sort.Slice(cands, func(a, b int) bool {
		if cands[a].goodFilms != cands[b].goodFilms {
			return cands[a].goodFilms > cands[b].goodFilms
		}
		return cands[a].hits > cands[b].hits
	})
	t.Logf("%8s %9s %12s   %s", "prefixe", "lectures", "films 4-4", "detail par film")
	for i, c := range cands {
		if i >= 8 {
			t.Logf("... et %d autres prefixes", len(cands)-8)
			break
		}
		t.Logf("%8d %9d %8d/%-3d   %s", c.w, c.hits, c.goodFilms, c.films, c.detail)
	}
}

func splitOf(counts map[uint8]int) string {
	keys := make([]int, 0, len(counts))
	for k := range counts {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	s := ""
	for _, k := range keys {
		if s != "" {
			s += "-"
		}
		s += fmt.Sprintf("e%d:%d", k, counts[uint8(k)])
	}
	if s == "" {
		return "vide"
	}
	return s
}

// forEachKeyframeRecordFilm : comme forEachKeyframeRecord, mais nomme le film — la
// discrimination se fait PAR MATCH (cf. l'en-tête).
func forEachKeyframeRecordFilm(
	t *testing.T, root string,
	fn func(film string, pay []byte, r KeyframeRec, reg *Registry),
) {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("REPLAY_FILMS illisible : %v", err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := root + "/" + e.Name()
		n := CountFilmChunks(dir)
		if n == 0 {
			continue
		}
		raw, err := os.ReadFile(dir + "/chunk_00.bin")
		if err != nil {
			continue
		}
		reg, err := ParseRegistryChunk(raw)
		if err != nil {
			continue
		}
		for c := 1; c <= n; c++ {
			chunk, err := ReadFilmChunk(dir, c)
			if err != nil {
				continue
			}
			for _, p := range WalkPackets(chunk) {
				if p.Type != PacketTypeKeyframe {
					continue
				}
				pay := p.Payload(chunk)
				for _, r := range WalkKeyframeWorld(pay) {
					fn(e.Name(), pay, r, reg)
				}
			}
		}
	}
}

func envInt(name string, def int) int {
	if v := os.Getenv(name); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
