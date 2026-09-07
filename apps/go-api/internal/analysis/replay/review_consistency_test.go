package replay

import (
	"levelup/go-api/internal/analysis/filmdec"
	"math"
	"testing"
)

func TestReviewReferenceMeasurements(t *testing.T) {
	g := loadGoldenInputs(t)
	tracks := indexBySlot(g.Positions)
	own := buildOwners(tracks, g.Deaths, g.Indices, g.Fire, g.Loadouts)
	births := projectileBirths(g.Projectiles)
	compared, distant, zeroSlot := 0, 0, 0
	maxDistance := float64(0)
	for _, throw := range g.Grenades {
		located, ok := locateThrow(throw, births, tracks, own.Owner)
		if !ok || located.Src != GrenadeSrcProjectile {
			continue
		}
		if located.Slot == 0 {
			zeroSlot++
		}
		slot, reason := slotFor(tracks, own.Owner, throw.FilmIndex, throw.TimestampUS)
		if reason != reasonAttached {
			continue
		}
		p, dt := tracks[slot].at(throw.TimestampUS)
		if dt > shotPosToleranceUS {
			continue
		}
		compared++
		distance := math.Hypot(float64(located.X-p.X), float64(located.Y-p.Y))
		if distance > 4 {
			distant++
		}
		maxDistance = math.Max(maxDistance, distance)
	}
	lives := buildLifeSpans(tracks)
	counts := map[uint32]int{}
	for _, life := range lives {
		counts[life.slot]++
	}
	reused := 0
	for _, count := range counts {
		if count > 1 {
			reused++
		}
	}
	t.Logf("reference: projectile-located throws with slot zero=%d; comparable to known thrower=%d; distance >4m=%d; max distance=%.2fm; slots with multiple life spans=%d; owner collisions=%d", zeroSlot, compared, distant, maxDistance, reused, own.SlotCollisions)
}

func TestReviewConsistency(t *testing.T) {
	t.Run("grenade must not use unrelated distant projectile", func(t *testing.T) {
		tracks := indexBySlot([]filmdec.BipedPosition{pos(512, 1000, 1, 2, 0)})
		g, ok := locateThrow(filmdec.GrenadeThrow{TimestampUS: 1000000, FilmIndex: 3},
			[]filmdec.ProjectileSample{{TimestampUS: 1000000, X: 200, Y: -180}}, tracks, map[uint32]int{512: 3})
		if ok && (g.X > 10 || g.Y < -10) {
			t.Fatalf("thrower at (1,2), published grenade at (%v,%v), slot=%d source=%s", g.X, g.Y, g.Slot, g.Src)
		}
	})
	t.Run("conflicting slot owner must be withheld", func(t *testing.T) {
		owners, xuids, _ := ownersFromLives([]lifeSpan{{slot: 512, xuid: 11}, {slot: 512, xuid: 22}}, map[uint64]int{11: 0, 22: 1})
		if len(owners) != 0 || len(xuids) != 0 {
			t.Fatalf("conflicting identities still published: owners=%v xuids=%v", owners, xuids)
		}
	})
	t.Run("one death cannot identify two equally near lives", func(t *testing.T) {
		lives := []lifeSpan{{slot: 512, to: 1000000}, {slot: 513, to: 1000000}}
		named := nameLivesByDeaths(lives, []Death{{XUID: 11, TimeMS: 1000}}, 0, nil).named
		if named != 0 {
			t.Fatalf("ambiguous death assigned arbitrarily: %+v", lives)
		}
	})
	t.Run("published tracks must preserve life gaps", func(t *testing.T) {
		positions := []filmdec.BipedPosition{pos(512, 0, 0, 0, 0), pos(512, 100, 1, 0, 0), pos(512, 10000, 100, 0, 0), pos(512, 10100, 101, 0, 0)}
		doc := BuildFromPositions("review", "halo_infinite", positions, nil, Options{})
		for _, tr := range doc.Tracks {
			if tr.StartFrame <= 50 && tr.EndFrame >= 50 {
				t.Fatalf("track marked alive through absent interval: start=%d end=%d", tr.StartFrame, tr.EndFrame)
			}
		}
	})
}
