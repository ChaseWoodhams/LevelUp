package filmdec

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// default_state_ti0_test.go — CALIBRER LA LARGEUR DE DEFAULT-STATE DE ti=0 (moteur de jeu).
//
// LE BLOCAGE. La sonde de présence (component_probe_test.go) traverse proprement ti=5, 6, 9,
// 13, 37 et 38, mais PAS ti=0 : `defaultStateDeserByTI` n'a pas d'entrée pour lui, donc ses
// 35 enregistrements sont sautés sans être lus. Or c'est ti=0 qui porte
// `game-engine-round-timer-component`, `game-engine-current-state-component` et la période de
// grâce — c'est-à-dire l'horloge RÉELLE du match, aujourd'hui seulement inférée.
//
// POURQUOI CALIBRER PLUTÔT QUE PORTER. Les desers existants viennent du désassemblage du jeu
// (`FUN_...`), qui n'est pas disponible ici. Mais le dépôt prévoit déjà la voie empirique :
// `SetDefaultStateBitsForTI` existe pour la « calibration keyframe ». Une largeur ne se
// devine pas pour autant — il lui faut un ORACLE.
//
// L'ORACLE : LA FIN DU RECORD. `WalkKeyframeWorld` rend le bit de début de CHAQUE record ;
// le début du suivant borne donc la fin du précédent. Une largeur correcte doit faire
// terminer la traversée exactement là. Ce test mesure d'abord ce résidu sur les archétypes
// qui traversent DÉJÀ sans désync — si l'oracle ne vaut pas sur eux, il ne vaudra pas sur
// ti=0, et la calibration serait une coïncidence choisie au hasard.
//
// LANCEMENT :
//
//	REPLAY_FILMS=<...>/data/cache/film_chunks go test ./internal/analysis/filmdec/ \
//	  -run TestDefaultStateOracle -v -timeout 60m

// oracleControlTIs : archétypes qui traversent à 0 désync (mesuré par la sonde de présence).
var oracleControlTIs = []int{5, 6, 9, 13, 37, 38}

type residual struct {
	ti   int
	dist map[int]int // (début du record suivant - fin de traversée) -> occurrences
}

// TestDefaultStateOracle mesure le résidu de fin de record sur les archétypes de contrôle.
// Il ne calibre rien : il dit si l'oracle est exploitable, ce qui conditionne tout le reste.
func TestDefaultStateOracle(t *testing.T) {
	root := os.Getenv("REPLAY_FILMS")
	if root == "" {
		t.Skip("REPLAY_FILMS non défini : banc hors CI")
	}
	control := map[int]bool{}
	for _, ti := range oracleControlTIs {
		control[ti] = true
	}
	res := map[int]*residual{}
	forEachKeyframeRecord(t, root, func(pay []byte, r KeyframeRec, next int, reg *Registry) {
		if !control[r.TI] {
			return
		}
		end, ok := traverseAt(pay, r, reg, -1)
		if !ok {
			return
		}
		if res[r.TI] == nil {
			res[r.TI] = &residual{ti: r.TI, dist: map[int]int{}}
		}
		res[r.TI].dist[next-end]++
	})
	if len(res) == 0 {
		t.Fatal("aucun record de contrôle traversé : l'oracle ne peut pas être mesuré")
	}
	t.Logf("=== RESIDU (debut du record suivant - fin de traversee), archetypes de controle ===")
	tis := make([]int, 0, len(res))
	for ti := range res {
		tis = append(tis, ti)
	}
	sort.Ints(tis)
	for _, ti := range tis {
		t.Logf("ti=%d : %s", ti, topResiduals(res[ti].dist, 6))
	}
}

// topResiduals rend les résidus les plus fréquents, du plus fréquent au moins.
func topResiduals(d map[int]int, n int) string {
	type kv struct{ k, v int }
	var all []kv
	total := 0
	for k, v := range d {
		all = append(all, kv{k, v})
		total += v
	}
	sort.Slice(all, func(i, j int) bool { return all[i].v > all[j].v })
	s := fmt.Sprintf("n=%d ", total)
	for i, e := range all {
		if i >= n {
			s += "..."
			break
		}
		s += fmt.Sprintf("[%+d]x%d ", e.k, e.v)
	}
	return s
}

// traverseAt traverse un record keyframe et rend le bit de fin. `forceWidth >= 0` impose une
// largeur de default-state au lieu du deser de l'archétype — c'est le levier de calibration.
func traverseAt(pay []byte, r KeyframeRec, reg *Registry, forceWidth int) (int, bool) {
	ti := uint32(r.TI)
	arch, ok := reg.Archetype(r.TI)
	if !ok {
		return 0, false
	}
	br := NewBitReader(pay)
	br.SetBitPos(r.Bit + 64)
	switch {
	case forceWidth >= 0:
		br.Skip(forceWidth)
	case ti == bipedDefaultStateTypeIndex:
		consumeBipedDefaultState(br)
		consumeBipedDefaultStateTail(br)
	default:
		fn, ok := defaultStateDeserByTI[ti]
		if !ok {
			return 0, false
		}
		fn(br)
	}
	br.ReadBit() // porte has-components
	tr := decodeDeltaWithArch(br, arch, ti)
	if tr.DesyncAt >= 0 {
		return br.BitPos(), false
	}
	return br.BitPos(), true
}

// forEachKeyframeRecord parcourt tous les records keyframe de tous les films, en donnant à
// `fn` le début du record SUIVANT (borne de fin), ou la fin du payload pour le dernier.
func forEachKeyframeRecord(
	t *testing.T, root string,
	fn func(pay []byte, r KeyframeRec, next int, reg *Registry),
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
		dir := filepath.Join(root, e.Name())
		n := CountFilmChunks(dir)
		if n == 0 {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, "chunk_00.bin"))
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
				recs := WalkKeyframeWorld(pay)
				sort.Slice(recs, func(i, j int) bool { return recs[i].Bit < recs[j].Bit })
				for i, r := range recs {
					next := len(pay) * 8
					if i+1 < len(recs) {
						next = recs[i+1].Bit
					}
					fn(pay, r, next, reg)
				}
			}
		}
	}
}
