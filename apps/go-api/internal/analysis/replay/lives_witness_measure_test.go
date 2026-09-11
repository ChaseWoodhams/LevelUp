package replay

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// lives_witness_measure_test.go — LE BANC DE MESURE DU TÉMOIN D'ARME.
//
// POURQUOI UN TEST ET PAS UN cmd/. Il doit appeler `buildOwners`, qui n'est pas exporté, et
// il ne doit RIEN publier : ce qu'on veut est le rapport du pont, pas un artefact. Un test
// in-package donne les deux.
//
// POURQUOI IL NE MESURE PAS LES BORNES DE CARTE. Le pont ne dépend que des slots, des
// horodatages et des armes ; la déquantification monde n'y entre pas. Le banc passe donc des
// bornes neutres et n'en tire aucune conclusion sur les positions.
//
// LANCEMENT (les films ne sont pas dans le dépôt) :
//
//	REPLAY_FILMS=../../data/cache/film_chunks go test ./internal/analysis/replay/ \
//	  -run TestWeaponWitnessMeasurement -v
func TestWeaponWitnessMeasurement(t *testing.T) {
	root := os.Getenv("REPLAY_FILMS")
	if root == "" {
		t.Skip("REPLAY_FILMS non défini : banc de mesure hors CI")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("REPLAY_FILMS illisible : %v", err)
	}
	t.Logf("%-10s %5s %5s %5s %6s %6s %6s %7s %7s %7s",
		"film", "vies", "nom.", "amb.", "levees", "report", "acc.", "contra.", "muet", "%acc")
	var tTies, tRes, tAgree, tContra int
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(root, e.Name())
		if filmdec.CountFilmChunks(dir) == 0 {
			continue
		}
		rep, ok := measureBridge(t, dir)
		if !ok {
			continue
		}
		den := rep.ControlAgree + rep.ControlContradict
		pct := "-"
		if den > 0 {
			pct = pctString(rep.ControlAgree, den)
		}
		t.Logf("%-10s %5d %5d %5d %6d %6d %6d %7d %7d %7s",
			e.Name(), rep.LivesTotal, rep.DeathsNamed, rep.AmbiguousTies,
			rep.TiesResolved, rep.WitnessDeferred,
			rep.ControlAgree, rep.ControlContradict, rep.ControlSilent, pct)
		tTies += rep.AmbiguousTies
		tRes += rep.TiesResolved
		tAgree += rep.ControlAgree
		tContra += rep.ControlContradict
	}
	t.Logf("TOTAL ambigues restantes=%d levees=%d | controle accord=%d contradiction=%d (%s)",
		tTies, tRes, tAgree, tContra, pctString(tAgree, tAgree+tContra))
}

// measureBridge refait, pour un film, exactement les lectures que BuildFromFilm enchaîne,
// puis rend le rapport du pont.
func measureBridge(t *testing.T, dir string) (OwnerReport, bool) {
	t.Helper()
	deaths, err := ScanFilmDeaths(dir)
	if err != nil {
		t.Logf("%s : fil des morts illisible (%v)", filepath.Base(dir), err)
		return OwnerReport{}, false
	}
	idx, err := ScanFilmPlayerIndices(dir, rosterFromDeaths(deaths))
	if err != nil {
		t.Logf("%s : index de joueur illisible (%v)", filepath.Base(dir), err)
		return OwnerReport{}, false
	}
	table, _ := injectiveOrEmpty(idx)
	scan := filmdec.DefaultScanFilmOptions()
	// Bornes neutres : le pont ne lit ni X ni Y (cf. l'en-tête). Elles ne servent qu'à
	// satisfaire le déquantificateur.
	neutral := filmdec.QuantRangeWorld100
	scan.WorldRange = &neutral
	pos, err := filmdec.ScanFilmBipedPositions(dir, scan)
	if err != nil {
		t.Logf("%s : positions illisibles (%v)", filepath.Base(dir), err)
		return OwnerReport{}, false
	}
	fire, err := filmdec.ScanFilmFireEvents(dir)
	if err != nil {
		fire = nil
	}
	loads, err := filmdec.ScanFilmKeyframeLoadouts(dir, loadoutFamilies())
	if err != nil {
		loads = nil
	}
	return buildOwners(indexBySlot(pos), deaths, table, fire, loads), true
}

func pctString(n, d int) string {
	if d == 0 {
		return "-"
	}
	v := float64(n) * 100 / float64(d)
	s := strings.TrimRight(strings.TrimRight(formatFloat(v), "0"), ".")
	return s + "%"
}

func formatFloat(v float64) string {
	whole := int(v)
	frac := int((v - float64(whole)) * 10)
	return itoa(whole) + "." + itoa(frac)
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var b []byte
	for v > 0 {
		b = append([]byte{byte('0' + v%10)}, b...)
		v /= 10
	}
	if neg {
		return "-" + string(b)
	}
	return string(b)
}
