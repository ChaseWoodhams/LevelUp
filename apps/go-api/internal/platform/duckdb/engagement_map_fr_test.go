// Package duckdb — engagement_map_fr_test.go : test interne de resolveMapName
// (resolution du nom de map FR via metadata.asset_translations, car
// match_registry.map_name_fr est systematiquement NULL).
//
// Test interne (package duckdb) pour acceder a la methode non exportee +
// construire un EngagementScoreRepo minimal avec seulement Metadata cable.
package duckdb

import (
	"context"
	"testing"
)

func TestResolveMapName(t *testing.T) {
	meta, err := OpenReadWrite(":memory:")
	if err != nil {
		t.Fatalf("OpenReadWrite metadata: %v", err)
	}
	t.Cleanup(func() { _ = meta.Close() })

	ctx := context.Background()
	ddl := []string{
		`CREATE TABLE asset_translations (
			asset_id    VARCHAR,
			asset_type  VARCHAR,
			lang        VARCHAR,
			name        VARCHAR,
			description VARCHAR,
			fetched_at  TIMESTAMP
		)`,
		`INSERT INTO asset_translations VALUES ('648ae7aa','map','en-US','The Pit','',now())`,
		`INSERT INTO asset_translations VALUES ('33c0766c','map','en-US','Aquarius','',now())`,
		`INSERT INTO asset_translations VALUES ('ffff0001','map','en','Fallback','',now())`,
		`INSERT INTO asset_translations VALUES ('648ae7aa','playlist','en-US','Do Not Use','',now())`,
	}
	for _, s := range ddl {
		if _, err := meta.Exec(ctx, s); err != nil {
			t.Fatalf("seed %q: %v", s, err)
		}
	}

	repo := &EngagementScoreRepo{pdb: &PlayerDB{Metadata: meta}}

	cases := []struct {
		name      string
		mapID     string
		wantName  string
		wantFound bool
	}{
		{"English name", "648ae7aa", "The Pit", true},
		{"second English name", "33c0766c", "Aquarius", true},
		{"fallback English lang", "ffff0001", "Fallback", true},
		{"unknown map id", "deadbeef", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := repo.resolveMapName(ctx, tc.mapID)
			if ok != tc.wantFound {
				t.Fatalf("found = %v, want %v (got %q)", ok, tc.wantFound, got)
			}
			if ok && got != tc.wantName {
				t.Errorf("name = %q, want %q", got, tc.wantName)
			}
		})
	}
}

// Metadata nil -> best-effort ("", false), jamais de panic.
func TestResolveMapName_NilMetadata(t *testing.T) {
	repo := &EngagementScoreRepo{pdb: &PlayerDB{}}
	if name, ok := repo.resolveMapName(context.Background(), "648ae7aa"); ok || name != "" {
		t.Errorf("expected empty result without metadata, got (%q, %v)", name, ok)
	}
}
