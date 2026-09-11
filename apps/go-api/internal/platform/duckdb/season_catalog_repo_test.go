package duckdb

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func openSeasonCatalogMemDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestLoadSeasonCatalogNames_RoundTripAndCase(t *testing.T) {
	db := openSeasonCatalogMemDB(t)
	if _, err := db.Exec(`CREATE TABLE season_catalog (
        title_slug VARCHAR, season_id VARCHAR, display_name VARCHAR, name_fr VARCHAR,
        season_major INTEGER, season_minor INTEGER, first_seen_at TIMESTAMP, last_fetched_at TIMESTAMP,
        PRIMARY KEY (title_slug, season_id))`); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO season_catalog
        (title_slug, season_id, display_name, name_fr, season_major, season_minor)
        VALUES ('halo_infinite','csrseason12-1','Shadows','Ombres',12,1)`); err != nil {
		t.Fatalf("insert: %v", err)
	}
	names, err := LoadSeasonCatalogNames(context.Background(), db, "halo_infinite")
	if err != nil {
		t.Fatalf("LoadSeasonCatalogNames: %v", err)
	}
	sn, ok := names["csrseason12-1"]
	if !ok || sn.DisplayName != "Shadows" || sn.Major != 12 || sn.Minor != 1 {
		t.Errorf("entry = %+v (ok=%v), want {Shadows,12,1}", sn, ok)
	}
	empty := openSeasonCatalogMemDB(t)
	got, err := LoadSeasonCatalogNames(context.Background(), empty, "halo_infinite")
	if err != nil || len(got) != 0 {
		t.Errorf("missing table: got (%v, %v)", got, err)
	}
}

func TestSeasonSelectorLabel(t *testing.T) {
	names := map[string]SeasonName{
		"csrseason13-2": {DisplayName: "Infinite", Major: 13, Minor: 2},
		"csrseason12-1": {DisplayName: "Shadows", Major: 12, Minor: 1},
		"csrseason0-0":  {DisplayName: "Bootstrap", Major: 0, Minor: 0},
	}
	cases := []struct{ name, locale, seasonID, fallback, want string }{
		{"English name", "fr", "csrseason12-1", "Season 12", "Season 12 · Shadows"},
		{"English name with English locale", "en", "csrseason12-1", "Season 12", "Season 12 · Shadows"},
		{"English fallback", "fr", "csrseason13-2", "Season 13", "Season 13 · Infinite"},
		{"case insensitive", "fr", "CsrSeason12-1", "Season 12", "Season 12 · Shadows"},
		{"missing catalog entry", "fr", "csrseason9-1", "Season 9", "Season 9"},
		{"name without number", "fr", "csrseason0-0", "raw", "Bootstrap"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := SeasonSelectorLabel(c.locale, c.seasonID, names, c.fallback); got != c.want {
				t.Errorf("SeasonSelectorLabel(%q,%q) = %q, want %q", c.locale, c.seasonID, got, c.want)
			}
		})
	}
	if got := SeasonSelectorLabel("fr", "csrseason12-1", nil, "Season 12"); got != "Season 12" {
		t.Errorf("nil map should return fallback, got %q", got)
	}
}

func TestFallbackSeasonLabel(t *testing.T) {
	cases := []struct{ locale, id, want string }{
		{"fr", "csrseason13-2", "Season 13"},
		{"en", "csrseason13-2", "Season 13"},
		{"fr", "garbage", "garbage"},
	}
	for _, c := range cases {
		if got := fallbackSeasonLabel(c.locale, c.id); got != c.want {
			t.Errorf("fallbackSeasonLabel(%q,%q) = %q, want %q", c.locale, c.id, got, c.want)
		}
	}
}
