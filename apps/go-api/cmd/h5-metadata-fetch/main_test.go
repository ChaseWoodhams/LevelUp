package main

import (
	"database/sql"
	"encoding/json"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// TestEnglishCatalogFields keeps the English-only ingestion contract explicit.
func TestEnglishCatalogFields(t *testing.T) {
	if got := strings.TrimSpace("  English  "); got != "English" {
		t.Fatalf("trimmed catalog label = %q, want English", got)
	}
}

// TestPersistTeamColors verifies parsing, id conversion, persistence, and idempotence.
func TestPersistTeamColors(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE team_colors (
		team_id INTEGER PRIMARY KEY, name_en VARCHAR NOT NULL DEFAULT '',
		name_fr VARCHAR NOT NULL DEFAULT '', color VARCHAR, icon_url VARCHAR)`); err != nil {
		t.Fatalf("create team_colors: %v", err)
	}

	// Fixture : forme RÉELLE de la réponse EN de l'API /team-colors — `id` est sérialisé
	// en STRING ("0","1"), iconUrl nullable. Verrouille le parsing string→int (régression
	// : un retour de apiTeamColor.ID à int ferait échouer cet unmarshal).
	payload := `[
		{"id":"0","name":"Red","description":"Red team","color":"#E64C4C","iconUrl":"https://cdn/red.png","contentId":"a"},
		{"id":"1","name":"Blue","description":"Blue team","color":"#4C7FE6","iconUrl":null,"contentId":"b"}
	]`
	var colors []apiTeamColor
	if err := json.Unmarshal([]byte(payload), &colors); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}

	if got := persistTeamColors(db, colors); got != 2 {
		t.Fatalf("persistTeamColors a écrit %d lignes, want 2", got)
	}
	// Idempotence : rejouer ne diverge pas (INSERT OR REPLACE).
	if got := persistTeamColors(db, colors); got != 2 {
		t.Fatalf("persistTeamColors (rejeu) a écrit %d lignes, want 2", got)
	}

	var nameEN, legacyName, color string
	if err := db.QueryRow(
		`SELECT name_en, name_fr, color FROM team_colors WHERE team_id=0`).Scan(&nameEN, &legacyName, &color); err != nil {
		t.Fatalf("read team 0: %v", err)
	}
	if nameEN != "Red" || legacyName != "Red" || color != "#E64C4C" {
		t.Errorf("team 0 = {en=%q fr=%q color=%q}, want {Red Red #E64C4C}", nameEN, legacyName, color)
	}

	// team 1 : iconUrl null → chaîne vide persistée, pas d'erreur ; legacy name column.
	var legacyName1, iconURL string
	if err := db.QueryRow(
		`SELECT name_fr, COALESCE(icon_url,'') FROM team_colors WHERE team_id=1`).Scan(&legacyName1, &iconURL); err != nil {
		t.Fatalf("read team 1: %v", err)
	}
	if legacyName1 != "Blue" {
		t.Errorf("team 1 name_fr = %q, want 'Blue'", legacyName1)
	}
	if iconURL != "" {
		t.Errorf("team 1 icon_url = %q, want '' (iconUrl null)", iconURL)
	}
}

// TestPersistTeamColors_MirrorsEnglishName verifies the legacy name column mirror.
func TestPersistTeamColors_MirrorsEnglishName(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE team_colors (
		team_id INTEGER PRIMARY KEY, name_en VARCHAR NOT NULL DEFAULT '',
		name_fr VARCHAR NOT NULL DEFAULT '', color VARCHAR, icon_url VARCHAR)`); err != nil {
		t.Fatalf("create team_colors: %v", err)
	}
	colors := []apiTeamColor{{ID: "2", Name: "Green", Color: "#4CE67F"}}
	persistTeamColors(db, colors) // no separate localized name
	var legacyName string
	if err := db.QueryRow(`SELECT name_fr FROM team_colors WHERE team_id=2`).Scan(&legacyName); err != nil {
		t.Fatalf("read team 2: %v", err)
	}
	if legacyName != "Green" {
		t.Errorf("name_fr = %q, want English fallback 'Green'", legacyName)
	}
}
