package replay

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// lives_split_measure_test.go — ARE THE EXTRA LIVES ONE LIFE CUT IN TWO?
//
// The ground-truth check shows films with more life spans than official lives (e01d80a1: 218
// spans for 206 lives) while every player is UNDER-named. buildLifeSpans opens a new life when a
// slot is silent for more than lifeGapUS; a replication dropout longer than that cuts one life in
// two, and the first half ends with no death — an end the naming can only leave unnamed or turn
// into a tie. This bench classifies every unnamed span, then simulates one rule — consecutive
// spans on a slot merge when no death of ANYONE falls at the earlier span's end — and compares the
// naming with official deaths before and after. Over-named (a player named on more lives than
// official deaths + 1) must stay 0.
//
//	REPLAY_FILMS=C:/…/data/cache/film_chunks REPLAY_PARTICIPANTS=C:/…/scratchpad \
//	REPLAY_FILM_IDS=e01d80a1,36e80b83 go test ./internal/analysis/replay/ -run TestSplitLivesMeasurement -v
func TestSplitLivesMeasurement(t *testing.T) {
	root, partDir, idList := os.Getenv("REPLAY_FILMS"), os.Getenv("REPLAY_PARTICIPANTS"), os.Getenv("REPLAY_FILM_IDS")
	if root == "" || partDir == "" || idList == "" {
		t.Skip("REPLAY_FILMS / REPLAY_PARTICIPANTS / REPLAY_FILM_IDS not set: bench, not CI")
	}
	for _, id := range strings.Split(idList, ",") {
		id = strings.TrimSpace(id)
		dir := filepath.Join(root, id)
		official, err := readOfficialDeaths(filepath.Join(partDir, "part_"+id+".json"))
		if err != nil {
			t.Errorf("%s: participants: %v", id, err)
			continue
		}
		deaths, err := ScanFilmDeaths(dir)
		if err != nil {
			t.Errorf("%s: deaths: %v", id, err)
			continue
		}
		idx, err := ScanFilmPlayerIndices(dir, rosterFromDeaths(deaths))
		if err != nil {
			t.Errorf("%s: player indices: %v", id, err)
			continue
		}
		table, _ := injectiveOrEmpty(idx)
		scan := filmdec.DefaultScanFilmOptions()
		// Naming reads no position, but the over-named dump does: real bounds when given.
		wr := filmdec.QuantRangeWorld100
		if bp, maps := os.Getenv("REPLAY_QUANT_BOUNDS"), os.Getenv("REPLAY_FILM_MAPS"); bp != "" && maps != "" {
			if cat, err := filmdec.LoadMapQuantCatalog(bp); err == nil {
				for _, pair := range strings.Split(maps, ",") {
					if fid, name, ok := strings.Cut(pair, "="); ok && fid == id {
						if e, err := cat.Lookup(name); err == nil {
							wr = e.Range()
						}
					}
				}
			}
		}
		scan.WorldRange = &wr
		pos, err := filmdec.ScanFilmBipedPositions(dir, scan)
		if err != nil {
			t.Errorf("%s: positions: %v", id, err)
			continue
		}
		fire, _ := filmdec.ScanFilmFireEvents(dir)
		loads, _ := filmdec.ScanFilmKeyframeLoadouts(dir, loadoutFamilies())
		w := newWeaponWitness(fire, loads, table.ByXUID)

		tracks := indexBySlot(pos)
		base := buildLifeSpans(tracks)
		off, _ := bestDeathOffset(base, deaths)
		repB := nameLivesByDeaths(base, deaths, off, w)
		t.Logf("%s BASE   %s ambiguous %d", id, judgeLives(base, official), repB.ambiguous)
		classifyUnnamed(t, id, base, deaths, off)

		delays := namedRespawnDelaysMS(base)
		if len(delays) == 0 {
			continue
		}
		pct := func(p float64) int64 { return delays[int(p*float64(len(delays)-1))] }
		t.Logf("%s respawn delay (death -> same player's next named life), n=%d: p05 %d p25 %d p50 %d p75 %d p95 %d ms",
			id, len(delays), pct(0.05), pct(0.25), pct(0.5), pct(0.75), pct(0.95))
		v0, h0, over0 := identityCheck(base, official)
		t.Logf("%s BASE   identity: violations %d handoffs %d over-named after handoffs %d", id, v0, h0, over0)
		named := append([]lifeSpan(nil), base...)
		sr := nameLivesByStart(named, deaths, off)
		v, h, over := identityCheck(named, official)
		t.Logf("%s START [%d..%d ms] calibrated %v added %d contested %d | violations %d handoffs %d over-named after handoffs %d | %s",
			id, sr.loMS, sr.hiMS, sr.calibrated, sr.named, sr.contested, v, h, over, judgeLives(named, official))
		dumpOverNamed(t, id, base, named, tracks, official)
	}
}

// dumpOverNamed prints, for every player named on more lives than official deaths + 1, each of
// their named spans: how it was named, the gap from the previous span, and how far its first
// position is from where the previous span ended. A life split across slots continues where
// it stopped; a respawn starts at a spawn point.
func dumpOverNamed(t *testing.T, id string, base, named []lifeSpan, tracks map[uint32]slotTrack, official map[uint64]int) {
	t.Helper()
	by := map[uint64][]int{}
	for i, l := range named {
		if l.xuid != 0 {
			by[l.xuid] = append(by[l.xuid], i)
		}
	}
	for x, idx := range by {
		if d, ok := official[x]; !ok || len(idx) <= d+1 {
			continue
		}
		sort.Slice(idx, func(a, b int) bool { return named[idx[a]].from < named[idx[b]].from })
		t.Logf("%s player %d: %d named spans, official deaths %d", id, x, len(idx), official[x])
		for k, i := range idx {
			l := named[i]
			how := "death"
			if base[i].xuid == 0 {
				how = "START"
			}
			first, _ := tracks[l.slot].at(uint64(l.from))
			line := "  #" + strconv.Itoa(k) + " slot " + strconv.Itoa(int(l.slot)) +
				" " + secs(l.from) + "->" + secs(l.to) + " by " + how
			if k > 0 {
				p := named[idx[k-1]]
				last, _ := tracks[p.slot].at(uint64(p.to))
				dx, dy, dz := float64(first.X-last.X), float64(first.Y-last.Y), float64(first.Z-last.Z)
				line += " gap " + secs(l.from-p.to) + " dist-from-prev-end " +
					strconv.FormatFloat(math.Sqrt(dx*dx+dy*dy+dz*dz), 'f', 1, 64) + "m"
			}
			t.Log(line)
		}
	}
}

func secs(us int64) string { return strconv.FormatFloat(float64(us)/1e6, 'f', 1, 64) + "s" }

// identityCheck reads the naming for what can be wrong about it, not for how many spans it names.
//
// A player cannot be alive twice: two spans named for one player that overlap by more than
// 500 ms are a VIOLATION — one of the two names is wrong. A span that ends within 2 s of the
// same player's next span starting on ANOTHER slot is a HANDOFF — one life carried across slots
// — and counts as one life. Over-named is then judged per player on lives, not spans.
func identityCheck(lives []lifeSpan, official map[uint64]int) (violations, handoffs, over int) {
	by := map[uint64][]lifeSpan{}
	for _, l := range lives {
		if l.xuid != 0 {
			by[l.xuid] = append(by[l.xuid], l)
		}
	}
	for x, ls := range by {
		sort.Slice(ls, func(i, j int) bool { return ls[i].from < ls[j].from })
		n := len(ls)
		for i := 1; i < len(ls); i++ {
			gapMS := (ls[i].from - ls[i-1].to) / 1000
			switch {
			case gapMS < -500:
				violations++
			case gapMS <= 2000 && ls[i].slot != ls[i-1].slot:
				handoffs++
				n--
			}
		}
		if d, ok := official[x]; ok && n > d+1 {
			over += n - (d + 1)
		}
	}
	return violations, handoffs, over
}

func deathNear(sorted []int64, endMS int64) bool {
	i := sort.Search(len(sorted), func(i int) bool { return sorted[i] >= endMS-deathMatchWindowMS })
	return i < len(sorted) && sorted[i] <= endMS+deathMatchWindowMS
}

// classifyUnnamed sorts every unnamed span by why it could not be named.
func classifyUnnamed(t *testing.T, id string, lives []lifeSpan, deaths []Death, off int64) {
	t.Helper()
	targets := make([]int64, 0, len(deaths))
	for _, d := range deaths {
		targets = append(targets, d.TimeMS+off)
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i] < targets[j] })
	var splitNext, deathlessLast, deathRefused int
	gaps := map[string]int{}
	for i, l := range lives {
		if l.xuid != 0 {
			continue
		}
		if deathNear(targets, l.to/1000) {
			deathRefused++
			continue
		}
		if i+1 < len(lives) && lives[i+1].slot == l.slot {
			splitNext++
			gaps[gapBin(lives[i+1].from-l.to)]++
			continue
		}
		deathlessLast++
	}
	t.Logf("%s unnamed: no death + same slot follows %d (gaps %v) | no death, last on slot %d | death at end but refused %d",
		id, splitNext, gaps, deathlessLast, deathRefused)
}

func gapBin(us int64) string {
	s := us / 1_000_000
	switch {
	case s < 6:
		return "5-6s"
	case s < 8:
		return "6-8s"
	case s < 10:
		return "8-10s"
	case s < 15:
		return "10-15s"
	default:
		return "15s+"
	}
}

// judgeLives compares named lives per player with official deaths.
func judgeLives(lives []lifeSpan, official map[uint64]int) string {
	named := map[uint64]int{}
	total := 0
	for _, l := range lives {
		if l.xuid != 0 {
			named[l.xuid]++
			total++
		}
	}
	expected, over, unlisted := 0, 0, 0
	for x, d := range official {
		expected += d + 1
		if n := named[x]; n > d+1 {
			over += n - (d + 1)
		}
	}
	for x, n := range named {
		if _, ok := official[x]; !ok {
			unlisted += n
		}
	}
	return "spans " + strconv.Itoa(len(lives)) + " expected " + strconv.Itoa(expected) +
		" named " + strconv.Itoa(total) + " over-named " + strconv.Itoa(over) +
		" unlisted " + strconv.Itoa(unlisted)
}

func readOfficialDeaths(path string) (map[uint64]int, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // bench input path from the environment
	if err != nil {
		return nil, err
	}
	var doc struct {
		Participants []struct {
			XUID   string `json:"xuid"`
			Deaths int    `json:"deaths"`
		} `json:"participants"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	out := map[uint64]int{}
	for _, p := range doc.Participants {
		if x, err := strconv.ParseUint(p.XUID, 10, 64); err == nil {
			out[x] = p.Deaths
		}
	}
	return out, nil
}
