//go:build integration

// halo5_commendation_defs_test.go — in-memory (DuckDB :memory:) test of reading the
// commendation_definitions catalogue (h5 metadata). NEVER touches the real h5
// metadata.
//
// Run: go test -tags=integration ./internal/platform/duckdb/ -run Halo5Commendation

package halo5

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func seedCommendationDefs(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	// The legacy name_fr / description_fr columns stay in the schema (read-compatible
	// databases still carry them); the reader must ignore them.
	if _, err := db.ExecContext(ctx, `CREATE TABLE commendation_definitions (
		commendation_id   VARCHAR PRIMARY KEY,
		name_en           VARCHAR NOT NULL,
		name_fr           VARCHAR NOT NULL,
		description_en    VARCHAR DEFAULT '',
		description_fr    VARCHAR DEFAULT '',
		commendation_type VARCHAR,
		category          VARCHAR,
		icon_url          VARCHAR,
		tier_targets      VARCHAR
	)`); err != nil {
		t.Fatalf("ddl: %v", err)
	}
	rows := []struct {
		id, nameEN, nameFR, icon string
	}{
		{"uuid-1", "Spartan Slayer", "Tueur de Spartans", "https://cdn/1.png"},
		{"uuid-2", "Headshot Honcho", "", "https://cdn/2.png"},
		{"uuid-3", "No Icon", "Sans Icône", ""}, // empty icon
	}
	for _, r := range rows {
		if _, err := db.ExecContext(ctx, `INSERT INTO commendation_definitions
			(commendation_id, name_en, name_fr, commendation_type, category, icon_url)
			VALUES (?,?,?,?,?,?)`, r.id, r.nameEN, r.nameFR, "Progressive", "MULTIPLAYER", r.icon); err != nil {
			t.Fatalf("insert %s: %v", r.id, err)
		}
	}
}

func TestHalo5CommendationDefs_LookupResolvesNameIconFallback(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	seedCommendationDefs(t, db)

	src := NewHalo5CommendationDefSource(db)
	// uuid-2 requested twice (dedup) + uuid-unknown (absent from the result).
	got, err := src.LookupCommendations(context.Background(),
		[]string{"uuid-1", "uuid-2", "uuid-2", "uuid-3", "uuid-unknown", ""})
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("result = %d, want 3 (uuid-1,2,3; unknown and empty excluded) — %+v", len(got), got)
	}
	// name_en is served even when a legacy name_fr is present.
	if got["uuid-1"].Name != "Spartan Slayer" {
		t.Errorf("uuid-1 name = %q, want 'Spartan Slayer'", got["uuid-1"].Name)
	}
	if got["uuid-1"].IconURL != "https://cdn/1.png" {
		t.Errorf("uuid-1 icon = %q", got["uuid-1"].IconURL)
	}
	if got["uuid-2"].Name != "Headshot Honcho" {
		t.Errorf("uuid-2 name = %q, want 'Headshot Honcho'", got["uuid-2"].Name)
	}
	// Empty icon → empty string (the adapter will not set an IconURL).
	if got["uuid-3"].IconURL != "" {
		t.Errorf("uuid-3 icon = %q, want empty", got["uuid-3"].IconURL)
	}
	if _, ok := got["uuid-unknown"]; ok {
		t.Error("uuid-unknown must not be in the result")
	}
}

func TestHalo5CommendationDefs_NilAndEmpty(t *testing.T) {
	// nil meta → empty map, no error.
	nilSrc := NewHalo5CommendationDefSource(nil)
	got, err := nilSrc.LookupCommendations(context.Background(), []string{"x"})
	if err != nil || len(got) != 0 {
		t.Errorf("nil meta: got=%v err=%v, want an empty map and no error", got, err)
	}
	// empty ids → empty map.
	db, _ := sql.Open("duckdb", ":memory:")
	defer db.Close()
	seedCommendationDefs(t, db)
	src := NewHalo5CommendationDefSource(db)
	got, err = src.LookupCommendations(context.Background(), nil)
	if err != nil || len(got) != 0 {
		t.Errorf("empty ids: got=%v err=%v, want an empty map", got, err)
	}
}
