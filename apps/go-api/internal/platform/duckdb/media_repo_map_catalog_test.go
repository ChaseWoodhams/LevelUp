//go:build integration

// media_repo_map_catalog_test.go — map name resolution through maps_catalog
// (loadMapCatalogNames + enrichMediaMapTranslations) and the anti-GUID guard.
//
// Covers the "map = GUID" bug (Cliffhanger / 5324364b): an unenriched match stores
// the map asset id in match_registry.map_name. It must be resolved to the
// catalogue's canonical English name, or hidden — never shown as a raw UUID.
//
// CGO required (DuckDB driver) → integration tag.
package duckdb

import (
	"context"
	"database/sql"
	"testing"

	"levelup/go-api/internal/domain"

	_ "github.com/duckdb/duckdb-go/v2"
)

func setupMetadataWithMapCatalog(t *testing.T) *DB {
	t.Helper()
	sqlDB, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	db := newTestDB(sqlDB, ":memory:")
	ctx := context.Background()
	for _, q := range []string{
		`CREATE TABLE maps_catalog (title_slug VARCHAR, map_asset_id VARCHAR, name_canonical VARCHAR, image_url VARCHAR)`,
		`CREATE TABLE asset_translations (asset_id VARCHAR, asset_type VARCHAR, lang VARCHAR, name VARCHAR, PRIMARY KEY (asset_id, asset_type, lang))`,
		// Cliffhanger: in the catalogue, plus a legacy French translation row that must
		// no longer be read.
		`INSERT INTO maps_catalog VALUES ('halo_infinite','5324364b-cliff','Cliffhanger','/static/maps/halo_infinite/Cliffhanger.jpg')`,
		`INSERT INTO asset_translations VALUES ('5324364b-cliff','map','fr-FR','Dévissage')`,
		// Domicile: in the catalogue, no translation row.
		`INSERT INTO maps_catalog VALUES ('halo_infinite','921aebb1-dom','Domicile',NULL)`,
	} {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatalf("setup %q: %v", q, err)
		}
	}
	return db
}

func TestEnrichMediaMapTranslations_ResolvesViaCatalog(t *testing.T) {
	meta := setupMetadataWithMapCatalog(t)
	repo := NewMediaRepo(&PlayerDB{Metadata: meta})

	rows := []domain.MediaFileRow{
		// Unenriched match: map_id AND map_name are the raw asset id → canonical name.
		{MapID: strPtr("5324364b-cliff"), MapName: strPtr("5324364b-cliff")},
		// Already resolved → canonical name.
		{MapID: strPtr("921aebb1-dom"), MapName: strPtr("Domicile")},
		// Unknown GUID (absent from the catalogue), no map_id → hidden, never the GUID.
		{MapID: nil, MapName: strPtr("deadbeef-0000-0000-0000-000000000000")},
		// A proper name outside the catalogue → unchanged.
		{MapID: nil, MapName: strPtr("Aquarius")},
	}
	repo.enrichMediaMapTranslations(context.Background(), rows)

	assertMapName(t, "row0 (catalogue, not the French translation)", rows[0].MapName, "Cliffhanger")
	assertMapName(t, "row1 (name_canonical)", rows[1].MapName, "Domicile")
	if rows[2].MapName != nil {
		t.Errorf("row2: an unknown GUID must be hidden, got %q", *rows[2].MapName)
	}
	assertMapName(t, "row3 (proper name unchanged)", rows[3].MapName, "Aquarius")
}

func TestLoadMapCatalogNames_UsesCanonicalEnglishName(t *testing.T) {
	meta := setupMetadataWithMapCatalog(t)
	repo := NewMediaRepo(&PlayerDB{Metadata: meta})

	got := repo.loadMapCatalogNames(context.Background(), []string{"5324364b-cliff", "921aebb1-dom", "inconnu"})
	if got["5324364b-cliff"].en != "Cliffhanger" {
		t.Errorf("Cliffhanger: got %+v, want en=Cliffhanger", got["5324364b-cliff"])
	}
	if got["921aebb1-dom"].en != "Domicile" {
		t.Errorf("Domicile: got %+v, want en=Domicile", got["921aebb1-dom"])
	}
	if _, ok := got["inconnu"]; ok {
		t.Errorf("an unknown id must not be present in the result")
	}
}

func TestLooksLikeAssetID(t *testing.T) {
	cases := map[string]bool{
		"5324364b-39a8-4f93-96a6-b80a1f18ce8a": true,
		"Domicile":                             false,
		"":                                     false,
		"343 Meowlnir":                         false,
		"Cliffhanger.jpg":                      false,
	}
	for in, want := range cases {
		if got := looksLikeAssetID(in); got != want {
			t.Errorf("looksLikeAssetID(%q) = %v, want %v", in, got, want)
		}
	}
}

func assertMapName(t *testing.T, label string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Errorf("%s : got <nil>, want %q", label, want)
		return
	}
	if *got != want {
		t.Errorf("%s : got %q, want %q", label, *got, want)
	}
}
