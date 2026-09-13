package replay

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// map_bounds_plausibility_measure_test.go — DOES A CANDIDATE BOX DECODE A MAP?
//
// A map's quant bounds are catalogued only when its module link is proven, but a proven module can
// still hand over the wrong box: several Forge canvases read back the SAME box, one of them equal
// to a box already shown wrong for Vagabond. This bench decodes a film's biped positions with a
// candidate catalogue and prints where the players land. Right bounds put 8 players in a compact
// arena on floors; wrong bounds scale or wrap positions into kilometres of noise. It writes
// nothing (cmd/replay-build and the archiver write the served cache).
//
//	REPLAY_FILMS=C:/…/data/cache/film_chunks REPLAY_QUANT_BOUNDS=C:/…/candidate.json \
//	REPLAY_FILM_MAPS=df31520e=Lattice,5a755599=Origin go test ./internal/analysis/replay/ -run TestMapBoundsPlausibility -v
func TestMapBoundsPlausibility(t *testing.T) {
	root, boundsPath, pairs := os.Getenv("REPLAY_FILMS"), os.Getenv("REPLAY_QUANT_BOUNDS"), os.Getenv("REPLAY_FILM_MAPS")
	if root == "" || boundsPath == "" || pairs == "" {
		t.Skip("REPLAY_FILMS / REPLAY_QUANT_BOUNDS / REPLAY_FILM_MAPS not set: bench, not CI")
	}
	cat, err := filmdec.LoadMapQuantCatalog(boundsPath)
	if err != nil {
		t.Fatalf("catalogue: %v", err)
	}
	for _, pair := range strings.Split(pairs, ",") {
		id, name, ok := strings.Cut(strings.TrimSpace(pair), "=")
		if !ok {
			continue
		}
		entry, err := cat.Lookup(name)
		if err != nil {
			t.Errorf("%s: %v", id, err)
			continue
		}
		wr := entry.Range()
		scan := filmdec.DefaultScanFilmOptions()
		scan.WorldRange = &wr
		pos, err := filmdec.ScanFilmBipedPositions(filepath.Join(root, id), scan)
		if err != nil {
			t.Errorf("%s: %v", id, err)
			continue
		}
		var xs, ys, zs []float64
		for _, p := range pos {
			if p.HasWorld {
				xs, ys, zs = append(xs, float64(p.X)), append(ys, float64(p.Y)), append(zs, float64(p.Z))
			}
		}
		if len(xs) == 0 {
			t.Logf("%s %-9s module %s: %d positions, none with world coordinates", id, name, entry.Module, len(pos))
			continue
		}
		t.Logf("%s %-9s module %-12s widths %v: %d positions (%d world) | x %s | y %s | z %s",
			id, name, entry.Module, entry.AxisWidths, len(pos), len(xs), spread(xs), spread(ys), spread(zs))
	}
}

// spread prints p1 / p50 / p99 of a sample.
func spread(v []float64) string {
	sort.Float64s(v)
	at := func(p float64) string { return strconv.FormatFloat(v[int(p*float64(len(v)-1))], 'f', 1, 64) }
	return at(0.01) + " / " + at(0.5) + " / " + at(0.99)
}
