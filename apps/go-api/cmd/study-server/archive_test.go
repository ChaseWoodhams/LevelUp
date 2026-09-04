package main

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	ddb "levelup/go-api/internal/platform/duckdb"
)

func TestListMatches_OnlyArchivedNewestFirst(t *testing.T) {
	a := newTestArchive(t)
	got := listIDs(t, a, rawFilter{})
	want := []string{aquariusID, streetsID, cliffhangerID}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ids = %v, want %v (newest first, the unbuilt match absent)", got, want)
	}
}

func TestListMatches_Filters(t *testing.T) {
	a := newTestArchive(t)
	cases := []struct {
		name string
		raw  rawFilter
		want []string
	}{
		// Case-folded: a map picked out of a table and a map typed by hand reach the same row.
		{"map", rawFilter{Map: "cliffhanger"}, []string{cliffhangerID}},
		{"mode", rawFilter{Mode: "ctf"}, []string{streetsID}},
		{"player by gamertag", rawFilter{Player: "rival"}, []string{streetsID, cliffhangerID}},
		{"player by xuid", rawFilter{Player: "2533274800000003"}, []string{streetsID}},
		{"day range", rawFilter{From: "2026-05-19", To: "2026-05-20"}, []string{streetsID, cliffhangerID}},
		{"single day", rawFilter{From: "2026-05-19", To: "2026-05-19"}, []string{cliffhangerID}},
		{"coverage floor", rawFilter{MinCoverage: "0.5"}, []string{cliffhangerID}},
		{"combined", rawFilter{Map: "Streets", Player: "Third"}, []string{streetsID}},
		{"nothing matches", rawFilter{Map: "Recharge"}, []string{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := listIDs(t, a, c.raw); !reflect.DeepEqual(got, c.want) {
				t.Errorf("ids = %v, want %v", got, c.want)
			}
		})
	}
}

// TestListMatches_UnknownCoverageIsNotZero — the Aquarius fixture built an artifact that
// named no lives at all. It is listed (it has an artifact, it can be replayed) with NO
// coverage figure, and a coverage floor of any value leaves it out.
func TestListMatches_UnknownCoverageIsNotZero(t *testing.T) {
	a := newTestArchive(t)
	page, err := a.listMatches(context.Background(), mustFilter(t, rawFilter{Map: "Aquarius"}))
	if err != nil {
		t.Fatalf("listMatches: %v", err)
	}
	if len(page.Matches) != 1 {
		t.Fatalf("matches = %d, want the Aquarius match", len(page.Matches))
	}
	if page.Matches[0].Coverage != nil {
		t.Errorf("coverage = %v, want absent - no lives is not zero coverage", *page.Matches[0].Coverage)
	}
	if got := listIDs(t, a, rawFilter{MinCoverage: "0"}); len(got) != 2 {
		t.Errorf("a floor of 0 selected %v, want only the two matches with a coverage figure", got)
	}
}

func TestListMatches_CoverageIsAFractionOfOne(t *testing.T) {
	a := newTestArchive(t)
	page, err := a.listMatches(context.Background(), mustFilter(t, rawFilter{Map: "Cliffhanger"}))
	if err != nil {
		t.Fatalf("listMatches: %v", err)
	}
	m := page.Matches[0]
	if m.Coverage == nil {
		t.Fatal("coverage absent on a match with 90 named lives out of 105")
	}
	if want := 90.0 / 105.0; *m.Coverage != want {
		t.Errorf("coverage = %v, want %v (0..1, ADR 0006)", *m.Coverage, want)
	}
	if m.NamedLives != 90 || m.TotalLives != 105 {
		t.Errorf("lives = %d/%d, want 90/105 alongside the ratio", m.NamedLives, m.TotalLives)
	}
}

// TestListMatches_TotalIgnoresPaging — the count answers "how many matched", which is what a
// browser needs to page; the rows answer "which ones are on this page".
func TestListMatches_TotalIgnoresPaging(t *testing.T) {
	a := newTestArchive(t)
	f := mustFilter(t, rawFilter{Limit: 1, Offset: 1})
	page, err := a.listMatches(context.Background(), f)
	if err != nil {
		t.Fatalf("listMatches: %v", err)
	}
	if page.Total != 3 {
		t.Errorf("total = %d, want the 3 archived matches", page.Total)
	}
	if len(page.Matches) != 1 || page.Matches[0].MatchID != streetsID {
		t.Errorf("page = %+v, want the second row alone", page.Matches)
	}
}

func TestLookupMatch(t *testing.T) {
	a := newTestArchive(t)
	ctx := context.Background()

	// Either form of the identifier reaches the row: the app speaks full match ids, the
	// film cache and its artifacts speak the short form.
	for _, id := range []string{cliffhangerID, "000d5950"} {
		got, err := a.lookupMatch(ctx, id)
		if err != nil {
			t.Fatalf("lookupMatch(%q): %v", id, err)
		}
		if got.MatchID != cliffhangerID || got.ShortID != "000d5950" {
			t.Errorf("lookupMatch(%q) = %+v", id, got)
		}
	}

	// A match the archiver recorded but never built is NOT servable: it has no artifact.
	if _, err := a.lookupMatch(ctx, unbuiltID); !errors.Is(err, errMatchUnknown) {
		t.Errorf("a match with no artifact must read as unknown, got %v", err)
	}
	if _, err := a.lookupMatch(ctx, "deadbeef"); !errors.Is(err, errMatchUnknown) {
		t.Errorf("an absent match must read as unknown, got %v", err)
	}
}

func TestReadParticipants_RosterShape(t *testing.T) {
	a := newTestArchive(t)
	rows, err := a.readParticipants(context.Background(), cliffhangerID)
	if err != nil {
		t.Fatalf("readParticipants: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("participants = %d, want 2", len(rows))
	}
	first := rows[0]
	if first.XUID != "2533274823110022" || first.Gamertag != "JGtm" {
		t.Errorf("identity = %+v", first)
	}
	// "t{N}" is the team_side the app's own scoreboard publishes, and the shape the copied
	// rosterLogic groups on. A bare 0 or an invented "Eagle" would both break the join.
	if first.TeamSide == nil || *first.TeamSide != "t0" {
		t.Errorf("team_side = %v, want t0", first.TeamSide)
	}
	if rows[1].TeamSide == nil || *rows[1].TeamSide != "t1" {
		t.Errorf("team_side = %v, want t1", rows[1].TeamSide)
	}
	if first.Kills == nil || *first.Kills != 15 || first.Deaths == nil || *first.Deaths != 9 ||
		first.Assists == nil || *first.Assists != 4 {
		t.Errorf("K/D/A = %v/%v/%v, want 15/9/4", first.Kills, first.Deaths, first.Assists)
	}
}

// TestOpenArchive_MissingFile — the state every machine is in before the first capture. The
// message has to name the tool that creates the archive, because DuckDB's own error names
// only the driver and the path.
func TestOpenArchive_MissingFile(t *testing.T) {
	_, err := openArchive(filepath.Join(t.TempDir(), "archive.duckdb"))
	if err == nil {
		t.Fatal("opening an archive that does not exist must fail")
	}
	if !errors.Is(err, errNoArchive) {
		t.Errorf("err = %v, want errNoArchive", err)
	}
}

// TestOpenArchive_WhileAWriterHoldsIt is the ticket's read-only criterion, stated as the
// thing it is actually for: browsing the archive during an hourly capture must work.
//
// A FORCED `OpenReadOnly` WOULD FAIL THIS TEST. DuckDB refuses a read-only handle on a file
// already held read-write, so the open has to go through `OpenReadForQuery`, which borrows
// the existing handle instead — and reads fine on it, because a SELECT does.
func TestOpenArchive_WhileAWriterHoldsIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "archive.duckdb")
	seedArchive(t, path)

	writer, err := ddb.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("standing in for the archiver's writer: %v", err)
	}
	defer func() { _ = writer.Close() }()

	a, err := openArchive(path)
	if err != nil {
		t.Fatalf("opening the archive while it is held read-write: %v", err)
	}
	defer a.Close()

	page, err := a.listMatches(context.Background(), mustFilter(t, rawFilter{}))
	if err != nil {
		t.Fatalf("listing while a writer holds the archive: %v", err)
	}
	if page.Total != 3 {
		t.Errorf("total = %d, want the 3 archived matches", page.Total)
	}
}

func mustFilter(t *testing.T, raw rawFilter) matchFilter {
	t.Helper()
	f, err := parseFilter(raw)
	if err != nil {
		t.Fatalf("parseFilter(%+v): %v", raw, err)
	}
	return f
}
