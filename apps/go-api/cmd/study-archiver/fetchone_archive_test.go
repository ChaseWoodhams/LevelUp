package main

// fetchone_archive_test.go — WHAT ARCHIVING LEAVES IN THE DATABASE.
//
// archive_test.go covers the store on its own. This file covers the ticket's own claim:
// that running fetch-one against the fake Halo server produces the right ROWS, on every
// path, and that running it twice does not fetch anything a second time.
//
// What those rows then MEAN to a later run — which states are retried, which one is
// terminal, and what a transient failure must not write — is fetchone_expiry_test.go.

import (
	"context"
	"os"
	"testing"

	"levelup/go-api/internal/analysis/replay"
)

// archived runs fetch-one and reads the resulting row back.
func archived(t *testing.T, d deps) (outcome, matchRecord) {
	t.Helper()
	out, err := fetchOne(context.Background(), d, testMatchID)
	if err != nil {
		t.Fatalf("fetchOne: %v", err)
	}
	rec, found, err := d.Archive.recorded(context.Background(), testMatchID)
	if err != nil {
		t.Fatalf("reading the archive row: %v", err)
	}
	if !found {
		t.Fatal("fetch-one archived a match without recording it")
	}
	return out, rec
}

func TestFetchOne_RecordsTheMatchAndItsRoster(t *testing.T) {
	srv := newFakeHalo(t, map[int][]byte{0: []byte("header"), 1: []byte("replication")},
		statsWithMap("Cliffhanger"))
	d := srv.deps(t, stubBuild(nil))
	ctx := context.Background()

	out, rec := archived(t, d)

	if rec.State != stateDownloaded {
		t.Errorf("film_state = %q, want %q", rec.State, stateDownloaded)
	}
	if rec.ArtifactPath != out.ArtifactPath || rec.ArtifactPath == "" {
		t.Errorf("artifact_path = %q, want the built artifact %q", rec.ArtifactPath, out.ArtifactPath)
	}
	if rec.MapName != "Cliffhanger" || rec.MapModule != "olympus" {
		t.Errorf("map = %q/%q, want Cliffhanger/olympus", rec.MapName, rec.MapModule)
	}
	if rec.SkipReason != "" {
		t.Errorf("skip_reason = %q on an archived match", rec.SkipReason)
	}

	// The columns the single reader does not carry, checked at the source.
	var mode, playlist, source string
	var durationMS int64
	var builtAt any
	if err := d.Archive.db.QueryRow(ctx, `
        SELECT mode, playlist, source_gamertag, duration_ms, built_at
        FROM matches WHERE match_id = ?`, testMatchID).
		Scan(&mode, &playlist, &source, &durationMS, &builtAt); err != nil {
		t.Fatalf("reading the recorded facts: %v", err)
	}
	if mode != "Slayer" || playlist != "Ranked Arena" {
		t.Errorf("mode/playlist = %q/%q, want Slayer/Ranked Arena", mode, playlist)
	}
	if source != "JGtm" {
		t.Errorf("source_gamertag = %q, want JGtm", source)
	}
	// PT9M13S, carried through sync.ExtractRegistry's own duration parsing.
	if durationMS != 553000 {
		t.Errorf("duration_ms = %d, want 553000", durationMS)
	}
	if builtAt == nil {
		t.Error("built_at is NULL on a match that produced an artifact")
	}

	// Team and outcome come from MATCH STATS, never from the film.
	var n, team, outcomeCol, kills, deaths, assists int
	var gamertag string
	if err := d.Archive.db.QueryRow(ctx,
		`SELECT count(*) FROM participants WHERE match_id = ?`, testMatchID).Scan(&n); err != nil {
		t.Fatalf("counting the roster: %v", err)
	}
	if n != 2 {
		t.Fatalf("roster = %d rows, want the 2 players in the stats payload", n)
	}
	if err := d.Archive.db.QueryRow(ctx, `
        SELECT gamertag, team, outcome, kills, deaths, assists
        FROM participants WHERE match_id = ? AND xuid = '1'`, testMatchID).
		Scan(&gamertag, &team, &outcomeCol, &kills, &deaths, &assists); err != nil {
		t.Fatalf("reading a participant: %v", err)
	}
	if gamertag != "JGtm" || team != 0 || outcomeCol != 2 {
		t.Errorf("gamertag/team/outcome = %q/%d/%d, want JGtm/0/2", gamertag, team, outcomeCol)
	}
	if kills != 15 || deaths != 9 || assists != 4 {
		t.Errorf("K/D/A = %d/%d/%d, want 15/9/4", kills, deaths, assists)
	}
}

// The acceptance criterion, and the reason it is asserted on API CALLS and not on row
// counts: "no needless re-download or rebuild". A second pass that re-fetched the film
// and rebuilt it would still leave exactly one row.
func TestFetchOne_SecondPassFetchesAndBuildsNothing(t *testing.T) {
	srv := newFakeHalo(t, map[int][]byte{0: []byte("header"), 1: []byte("replication")},
		statsWithMap("Cliffhanger"))
	builds := 0
	d := srv.deps(t, func(matchID, titleSlug, filmDir string, opt replay.Options) (replay.ReplayDocument, error) {
		builds++
		return stubBuild(nil)(matchID, titleSlug, filmDir, opt)
	})

	first, _ := archived(t, d)
	statsAfterFirst := srv.statsCalls.Load()
	manifestAfterFirst := srv.manifestCalls.Load()

	second, rec := archived(t, d)

	if !second.Settled {
		t.Error("the second pass did not recognise the match as already archived")
	}
	if builds != 1 {
		t.Errorf("%d builds, want 1 - the second pass rebuilt the replay", builds)
	}
	if got := srv.manifestCalls.Load(); got != manifestAfterFirst {
		t.Errorf("manifest fetched %d times, want %d - the film was re-downloaded",
			got, manifestAfterFirst)
	}
	if got := srv.statsCalls.Load(); got != statsAfterFirst {
		t.Errorf("stats fetched %d times, want %d", got, statsAfterFirst)
	}

	// One row, and the second pass still reports what the archive holds.
	var matches, parts int
	ctx := context.Background()
	if err := d.Archive.db.QueryRow(ctx, `SELECT count(*) FROM matches`).Scan(&matches); err != nil {
		t.Fatalf("counting matches: %v", err)
	}
	if err := d.Archive.db.QueryRow(ctx, `SELECT count(*) FROM participants`).Scan(&parts); err != nil {
		t.Fatalf("counting participants: %v", err)
	}
	if matches != 1 || parts != 2 {
		t.Errorf("rows = %d matches / %d participants, want 1/2", matches, parts)
	}
	if second.ArtifactPath != first.ArtifactPath || second.Tracks != first.Tracks {
		t.Errorf("the second pass reported %q/%d, want the archived %q/%d",
			second.ArtifactPath, second.Tracks, first.ArtifactPath, first.Tracks)
	}
	if rec.ArtifactPath != first.ArtifactPath {
		t.Errorf("the row changed: %q, want %q", rec.ArtifactPath, first.ArtifactPath)
	}
}

// A recorded artifact that has gone missing from disk is not an archive. Re-running must
// rebuild it rather than trust the row — otherwise a deleted cache is invisible forever.
func TestFetchOne_MissingArtifactIsRebuilt(t *testing.T) {
	srv := newFakeHalo(t, map[int][]byte{0: []byte("header"), 1: []byte("replication")},
		statsWithMap("Cliffhanger"))
	builds := 0
	d := srv.deps(t, func(matchID, titleSlug, filmDir string, opt replay.Options) (replay.ReplayDocument, error) {
		builds++
		return stubBuild(nil)(matchID, titleSlug, filmDir, opt)
	})

	first, _ := archived(t, d)
	if err := os.Remove(first.ArtifactPath); err != nil {
		t.Fatalf("removing the artifact: %v", err)
	}

	second, rec := archived(t, d)
	if second.Settled {
		t.Error("a match whose artifact is gone was reported as already archived")
	}
	if builds != 2 {
		t.Errorf("%d builds, want 2 - the missing artifact was not rebuilt", builds)
	}
	if _, err := os.Stat(rec.ArtifactPath); err != nil {
		t.Errorf("the artifact was not written again: %v", err)
	}
}

// Every skip is recorded, with the film state that says what a later run should make of
// it. The three differ, and collapsing them would lose the distinction #7 is built on.
func TestFetchOne_SkipsAreRecordedWithTheirState(t *testing.T) {
	film := map[int][]byte{0: []byte("header"), 1: []byte("replication")}

	t.Run("unsupported map stays downloadable", func(t *testing.T) {
		srv := newFakeHalo(t, film, statsWithMap("Forbidden Sands"))
		_, rec := archived(t, srv.deps(t, stubBuild(nil)))
		if rec.State != stateDownloaded || rec.SkipReason != skipUnsupportedMap {
			t.Errorf("state/reason = %q/%q, want %q/%q",
				rec.State, rec.SkipReason, stateDownloaded, skipUnsupportedMap)
		}
		if rec.ArtifactPath != "" {
			t.Errorf("artifact_path = %q on an unsupported map", rec.ArtifactPath)
		}
	})

	t.Run("expired film", func(t *testing.T) {
		srv := newFakeHalo(t, film, statsWithMap("Cliffhanger"))
		srv.manifestStatus = 410
		_, rec := archived(t, srv.deps(t, stubBuild(nil)))
		if rec.State != stateExpired || rec.SkipReason != skipFilmAbsent {
			t.Errorf("state/reason = %q/%q, want %q/%q",
				rec.State, rec.SkipReason, stateExpired, skipFilmAbsent)
		}
	})

	t.Run("nothing decoded", func(t *testing.T) {
		srv := newFakeHalo(t, film, statsWithMap("Cliffhanger"))
		empty := replay.ReplayDocument{SchemaVersion: replay.SchemaVersion}
		_, rec := archived(t, srv.deps(t, stubBuild(&empty)))
		if rec.State != stateFailed || rec.SkipReason != skipNoTracks {
			t.Errorf("state/reason = %q/%q, want %q/%q",
				rec.State, rec.SkipReason, stateFailed, skipNoTracks)
		}
	})

	// A skipped match still records its roster: the players are known from the stats
	// whether or not the film decoded, and #9's status counts read them.
	t.Run("roster recorded even when skipped", func(t *testing.T) {
		srv := newFakeHalo(t, film, statsWithMap("Forbidden Sands"))
		d := srv.deps(t, stubBuild(nil))
		archived(t, d)
		var n int
		if err := d.Archive.db.QueryRow(context.Background(),
			`SELECT count(*) FROM participants WHERE match_id = ?`, testMatchID).Scan(&n); err != nil {
			t.Fatalf("counting the roster: %v", err)
		}
		if n != 2 {
			t.Errorf("roster = %d rows, want 2", n)
		}
	})
}

// The decoded counts come from the document's own coverage report — including when there
// is none. Coverage is a nillable, omitempty pointer, so this is the crash path.
func TestFetchOne_CountsSurviveADocumentWithoutCoverage(t *testing.T) {
	srv := newFakeHalo(t, map[int][]byte{0: []byte("header"), 1: []byte("replication")},
		statsWithMap("Cliffhanger"))

	withCoverage := replay.ReplayDocument{
		SchemaVersion: replay.SchemaVersion,
		Tracks:        []replay.Track{{Slot: 1, Team: -1, Points: []replay.Point{{T: 0}, {T: 1}}}},
		Coverage:      &replay.Coverage{Bridge: replay.BridgeHealth{LivesNamed: 90, LivesTotal: 105}},
	}
	_, rec := archived(t, srv.deps(t, stubBuild(&withCoverage)))
	if rec.NamedLives != 90 || rec.TotalLives != 105 {
		t.Errorf("lives = %d/%d, want 90/105", rec.NamedLives, rec.TotalLives)
	}

	// The same build with no coverage report at all must record zero, not panic.
	srv2 := newFakeHalo(t, map[int][]byte{0: []byte("header"), 1: []byte("replication")},
		statsWithMap("Cliffhanger"))
	noCoverage := withCoverage
	noCoverage.Coverage = nil
	_, rec2 := archived(t, srv2.deps(t, stubBuild(&noCoverage)))
	if rec2.NamedLives != 0 || rec2.TotalLives != 0 {
		t.Errorf("lives = %d/%d, want 0/0", rec2.NamedLives, rec2.TotalLives)
	}
	if rec2.Tracks != 1 {
		t.Errorf("tracks = %d, want 1 - the other counts must survive too", rec2.Tracks)
	}
}
