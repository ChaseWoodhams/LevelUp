//go:build integration

package migrations

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func TestApplyPlaylistSeeds_IsHistoricalNoOp(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	if err := applyPlaylistFRSeeds(db); err != nil {
		t.Fatalf("apply: %v", err)
	}
}

func TestApplyPlaylistSeeds_DoesNotRewriteExistingRows(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE asset_translations (asset_id VARCHAR, asset_type VARCHAR, lang VARCHAR, name VARCHAR)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO asset_translations VALUES ('p','playlist','en-US','Quick Play'), ('p','playlist','fr-FR','legacy')`); err != nil {
		t.Fatal(err)
	}
	if err := applyPlaylistFRSeeds(db); err != nil {
		t.Fatalf("apply: %v", err)
	}
	var got string
	if err := db.QueryRow(`SELECT name FROM asset_translations WHERE asset_id='p' AND lang='fr-FR'`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != "legacy" {
		t.Fatalf("historical row changed: %q", got)
	}
}
