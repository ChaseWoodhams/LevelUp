package main

// teamside_guard_test.go — THE SECOND COPY OF `t%d`, PINNED TO THE FIRST.
//
// `team_side` is the app's own encoding of a team id, written in
// `internal/service/match_view_builders_team.go` and read back by `teamSideToID` beside it and
// by `rosterLogic.ts` in the front end. This server writes it too, because it publishes the
// same field for the same consumer — and it cannot share the app's copy: that one is an
// unexported literal inside a package this ticket puts out of bounds for edits.
//
// CLAUDE.md rule 6 allows two copies. What it asks for at the second is a GUARD-RAIL, because
// a copy that can drift in silence eventually does: the app changes the encoding, this server
// keeps emitting the old one, and the symptom is a roster panel that silently groups all eight
// players into one nameless team — no error, no failing test, just a viewer that looks broken.
//
// So this test reads the app's source and fails the moment the two stop agreeing. A third copy
// owes a shared helper instead.

import (
	"os"
	"strings"
	"testing"
)

// appTeamSideSource is where the app writes a scoreboard row's team_side.
const appTeamSideSource = "../../internal/service/match_view_builders_team.go"

func TestTeamSideEncodingMatchesTheApp(t *testing.T) {
	raw, err := os.ReadFile(appTeamSideSource)
	if err != nil {
		t.Fatalf("reading the app's scoreboard builder at %s: %v", appTeamSideSource, err)
	}
	// The exact call the app makes. Quoted with the verb so this cannot match some other
	// Sprintf that happens to live in the same file.
	want := `fmt.Sprintf("` + teamSideFormat + `"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("%s no longer encodes team_side as %q.\n"+
			"This server publishes the same field to the same consumer, so the two must agree: "+
			"update teamSideFormat in participants.go to match, or — if there is now a third "+
			"copy — extract a shared helper and point both at it.",
			appTeamSideSource, teamSideFormat)
	}
}

// TestTeamSideGuardWouldCatchADrift proves the guard is not vacuous: a test that passes because
// it searches for something always present would protect nothing.
func TestTeamSideGuardWouldCatchADrift(t *testing.T) {
	raw, err := os.ReadFile(appTeamSideSource)
	if err != nil {
		t.Fatalf("reading %s: %v", appTeamSideSource, err)
	}
	if strings.Contains(string(raw), `fmt.Sprintf("team-%d"`) {
		t.Fatal("the drifted encoding this guard looks for is present; the guard proves nothing")
	}
}
