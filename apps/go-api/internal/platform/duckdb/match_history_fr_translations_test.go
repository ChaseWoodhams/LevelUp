//go:build integration

// match_history_fr_translations_test.go — integration tests for
// applyMatchHistoryFRTranslations (see thought_log 2026-05-09 root cause P2).
//
// Covers the case where the raw pair_name is empty in DB and
// asset_translations[pair_id] returns the raw EN "Arena:CTF on X" — the helper
// analysis.ResolvePairName must re-normalise it to the mode name. The product is
// English-only: the legacy PairNameFR field carries that English mode name, and the
// French mode_name_tr rows seeded below must never surface.
package duckdb

import (
	"context"
	"database/sql"
	"testing"

	"levelup/go-api/internal/domain"

	_ "github.com/duckdb/duckdb-go/v2"
)

func setupMetadataWithModeTranslations(t *testing.T) *DB {
	t.Helper()
	sqlDB, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	db := newTestDB(sqlDB, ":memory:")

	ctx := context.Background()
	for _, q := range []string{
		`CREATE TABLE mode_name_tr (mode_en VARCHAR, lang VARCHAR, name VARCHAR, PRIMARY KEY (mode_en, lang))`,
		`CREATE TABLE asset_translations (asset_id VARCHAR, asset_type VARCHAR, lang VARCHAR, name VARCHAR, PRIMARY KEY (asset_id, asset_type, lang))`,
	} {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatalf("schema: %v", err)
		}
	}

	// mode_name_tr: legacy French rows the English reader must ignore.
	for _, kv := range [][3]string{
		{"CTF", "fr", "Capture du drapeau"},
		{"Strongholds", "fr", "Bases"},
		{"Slayer", "fr", "Assassin"},
		{"Team Slayer", "fr", "Assassin en équipe"},
		{"Neutral Flag CTF", "fr", "Drapeau neutre"},
	} {
		if _, err := db.Exec(ctx, `INSERT INTO mode_name_tr (mode_en, lang, name) VALUES (?, ?, ?)`, kv[0], kv[1], kv[2]); err != nil {
			t.Fatalf("seed mode_name_tr: %v", err)
		}
	}

	// asset_translations: corrupted pair_id — every language returns the raw EN.
	pairID := "bd1457cc-corrupted-pair"
	for _, lang := range []string{"en-US", "fr", "fr-FR", "de-DE"} {
		_, _ = db.Exec(ctx, `INSERT INTO asset_translations (asset_id, asset_type, lang, name) VALUES (?, ?, ?, ?)`,
			pairID, "pair", lang, "Arena:CTF on Shiro")
	}
	// Healthy pair_id — fr-FR carries a translation.
	pairIDOK := "ok-pair-001"
	_, _ = db.Exec(ctx, `INSERT INTO asset_translations (asset_id, asset_type, lang, name) VALUES (?, ?, ?, ?)`,
		pairIDOK, "pair", "fr-FR", "Capture du drapeau sur Aquarius")
	_, _ = db.Exec(ctx, `INSERT INTO asset_translations (asset_id, asset_type, lang, name) VALUES (?, ?, ?, ?)`,
		pairIDOK, "pair", "en-US", "Arena:CTF on Aquarius")

	return db
}

func TestApplyMatchHistoryFRTranslations_HandlesCorruptedAssetTranslations(t *testing.T) {
	ctx := context.Background()
	meta := setupMetadataWithModeTranslations(t)
	pdb := &PlayerDB{Metadata: meta}

	// Reproduces the bug observed on Chocoboflor:
	corruptedID := "bd1457cc-corrupted-pair"
	pairNameOK := "Arena:CTF on Aquarius"
	pairNameFROK := pairNameOK // COALESCE(NULL, EN)
	emptyPairName := ""
	pairNameFRPlaceholder := "Arena:CTF on Shiro" // what COALESCE returns when pair_name_fr holds EN

	rows := []domain.MatchHistoryRawRow{
		// Case A: raw pair_name present, normalised directly.
		{
			MatchID:    "match-A",
			PairName:   &pairNameOK,
			PairNameFR: &pairNameFROK,
		},
		// Case B (root cause P2): pair_name empty, asset_translations holds the raw EN;
		// the helper must re-normalise and find "CTF".
		{
			MatchID:    "match-B",
			PairName:   &emptyPairName,
			PairNameFR: &pairNameFRPlaceholder,
			PairID:     &corruptedID,
		},
	}

	applyMatchHistoryFRTranslations(ctx, pdb, rows)

	if got := derefString(rows[0].PairNameFR); got != "CTF" {
		t.Errorf("Case A (pair_name OK): PairNameFR = %q, want %q", got, "CTF")
	}
	if got := derefString(rows[1].PairNameFR); got != "CTF" {
		t.Errorf("Case B (corrupted asset): PairNameFR = %q, want %q", got, "CTF")
	}
}
