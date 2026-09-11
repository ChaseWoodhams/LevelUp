package main

import "testing"

// The mapping is asserted end to end in fetchone_expiry_test.go; this pins it at the
// source, where a new skip reason is added and where forgetting to classify it would
// silently record `downloaded` on a match that produced nothing.
func TestFilmStateOf(t *testing.T) {
	cases := []struct {
		name string
		out  outcome
		want filmState
	}{
		{"an artifact was built", outcome{ChunksWritten: 4}, stateDownloaded},
		{"the film is gone", outcome{SkipReason: skipFilmAbsent}, stateExpired},
		{"the decoder found nothing", outcome{ChunksWritten: 4, SkipReason: skipNoTracks}, stateFailed},
		{"the decoder refused the film", outcome{ChunksWritten: 4, SkipReason: skipBuildFailed}, stateFailed},
		// The film is on disk and the map catalogue is what is missing: neither terminal
		// state, so the next run tries again.
		{"the map has no bounds yet", outcome{ChunksWritten: 4, SkipReason: skipUnsupportedMap}, stateDownloaded},
		{"nothing happened at all", outcome{}, statePending},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := filmStateOf(tc.out); got != tc.want {
				t.Errorf("filmStateOf = %q, want %q", got, tc.want)
			}
		})
	}
}

// The retry policy in one assertion: expired stops a later run, and NOTHING else does.
// A second terminal state added here without a deliberate decision would silently bury
// matches a decoder fix or a catalogue update could still rescue.
func TestFilmState_OnlyExpiredIsTerminal(t *testing.T) {
	for _, s := range []filmState{statePending, stateDownloaded, stateFailed} {
		if s.terminal() {
			t.Errorf("%q is terminal: matches in that state can never be rescued", s)
		}
	}
	if !stateExpired.terminal() {
		t.Errorf("%q is not terminal: an expired film would be re-fetched forever", stateExpired)
	}
}
