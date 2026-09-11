package migrations

// mode_playlist_fr.go contains the historical mode and playlist migration
// hooks. The migration names remain in migration.canonicalOrder for upgrade
// compatibility, while new catalog rows are English-only.

import (
	"database/sql"
	"fmt"

	"levelup/go-api/internal/migration"
)

// Canonical mode keys used by mode_name_tr.
const (
	modeAttrition  = "Attrition"
	modeExtraction = "Extraction"
	modeOddball    = "Oddball"
)

// Mode labels shared by the mode catalog and the historical playlist hook.
const (
	modeTeamSlayer  = "Team Slayer"
	modeTeamSnipers = "Team Snipers"
)

// applyModeNameTr creates and populates mode_name_tr with English catalog rows.
func applyModeNameTr(db *sql.DB) error {
	if _, err := db.ExecContext(migration.BootCtx(), `
		CREATE TABLE IF NOT EXISTS mode_name_tr (
			mode_en VARCHAR NOT NULL,
			lang    VARCHAR NOT NULL,
			name    VARCHAR NOT NULL,
			PRIMARY KEY (mode_en, lang)
		)
	`); err != nil {
		return err
	}

	type modeRow struct{ modeEN, lang, name string }
	rows := []modeRow{
		{"Assault", "en", "Assault"},
		{modeAttrition, "en", modeAttrition},
		{"CTF", "en", "CTF"},
		{"CTF 3 Captures", "en", "CTF (3 Captures)"},
		{"Escalation Slayer", "en", "Escalation Slayer"},
		{modeExtraction, "en", modeExtraction},
		{"FFA Slayer", "en", "FFA Slayer"},
		{"Fiesta CTF", "en", "Fiesta CTF"},
		{"Fiesta Slayer", "en", "Fiesta Slayer"},
		{"Fiesta Total Control", "en", "Fiesta Total Control"},
		{"Heroic KOTH", "en", "King of the Hill (Heroic)"},
		{"Heroic King of the Hill", "en", "King of the Hill (Heroic)"},
		{"King of the Hill", "en", "King of the Hill"},
		{"Land Grab", "en", "Land Grab"},
		{"Legendary King of the Hill", "en", "King of the Hill (Legendary)"},
		{"Neutral Bomb", "en", "Neutral Bomb"},
		{"Neutral Bomb Squad", "en", "Neutral Bomb Squad"},
		{"Neutral Flag CTF", "en", "Neutral Flag CTF"},
		{modeOddball, "en", modeOddball},
		{"One Bomb", "en", "One Bomb"},
		{"One Flag CTF", "en", "One Flag CTF"},
		{"Sentry Defense", "en", "Sentry Defense"},
		{"Shotty Snipe Slayer FFA", "en", "Shotty Snipers FFA"},
		{"Shotty Snipes Slayer", "en", "Shotty Snipers"},
		{"Slayer", "en", "Slayer"},
		{"Stockpile", "en", "Stockpile"},
		{"Strongholds", "en", "Strongholds"},
		{modeTeamSlayer, "en", modeTeamSlayer},
		{modeTeamSnipers, "en", modeTeamSnipers},
		{"Total Control", "en", "Total Control"},
		{"VIP", "en", "VIP"},
	}

	for _, r := range rows {
		if _, err := db.ExecContext(migration.BootCtx(),
			"INSERT OR IGNORE INTO mode_name_tr (mode_en, lang, name) VALUES (?, ?, ?)",
			r.modeEN, r.lang, r.name,
		); err != nil {
			return err
		}
	}
	return nil
}

// applyPlaylistFRSeeds is retained as a historical migration hook. Existing
// translation rows remain readable for upgrades, while new databases receive
// English catalog rows only.
func applyPlaylistFRSeeds(_ *sql.DB) error { return nil }

// ReconcileMetadataSeeds reapplies the idempotent metadata hooks after the
// migration runner. Historical migration IDs and the no-op playlist hook stay
// in place so existing databases can upgrade without creating new non-English
// rows.
func ReconcileMetadataSeeds(db *sql.DB) error {
	if db == nil {
		return nil
	}
	if err := applyModeNameTr(db); err != nil {
		return fmt.Errorf("reconcile mode_name_tr: %w", err)
	}
	if err := applyPlaylistFRSeeds(db); err != nil {
		return fmt.Errorf("reconcile historical playlist seed: %w", err)
	}
	return nil
}
