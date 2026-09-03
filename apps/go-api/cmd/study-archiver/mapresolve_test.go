package main

import (
	"errors"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// testCatalog is a two-map quant-bounds catalogue. Hand-built rather than read from
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

// statsWithMap builds the slice of a match-stats payload the archiver actually reads.
func statsWithMap(name string) map[string]any {
	return map[string]any{
		"MatchInfo": map[string]any{
			"MapVariant": map[string]any{"PublicName": name},
		},
	}
}

func TestResolveMatchMap_SupportedMapCarriesModuleAndBounds(t *testing.T) {
	got, err := resolveMatchMap(statsWithMap("Cliffhanger"), testCatalog())
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
	got, err := resolveMatchMap(statsWithMap("Cliffhanger - Ranked"), testCatalog())
	if err != nil {
		t.Fatalf("resolveMatchMap: %v", err)
	}
	if got.Module != "olympus" {
		t.Errorf("module = %q, want olympus", got.Module)
	}
}

// A map absent from the catalogue must yield the NAMED reason, not a bare error:
// ticket #6 records it and #7 branches on it. Building with another map's bounds
// would be wrong by an arbitrary scale factor and nothing on screen would say so.
func TestResolveMatchMap_UnsupportedMapIsANamedSkip(t *testing.T) {
	_, err := resolveMatchMap(statsWithMap("Forbidden Sands"), testCatalog())

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

func TestResolveMatchMap_StatsWithoutMapNameIsANamedSkip(t *testing.T) {
	for _, tc := range []struct {
		name  string
		stats map[string]any
	}{
		{"no MatchInfo", map[string]any{}},
		{"no MapVariant", map[string]any{"MatchInfo": map[string]any{}}},
		{"empty PublicName", statsWithMap("")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := resolveMatchMap(tc.stats, testCatalog())
			var skip skipError
			if !errors.As(err, &skip) || skip.Reason != skipNoMapInStats {
				t.Errorf("err = %v, want skipError(%s)", err, skipNoMapInStats)
			}
		})
	}
}
