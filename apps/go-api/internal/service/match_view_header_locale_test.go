package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
)

// TestBuildMatchHeader_LocaleAwareLabels — GH-9 : le header Match View servait des
// libellés FR (map/mode/playlist) et une date FR quelle que soit la locale de la
// requête → sous UI EN « Assassin en équipe on Bazaar » / « Partie rapide » /
// « 06 avr. 2026 ». La locale de requête (ctxkeys.Locale) doit choisir EN vs FR.
func TestBuildMatchHeader_LocaleAwareLabels(t *testing.T) {
	start := time.Date(2026, 4, 6, 23, 40, 0, 0, time.UTC)
	meta := &domain.MatchMetaRaw{
		MatchID:        "m1",
		StartTime:      &start,
		MapName:        strPtr("Bazaar"),
		MapNameFR:      strPtr("Bazar"),
		MapNameEN:      strPtr("Bazaar"),
		PairName:       strPtr("Arena:Team Slayer on Bazaar"),
		PairNameFR:     strPtr("Arene : Assassin en equipe"),
		ModeNameFR:     strPtr("Assassin en equipe"),
		PlaylistName:   strPtr("Quick Play"),
		PlaylistNameFR: strPtr("Partie rapide"),
	}

	// UI EN → libellés canoniques EN + date EN.
	hEN := buildMatchHeader(ctxkeys.WithLocale(context.Background(), "en"), "m1", meta, nil, nil, nil, nil, false)
	if hEN.MapUI != "Bazaar" {
		t.Errorf("EN MapUI = %q, want %q", hEN.MapUI, "Bazaar")
	}
	if hEN.ModeUI != "Team Slayer" {
		t.Errorf("EN ModeUI = %q, want %q (pair EN normalise, pas de FR)", hEN.ModeUI, "Team Slayer")
	}
	if hEN.PlaylistLabel != "Quick Play" {
		t.Errorf("EN PlaylistLabel = %q, want %q", hEN.PlaylistLabel, "Quick Play")
	}
	if !strings.Contains(hEN.StartTimeLabel, "Apr") {
		t.Errorf("EN StartTimeLabel = %q, want mois EN (Apr)", hEN.StartTimeLabel)
	}

	// Legacy locale inputs still produce the English-only contract.
	hFR := buildMatchHeader(ctxkeys.WithLocale(context.Background(), "fr"), "m1", meta, nil, nil, nil, nil, false)
	if hFR.MapUI != "Bazaar" {
		t.Errorf("legacy locale MapUI = %q, want %q", hFR.MapUI, "Bazaar")
	}
	if hFR.ModeUI != "Team Slayer" {
		t.Errorf("legacy locale ModeUI = %q, want %q", hFR.ModeUI, "Team Slayer")
	}
	if hFR.PlaylistLabel != "Quick Play" {
		t.Errorf("legacy locale PlaylistLabel = %q, want %q", hFR.PlaylistLabel, "Quick Play")
	}
	if !strings.Contains(hFR.StartTimeLabel, "Apr") {
		t.Errorf("legacy locale StartTimeLabel = %q, want English month (Apr)", hFR.StartTimeLabel)
	}
}
