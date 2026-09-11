package main

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"
)

// testArchive opens a throwaway archive database.
func testArchive(t *testing.T) *archive {
	t.Helper()
	a, err := openArchive(filepath.Join(t.TempDir(), "archive.duckdb"))
	if err != nil {
		t.Fatalf("openArchive: %v", err)
	}
	t.Cleanup(func() { _ = a.Close() })
	return a
}

func intPtr(v int) *int { return &v }

func sampleRecord() (matchRecord, []participantRecord) {
	played := time.Date(2026, 5, 19, 20, 15, 0, 0, time.UTC)
	built := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	dur := int64(553000)
	m := matchRecord{
		MatchID: testMatchID, ShortID: "000d5950",
		PlayedAt: &played, MapName: "Cliffhanger", MapModule: "olympus",
		Mode: "Slayer", Playlist: "Ranked Arena", DurationMS: &dur,
		SourceGT: "JGtm", State: stateDownloaded,
		ArtifactPath: "data/cache/replays/halo_infinite/000d5950.json",
		BuiltAt:      &built, DecoderRev: "abc1234",
		Tracks: 8, Points: 4200, Shots: 519, NamedLives: 90, TotalLives: 105,
	}
	ps := []participantRecord{
		{XUID: "1", Gamertag: "JGtm", Team: intPtr(0), Outcome: intPtr(2),
			Kills: intPtr(15), Deaths: intPtr(9), Assists: intPtr(4)},
		{XUID: "2", Gamertag: "Rival", Team: intPtr(1), Outcome: intPtr(3),
			Kills: intPtr(9), Deaths: intPtr(15), Assists: intPtr(2)},
	}
	return m, ps
}

func TestArchive_RecordsMatchAndRoster(t *testing.T) {
	a := testArchive(t)
	ctx := context.Background()
	rec, roster := sampleRecord()

	if err := a.recordMatch(ctx, rec, roster); err != nil {
		t.Fatalf("recordMatch: %v", err)
	}

	var (
		shortID, mapName, mapModule, mode, playlist, state, artifact, rev string
		playedAt, builtAt                                                 time.Time
		durationMS                                                        int64
		tracks, points, shots, named, total                               int
	)
	err := a.db.QueryRow(ctx, `
        SELECT short_id, played_at, map_name, map_module, mode, playlist, duration_ms,
               film_state, artifact_path, built_at, decoder_rev,
               tracks, points, shots, named_lives, total_lives
        FROM matches WHERE match_id = ?`, testMatchID).
		Scan(&shortID, &playedAt, &mapName, &mapModule, &mode, &playlist, &durationMS,
			&state, &artifact, &builtAt, &rev, &tracks, &points, &shots, &named, &total)
	if err != nil {
		t.Fatalf("reading back the match: %v", err)
	}
	if shortID != "000d5950" || mapName != "Cliffhanger" || mapModule != "olympus" {
		t.Errorf("identity/map = %q/%q/%q", shortID, mapName, mapModule)
	}
	if !playedAt.Equal(*rec.PlayedAt) {
		t.Errorf("played_at = %s, want %s", playedAt, *rec.PlayedAt)
	}
	if mode != "Slayer" || playlist != "Ranked Arena" || durationMS != 553000 {
		t.Errorf("mode/playlist/duration = %q/%q/%d", mode, playlist, durationMS)
	}
	if state != string(stateDownloaded) || artifact == "" || rev != "abc1234" {
		t.Errorf("state/artifact/rev = %q/%q/%q", state, artifact, rev)
	}
	if tracks != 8 || points != 4200 || shots != 519 || named != 90 || total != 105 {
		t.Errorf("counts = %d/%d/%d/%d/%d", tracks, points, shots, named, total)
	}

	var n int
	if err := a.db.QueryRow(ctx,
		`SELECT count(*) FROM participants WHERE match_id = ?`, testMatchID).Scan(&n); err != nil {
		t.Fatalf("counting roster: %v", err)
	}
	if n != 2 {
		t.Errorf("roster = %d rows, want 2", n)
	}

	var team, outcome, kills, deaths, assists int
	var gamertag string
	err = a.db.QueryRow(ctx, `
        SELECT gamertag, team, outcome, kills, deaths, assists
        FROM participants WHERE match_id = ? AND xuid = '1'`, testMatchID).
		Scan(&gamertag, &team, &outcome, &kills, &deaths, &assists)
	if err != nil {
		t.Fatalf("reading back a participant: %v", err)
	}
	if gamertag != "JGtm" || team != 0 || outcome != 2 {
		t.Errorf("gamertag/team/outcome = %q/%d/%d", gamertag, team, outcome)
	}
	if kills != 15 || deaths != 9 || assists != 4 {
		t.Errorf("K/D/A = %d/%d/%d", kills, deaths, assists)
	}
}

// Recording the same match twice must leave ONE row and ONE roster — the acceptance
// criterion "no duplicate rows", at the storage layer.
func TestArchive_RecordingTwiceReplacesRatherThanDuplicates(t *testing.T) {
	a := testArchive(t)
	ctx := context.Background()
	rec, roster := sampleRecord()

	if err := a.recordMatch(ctx, rec, roster); err != nil {
		t.Fatalf("first record: %v", err)
	}
	// A second pass that sees fewer players and a different verdict — the shape a
	// corrected re-read would take.
	rec.Tracks, rec.State = 4, stateFailed
	if err := a.recordMatch(ctx, rec, roster[:1]); err != nil {
		t.Fatalf("second record: %v", err)
	}

	var matches, parts, tracks int
	var state string
	if err := a.db.QueryRow(ctx, `SELECT count(*) FROM matches`).Scan(&matches); err != nil {
		t.Fatalf("counting matches: %v", err)
	}
	if err := a.db.QueryRow(ctx, `SELECT count(*) FROM participants`).Scan(&parts); err != nil {
		t.Fatalf("counting participants: %v", err)
	}
	if err := a.db.QueryRow(ctx,
		`SELECT tracks, film_state FROM matches`).Scan(&tracks, &state); err != nil {
		t.Fatalf("reading back: %v", err)
	}
	if matches != 1 {
		t.Errorf("matches = %d rows, want 1", matches)
	}
	// The roster is a SET: the player dropped by the second read must be gone, not
	// left behind by a keyed upsert.
	if parts != 1 {
		t.Errorf("participants = %d rows, want 1 - the stale player was not cleared", parts)
	}
	if tracks != 4 || state != string(stateFailed) {
		t.Errorf("the second record did not replace the first: tracks=%d state=%q", tracks, state)
	}
}

func TestArchive_RecordedReadsBackWhatWasWritten(t *testing.T) {
	a := testArchive(t)
	ctx := context.Background()

	if _, found, err := a.recorded(ctx, testMatchID); err != nil || found {
		t.Fatalf("unknown match reported found=%v err=%v", found, err)
	}

	rec, roster := sampleRecord()
	if err := a.recordMatch(ctx, rec, roster); err != nil {
		t.Fatalf("recordMatch: %v", err)
	}
	got, found, err := a.recorded(ctx, testMatchID)
	if err != nil || !found {
		t.Fatalf("recorded: found=%v err=%v", found, err)
	}
	if got.State != stateDownloaded || got.ArtifactPath != rec.ArtifactPath {
		t.Errorf("state/artifact = %q/%q, want %q/%q",
			got.State, got.ArtifactPath, stateDownloaded, rec.ArtifactPath)
	}
	if got.MapName != "Cliffhanger" || got.MapModule != "olympus" {
		t.Errorf("map = %q/%q", got.MapName, got.MapModule)
	}
	if got.Tracks != 8 || got.NamedLives != 90 || got.TotalLives != 105 {
		t.Errorf("counts = %d/%d/%d", got.Tracks, got.NamedLives, got.TotalLives)
	}
}

// A row whose stats named no map at all stores NULL in map_name/map_module. Reading it
// back must not fail — this is exactly the row the idempotency check has to survive.
func TestArchive_RecordedSurvivesNullMap(t *testing.T) {
	a := testArchive(t)
	ctx := context.Background()
	rec, _ := sampleRecord()
	rec.MapName, rec.MapModule, rec.ArtifactPath = "", "", ""
	rec.BuiltAt, rec.DecoderRev = nil, ""
	rec.SkipReason = skipNoMapInStats

	if err := a.recordMatch(ctx, rec, nil); err != nil {
		t.Fatalf("recordMatch: %v", err)
	}
	got, found, err := a.recorded(ctx, testMatchID)
	if err != nil || !found {
		t.Fatalf("recorded: found=%v err=%v", found, err)
	}
	if got.MapName != "" || got.SkipReason != skipNoMapInStats {
		t.Errorf("map=%q skip=%q", got.MapName, got.SkipReason)
	}
}

// A match recorded WITHOUT an artifact (unsupported map) must read back as a NULL
// artifact path, not as an empty string that a later query would mistake for a path.
func TestArchive_SkippedMatchHasNullArtifact(t *testing.T) {
	a := testArchive(t)
	ctx := context.Background()
	rec, roster := sampleRecord()
	rec.ArtifactPath, rec.BuiltAt, rec.DecoderRev = "", nil, ""
	rec.State, rec.SkipReason = stateDownloaded, skipUnsupportedMap

	if err := a.recordMatch(ctx, rec, roster); err != nil {
		t.Fatalf("recordMatch: %v", err)
	}
	var artifact, builtAt sql.NullString
	var skip string
	if err := a.db.QueryRow(ctx,
		`SELECT artifact_path, built_at, skip_reason FROM matches WHERE match_id = ?`,
		testMatchID).Scan(&artifact, &builtAt, &skip); err != nil {
		t.Fatalf("reading back: %v", err)
	}
	if artifact.Valid || builtAt.Valid {
		t.Errorf("artifact_path/built_at should be NULL, got %v/%v", artifact, builtAt)
	}
	if skip != string(skipUnsupportedMap) {
		t.Errorf("skip_reason = %q, want %q", skip, skipUnsupportedMap)
	}
}

// The watchlist table is created by the schema even though #8 is what fills it: a table
// that appears only once something writes it would make #8 a schema migration.
func TestArchive_WatchlistTableExists(t *testing.T) {
	a := testArchive(t)
	var n int
	if err := a.db.QueryRow(context.Background(), `SELECT count(*) FROM watchlist`).Scan(&n); err != nil {
		t.Fatalf("watchlist table missing: %v", err)
	}
	if n != 0 {
		t.Errorf("a fresh watchlist holds %d rows, want 0", n)
	}
}
