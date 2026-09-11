package replay

import "testing"

// respawnMS is the fixed delay measured on the archived films, death -> next life.
const respawnMS = 10_060

// chainedLives builds n lives of one player, each named by the death that ends it and each
// starting respawnMS after the previous death, beginning at startMS on slots from slot0. It returns
// the lives, their deaths, and the instant of the last death.
func chainedLives(xuid uint64, slot0 uint32, startMS int64, n int) ([]lifeSpan, []Death, int64) {
	var lives []lifeSpan
	var deaths []Death
	t := startMS
	for i := 0; i < n; i++ {
		end := t + 20_000
		lives = append(lives, lifeSpan{slot: slot0 + uint32(i), from: t * 1000, to: end * 1000, xuid: xuid})
		deaths = append(deaths, Death{XUID: xuid, TimeMS: end})
		t = end + respawnMS
	}
	return lives, deaths, t - respawnMS
}

// The last life of a match ends with no death, but it starts one respawn after its owner's last
// death: named, and only that life.
func TestNameLivesByStart_NamesALastLife(t *testing.T) {
	lives, deaths, lastDeath := chainedLives(111, 500, 0, 25)
	lives = append(lives, lifeSpan{slot: 900, from: (lastDeath + respawnMS) * 1000, to: (lastDeath + 60_000) * 1000})

	rep := nameLivesByStart(lives, deaths, 0)

	if !rep.calibrated || rep.named != 1 || lives[len(lives)-1].xuid != 111 {
		t.Fatalf("report %+v, last life xuid %d; want calibrated, 1 named, owner 111", rep, lives[len(lives)-1].xuid)
	}
}

// Two players dying at the same instant both have a death in the window: no vote, no name.
func TestNameLivesByStart_TwoCandidatesStayUnnamed(t *testing.T) {
	lives, deaths, lastDeath := chainedLives(111, 500, 0, 25)
	deaths = append(deaths, Death{XUID: 222, TimeMS: lastDeath})
	lives = append(lives, lifeSpan{slot: 900, from: (lastDeath + respawnMS) * 1000, to: (lastDeath + 60_000) * 1000})

	rep := nameLivesByStart(lives, deaths, 0)

	if rep.named != 0 || rep.contested != 1 || lives[len(lives)-1].xuid != 0 {
		t.Errorf("report %+v, want nothing named and one contested life", rep)
	}
}

// A film whose named respawns spread over seconds has no fixed delay to read: nothing is named.
func TestNameLivesByStart_NoFixedDelayNamesNothing(t *testing.T) {
	var lives []lifeSpan
	var deaths []Death
	t0 := int64(0)
	for i := 0; i < 25; i++ {
		end := t0 + 20_000
		lives = append(lives, lifeSpan{slot: uint32(500 + i), from: t0 * 1000, to: end * 1000, xuid: 111})
		deaths = append(deaths, Death{XUID: 111, TimeMS: end})
		t0 = end + 5_000 + int64(i%2)*10_000 // 5 s, 15 s, 5 s, ...
	}
	lives = append(lives, lifeSpan{slot: 900, from: (t0 + 5_000) * 1000, to: (t0 + 40_000) * 1000})

	if rep := nameLivesByStart(lives, deaths, 0); rep.calibrated || rep.named != 0 {
		t.Errorf("report %+v, want an uncalibrated film and nothing named", rep)
	}
}

// A player is never alive twice. A feed anomaly that gives one player two deaths ten seconds apart
// makes two overlapping lives each pick that player: both are refused, not one kept by order.
func TestNameLivesByStart_OverlappingPicksAreBothRefused(t *testing.T) {
	lives, deaths, lastDeath := chainedLives(111, 500, 0, 25)
	deaths = append(deaths, Death{XUID: 111, TimeMS: lastDeath + 5_000})
	lives = append(lives,
		lifeSpan{slot: 900, from: (lastDeath + respawnMS) * 1000, to: (lastDeath + 60_000) * 1000},
		lifeSpan{slot: 901, from: (lastDeath + 5_000 + respawnMS) * 1000, to: (lastDeath + 60_000) * 1000},
	)

	rep := nameLivesByStart(lives, deaths, 0)

	if rep.named != 0 || rep.overlapping != 2 {
		t.Errorf("report %+v, want nothing named and both overlapping picks refused", rep)
	}
}

// A death whose respawn a named life already accounts for does not name a second life.
func TestNameLivesByStart_AccountedRespawnIsNotReused(t *testing.T) {
	lives, deaths, _ := chainedLives(111, 500, 0, 25)
	// A second, unnamed life starting at the same instant as the 11th named one.
	lives = append(lives, lifeSpan{slot: 900, from: lives[10].from, to: lives[10].from + 5_000_000})

	if rep := nameLivesByStart(lives, deaths, 0); rep.named != 0 || lives[len(lives)-1].xuid != 0 {
		t.Errorf("report %+v, want the accounted respawn not reused", rep)
	}
}
