package replay

import (
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// projectile_wrap_measure_test.go — THE PROJECTILE QUANTUM-WRAP BENCH.
//
// WHAT IT MEASURES. `object-position-component` is quantised over the map's bounds, so a
// projectile that leaves them on one axis comes back on the other side, exactly one extent away.
// filmdec.unwrapLife follows that wrap; before it, buildProjectiles cut the flight at the step
// (projectileMaxStepM). Per film, this bench reports the steps still jumping by more than half an
// extent, the flights buildProjectiles truncates, the points outside the box by side, and — the
// control that the unwrap does not invent positions — the kink rate of flights that leave the box
// against flights that stay inside it.
//
// Measured 2026-09-11 on the six archived films: truncated 359 -> 19 of 2,115 flights; 19,419
// points outside the box, all on Streets' Y axis (both ends of the streets) bar a few on Aquarius
// Z; kinks 0.06 per 1000 steps for both populations.
//
// It stays a TEST, not a cmd/: buildProjectiles is unexported, and nothing here may write an
// artifact (cmd/replay-build writes into the cache the study server serves).
//
// LAUNCH (films and bounds are not in the repository; paths must be ABSOLUTE — go test runs in
// the package directory):
//
//	REPLAY_FILMS=C:/…/LevelUp/data/cache/film_chunks \
//	REPLAY_QUANT_BOUNDS=C:/…/LevelUp/data/titles/halo_infinite/reference/map_quant_bounds.json \
//	REPLAY_FILM_MAPS=36e80b83=Streets,c3d46dbc=Aquarius \
//	go test ./internal/analysis/replay/ -run TestProjectileWrapMeasurement -v
func TestProjectileWrapMeasurement(t *testing.T) {
	root, boundsPath := os.Getenv("REPLAY_FILMS"), os.Getenv("REPLAY_QUANT_BOUNDS")
	if root == "" || boundsPath == "" || os.Getenv("REPLAY_FILM_MAPS") == "" {
		t.Skip("REPLAY_FILMS / REPLAY_QUANT_BOUNDS / REPLAY_FILM_MAPS non définis : banc hors CI")
	}
	cat, err := filmdec.LoadMapQuantCatalog(boundsPath)
	if err != nil {
		t.Fatalf("catalogue de bornes : %v", err)
	}
	maps := map[string]string{}
	for _, pair := range strings.Split(os.Getenv("REPLAY_FILM_MAPS"), ",") {
		if id, name, ok := strings.Cut(strings.TrimSpace(pair), "="); ok {
			maps[id] = name
		}
	}
	ids := make([]string, 0, len(maps))
	for id := range maps {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	t.Logf("%-10s %-10s %7s %7s %9s %9s %8s %8s", "film", "map", "flights", "points", "wrapSteps",
		"truncated", "outside", "maxOutM")
	var tFlights, tWraps, tTrunc, tOutside int
	// Smoothness control: if the unwrap invented positions, flights that leave the box would
	// kink more often than flights that stay inside it.
	var kinkLeave, stepLeave, kinkStay, stepStay int
	for _, id := range ids {
		entry, err := cat.Lookup(maps[id])
		if err != nil {
			t.Errorf("%s: carte %q absente du catalogue : %v", id, maps[id], err)
			continue
		}
		wr := entry.Range()
		tracks, err := filmdec.ScanFilmProjectiles(filepath.Join(root, id), &wr)
		if err != nil {
			t.Errorf("%s: %v", id, err)
			continue
		}
		points, wraps, outside, restOutside := 0, 0, 0, 0
		var sides [6]int // X-, X+, Y-, Y+, Z-, Z+
		var maxOut float32
		var origin uint64 = math.MaxUint64
		for _, tr := range tracks {
			points += len(tr.Pts)
			if len(tr.Pts) > 0 && tr.Pts[0].TimestampUS < origin {
				origin = tr.Pts[0].TimestampUS
			}
			// Distance past the box, per point: an unwrapped flight legitimately leaves it, but
			// only by what a projectile can cover, never by a whole extent.
			leaves := false
			for _, p := range tr.Pts {
				if p.X < wr[0].Min || p.X > wr[0].Max || p.Y < wr[1].Min || p.Y > wr[1].Max ||
					p.Z < wr[2].Min || p.Z > wr[2].Max {
					leaves = true
				}
			}
			k, s := kinks(tr.Pts)
			if leaves {
				kinkLeave, stepLeave = kinkLeave+k, stepLeave+s
			} else {
				kinkStay, stepStay = kinkStay+k, stepStay+s
			}
			for i, p := range tr.Pts {
				out := false
				for a, v := range [3]float32{p.X, p.Y, p.Z} {
					if past := wr[a].Min - v; past > 0 {
						out, maxOut = true, max(maxOut, past)
						sides[2*a]++
					}
					if past := v - wr[a].Max; past > 0 {
						out, maxOut = true, max(maxOut, past)
						sides[2*a+1]++
					}
				}
				if out {
					outside++
					if i == len(tr.Pts)-1 && p.AtRest {
						restOutside++
					}
				}
			}
			for i := 1; i < len(tr.Pts); i++ {
				a, b := tr.Pts[i-1], tr.Pts[i]
				if float64(absF32(b.X-a.X)) > float64(wr[0].Max-wr[0].Min)/2 ||
					float64(absF32(b.Y-a.Y)) > float64(wr[1].Max-wr[1].Min)/2 ||
					float64(absF32(b.Z-a.Z)) > float64(wr[2].Max-wr[2].Min)/2 {
					wraps++
				}
			}
		}
		if origin == math.MaxUint64 {
			origin = 0
		}
		_, truncated := buildProjectiles(tracks, origin, uint64(DefaultFrameIntervalMS)*1000)
		t.Logf("%-10s %-10s %7d %7d %9d %9d %8d %8.2f", id, maps[id], len(tracks), points, wraps,
			truncated, outside, maxOut)
		t.Logf("%-10s   outside by side X-/X+ %d/%d  Y-/Y+ %d/%d  Z-/Z+ %d/%d; at-rest endpoints outside %d",
			id, sides[0], sides[1], sides[2], sides[3], sides[4], sides[5], restOutside)
		tFlights, tWraps, tTrunc, tOutside = tFlights+len(tracks), tWraps+wraps, tTrunc+truncated, tOutside+outside
	}
	t.Logf("TOTAL flights %d, wrap steps %d, truncated flights %d, points outside the box %d",
		tFlights, tWraps, tTrunc, tOutside)
	t.Logf("kinks per 1000 steps: flights leaving the box %.2f (%d/%d), staying inside %.2f (%d/%d)",
		perMille(kinkLeave, stepLeave), kinkLeave, stepLeave, perMille(kinkStay, stepStay), kinkStay, stepStay)
}

// kinks counts the steps of a flight longer than four times its median step (and over 1 m):
// a mis-placed sample shows up as one, a smooth arc does not.
func kinks(pts []filmdec.ProjectileSample) (kinked, steps int) {
	if len(pts) < 3 {
		return 0, 0
	}
	d := make([]float64, 0, len(pts)-1)
	for i := 1; i < len(pts); i++ {
		a, b := pts[i-1], pts[i]
		d = append(d, math.Sqrt(float64((b.X-a.X)*(b.X-a.X)+(b.Y-a.Y)*(b.Y-a.Y)+(b.Z-a.Z)*(b.Z-a.Z))))
	}
	sorted := append([]float64(nil), d...)
	sort.Float64s(sorted)
	limit := max(4*sorted[len(sorted)/2], 1.0)
	for _, s := range d {
		if s > limit {
			kinked++
		}
	}
	return kinked, len(d)
}

func perMille(n, of int) float64 {
	if of == 0 {
		return 0
	}
	return 1000 * float64(n) / float64(of)
}

func absF32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
