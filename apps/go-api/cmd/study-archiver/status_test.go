package main

// status_test.go — THE HEALTH REPORT (#9).
//
// The numbers are asserted on the REPORT VALUE, not on the rendered text: an assertion
// that parsed the table back would be testing column widths, and would go red the first
// time somebody improved the layout.

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	ddb "levelup/go-api/internal/platform/duckdb"
)

// seededArchive writes a known archive and returns its path. The handle is CLOSED before
// the report is read, which is how the two run in production: a separate `status` process
// beside the archiver.
func seededArchive(t *testing.T, seed func(t *testing.T, a *archive)) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "archive.duckdb")
	a, err := openArchive(path)
	if err != nil {
		t.Fatalf("openArchive: %v", err)
	}
	seed(t, a)
	if err := a.Close(); err != nil {
		t.Fatalf("closing the seeded archive: %v", err)
	}
	return path
}

// record writes one match row with the fields this report actually counts.
func record(t *testing.T, a *archive, matchID, mapName, mode, source string,
	state filmState, skip reason, artifact string) {
	t.Helper()
	rec, _ := sampleRecord()
	rec.MatchID, rec.MapName, rec.Mode, rec.SourceGT = matchID, mapName, mode, source
	rec.State, rec.SkipReason, rec.ArtifactPath = state, skip, artifact
	if artifact == "" {
		rec.BuiltAt, rec.DecoderRev = nil, ""
	}
	if err := a.recordMatch(context.Background(), rec, nil); err != nil {
		t.Fatalf("recording %s: %v", matchID, err)
	}
}

func reportOf(t *testing.T, path string) statusReport {
	t.Helper()
	db, release, err := ddb.OpenReadForQuery(path)
	if err != nil {
		t.Fatalf("opening the archive for reading: %v", err)
	}
	defer release()
	rep, err := readStatus(context.Background(), db, path)
	if err != nil {
		t.Fatalf("readStatus: %v", err)
	}
	return rep
}

func TestStatus_CountsWhatIsArchivedAndWhatWentWrong(t *testing.T) {
	path := seededArchive(t, func(t *testing.T, a *archive) {
		record(t, a, "m-1", "Cliffhanger", "Slayer", "ProOne", stateDownloaded, "", "/a/1.json")
		record(t, a, "m-2", "Cliffhanger", "Slayer", "ProOne", stateDownloaded, "", "/a/2.json")
		record(t, a, "m-3", "Recharge", "Oddball", "ProTwo", stateDownloaded, "", "/a/3.json")
		// Not archived, for three DIFFERENT reasons - the distinction this report exists
		// to show.
		record(t, a, "m-4", "Recharge", "Slayer", "ProOne", stateFailed, skipNoTracks, "")
		record(t, a, "m-5", "Recharge", "Slayer", "ProOne", stateFailed, skipBuildFailed, "")
		record(t, a, "m-6", "Streets", "Slayer", "ProTwo", stateExpired, skipFilmAbsent, "")
		record(t, a, "m-7", "Streets", "Slayer", "ProTwo", stateDownloaded, skipUnsupportedMap, "")
		if err := a.rememberWatched(context.Background(), "ProOne", "1111"); err != nil {
			t.Fatalf("watchlist: %v", err)
		}
		if err := a.markChecked(context.Background(), "ProOne"); err != nil {
			t.Fatalf("markChecked: %v", err)
		}
		// Followed but never yet resolved or checked: the row that proves the report can
		// tell "no xuid yet" from "checked a moment ago".
		if err := a.rememberWatched(context.Background(), "ProTwo", ""); err != nil {
			t.Fatalf("watchlist: %v", err)
		}
	})

	rep := reportOf(t, path)

	if rep.Recorded != 7 || rep.Archived != 3 {
		t.Errorf("recorded/archived = %d/%d, want 7/3", rep.Recorded, rep.Archived)
	}
	// Counted SEPARATELY and never summed: one is rebuildable, the other is gone.
	if rep.Failed != 2 || rep.Expired != 1 {
		t.Errorf("failed/expired = %d/%d, want 2/1", rep.Failed, rep.Expired)
	}
	// Captured-but-unbuilt counts ONLY the unsupported map, not the three archived
	// matches that are also in the `downloaded` state: a reader must not see a backlog
	// of four when there is one.
	if rep.Captured != 1 {
		t.Errorf("captured but unbuilt = %d, want 1 (the unsupported map alone)", rep.Captured)
	}

	// The breakdowns count ARCHIVED matches only: "what have I got to study?".
	wantMap := []countedRow{{"Cliffhanger", 2}, {"Recharge", 1}}
	if !sameCounts(rep.ByMap, wantMap) {
		t.Errorf("by map = %v, want %v", rep.ByMap, wantMap)
	}
	wantMode := []countedRow{{"Slayer", 2}, {"Oddball", 1}}
	if !sameCounts(rep.ByMode, wantMode) {
		t.Errorf("by mode = %v, want %v", rep.ByMode, wantMode)
	}
	wantPlayer := []countedRow{{"ProOne", 2}, {"ProTwo", 1}}
	if !sameCounts(rep.ByPlayer, wantPlayer) {
		t.Errorf("by player = %v, want %v", rep.ByPlayer, wantPlayer)
	}

	// Every unarchived match, under the reason that says what to do about it.
	wantReasons := map[string]int{
		string(skipNoTracks): 1, string(skipBuildFailed): 1,
		string(skipFilmAbsent): 1, string(skipUnsupportedMap): 1,
	}
	if len(rep.UnArchived) != len(wantReasons) {
		t.Fatalf("unarchived reasons = %v, want %d distinct", rep.UnArchived, len(wantReasons))
	}
	for _, row := range rep.UnArchived {
		if wantReasons[row.Label] != row.N {
			t.Errorf("reason %q = %d, want %d", row.Label, row.N, wantReasons[row.Label])
		}
	}

	if len(rep.Watchlist) != 2 {
		t.Fatalf("watchlist = %d rows, want 2", len(rep.Watchlist))
	}
	if rep.Watchlist[0].Gamertag != "ProOne" || rep.Watchlist[0].LastChecked == nil {
		t.Errorf("ProOne = %+v, want a recorded last_checked", rep.Watchlist[0])
	}
	if rep.Watchlist[1].XUID != "" || rep.Watchlist[1].LastChecked != nil {
		t.Errorf("ProTwo = %+v, want an unresolved, never-checked row", rep.Watchlist[1])
	}
}

// A brand-new archive must report zeroes and render, rather than crash on empty results —
// this is the state the report is in the first time anybody runs it.
func TestStatus_EmptyArchiveReportsNothingRatherThanFailing(t *testing.T) {
	path := seededArchive(t, func(*testing.T, *archive) {})
	rep := reportOf(t, path)
	if rep.Recorded != 0 || len(rep.ByMap) != 0 || len(rep.Watchlist) != 0 {
		t.Errorf("a fresh archive reported %+v", rep)
	}
	var buf bytes.Buffer
	if err := rep.render(&buf, time.Now()); err != nil {
		t.Fatalf("rendering an empty report: %v", err)
	}
	if !strings.Contains(buf.String(), "nothing yet") {
		t.Errorf("an empty report does not say so:\n%s", buf.String())
	}
}

// The rendering is checked for the few things a reader depends on: that failed and expired
// appear as DIFFERENT numbers, and that an unresolved player is spelled out rather than
// shown as an empty column that reads like a bug.
func TestStatus_RenderSeparatesFailedFromExpired(t *testing.T) {
	rep := statusReport{
		Path: "data/study/archive.duckdb", Recorded: 7, Archived: 3, Failed: 2, Expired: 1,
		ByMap:      []countedRow{{"Cliffhanger", 2}},
		UnArchived: []countedRow{{string(skipFilmAbsent), 1}},
		Watchlist:  []watchlistStatus{{Gamertag: "ProTwo"}},
	}
	var buf bytes.Buffer
	if err := rep.render(&buf, time.Now()); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := buf.String()
	// Logged so `go test -v` shows the actual layout: the one criterion this ticket has is
	// that a human can read it, and that is not something an assertion can check.
	t.Logf("rendered report:\n%s", out)
	for _, want := range []string{
		"2 failed", "1 expired", "Cliffhanger", string(skipFilmAbsent),
		"(unresolved)", "never",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("the report does not mention %q:\n%s", want, out)
		}
	}
}

// A long tail is summarised rather than scrolled off the screen: the one criterion this
// ticket has is that the report be readable at a glance.
func TestStatus_RenderSummarisesALongTail(t *testing.T) {
	rows := make([]countedRow, 0, maxBreakdownRows+3)
	for i := 0; i < maxBreakdownRows+3; i++ {
		rows = append(rows, countedRow{Label: "Map", N: 1})
	}
	var buf bytes.Buffer
	if err := (statusReport{ByMap: rows}).render(&buf, time.Now()); err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(buf.String(), "and 3 more") {
		t.Errorf("the tail was not summarised:\n%s", buf.String())
	}
}

func sameCounts(got, want []countedRow) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
