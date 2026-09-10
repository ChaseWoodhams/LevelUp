package filmdec

import (
	"os"
	"sort"
	"testing"
)

// respawn_validate_test.go — DÉPARTAGER LES PRÉFIXES ti=5 SUR LA VALEUR, PAS SUR L'ABSENCE
// D'ERREUR.
//
// LE PIÈGE, POUR LA TROISIÈME FOIS. Le balayage ti=5 rend QUATRE-VINGT-DIX préfixes qui
// décodent 7 680 enregistrements sans un seul désync. « Ça décode » ne distingue rien : sur un
// archétype à 27 composants courts, un masque mal aligné produit sans peine une lecture
// plausible. C'est le critère qui avait déjà fait remonter cinq faux candidats sur ti=9, et
// que seule la confrontation à une vérité extérieure (4-4) avait tranché.
//
// LA VÉRITÉ EXTÉRIEURE ICI : COMBIEN DE TEMPS UN JOUEUR EST MORT. Le compteur de réapparition
// est actif pendant la mort et pas autrement. On sait le mesurer sans le film : le fil des
// morts donne N morts, la réapparition dure ~8 s, le match dure D secondes pour 8 joueurs.
// La part de mesures « en attente de réapparition » doit donc tomber autour de
// N x 8 / (8 x D) — de l'ordre de 15 a 25 % sur ces films. Un préfixe qui rend 0 % (jamais
// actif) ou ~100 % (toujours actif) lit autre chose que ce compteur, quel que soit son
// silence en désync.
//
// LANCEMENT :
//
//	REPLAY_FILMS=<...> go test ./internal/analysis/filmdec/ -run TestRespawnPrefixValidate -v -timeout 60m

func TestRespawnPrefixValidate(t *testing.T) {
	root := os.Getenv("REPLAY_FILMS")
	if root == "" {
		t.Skip("REPLAY_FILMS non défini : banc hors CI")
	}
	maxW := envInt("SWEEP_MAX", 300)
	const targetTI = 5

	type acc struct {
		samples, active int
		t0min, t0max    int
	}
	sc := make([]*acc, maxW+1)
	for i := range sc {
		sc[i] = &acc{t0min: 1 << 30}
	}

	forEachKeyframeRecordFilm(t, root, func(_ string, pay []byte, r KeyframeRec, reg *Registry) {
		if r.TI != targetTI {
			return
		}
		arch, ok := reg.Archetype(r.TI)
		if !ok {
			return
		}
		for w := 0; w <= maxW; w++ {
			a := sc[w]
			var rt RespawnTimer
			var got bool
			compProbeHook = func(_ uint32, name string, payload any, _ bool) {
				if name != "player-respawn-timer-component" {
					return
				}
				if v, ok := payload.(RespawnTimer); ok {
					rt, got = v, true
				}
			}
			br := NewBitReader(pay)
			br.SetBitPos(r.Bit)
			br.Skip(w)
			br.ReadBit()
			tr := decodeDeltaWithArch(br, arch, uint32(targetTI))
			compProbeHook = nil
			if tr.DesyncAt >= 0 || !got {
				continue
			}
			a.samples++
			if rt.Active {
				a.active++
			}
			if int(rt.T0) < a.t0min {
				a.t0min = int(rt.T0)
			}
			if int(rt.T0) > a.t0max {
				a.t0max = int(rt.T0)
			}
		}
	})

	type row struct {
		w, samples, active int
		pct                float64
		t0min, t0max       int
	}
	var rows []row
	for w := 0; w <= maxW; w++ {
		a := sc[w]
		if a.samples < 500 {
			continue
		}
		rows = append(rows, row{w, a.samples, a.active,
			100 * float64(a.active) / float64(a.samples), a.t0min, a.t0max})
	}
	// Le meilleur candidat est celui dont la part d'actifs tombe le plus pres de la fourchette
	// attendue (15-25 %). On classe par ecart au centre, 20 %.
	sort.Slice(rows, func(a, b int) bool {
		da, db := abs(rows[a].pct-20), abs(rows[b].pct-20)
		return da < db
	})
	t.Logf("%8s %9s %9s %9s %14s   %s", "prefixe", "mesures", "actifs", "part", "T0 min..max", "verdict")
	for i, r := range rows {
		if i >= 12 {
			t.Logf("... et %d autres prefixes", len(rows)-12)
			break
		}
		verdict := ""
		if r.pct >= 10 && r.pct <= 35 {
			verdict = "<<< PLAUSIBLE"
		} else if r.pct == 0 {
			verdict = "jamais actif"
		} else if r.pct > 95 {
			verdict = "toujours actif"
		}
		t.Logf("%8d %9d %9d %8.1f%% %6d..%-6d   %s", r.w, r.samples, r.active, r.pct, r.t0min, r.t0max, verdict)
	}
	if len(rows) == 0 {
		t.Logf("aucun prefixe ne lit le compteur de reapparition")
	}
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
