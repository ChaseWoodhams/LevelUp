package main

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	ddb "levelup/go-api/internal/platform/duckdb"
)

// testCatalog is a one-map quant-bounds catalogue. Hand-built rather than read from
// data/titles/**: this test is about the LOOKUP, and reading the versioned catalogue
// would make it fail whenever the game ships a new map.
func testCatalog() *filmdec.MapQuantCatalog {
	return &filmdec.MapQuantCatalog{
		SchemaVersion: filmdec.MapQuantSchemaVersion,
		Maps: map[string]filmdec.MapQuantEntry{
			"cliffhanger": {
				Module:     "olympus",
				Min:        [3]float32{-40, -40, -10},
				Max:        [3]float32{40, 40, 20},
				AxisWidths: [3]uint{13, 13, 14},
			},
		},
	}
}

func TestResolveMatchMap_SupportedMapCarriesModuleAndBounds(t *testing.T) {
	got, err := resolveMatchMap(context.Background(), "Cliffhanger", testCatalog(), nil)
	if err != nil {
		t.Fatalf("resolveMatchMap: %v", err)
	}
	if got.Name != "Cliffhanger" || got.Module != "olympus" {
		t.Errorf("name/module = %q/%q, want Cliffhanger/olympus", got.Name, got.Module)
	}
	if got.Range[0].Min != -40 || got.Range[2].Max != 20 {
		t.Errorf("bounds not carried through: %+v", got.Range)
	}
}

// The API suffixes ranked playlists onto the map name; the geometry is the same map.
// NormalizeMapName owns that rule, and this pins that the archiver goes through it.
func TestResolveMatchMap_RankedSuffixStillResolves(t *testing.T) {
	got, err := resolveMatchMap(context.Background(), "Cliffhanger - Ranked", testCatalog(), nil)
	if err != nil {
		t.Fatalf("resolveMatchMap: %v", err)
	}
	if got.Module != "olympus" {
		t.Errorf("module = %q, want olympus", got.Module)
	}
}

// A map absent from the catalogue must yield the NAMED reason, not a bare error:
// #7 branches on it and the archive row records it. Building with another map's bounds
// would be wrong by an arbitrary scale factor and nothing on screen would say so.
func TestResolveMatchMap_UnsupportedMapIsANamedSkip(t *testing.T) {
	_, err := resolveMatchMap(context.Background(), "Forbidden Sands", testCatalog(), nil)

	var skip skipError
	if !errors.As(err, &skip) {
		t.Fatalf("err = %v, want a skipError", err)
	}
	if skip.Reason != skipUnsupportedMap {
		t.Errorf("reason = %q, want %q", skip.Reason, skipUnsupportedMap)
	}
	if !errors.Is(err, filmdec.ErrUnknownMapBounds) {
		t.Errorf("the catalogue's own sentinel was lost: %v", err)
	}
}

// Stats that named no map at all: a different reason, because nothing says a rebuild
// would ever succeed.
func TestResolveMatchMap_NoMapNameIsANamedSkip(t *testing.T) {
	_, err := resolveMatchMap(context.Background(), "", testCatalog(), nil)
	var skip skipError
	if !errors.As(err, &skip) || skip.Reason != skipNoMapInStats {
		t.Errorf("err = %v, want skipError(%s)", err, skipNoMapInStats)
	}
}

// testMetadataDB builds a throwaway metadata.duckdb carrying one map's translation —
// the shape `sync.LookupAssetCanonicalEN` reads, not a copy of it.
func testMetadataDB(t *testing.T, assetID, name string) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "metadata.duckdb")
	db, err := ddb.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("creating the fixture metadata db: %v", err)
	}
	ctx := context.Background()
	if _, err := db.Exec(ctx, `CREATE TABLE asset_translations (
		asset_id VARCHAR, asset_type VARCHAR, lang VARCHAR, name VARCHAR,
		PRIMARY KEY (asset_id, asset_type, lang))`); err != nil {
		t.Fatalf("creating asset_translations: %v", err)
	}
	if _, err := db.Exec(ctx,
		`INSERT INTO asset_translations VALUES (?, 'map', 'en-US', ?)`, assetID, name); err != nil {
		t.Fatalf("seeding asset_translations: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("closing the fixture metadata db: %v", err)
		}
	})
	return db.SQLDb()
}

// A GUID is what `sync.ExtractRegistry` leaves behind when the stats payload carried no
// display name — confirmed against real captures, where PublicName was absent every
// time. This is the case the metadata fallback exists for.
const cliffhangerAssetID = "c9a5b1b1-0000-4000-8000-000000000001"

func TestResolveMatchMap_RecoversAnAssetIDViaMetadata(t *testing.T) {
	metaDB := testMetadataDB(t, cliffhangerAssetID, "Cliffhanger - Ranked")
	got, err := resolveMatchMap(context.Background(), cliffhangerAssetID, testCatalog(), metaDB)
	if err != nil {
		t.Fatalf("resolveMatchMap: %v", err)
	}
	if got.Name != "Cliffhanger - Ranked" || got.Module != "olympus" {
		t.Errorf("name/module = %q/%q, want the resolved name and olympus", got.Name, got.Module)
	}
}

func TestResolveMatchMap_StillUnsupportedWhenMetadataHasNoRow(t *testing.T) {
	metaDB := testMetadataDB(t, cliffhangerAssetID, "Cliffhanger - Ranked")
	_, err := resolveMatchMap(context.Background(), "some-other-guid", testCatalog(), metaDB)

	var skip skipError
	if !errors.As(err, &skip) || skip.Reason != skipUnsupportedMap {
		t.Errorf("err = %v, want skipError(%s) - metadata has no row for this id", err, skipUnsupportedMap)
	}
}

// The metadata catalogue can resolve a NAME the quant-bounds catalogue itself still does
// not carry (a map with a real display name but no reconstructed bounds yet): the
// fallback must not paper over that with the wrong reason.
func TestResolveMatchMap_MetadataResolvesButBoundsStillMissing(t *testing.T) {
	const otherAssetID = "d9a5b1b1-0000-4000-8000-000000000002"
	metaDB := testMetadataDB(t, otherAssetID, "Forbidden Sands")
	_, err := resolveMatchMap(context.Background(), otherAssetID, testCatalog(), metaDB)

	var skip skipError
	if !errors.As(err, &skip) || skip.Reason != skipUnsupportedMap {
		t.Errorf("err = %v, want skipError(%s)", err, skipUnsupportedMap)
	}
	if !errors.Is(err, filmdec.ErrUnknownMapBounds) {
		t.Errorf("the catalogue's own sentinel was lost: %v", err)
	}
}

func TestResolveMatchMap_NilMetadataDBIsFine(t *testing.T) {
	_, err := resolveMatchMap(context.Background(), cliffhangerAssetID, testCatalog(), nil)
	var skip skipError
	if !errors.As(err, &skip) || skip.Reason != skipUnsupportedMap {
		t.Errorf("err = %v, want skipError(%s) - no metadata db to fall back to", err, skipUnsupportedMap)
	}
}
