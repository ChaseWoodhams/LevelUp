package main

// groundtruth_test.go — THE REPLAY AGAINST THE MATCH STATS.
//
// The comparison itself is asserted on its value; the archive tests check that fetch-one and
// rebuild both leave it where `status` reads it, and that a match with nothing to compare stores
// NULL rather than a row of zeros that would read as "compared, and perfect".

import (
	"bytes"
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/replay"
)

// livesDoc builds a document naming `lives[xuid]` lives per player, plus `unnamed` lives nobody
// was named for, with the given life total in its coverage report.
func livesDoc(lives map[string]int, unnamed, livesTotal int) replay.ReplayDocument {
	var tracks []replay.Track
	slot := uint32(1)
	for xuid, n := range lives {
		for i := 0; i < n; i++ {
			tracks = append(tracks, replay.Track{Slot: slot, Team: -1, XUID: xuid, Points: []replay.Point{{T: 0}}})
		}
		slot++
	}
	for i := 0; i < unnamed; i++ {
		tracks = append(tracks, replay.Track{Slot: slot, Team: -1, Points: []replay.Point{{T: 0}}})
	}
	return replay.ReplayDocument{
		SchemaVersion: replay.SchemaVersion,
		Tracks:        tracks,
		Coverage:      &replay.Coverage{Bridge: replay.BridgeHealth{LivesTotal: livesTotal}},
	}
}

func officialPlayer(xuid string, deaths *int) participantRecord {
	return participantRecord{XUID: xuid, Gamertag: "P" + xuid, Deaths: deaths}
}

func TestCompareGroundTruth_ExactReplayIsClean(t *testing.T) {
	doc := livesDoc(map[string]int{"1": 3, "2": 2}, 0, 5)
	gt := compareGroundTruth(doc, []participantRecord{officialPlayer("1", intPtr(2)), officialPlayer("2", intPtr(1))})

	if !gt.Compared || gt.Players != 2 || gt.ExpectedLives != 5 || gt.NamedLives != 5 {
		t.Fatalf("gt = %+v, want compared, 2 players, 5 expected and 5 named lives", gt)
	}
	if gt.OverNamed != 0 || gt.MissingLives != 0 || gt.UnknownNamed != 0 || gt.LivesGap != 0 {
		t.Errorf("gt = %+v, want no over-named, missing, unknown lives and no gap", gt)
	}
}

// A life named beyond a player's deaths + 1 belongs to somebody else: counted, never absorbed
// by another player's shortfall.
func TestCompareGroundTruth_OverNamedIsNotOffsetByMissing(t *testing.T) {
	doc := livesDoc(map[string]int{"1": 4, "2": 1}, 0, 5)
	gt := compareGroundTruth(doc, []participantRecord{officialPlayer("1", intPtr(2)), officialPlayer("2", intPtr(1))})

	if gt.OverNamed != 1 || gt.MissingLives != 1 {
		t.Errorf("over/missing = %d/%d, want 1/1", gt.OverNamed, gt.MissingLives)
	}
	if gt.NamedByPlayer["1"] != 4 || gt.NamedByPlayer["2"] != 1 {
		t.Errorf("named by player = %v, want 1:4 2:1", gt.NamedByPlayer)
	}
}

func TestCompareGroundTruth_UnlistedPlayersAndGap(t *testing.T) {
	// Player 1 died twice (3 lives) and has 2 named; a life is named for "9", whom the stats do
	// not list; 3 lives are unnamed. The document segmented 8 lives against 3 expected.
	doc := livesDoc(map[string]int{"1": 2, "9": 1}, 3, 8)
	gt := compareGroundTruth(doc, []participantRecord{officialPlayer("1", intPtr(2))})

	if gt.MissingLives != 1 || gt.UnknownNamed != 1 || gt.LivesGap != 5 {
		t.Errorf("missing/unknown/gap = %d/%d/%d, want 1/1/+5", gt.MissingLives, gt.UnknownNamed, gt.LivesGap)
	}
}

// A player with no official death count is left out, not guessed at; with none at all, or with
// no coverage report to take the life total from, nothing is compared.
func TestCompareGroundTruth_NothingToCompareAgainst(t *testing.T) {
	doc := livesDoc(map[string]int{"1": 3, "2": 2}, 0, 5)

	partial := compareGroundTruth(doc, []participantRecord{officialPlayer("1", intPtr(2)), officialPlayer("2", nil)})
	if !partial.Compared || partial.Players != 1 || partial.ExpectedLives != 3 || partial.UnknownNamed != 0 {
		t.Errorf("partial = %+v, want player 2 left out (not counted as unlisted)", partial)
	}
	if _, ok := partial.NamedByPlayer["2"]; ok {
		t.Error("player 2 has no death count but was given a replay count")
	}

	if gt := compareGroundTruth(doc, []participantRecord{officialPlayer("1", nil)}); gt.Compared {
		t.Errorf("no death counts: gt = %+v, want not compared", gt)
	}
	noCoverage := doc
	noCoverage.Coverage = nil
	if gt := compareGroundTruth(noCoverage, []participantRecord{officialPlayer("1", intPtr(2))}); gt.Compared {
		t.Errorf("no coverage report: gt = %+v, want not compared", gt)
	}
}

// groundTruthRow reads a match's stored comparison; an invalid NullInt64 is a NULL column.
func groundTruthRow(t *testing.T, a *archive, matchID string) []sql.NullInt64 {
	t.Helper()
	cols := make([]sql.NullInt64, 7)
	if err := a.db.QueryRow(context.Background(), `
        SELECT gt_players, gt_expected_lives, gt_named_lives, gt_over_named,
               gt_missing_lives, gt_unknown_named, gt_lives_gap
        FROM matches WHERE match_id = ?`, matchID).
		Scan(&cols[0], &cols[1], &cols[2], &cols[3], &cols[4], &cols[5], &cols[6]); err != nil {
		t.Fatalf("reading the ground-truth columns: %v", err)
	}
	return cols
}

// replayLivesByPlayer reads each participant's stored replay-named lives, -1 for NULL.
func replayLivesByPlayer(t *testing.T, a *archive, matchID string) map[string]int64 {
	t.Helper()
	rows, err := a.db.SQLDb().QueryContext(context.Background(),
		`SELECT xuid, replay_named_lives FROM participants WHERE match_id = ?`, matchID)
	if err != nil {
		t.Fatalf("reading the participants: %v", err)
	}
	defer func() { _ = rows.Close() }()
	got := map[string]int64{}
	for rows.Next() {
		var (
			xuid  string
			lives sql.NullInt64
		)
		if err := rows.Scan(&xuid, &lives); err != nil {
			t.Fatalf("scanning a participant: %v", err)
		}
		got[xuid] = -1
		if lives.Valid {
			got[xuid] = lives.Int64
		}
	}
	return got
}

func assertColumns(t *testing.T, got []sql.NullInt64, want []int64) {
	t.Helper()
	names := []string{"players", "expected", "named", "over_named", "missing", "unknown", "gap"}
	for i, w := range want {
		if !got[i].Valid || got[i].Int64 != w {
			t.Errorf("gt_%s = %+v, want %d", names[i], got[i], w)
		}
	}
}

// THE REBUILD PATH: the rebuilt replay is compared with the roster the archive already holds,
// with no match-stats read. sampleRecord's JGtm ("1") died 9 times, Rival ("2") 15 times.
func TestRebuild_RecordsTheGroundTruthComparison(t *testing.T) {
	doc := livesDoc(map[string]int{"1": 10, "2": 14}, 1, 27)
	d := offlineDeps(t, stubBuild(&doc))
	captured(t, d, stateFailed, skipNoTracks)

	out, err := rebuildOne(context.Background(), d, testMatchID)
	if err != nil {
		t.Fatalf("rebuildOne: %v", err)
	}
	if !out.GroundTruth.Compared {
		t.Fatal("the rebuild did not compare the replay with the recorded roster")
	}

	assertColumns(t, groundTruthRow(t, d.Archive, testMatchID), []int64{2, 26, 24, 0, 2, 0, 1})
	lives := replayLivesByPlayer(t, d.Archive, testMatchID)
	if lives["1"] != 10 || lives["2"] != 14 {
		t.Errorf("replay lives by player = %v, want 1:10 2:14", lives)
	}
}

// A rebuilt replay with no coverage report is NOT compared, and says so with NULLs.
func TestRebuild_WithoutCoverageStoresNoComparison(t *testing.T) {
	d := offlineDeps(t, stubBuild(nil))
	captured(t, d, stateFailed, skipNoTracks)

	if _, err := rebuildOne(context.Background(), d, testMatchID); err != nil {
		t.Fatalf("rebuildOne: %v", err)
	}
	for i, col := range groundTruthRow(t, d.Archive, testMatchID) {
		if col.Valid {
			t.Errorf("column %d = %d, want NULL for a match that was not compared", i, col.Int64)
		}
	}
	for xuid, lives := range replayLivesByPlayer(t, d.Archive, testMatchID) {
		if lives != -1 {
			t.Errorf("player %s replay lives = %d, want NULL", xuid, lives)
		}
	}
}

// THE FETCH-ONE PATH: the row and the roster are written together, the replay's count beside
// each official death count.
func TestRecordOutcome_StoresReplayLivesBesideOfficialDeaths(t *testing.T) {
	d := deps{Archive: testArchive(t)}
	rec, roster := sampleRecord()
	facts := matchFacts{MapName: rec.MapName, Mode: rec.Mode, Playlist: rec.Playlist, Roster: roster}
	out := outcome{
		MatchID: testMatchID, ShortID: rec.ShortID, MapName: rec.MapName, ArtifactPath: rec.ArtifactPath,
		GroundTruth: groundTruth{
			Compared: true, Players: 2, ExpectedLives: 26, NamedLives: 23, OverNamed: 1,
			MissingLives: 4, LivesGap: -3, NamedByPlayer: map[string]int{"1": 11, "2": 12},
		},
	}
	if err := recordOutcome(context.Background(), d, out, facts); err != nil {
		t.Fatalf("recordOutcome: %v", err)
	}

	assertColumns(t, groundTruthRow(t, d.Archive, testMatchID), []int64{2, 26, 23, 1, 4, 0, -3})
	lives := replayLivesByPlayer(t, d.Archive, testMatchID)
	if lives["1"] != 11 || lives["2"] != 12 {
		t.Errorf("replay lives by player = %v, want 1:11 2:12", lives)
	}
	if roster[0].ReplayNamedLives != nil {
		t.Error("recordOutcome modified the caller's roster")
	}
}

func TestStatus_SummarisesTheGroundTruthComparison(t *testing.T) {
	path := seededArchive(t, func(t *testing.T, a *archive) {
		for _, m := range []struct {
			id, short, artifact string
			gt                  groundTruth
		}{
			{"m-1", "aaaa0001", "/a/1.json", groundTruth{Compared: true, Players: 8,
				ExpectedLives: 100, NamedLives: 90, MissingLives: 10, UnknownNamed: 1, LivesGap: 3}},
			{"m-2", "aaaa0002", "/a/2.json", groundTruth{Compared: true, Players: 8,
				ExpectedLives: 50, NamedLives: 48, OverNamed: 1, MissingLives: 3, LivesGap: -2}},
			// No artifact: its comparison, however stale, is not part of what can be studied.
			{"m-3", "aaaa0003", "", groundTruth{Compared: true, Players: 8,
				ExpectedLives: 999, OverNamed: 5, LivesGap: 40}},
			// Archived but never compared: must not drag the totals towards zero.
			{"m-4", "aaaa0004", "/a/4.json", groundTruth{}},
		} {
			rec, _ := sampleRecord()
			rec.MatchID, rec.ShortID, rec.ArtifactPath = m.id, m.short, m.artifact
			if m.artifact == "" {
				rec.BuiltAt, rec.DecoderRev = nil, ""
			}
			rec.GroundTruth = groundTruthColumns(m.gt)
			if err := a.recordMatch(context.Background(), rec, nil); err != nil {
				t.Fatalf("recording %s: %v", m.id, err)
			}
		}
	})

	s := reportOf(t, path).GroundTruth
	if s.Compared != 2 || s.ExpectedLives != 150 || s.NamedLives != 138 {
		t.Errorf("compared/expected/named = %d/%d/%d, want 2/150/138", s.Compared, s.ExpectedLives, s.NamedLives)
	}
	if s.OverNamed != 1 || s.MissingLives != 13 || s.UnknownNamed != 1 || s.GapMin != -2 || s.GapMax != 3 {
		t.Errorf("summary = %+v, want over 1, missing 13, unknown 1, gap -2..+3", s)
	}
	if len(s.OverNamedMatches) != 1 || s.OverNamedMatches[0].ShortID != "aaaa0002" {
		t.Errorf("over-named matches = %+v, want only aaaa0002", s.OverNamedMatches)
	}
	if len(s.WidestGaps) != 2 || s.WidestGaps[0].ShortID != "aaaa0001" || s.WidestGaps[1].LivesGap != -2 {
		t.Errorf("widest gaps = %+v, want aaaa0001 (+3) then aaaa0002 (-2)", s.WidestGaps)
	}

	var buf bytes.Buffer
	if err := (statusReport{Path: path, GroundTruth: s}).render(&buf, time.Now()); err != nil {
		t.Fatalf("render: %v", err)
	}
	for _, want := range []string{"Replay vs official match stats", "must stay 0", "aaaa0002"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("rendered report lacks %q:\n%s", want, buf.String())
		}
	}
}
