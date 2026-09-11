//go:build integration

package migrations

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func TestReconcileMetadataSeeds_EnglishCatalogOnly(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	for _, q := range []string{
		`CREATE TABLE mode_name_tr (mode_en VARCHAR NOT NULL, lang VARCHAR NOT NULL, name VARCHAR NOT NULL, PRIMARY KEY (mode_en, lang))`,
		`CREATE TABLE asset_translations (asset_id VARCHAR NOT NULL, asset_type VARCHAR NOT NULL, lang VARCHAR NOT NULL, name VARCHAR, PRIMARY KEY (asset_id, asset_type, lang))`,
		`INSERT INTO asset_translations VALUES ('quick','playlist','en-US','Quick Play')`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("setup: %v", err)
		}
	}
	if err := ReconcileMetadataSeeds(db); err != nil {
		t.Fatalf("ReconcileMetadataSeeds: %v", err)
	}
	for _, mode := range []string{"Slayer", "Team Slayer", "Neutral Flag CTF"} {
		var got string
		if err := db.QueryRow(`SELECT name FROM mode_name_tr WHERE mode_en = ? AND lang = 'en'`, mode).Scan(&got); err != nil {
			t.Fatalf("mode %q absent: %v", mode, err)
		}
		if got == "" {
			t.Errorf("mode %q is empty", mode)
		}
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM asset_translations WHERE lang <> 'en-US'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("unexpected non-English rows: %d", count)
	}
}
