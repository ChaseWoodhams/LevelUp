//go:build integration

// home_repo_fr_translations_test.go — integration tests for
// enrichHomeMatchTranslations and EnrichCanonicalAssetTranslations.
//
// Covers the same corruption scenarios as match_history_fr_translations_test.go
// (asset_translations[fr-FR] == raw EN) on the Home/canonical path, plus playlist
// resolution from asset_translations. The product is English-only: the legacy *FR
// fields mirror the resolved English name, and the French rows seeded below must
// never surface.
package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/legacymatch"
)

// seedHomeFRFixtures fills mode_name_tr + asset_translations in a test meta DB,
// including French rows that the English reader must ignore. Reuses
// newHomeRepoTestMetaDB (same package).
func seedHomeFRFixtures(t *testing.T, meta *DB) {
	t.Helper()
	ctx := context.Background()

	for _, row := range [][3]string{
		{"CTF", "fr", "Capture du drapeau"},
		{"Strongholds", "fr", "Bases"},
		{"Slayer", "fr", "Assassin"},
	} {
		if _, err := meta.Exec(ctx,
			`INSERT INTO mode_name_tr (lang, mode_en, name) VALUES (?, ?, ?)`,
			row[1], row[0], row[2],
		); err != nil {
			t.Fatalf("seed mode_name_tr: %v", err)
		}
	}

	// Corrupted pair: every language returns the raw EN.
	for _, lang := range []string{"en-US", "fr-FR", "fr"} {
		if _, err := meta.Exec(ctx,
			`INSERT INTO asset_translations VALUES (?, 'pair', ?, ?, '', now())`,
			"corrupted-pair-id", lang, "Arena:CTF on Shiro",
		); err != nil {
			t.Fatalf("seed corrupted pair (%s): %v", lang, err)
		}
	}

	// Quick Play playlist: correct fr-FR and en-US rows.
	for _, row := range [][3]string{
		{"qp-playlist-id", "en-US", "Quick Play"},
		{"qp-playlist-id", "fr-FR", "Partie rapide"},
	} {
		if _, err := meta.Exec(ctx,
			`INSERT INTO asset_translations VALUES (?, 'playlist', ?, ?, '', now())`,
			row[0], row[1], row[2],
		); err != nil {
			t.Fatalf("seed playlist: %v", err)
		}
	}

	// Aquarius map: same name in both languages.
	for _, row := range [][3]string{
		{"aquarius-map-id", "en-US", "Aquarius"},
		{"aquarius-map-id", "fr-FR", "Aquarius"},
	} {
		if _, err := meta.Exec(ctx,
			`INSERT INTO asset_translations VALUES (?, 'map', ?, ?, '', now())`,
			row[0], row[1], row[2],
		); err != nil {
			t.Fatalf("seed map: %v", err)
		}
	}
}

// ── enrichHomeMatchTranslations ───────────────────────────────────────────────

// TestEnrichHomeMatchTranslations_CorruptedPairName: pair_name stored, but the legacy
// pair_name_fr holds the raw EN pair ("Arena:CTF on Shiro") — the home path must
// normalise it to the mode name "CTF", never keep the raw pair string.
func TestEnrichHomeMatchTranslations_CorruptedPairName(t *testing.T) {
	meta := newHomeRepoTestMetaDB(t)
	seedHomeFRFixtures(t, meta)

	repo := NewHomeRepo(&PlayerDB{Metadata: meta})
	matches := []legacymatch.HomeMatchRow{
		{
			MatchID:    "m1",
			PairID:     "corrupted-pair-id",
			PairName:   "Arena:CTF on Shiro",
			PairNameFR: "Arena:CTF on Shiro",
		},
	}
	repo.enrichHomeMatchTranslations(context.Background(), matches)

	if got := matches[0].PairNameFR; got != "CTF" {
		t.Errorf("PairNameFR = %q, want %q", got, "CTF")
	}
}

// TestEnrichHomeMatchTranslations_NullPairNameResolvedViaPairID: pair_name NULL in DB
// (legacy pair_name_fr NULL too) — home must resolve via pair_id →
// asset_translations → re-normalise to the mode name.
func TestEnrichHomeMatchTranslations_NullPairNameResolvedViaPairID(t *testing.T) {
	meta := newHomeRepoTestMetaDB(t)
	seedHomeFRFixtures(t, meta)

	repo := NewHomeRepo(&PlayerDB{Metadata: meta})
	matches := []legacymatch.HomeMatchRow{
		{
			MatchID:    "m2",
			PairID:     "corrupted-pair-id",
			PairName:   "", // NULL in DB
			PairNameFR: "", // NULL in DB
		},
	}
	repo.enrichHomeMatchTranslations(context.Background(), matches)

	if got := matches[0].PairNameFR; got != "CTF" {
		t.Errorf("PairNameFR = %q, want %q", got, "CTF")
	}
}

// TestEnrichHomeMatchTranslations_PlaylistEnglish: the playlist keeps its English name
// even though a fr-FR translation exists.
func TestEnrichHomeMatchTranslations_PlaylistEnglish(t *testing.T) {
	meta := newHomeRepoTestMetaDB(t)
	seedHomeFRFixtures(t, meta)

	repo := NewHomeRepo(&PlayerDB{Metadata: meta})
	matches := []legacymatch.HomeMatchRow{
		{
			MatchID:        "m3",
			PlaylistID:     "qp-playlist-id",
			PlaylistName:   "Quick Play",
			PlaylistNameFR: "Quick Play",
		},
	}
	repo.enrichHomeMatchTranslations(context.Background(), matches)

	if got := matches[0].PlaylistNameFR; got != "Quick Play" {
		t.Errorf("PlaylistNameFR = %q, want %q", got, "Quick Play")
	}
}

// TestEnrichHomeMatchTranslations_StaleFrenchLabelReplaced: a French label left in the
// legacy column ("Assassin") is replaced by the English mode name.
func TestEnrichHomeMatchTranslations_StaleFrenchLabelReplaced(t *testing.T) {
	meta := newHomeRepoTestMetaDB(t)
	seedHomeFRFixtures(t, meta)

	repo := NewHomeRepo(&PlayerDB{Metadata: meta})
	matches := []legacymatch.HomeMatchRow{
		{
			MatchID:    "m4",
			PairID:     "corrupted-pair-id",
			PairName:   "Arena:Slayer on Live Fire",
			PairNameFR: "Assassin",
		},
	}
	repo.enrichHomeMatchTranslations(context.Background(), matches)

	if got := matches[0].PairNameFR; got != "Slayer" {
		t.Errorf("PairNameFR = %q, want %q", got, "Slayer")
	}
}

// ── EnrichCanonicalAssetTranslations ─────────────────────────────────────────

// TestEnrichCanonicalAssetTranslations_PairModeFromCorruptedPair: PairMode whose
// DefaultLabel is the raw corrupted EN pair — the resolved label is the normalised
// mode name, never the French translation.
func TestEnrichCanonicalAssetTranslations_PairModeFromCorruptedPair(t *testing.T) {
	meta := newHomeRepoTestMetaDB(t)
	seedHomeFRFixtures(t, meta)

	repo := NewHomeRepo(&PlayerDB{Metadata: meta})
	rows := []canonical.PlayerMatchRow{
		{
			Summary: canonical.MatchSummary{
				MatchID: "m5",
				PairMode: &canonical.AssetReference{
					ID:           "corrupted-pair-id",
					DefaultLabel: "Arena:CTF on Shiro",
					Labels:       map[string]string{},
				},
			},
		},
	}
	if err := repo.EnrichCanonicalAssetTranslations(context.Background(), rows); err != nil {
		t.Fatalf("EnrichCanonicalAssetTranslations: %v", err)
	}

	got := rows[0].Summary.PairMode.Labels["fr"]
	if got != "CTF" {
		t.Errorf("Labels[fr] = %q, want %q", got, "CTF")
	}
}

// TestEnrichCanonicalAssetTranslations_PlaylistEnglish: a playlist with a fr-FR row in
// asset_translations still resolves to its English name.
func TestEnrichCanonicalAssetTranslations_PlaylistEnglish(t *testing.T) {
	meta := newHomeRepoTestMetaDB(t)
	seedHomeFRFixtures(t, meta)

	repo := NewHomeRepo(&PlayerDB{Metadata: meta})
	rows := []canonical.PlayerMatchRow{
		{
			Summary: canonical.MatchSummary{
				MatchID: "m6",
				Playlist: &canonical.AssetReference{
					ID:           "qp-playlist-id",
					DefaultLabel: "Quick Play",
					Labels:       map[string]string{},
				},
			},
		},
	}
	if err := repo.EnrichCanonicalAssetTranslations(context.Background(), rows); err != nil {
		t.Fatalf("EnrichCanonicalAssetTranslations: %v", err)
	}

	got := rows[0].Summary.Playlist.Labels["fr"]
	if got != "Quick Play" {
		t.Errorf("Labels[fr] = %q, want %q", got, "Quick Play")
	}
}

// TestEnrichCanonicalAssetTranslations_NilMetadataNoOp: without a metadata DB the
// function returns nil and leaves the rows untouched.
func TestEnrichCanonicalAssetTranslations_NilMetadataNoOp(t *testing.T) {
	repo := NewHomeRepo(&PlayerDB{}) // Metadata = nil
	rows := []canonical.PlayerMatchRow{
		{
			Summary: canonical.MatchSummary{
				MatchID: "m7",
				PairMode: &canonical.AssetReference{
					ID:           "some-pair",
					DefaultLabel: "Arena:CTF on X",
					Labels:       map[string]string{},
				},
			},
		},
	}
	if err := repo.EnrichCanonicalAssetTranslations(context.Background(), rows); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := rows[0].Summary.PairMode.Labels["fr"]; got != "" {
		t.Errorf("Labels[fr] = %q, want empty (nil metadata must be a no-op)", got)
	}
}
