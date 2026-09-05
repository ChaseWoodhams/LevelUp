package main

import (
	"context"
	"path/filepath"
	"testing"

	ddb "levelup/go-api/internal/platform/duckdb"
)

func TestMatches_FinalScoresAndLegacyArchive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "archive.duckdb")
	seedArchive(t, path)
	db, err := ddb.OpenReadWrite(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := db.Exec(ctx, `UPDATE matches SET team0_score = 3, team1_score = 0 WHERE match_id = ?`, streetsID); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	ddb.EvictAndCloseCached(path)
	a, err := openTestArchive(t, path)
	if err != nil {
		t.Fatal(err)
	}
	page, err := a.listMatches(ctx, mustFilter(t, rawFilter{Map: "Streets"}))
	if err != nil {
		t.Fatal(err)
	}
	m := page.Matches[0]
	if m.Team0Score == nil || *m.Team0Score != 3 || m.Team1Score == nil || *m.Team1Score != 0 {
		t.Fatalf("scores = %v / %v", m.Team0Score, m.Team1Score)
	}
	single, err := a.getMatch(ctx, streetsID)
	if err != nil || single.Team0Score == nil || *single.Team0Score != 3 {
		t.Fatalf("single match: %+v, %v", single, err)
	}
	// Recreate an old archive shape without requiring the read-only server to migrate it.
	a.Close()
	db, err = ddb.OpenReadWrite(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, column := range []string{"team0_score", "team1_score"} {
		if _, err := db.Exec(ctx, `ALTER TABLE matches DROP COLUMN `+column); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	ddb.EvictAndCloseCached(path)
	a, err = openTestArchive(t, path)
	if err != nil {
		t.Fatal(err)
	}
	page, err = a.listMatches(ctx, mustFilter(t, rawFilter{Map: "Streets"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Matches) != 1 || page.Matches[0].Team0Score != nil || page.Matches[0].Team1Score != nil {
		t.Fatalf("legacy scores: %+v", page)
	}
}
