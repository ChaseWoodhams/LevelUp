package main

// helpers_test.go — a throwaway archive, built from THE ARCHIVER'S OWN SCHEMA.
//
// WHY THE DDL IS READ OUT OF cmd/study-archiver RATHER THAN COPIED HERE. This server owns no
// part of the archive's shape: the archiver declares it (archive.go, ticket #6) and is its
// only writer. A fixture with its own CREATE TABLE would be a second declaration of the same
// tables, and the failure of that copy is the quiet kind — the archiver renames a column, the
// server's queries break in production, and every test here stays green against the schema
// the fixture still remembers.
//
// So the fixture EXECUTES the archiver's own const. There is one schema, the tests run
// against it, and a rename that breaks this server breaks these tests in the same commit.
// The coupling is a test-only file path, and when it breaks it says so by name.

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	ddb "levelup/go-api/internal/platform/duckdb"
)

// archiverSchemaSource is where the archive's shape is declared. Test-only coupling.
const (
	archiverSchemaSource = "../study-archiver/archive.go"
	archiverSchemaMarker = "const archiveSchema = `"
)

// archiverSchema lifts the DDL out of the archiver's source.
func archiverSchema(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(archiverSchemaSource)
	if err != nil {
		t.Fatalf("reading the archiver's schema from %s: %v", archiverSchemaSource, err)
	}
	i := strings.Index(string(raw), archiverSchemaMarker)
	if i < 0 {
		t.Fatalf("%s no longer declares %s - the archive's shape moved, "+
			"point this fixture at its new home", archiverSchemaSource, archiverSchemaMarker)
	}
	rest := string(raw)[i+len(archiverSchemaMarker):]
	end := strings.IndexByte(rest, '`')
	if end < 0 {
		t.Fatalf("%s: the archiveSchema literal is not terminated", archiverSchemaSource)
	}
	return rest[:end]
}

// newTestArchive writes a populated archive to a temp directory and returns it opened for
// reading, exactly as a request opens the real one.
func newTestArchive(t *testing.T) *archive {
	t.Helper()
	path := filepath.Join(t.TempDir(), "archive.duckdb")
	seedArchive(t, path)
	a, err := openTestArchive(t, path)
	if err != nil {
		t.Fatalf("openArchive: %v", err)
	}
	return a
}

// newTestSource is the archive as the handler holds it: an address, borrowed per request.
func newTestSource(t *testing.T) *archiveSource {
	t.Helper()
	path := filepath.Join(t.TempDir(), "archive.duckdb")
	seedArchive(t, path)
	src, err := newArchiveSource(path)
	if err != nil {
		t.Fatalf("newArchiveSource: %v", err)
	}
	return src
}

// openTestArchive borrows an archive for the length of one test.
func openTestArchive(t *testing.T, path string) (*archive, error) {
	t.Helper()
	src, err := newArchiveSource(path)
	if err != nil {
		return nil, err
	}
	a, err := src.open(context.Background())
	if err != nil {
		return nil, err
	}
	t.Cleanup(a.Close)
	return a, nil
}

// seedArchive creates the database and fills it with the fixture matches.
func seedArchive(t *testing.T, path string) {
	t.Helper()
	db, err := ddb.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("creating the fixture archive: %v", err)
	}
	ctx := context.Background()
	if _, err := db.Exec(ctx, archiverSchema(t)); err != nil {
		t.Fatalf("applying the archiver's schema: %v", err)
	}
	for _, f := range fixtureMatches() {
		insertFixture(t, db.SQLDb(), f)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("closing the fixture archive: %v", err)
	}
	// Evicted so the server's own open is a genuine read-only open of a file on disk,
	// not a borrow of the writer's cached handle (cf. duckdb.OpenReadForQuery).
	ddb.EvictAndCloseCached(path)
}

// fixtureMatch is one row of the fixture, with its roster.
type fixtureMatch struct {
	MatchID string
	ShortID string
	// Artifact empty is the "recorded but never built" case, and SkipReason says why.
	Artifact   string
	PlayedAt   time.Time
	MapName    string
	Mode       string
	SourceGT   string
	State      string
	SkipReason string
	NamedLives int
	TotalLives int
	Roster     []fixtureParticipant
}

// fixtureParticipant carries POINTERS so a fixture can express the row that matters most and
// is easiest to forget: a player the match stats named no team or no counters for. A fixture
// where every field is populated cannot tell a null apart from a missing key.
type fixtureParticipant struct {
	XUID     string
	Gamertag string
	Team     *int
	Kills    *int
	Deaths   *int
	Assists  *int
}

func ip(n int) *int { return &n }

// Fixture identities, named so an assertion reads as a sentence.
const (
	cliffhangerID = "000d5950-8b0e-4a2c-9a1f-1c2d3e4f5a6b"
	streetsID     = "111a2b3c-4d5e-6f70-8192-a3b4c5d6e7f8"
	unbuiltID     = "222b3c4d-5e6f-7081-92a3-b4c5d6e7f809"
	aquariusID    = "333c4d5e-6f70-8192-a3b4-c5d6e7f8091a"
)

// fixtureMatches spans every case the list endpoint has to tell apart: two archived matches
// on different maps and modes, one recorded but never built, and one built with no lives at
// all (coverage unknown, not zero).
func fixtureMatches() []fixtureMatch {
	day := func(d int) time.Time { return time.Date(2026, 5, d, 20, 15, 0, 0, time.UTC) }
	return []fixtureMatch{
		{
			MatchID: cliffhangerID, ShortID: "000d5950", PlayedAt: day(19),
			MapName: "Cliffhanger", Mode: "Slayer", SourceGT: "JGtm", State: "downloaded",
			Artifact:   "data/cache/replays/halo_infinite/000d5950.json",
			NamedLives: 90, TotalLives: 105,
			Roster: []fixtureParticipant{
				{XUID: "2533274823110022", Gamertag: "JGtm", Team: ip(0), Kills: ip(15), Deaths: ip(9), Assists: ip(4)},
				{XUID: "2533274800000002", Gamertag: "Rival", Team: ip(1), Kills: ip(9), Deaths: ip(15), Assists: ip(2)},
			},
		},
		{
			MatchID: streetsID, ShortID: "111a2b3c", PlayedAt: day(20),
			MapName: "Streets", Mode: "CTF", SourceGT: "Rival", State: "downloaded",
			Artifact:   "data/cache/replays/halo_infinite/111a2b3c.json",
			NamedLives: 40, TotalLives: 100,
			Roster: []fixtureParticipant{
				{XUID: "2533274800000002", Gamertag: "Rival", Team: ip(0), Kills: ip(12), Deaths: ip(11), Assists: ip(5)},
				{XUID: "2533274800000003", Gamertag: "Third", Team: ip(1), Kills: ip(11), Deaths: ip(12), Assists: ip(1)},
			},
		},
		{
			MatchID: unbuiltID, ShortID: "222b3c4d", PlayedAt: day(18),
			MapName: "Cliffhanger", Mode: "Slayer", SourceGT: "JGtm", State: "downloaded",
			SkipReason: "map_without_bounds",
			// A roster the archiver recorded perfectly for a match whose ARTIFACT never
			// built: the case that proves the two are independent.
			Roster: []fixtureParticipant{
				{XUID: "2533274823110022", Gamertag: "JGtm", Team: ip(0), Kills: ip(20), Deaths: ip(5), Assists: ip(3)},
			},
		},
		{
			MatchID: aquariusID, ShortID: "333c4d5e", PlayedAt: day(21),
			MapName: "Aquarius", Mode: "Slayer", SourceGT: "JGtm", State: "downloaded",
			Artifact:   "data/cache/replays/halo_infinite/333c4d5e.json",
			NamedLives: 0, TotalLives: 0,
			Roster: []fixtureParticipant{
				{XUID: "2533274823110022", Gamertag: "JGtm", Team: ip(0), Kills: ip(8), Deaths: ip(8), Assists: ip(8)},
				// The row the payload's shape hangs on: a player the stats named NO team and
				// no counters for. Every field of this one is NULL, and the endpoint has to
				// publish nulls rather than dropping the keys.
				{XUID: "2533274800000009", Gamertag: "Inconnu"},
			},
		},
	}
}

func insertFixture(t *testing.T, db *sql.DB, f fixtureMatch) {
	t.Helper()
	var artifact, skip any
	if f.Artifact != "" {
		artifact = f.Artifact
	}
	if f.SkipReason != "" {
		skip = f.SkipReason
	}
	if _, err := db.Exec(`
        INSERT INTO matches (match_id, short_id, played_at, map_name, map_module, mode,
            playlist, duration_ms, source_gamertag, film_state, skip_reason, artifact_path,
            built_at, decoder_rev, tracks, points, shots, named_lives, total_lives, recorded_at)
        VALUES (?, ?, ?, ?, 'olympus', ?, 'Ranked Arena', 553000, ?, ?, ?, ?,
                ?, 'abc1234', 8, 4200, 519, ?, ?, now())`,
		f.MatchID, f.ShortID, f.PlayedAt, f.MapName, f.Mode, f.SourceGT, f.State,
		skip, artifact, f.PlayedAt, f.NamedLives, f.TotalLives); err != nil {
		t.Fatalf("inserting fixture match %s: %v", f.MatchID, err)
	}
	for _, p := range f.Roster {
		if _, err := db.Exec(`
            INSERT INTO participants (match_id, xuid, gamertag, team, outcome, kills, deaths, assists)
            VALUES (?, ?, ?, ?, 2, ?, ?, ?)`,
			f.MatchID, p.XUID, p.Gamertag, p.Team, p.Kills, p.Deaths, p.Assists); err != nil {
			t.Fatalf("inserting fixture participant %s of %s: %v", p.XUID, f.MatchID, err)
		}
	}
}

// Two archived matches whose ids collide on their first 8 characters. short_id is not a key,
// so this is possible, and the lookup has to resolve it the same way every time.
const (
	sharedShortID   = "abcd1234"
	collisionLowID  = "abcd1234-0000-4000-8000-000000000001"
	collisionHighID = "abcd1234-ffff-4fff-8fff-ffffffffffff"
)

// newCollisionArchive is a SEPARATE archive rather than two more rows in the shared fixture:
// the collision is a property of the lookup, and adding it to the fixture every other test
// counts rows in would have made those tests about it too.
func newCollisionArchive(t *testing.T) *archive {
	t.Helper()
	path := filepath.Join(t.TempDir(), "archive.duckdb")
	db, err := ddb.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("creating the collision archive: %v", err)
	}
	if _, err := db.Exec(context.Background(), archiverSchema(t)); err != nil {
		t.Fatalf("applying the archiver's schema: %v", err)
	}
	played := time.Date(2026, 5, 19, 20, 15, 0, 0, time.UTC)
	for _, id := range []string{collisionHighID, collisionLowID} { // inserted high first
		insertFixture(t, db.SQLDb(), fixtureMatch{
			MatchID: id, ShortID: sharedShortID, PlayedAt: played,
			MapName: "Cliffhanger", Mode: "Slayer", SourceGT: "JGtm", State: "downloaded",
			Artifact:   "data/cache/replays/halo_infinite/" + sharedShortID + ".json",
			NamedLives: 90, TotalLives: 105,
		})
	}
	if err := db.Close(); err != nil {
		t.Fatalf("closing the collision archive: %v", err)
	}
	ddb.EvictAndCloseCached(path)

	a, err := openTestArchive(t, path)
	if err != nil {
		t.Fatalf("openArchive: %v", err)
	}
	return a
}

// listIDs runs a filter and returns the match ids it selected, in order.
func listIDs(t *testing.T, a *archive, raw rawFilter) []string {
	t.Helper()
	f, err := parseFilter(raw)
	if err != nil {
		t.Fatalf("parseFilter(%+v): %v", raw, err)
	}
	page, err := a.listMatches(context.Background(), f)
	if err != nil {
		t.Fatalf("listMatches(%+v): %v", raw, err)
	}
	ids := make([]string, 0, len(page.Matches))
	for _, m := range page.Matches {
		ids = append(ids, m.MatchID)
	}
	return ids
}
