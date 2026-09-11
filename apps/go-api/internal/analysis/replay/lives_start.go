package replay

import "sort"

// lives_start.go — NAMING A LIFE BY THE DEATH BEFORE IT.
//
// nameLivesByDeaths names a life by the death that ENDS it. Two kinds of life have no such death:
// a player's last life of the match, and a life the clock could not tell from a rival's (a refused
// tie). Both still START after a death — the player's own, one respawn delay earlier — and that
// delay is not a distribution but a constant. Measured death -> same player's next named life on
// six archived films (lives_split_measure_test.go): p05 10,059 ms, p25 10,060 ms, median
// 10,060-10,127 ms.
//
// THE WINDOW IS READ FROM THE FILM, NOT WIRED. A mode with another respawn time, or with no fixed
// one, must not inherit 10 s: the delays of the lives the death feed already named give the window
// [p05 − 200 ms, p75 + 300 ms], and a film that named fewer than startMinPairs pairs, or whose
// delays spread over more than startMaxSpreadMS, is not named this way at all.
//
// NO VOTE, as everywhere in this package. A life is named only when exactly ONE player has a death
// in its window whose respawn no named life already accounts for, and only if no other unnamed
// life claims that same death. Measured on the six films: +7 to +30 lives per film, and no player
// named on two lives that overlap in time.

const (
	// startMinPairs is the fewest named (death, next life) pairs a window is read from.
	startMinPairs = 20
	// startMaxSpreadMS is the widest p05..p75 spread still read as a fixed respawn delay.
	startMaxSpreadMS = 1000
	// startWindowBelowMS / startWindowAboveMS widen [p05, p75] into the naming window.
	startWindowBelowMS = 200
	startWindowAboveMS = 300
	// startOverlapToleranceUS is how much two lives of one player may overlap before one of them
	// cannot be theirs (a handoff across slots shares a few samples, never seconds).
	startOverlapToleranceUS = 500_000
)

// startReport is what start-side naming did on one film.
type startReport struct {
	// overlapping counts picks refused because their player was already alive on another life.
	named, contested, overlapping int
	// calibrated is false when the film gave no window (too few pairs, or no fixed delay).
	calibrated bool
	loMS, hiMS int64
}

// namedRespawnDelaysMS measures, on lives the death feed already named, the gap between a
// player's death (the end of a named life) and the start of that player's next named life.
// Sorted ascending.
func namedRespawnDelaysMS(lives []lifeSpan) []int64 {
	by := map[uint64][]lifeSpan{}
	for _, l := range lives {
		if l.xuid != 0 {
			by[l.xuid] = append(by[l.xuid], l)
		}
	}
	var out []int64
	for _, ls := range by {
		sort.Slice(ls, func(i, j int) bool { return ls[i].from < ls[j].from })
		for i := 1; i < len(ls); i++ {
			out = append(out, (ls[i].from-ls[i-1].to)/1000)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// nameLivesByStart names, in place, the unnamed lives whose start follows exactly one player's
// unaccounted death by the film's own respawn delay.
func nameLivesByStart(lives []lifeSpan, deaths []Death, off int64) startReport {
	var rep startReport
	delays := namedRespawnDelaysMS(lives)
	if len(delays) < startMinPairs {
		return rep
	}
	at := func(p float64) int64 { return delays[int(p*float64(len(delays)-1))] }
	p05, p75 := at(0.05), at(0.75)
	if p75-p05 > startMaxSpreadMS {
		return rep
	}
	rep.calibrated, rep.loMS, rep.hiMS = true, p05-startWindowBelowMS, p75+startWindowAboveMS

	// accounted: a named life of x already starts in the respawn window of a death at tMS. Read on
	// the names as they stood BEFORE this pass, so no assignment below depends on another.
	accounted := func(x uint64, tMS int64) bool {
		for _, l := range lives {
			if l.xuid == x {
				if d := l.from/1000 - tMS; d >= rep.loMS && d <= rep.hiMS {
					return true
				}
			}
		}
		return false
	}
	claims := map[int][]int{} // death index -> unnamed lives claiming it
	pick := map[int]int{}     // unnamed life index -> its one candidate death
	for li, l := range lives {
		if l.xuid != 0 {
			continue
		}
		cands := map[uint64]int{}
		for di, d := range deaths {
			tMS := d.TimeMS + off
			if delta := l.from/1000 - tMS; delta < rep.loMS || delta > rep.hiMS || accounted(d.XUID, tMS) {
				continue
			}
			cands[d.XUID] = di
		}
		if len(cands) > 1 {
			rep.contested++
			continue
		}
		for _, di := range cands {
			claims[di] = append(claims[di], li)
			pick[li] = di
		}
	}
	accepted := map[int]uint64{}
	for li, di := range pick {
		if len(claims[di]) > 1 {
			rep.contested++
			continue
		}
		accepted[li] = deaths[di].XUID
	}
	// A PLAYER IS NEVER ALIVE TWICE. A pick whose life overlaps another life of the same player —
	// one already named, or another pick of this pass — is refused, and so is the other pick: the
	// clock cannot say which of the two is theirs. The golden film had one such pair (two lives of
	// one player running together to the end of the film), which gave 5 shots two candidate slots.
	type interval struct{ from, to int64 }
	alive := map[uint64][]interval{}
	for _, l := range lives {
		if l.xuid != 0 {
			alive[l.xuid] = append(alive[l.xuid], interval{l.from, l.to})
		}
	}
	for li, x := range accepted {
		alive[x] = append(alive[x], interval{lives[li].from, lives[li].to})
	}
	for li, x := range accepted {
		me, n := interval{lives[li].from, lives[li].to}, 0
		for _, s := range alive[x] {
			if s.from < me.to-startOverlapToleranceUS && me.from < s.to-startOverlapToleranceUS {
				n++ // counts the pick itself once
			}
		}
		if n > 1 {
			rep.overlapping++
			continue
		}
		lives[li].xuid = x
		rep.named++
	}
	return rep
}
