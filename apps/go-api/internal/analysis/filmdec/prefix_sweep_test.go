package filmdec

import (
	"fmt"
	"os"
	"sort"
	"testing"
)

// prefix_sweep_test.go — CALIBRER LE PRÉFIXE D'UN TYPE D'ENTITÉ QUELCONQUE.
//
// CE QUE LE BANC ti=9 A ÉTABLI. Le préfixe d'un enregistrement keyframe (en-tête + default-
// state, tout ce qui précède la porte has-components) vaut 61 bits pour ti=9 : seul préfixe,
// sur 300 essayés, qui donne 8 entités joueur d'équipe STABLE réparties 4-4 dans les SIX
// films. Tous les autres candidats rendent une constante.
//
// CE QUI EN DÉCOULE, ET QUI SURPREND. 61 n'est pas 64. Le default-state de ti=9 consomme 14
// bits (V=1 + 6 + 6 + 1), donc l'en-tête ferait 47 bits — alors que le biped se décode
// correctement à +64. Les deux ne peuvent pas être vrais avec un en-tête unique : LE PRÉFIXE
// EST DONC PROPRE AU TYPE D'ENTITÉ, et se calibre type par type. C'est la raison d'être de ce
// banc générique.
//
// LE CRITÈRE, ET SA LIMITE. Sans vérité extérieure comparable à celle de l'équipe, on juge
// sur la STRUCTURE : un préfixe correct doit décoder des composants sans désync, sur une
// forte proportion des enregistrements, et rendre un nombre d'entités stable d'un film à
// l'autre. C'est plus faible que le critère 4-4, et le rapport le dit — un candidat trouvé
// ici est une PISTE à confirmer sur une grandeur connue, pas une conclusion.
//
// LANCEMENT :
//
//	REPLAY_FILMS=<...> SWEEP_TI=5 SWEEP_MAX=300 go test ./internal/analysis/filmdec/ \
//	  -run TestPrefixSweep -v -timeout 60m

func TestPrefixSweep(t *testing.T) {
	root := os.Getenv("REPLAY_FILMS")
	if root == "" {
		t.Skip("REPLAY_FILMS non défini : banc hors CI")
	}
	target := envInt("SWEEP_TI", 5)
	maxW := envInt("SWEEP_MAX", 300)

	type acc struct {
		ok, desync, empty int
		comps             int
		slots             map[string]map[int]bool
		names             map[string]int
	}
	sc := make([]*acc, maxW+1)
	for i := range sc {
		sc[i] = &acc{slots: map[string]map[int]bool{}, names: map[string]int{}}
	}
	records := 0

	forEachKeyframeRecordFilm(t, root, func(film string, pay []byte, r KeyframeRec, reg *Registry) {
		if r.TI != target {
			return
		}
		arch, ok := reg.Archetype(r.TI)
		if !ok {
			return
		}
		records++
		for w := 0; w <= maxW; w++ {
			a := sc[w]
			names := map[string]int{}
			compProbeHook = func(_ uint32, name string, _ any, _ bool) { names[name]++ }
			br := NewBitReader(pay)
			br.SetBitPos(r.Bit)
			br.Skip(w)
			br.ReadBit()
			tr := decodeDeltaWithArch(br, arch, uint32(r.TI))
			compProbeHook = nil
			switch {
			case tr.DesyncAt >= 0:
				a.desync++
			case len(tr.Comps) == 0:
				a.empty++
			default:
				a.ok++
				a.comps += len(tr.Comps)
				if a.slots[film] == nil {
					a.slots[film] = map[int]bool{}
				}
				a.slots[film][r.Slot] = true
				for n, c := range names {
					a.names[n] += c
				}
			}
		}
	})
	if records == 0 {
		t.Fatalf("aucun enregistrement ti=%d", target)
	}
	t.Logf("ti=%d : %d enregistrements, prefixes 0..%d", target, records, maxW)

	idx := make([]int, 0, maxW+1)
	for w := 0; w <= maxW; w++ {
		if sc[w].ok > 0 {
			idx = append(idx, w)
		}
	}
	sort.Slice(idx, func(a, b int) bool { return sc[idx[a]].ok > sc[idx[b]].ok })
	t.Logf("%8s %9s %9s %8s %8s   %s", "prefixe", "decodent", "desync", "vides", "comps", "slots par film / composants")
	for i, w := range idx {
		if i >= 10 {
			t.Logf("... et %d autres prefixes", len(idx)-10)
			break
		}
		a := sc[w]
		var per []string
		films := make([]string, 0, len(a.slots))
		for f := range a.slots {
			films = append(films, f)
		}
		sort.Strings(films)
		for _, f := range films {
			per = append(per, fmt.Sprintf("%s:%d", f[:4], len(a.slots[f])))
		}
		type kv struct {
			n string
			c int
		}
		var top []kv
		for n, c := range a.names {
			top = append(top, kv{n, c})
		}
		sort.Slice(top, func(x, y int) bool { return top[x].c > top[y].c })
		names := ""
		for j, e := range top {
			if j >= 3 {
				break
			}
			names += fmt.Sprintf(" %s=%d", trunc(e.n, 40), e.c)
		}
		t.Logf("%8d %9d %9d %8d %8d   [%v]%s", w, a.ok, a.desync, a.empty, a.comps, per, names)
	}
}
