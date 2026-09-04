package main

// watch_test.go — ONE UNATTENDED PASS, DRIVEN THROUGH THE FAKE HALO SERVER (#8).
//
// The pass is judged on what it FETCHES as much as on what it records: "already known
// matches are not re-processed" and "expired or failed are not retried" are claims about
// not calling the API, and a row-count assertion would pass on a tool that re-downloaded
// everything every hour and rewrote the same verdicts.

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/replay"
)

// The three matches a pass sees: a 4v4 from matchmaking, a 4v4 from customs (scrims are
// exactly the games worth reviewing), and a two-player game that is not what the archive
// is for. All well-formed UUIDs — the client rejects anything else.
const (
	arenaMatchID  = "000d5950-1234-4abc-9def-0123456789ab" // == testMatchID
	customMatchID = "111e6a61-2345-4bcd-8ef0-123456789abc"
	duoMatchID    = "222f7b72-3456-4cde-9f01-23456789abcd"
)

// watchFixture wires a fake server carrying the three matches above and a watchlist of one
// player, with the xuid resolution counted.
type watchFixture struct {
	srv       *fakeHalo
	deps      deps
	watch     watchDeps
	list      watchlist
	resolved  int
	buildRuns int
}

func newWatchFixture(t *testing.T) *watchFixture {
	t.Helper()
	srv := newFakeHalo(t, map[int][]byte{0: []byte("header"), 1: []byte("replication")}, nil)
	srv.statsByMatch = map[string]map[string]any{
		arenaMatchID:  statsFor(arenaMatchID, "Cliffhanger", fourVFourRoster("TrackedPro")),
		customMatchID: statsFor(customMatchID, "Cliffhanger", fourVFourRoster("TrackedPro")),
		duoMatchID:    statsFor(duoMatchID, "Cliffhanger", twoPlayerRoster()),
	}
	srv.history = map[string][]string{
		"matchmaking": {arenaMatchID, duoMatchID},
		"custom":      {customMatchID},
	}

	f := &watchFixture{srv: srv, list: watchlist{Gamertags: []string{"TrackedPro"}}}
	f.deps = srv.deps(t, countingBuild(&f.buildRuns, stubBuild(nil)))
	f.deps.SourceGamertag = "" // watch fills it per match, from the watchlist entry
	f.watch = watchDeps{ResolveXUID: func(context.Context, string) (string, error) {
		f.resolved++
		return "2533274823110022", nil
	}}
	return f
}

func (f *watchFixture) pass(t *testing.T) watchSummary {
	t.Helper()
	return watchPass(context.Background(), f.deps, f.watch, f.list)
}

func (f *watchFixture) row(t *testing.T, matchID string) (matchRecord, bool) {
	t.Helper()
	rec, found, err := f.deps.Archive.recorded(context.Background(), matchID)
	if err != nil {
		t.Fatalf("reading the archive row of %s: %v", matchID, err)
	}
	return rec, found
}

// THE TICKET'S OWN ACCEPTANCE TEST: a pass archives the unseen 4v4s from both histories
// and leaves everything else alone.
func TestWatch_ArchivesTheUnseenFourVFours(t *testing.T) {
	f := newWatchFixture(t)

	sum := f.pass(t)

	if sum.Failed != 0 {
		t.Fatalf("%d failures in a pass where nothing should fail", sum.Failed)
	}
	if sum.Archived != 2 {
		t.Errorf("archived %d, want 2 (the matchmaking 4v4 and the custom one)", sum.Archived)
	}
	if sum.Skipped != 1 {
		t.Errorf("skipped %d, want 1 (the two-player game)", sum.Skipped)
	}
	for _, id := range []string{arenaMatchID, customMatchID} {
		rec, found := f.row(t, id)
		if !found || rec.ArtifactPath == "" {
			t.Errorf("%s was not archived (found=%v artifact=%q)", id, found, rec.ArtifactPath)
		}
	}
	// The match that is not a 4v4 leaves NO row: it was never archived, and recording it
	// would put a match the archive is not for into every count #9 reports.
	if _, found := f.row(t, duoMatchID); found {
		t.Error("a two-player game was recorded - the archive is for 4v4s")
	}

	// Both histories are read, matchmaking and customs.
	if got := f.srv.historyCalls.Load(); got != 2 {
		t.Errorf("%d history calls, want 2 (matchmaking + custom)", got)
	}

	// The resolution happened once and was recorded, so no later pass repeats it.
	if f.resolved != 1 {
		t.Errorf("resolved the gamertag %d times, want 1", f.resolved)
	}
	player, found, err := f.deps.Archive.watched(context.Background(), "TrackedPro")
	if err != nil || !found {
		t.Fatalf("watchlist row: found=%v err=%v", found, err)
	}
	if player.XUID != "2533274823110022" {
		t.Errorf("recorded xuid = %q, want the resolved one", player.XUID)
	}
	if player.LastChecked == nil {
		t.Error("last_checked was not stamped: a quiet pass is indistinguishable from a job that never ran")
	}

	// source_gamertag records WHOSE pass surfaced the match — what #9 counts by player.
	var source string
	if err := f.deps.Archive.db.QueryRow(context.Background(),
		`SELECT source_gamertag FROM matches WHERE match_id = ?`, arenaMatchID).Scan(&source); err != nil {
		t.Fatalf("reading source_gamertag: %v", err)
	}
	if source != "TrackedPro" {
		t.Errorf("source_gamertag = %q, want TrackedPro", source)
	}
}

// A second pass an hour later must cost stats calls for nothing it already has, and must
// not resolve the gamertag again.
func TestWatch_SecondPassReprocessesNothing(t *testing.T) {
	f := newWatchFixture(t)
	f.pass(t)

	statsAfterFirst := f.srv.statsCalls.Load()
	buildsAfterFirst := f.buildRuns

	sum := f.pass(t)

	if sum.Archived != 0 {
		t.Errorf("archived %d on the second pass, want 0", sum.Archived)
	}
	if f.buildRuns != buildsAfterFirst {
		t.Errorf("%d builds, want %d - the second pass rebuilt an archived match",
			f.buildRuns, buildsAfterFirst)
	}
	if f.resolved != 1 {
		t.Errorf("resolved %d times over two passes, want 1 - the recorded xuid was ignored", f.resolved)
	}
	// The two archived matches cost no stats call at all; only the two-player game is
	// re-examined, because nothing about it was recorded.
	if got := f.srv.statsCalls.Load(); got != statsAfterFirst+1 {
		t.Errorf("%d stats calls after the second pass, want %d (only the unrecorded "+
			"non-4v4 is re-read)", got, statsAfterFirst+1)
	}
}

// The retry policy of #7, enforced at the DISCOVERY layer: neither terminal state is
// re-attempted by the hourly loop. They are skipped for OPPOSITE reasons, which is why
// both are asserted — and why `failed` is skipped here but stays rebuildable elsewhere.
func TestWatch_NeverRetriesExpiredOrFailedMatches(t *testing.T) {
	cases := map[string]struct {
		state  filmState
		reason reason
	}{
		"expired films are gone for good":  {stateExpired, skipFilmAbsent},
		"failed builds wait for a rebuild": {stateFailed, skipNoTracks},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f := newWatchFixture(t)
			// Only the matchmaking 4v4 is on the history, already recorded in the state
			// under test with no artifact.
			f.srv.history = map[string][]string{"matchmaking": {arenaMatchID}, "custom": nil}
			rec, roster := sampleRecord()
			rec.ArtifactPath, rec.BuiltAt, rec.DecoderRev = "", nil, ""
			rec.State, rec.SkipReason = tc.state, tc.reason
			if err := f.deps.Archive.recordMatch(context.Background(), rec, roster); err != nil {
				t.Fatalf("seeding the archive: %v", err)
			}
			statsBefore := f.srv.statsCalls.Load()

			sum := f.pass(t)

			if sum.Archived != 0 || sum.Failed != 0 {
				t.Errorf("archived %d / failed %d, want 0/0", sum.Archived, sum.Failed)
			}
			if got := f.srv.statsCalls.Load(); got != statsBefore {
				t.Errorf("%d stats calls, want %d - a %s match was re-examined",
					got, statsBefore, tc.state)
			}
			if f.buildRuns != 0 {
				t.Errorf("%d builds on a %s match", f.buildRuns, tc.state)
			}
			after, _ := f.row(t, arenaMatchID)
			if after.State != tc.state {
				t.Errorf("state = %q, want %q left untouched", after.State, tc.state)
			}
		})
	}
}

// A match recorded `downloaded` with NO artifact is not settled: its map may have entered
// the quant-bounds catalogue since. The loop must pick it up again.
func TestWatch_RetriesAMatchWhoseMapWasUnsupported(t *testing.T) {
	f := newWatchFixture(t)
	f.srv.history = map[string][]string{"matchmaking": {arenaMatchID}, "custom": nil}
	rec, roster := sampleRecord()
	rec.ArtifactPath, rec.BuiltAt, rec.DecoderRev = "", nil, ""
	rec.State, rec.SkipReason = stateDownloaded, skipUnsupportedMap
	if err := f.deps.Archive.recordMatch(context.Background(), rec, roster); err != nil {
		t.Fatalf("seeding the archive: %v", err)
	}

	if sum := f.pass(t); sum.Archived != 1 {
		t.Errorf("archived %d, want 1 - the catalogue may have grown since", sum.Archived)
	}
	if after, _ := f.row(t, arenaMatchID); after.ArtifactPath == "" {
		t.Error("the retried match produced no artifact")
	}
}

// One player's failure must not cost the pass: the films of everybody else are expiring
// while it would be giving up.
func TestWatch_APlayerFailureDoesNotStopThePass(t *testing.T) {
	f := newWatchFixture(t)
	f.list = watchlist{Gamertags: []string{"Unresolvable", "TrackedPro"}}
	f.watch = watchDeps{ResolveXUID: func(_ context.Context, gt string) (string, error) {
		if gt == "Unresolvable" {
			return "", errors.New("no such xbox profile")
		}
		return "2533274823110022", nil
	}}

	sum := f.pass(t)

	if sum.Failed != 1 {
		t.Errorf("failed = %d, want 1 (the unresolvable player)", sum.Failed)
	}
	if sum.Archived != 2 {
		t.Errorf("archived %d, want 2 - the second player's matches were lost to the first's failure",
			sum.Archived)
	}
}

// A match that fails to ARCHIVE is counted and left behind, and the pass carries on.
func TestWatch_AnArchivingFailureDoesNotStopThePass(t *testing.T) {
	f := newWatchFixture(t)
	f.deps.Build = func(matchID, titleSlug, filmDir string, opt replay.Options) (replay.ReplayDocument, error) {
		if matchID == arenaMatchID {
			return replay.ReplayDocument{}, errors.New("decoder exploded")
		}
		return stubBuild(nil)(matchID, titleSlug, filmDir, opt)
	}

	sum := f.pass(t)

	if sum.Failed != 1 {
		t.Errorf("failed = %d, want 1", sum.Failed)
	}
	if sum.Archived != 1 {
		t.Errorf("archived %d, want 1 - the custom match was lost to the other's failure", sum.Archived)
	}
	// #7's contract holds inside the loop: the decoder's refusal is recorded as `failed`.
	if rec, found := f.row(t, arenaMatchID); !found || rec.State != stateFailed {
		t.Errorf("state = %q (found=%v), want %q", rec.State, found, stateFailed)
	}
}

// The shape test, at the source. It is the filter, in place of a playlist NAME: playlists
// get renamed between seasons and their labels are localised.
func TestIsArenaFourVFour(t *testing.T) {
	team := func(t int) *int { return &t }
	roster := func(teams ...int) []participantRecord {
		out := make([]participantRecord, 0, len(teams))
		for _, tm := range teams {
			out = append(out, participantRecord{XUID: "x", Team: team(tm)})
		}
		return out
	}
	cases := []struct {
		name  string
		given []participantRecord
		want  bool
	}{
		{"four against four", roster(0, 0, 0, 0, 1, 1, 1, 1), true},
		{"five against three", roster(0, 0, 0, 0, 0, 1, 1, 1), false},
		{"two against two", roster(0, 0, 1, 1), false},
		{"eight players, four teams", roster(0, 0, 1, 1, 2, 2, 3, 3), false},
		{"free-for-all of eight", roster(0, 1, 2, 3, 4, 5, 6, 7), false},
		{"empty", nil, false},
	}
	for _, c := range cases {
		if got := isArenaFourVFour(c.given); got != c.want {
			t.Errorf("%s: isArenaFourVFour = %v, want %v", c.name, got, c.want)
		}
	}

	// A roster entry with no team at all cannot be reasoned about: everything downstream
	// slices by team.
	teamless := roster(0, 0, 0, 0, 1, 1, 1)
	teamless = append(teamless, participantRecord{XUID: "x"})
	if isArenaFourVFour(teamless) {
		t.Error("a roster with a team-less player passed for a 4v4")
	}
}

func TestLoadWatchlist(t *testing.T) {
	write := func(t *testing.T, body string) string {
		t.Helper()
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, watchlistFileName), []byte(body), 0o600); err != nil {
			t.Fatalf("writing the watchlist: %v", err)
		}
		return root
	}

	t.Run("gamertags in file order", func(t *testing.T) {
		wl, err := loadWatchlist(write(t, "gamertags = [\"Bravo\", \"Alpha\"]\n"))
		if err != nil {
			t.Fatalf("loadWatchlist: %v", err)
		}
		if len(wl.Gamertags) != 2 || wl.Gamertags[0] != "Bravo" {
			t.Errorf("gamertags = %v, want [Bravo Alpha] in file order", wl.Gamertags)
		}
	})

	// Xbox gamertags are case-insensitive: two spellings are one player, and keeping both
	// would resolve, pull and count them twice.
	t.Run("duplicates and blanks are dropped", func(t *testing.T) {
		wl, err := loadWatchlist(write(t, "gamertags = [\"Alpha\", \" alpha \", \"\", \"Bravo\"]\n"))
		if err != nil {
			t.Fatalf("loadWatchlist: %v", err)
		}
		if len(wl.Gamertags) != 2 {
			t.Errorf("gamertags = %v, want 2 entries", wl.Gamertags)
		}
	})

	// An absent or empty file is an ERROR: a watch that archives nothing looks exactly
	// like a watch with nothing to archive, and the job would report success forever.
	t.Run("an absent file is an error", func(t *testing.T) {
		if _, err := loadWatchlist(t.TempDir()); err == nil {
			t.Fatal("a missing watchlist was accepted - the pass would silently archive nothing")
		}
	})
	t.Run("an empty list is an error", func(t *testing.T) {
		if _, err := loadWatchlist(write(t, "gamertags = []\n")); err == nil {
			t.Fatal("an empty watchlist was accepted")
		}
	})
	t.Run("a malformed file is an error", func(t *testing.T) {
		if _, err := loadWatchlist(write(t, "gamertags = \"not a list\"\n")); err == nil {
			t.Fatal("a malformed watchlist was accepted")
		}
	})
}

func TestLastPassAge(t *testing.T) {
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	if got := lastPassAge(watchedPlayer{}, now); got != "never" {
		t.Errorf("a player never checked reads %q, want \"never\" - a zero duration would "+
			"read as \"checked a moment ago\"", got)
	}
	earlier := now.Add(-90 * time.Minute)
	if got := lastPassAge(watchedPlayer{LastChecked: &earlier}, now); got != "1h30m0s ago" {
		t.Errorf("last checked = %q, want \"1h30m0s ago\"", got)
	}
}
