package service

import (
	"testing"

	"levelup/go-api/internal/domain"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func makeMapRows() []domain.FilterMatchRow {
	// Rows carry canonical English names. Legacy fields are mirrors only.
	rows := make([]domain.FilterMatchRow, 0, 7)
	for i := 0; i < 4; i++ {
		en := "Aquarius"
		fr := en
		rows = append(rows, domain.FilterMatchRow{
			MatchID:   "m" + string(rune('0'+i)),
			MapName:   strPtr(en),
			MapNameFR: strPtr(fr), // applyMapFRTranslations enriched this
		})
	}
	for i := 0; i < 3; i++ {
		en := "Recharge"
		fr := en
		rows = append(rows, domain.FilterMatchRow{
			MatchID:   "n" + string(rune('0'+i)),
			MapName:   strPtr(en),
			MapNameFR: strPtr(fr),
		})
	}
	return rows
}

func makePlaylistRows() []domain.FilterMatchRow {
	rows := make([]domain.FilterMatchRow, 0, 6)
	for i := 0; i < 3; i++ {
		en := "Ranked Arena"
		fr := en
		rows = append(rows, domain.FilterMatchRow{
			MatchID:        "p" + string(rune('0'+i)),
			PlaylistNameEN: strPtr(en),
			PlaylistName:   strPtr(fr),
		})
	}
	for i := 0; i < 3; i++ {
		en := "Quick Play"
		fr := en
		rows = append(rows, domain.FilterMatchRow{
			MatchID:        "q" + string(rune('0'+i)),
			PlaylistNameEN: strPtr(en),
			PlaylistName:   strPtr(fr),
		})
	}
	return rows
}

// ---------------------------------------------------------------------------
// buildMapTranslationMap
// ---------------------------------------------------------------------------

func TestBuildMapTranslationMap_WithTranslations(t *testing.T) {
	rows := makeMapRows()
	tr := buildMapTranslationMap(rows)
	if len(tr) != 0 {
		t.Fatalf("expected no translation entries for English-only rows, got %d", len(tr))
	}
}

func TestBuildMapTranslationMap_SameName_NotIncluded(t *testing.T) {
	rows := []domain.FilterMatchRow{
		{MatchID: "x1", MapName: strPtr("Catalyst"), MapNameFR: strPtr("Catalyst")},
	}
	tr := buildMapTranslationMap(rows)
	if len(tr) != 0 {
		t.Errorf("expected empty map when EN==FR, got %d entries", len(tr))
	}
}

func TestBuildMapTranslationMap_NilMapName(t *testing.T) {
	rows := []domain.FilterMatchRow{
		{MatchID: "x1", MapName: nil, MapNameFR: strPtr("Legacy Map")},
	}
	tr := buildMapTranslationMap(rows)
	if len(tr) != 0 {
		t.Errorf("expected empty map for nil MapName, got %d entries", len(tr))
	}
}

// ---------------------------------------------------------------------------
// buildPlaylistTranslationMap
// ---------------------------------------------------------------------------

func TestBuildPlaylistTranslationMap_WithTranslations(t *testing.T) {
	rows := makePlaylistRows()
	tr := buildPlaylistTranslationMap(rows)
	if len(tr) != 0 {
		t.Fatalf("expected no translation entries for English-only rows, got %d", len(tr))
	}
}

func TestBuildPlaylistTranslationMap_SameName_NotIncluded(t *testing.T) {
	rows := []domain.FilterMatchRow{
		{MatchID: "x1", PlaylistNameEN: strPtr("Ranked Arena"), PlaylistName: strPtr("Ranked Arena")},
	}
	tr := buildPlaylistTranslationMap(rows)
	if len(tr) != 0 {
		t.Errorf("expected empty map when EN==FR, got %d entries", len(tr))
	}
}

func TestBuildPlaylistTranslationMap_NilEN(t *testing.T) {
	rows := []domain.FilterMatchRow{
		{MatchID: "x1", PlaylistNameEN: nil, PlaylistName: strPtr("Arène classée")},
	}
	tr := buildPlaylistTranslationMap(rows)
	if len(tr) != 0 {
		t.Errorf("expected empty map for nil PlaylistNameEN, got %d entries", len(tr))
	}
}

// ---------------------------------------------------------------------------
// migrateCascadeValues
// ---------------------------------------------------------------------------

func TestMigrateCascadeValues_TranslatesKnown(t *testing.T) {
	tr := map[string]string{"Aquarius": "Aquarius", "Recharge": "Recharge"}
	in := []string{"Aquarius", "Recharge", "Unknown Map"}
	out := migrateCascadeValues(in, tr)
	if out[0] != "Aquarius" {
		t.Errorf("[0] want 'Aquarius', got %q", out[0])
	}
	if out[1] != "Recharge" {
		t.Errorf("[1] want 'Recharge', got %q", out[1])
	}
	if out[2] != "Unknown Map" {
		t.Errorf("[2] want 'Unknown Map' preserved, got %q", out[2])
	}
}

func TestMigrateCascadeValues_EmptyTranslation(t *testing.T) {
	in := []string{"Aquarius", "Recharge"}
	out := migrateCascadeValues(in, map[string]string{})
	for i, v := range out {
		if v != in[i] {
			t.Errorf("[%d] want %q preserved, got %q", i, in[i], v)
		}
	}
}

// ---------------------------------------------------------------------------
// ResolveFiltersFromRows — cascade migration for maps
// ---------------------------------------------------------------------------

func makeMapFilterRows() []domain.FilterMatchRow {
	rows := make([]domain.FilterMatchRow, 0, 8)
	for i := 0; i < 5; i++ {
		rows = append(rows, domain.FilterMatchRow{
			MatchID:   "a" + string(rune('0'+i)),
			MapName:   strPtr("Aquarius"),
			MapNameFR: strPtr("Aquarius"),
		})
	}
	for i := 0; i < 3; i++ {
		rows = append(rows, domain.FilterMatchRow{
			MatchID:   "r" + string(rune('0'+i)),
			MapName:   strPtr("Recharge"),
			MapNameFR: strPtr("Recharge"),
		})
	}
	return rows
}

func TestResolveFilters_EnglishMapFilter(t *testing.T) {
	rows := makeMapFilterRows()
	input := domain.FilterContextInput{
		Cascade: domain.CascadeFilter{
			Maps: []string{"Aquarius"},
		},
	}
	result := ResolveFiltersFromRows(rows, input)
	if result.Counts.TotalMatchesAfterFilters != 5 {
		t.Errorf("expected 5 matches for English playlist filter, got %d", result.Counts.TotalMatchesAfterFilters)
	}
}

func TestResolveFilters_SecondEnglishMapFilter_WorksDirectly(t *testing.T) {
	rows := makeMapFilterRows()
	input := domain.FilterContextInput{
		Cascade: domain.CascadeFilter{
			Maps: []string{"Aquarius"},
		},
	}
	result := ResolveFiltersFromRows(rows, input)
	if result.Counts.TotalMatchesAfterFilters != 5 {
		t.Errorf("expected 5 matches for direct English playlist filter, got %d", result.Counts.TotalMatchesAfterFilters)
	}
}

func TestResolveFilters_MapOptions_AreEnglish(t *testing.T) {
	rows := makeMapFilterRows()
	result := ResolveFiltersFromRows(rows, domain.FilterContextInput{})
	maps := result.AvailableOptions.Maps
	for _, m := range maps {
		_ = m
	}
	found := false
	for _, m := range maps {
		if m.Value == "Aquarius" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'Aquarius' in map options")
	}
}

// ---------------------------------------------------------------------------
// ResolveFiltersFromRows — cascade migration for playlists
// ---------------------------------------------------------------------------

func makePlaylistFilterRows() []domain.FilterMatchRow {
	rows := make([]domain.FilterMatchRow, 0, 8)
	for i := 0; i < 5; i++ {
		rows = append(rows, domain.FilterMatchRow{
			MatchID:        "p" + string(rune('0'+i)),
			PlaylistNameEN: strPtr("Ranked Arena"),
			PlaylistName:   strPtr("Ranked Arena"),
		})
	}
	for i := 0; i < 3; i++ {
		rows = append(rows, domain.FilterMatchRow{
			MatchID:        "q" + string(rune('0'+i)),
			PlaylistNameEN: strPtr("Quick Play"),
			PlaylistName:   strPtr("Quick Play"),
		})
	}
	return rows
}

func TestResolveFilters_EnglishPlaylistFilter(t *testing.T) {
	rows := makePlaylistFilterRows()
	input := domain.FilterContextInput{
		Cascade: domain.CascadeFilter{
			Playlists: []string{"Ranked Arena"}, // stored as EN
		},
	}
	result := ResolveFiltersFromRows(rows, input)
	if result.Counts.TotalMatchesAfterFilters != 5 {
		t.Errorf("expected 5 matches for English playlist filter, got %d", result.Counts.TotalMatchesAfterFilters)
	}
}

func TestResolveFilters_SecondEnglishPlaylistFilter_WorksDirectly(t *testing.T) {
	rows := makePlaylistFilterRows()
	input := domain.FilterContextInput{
		Cascade: domain.CascadeFilter{
			Playlists: []string{"Ranked Arena"},
		},
	}
	result := ResolveFiltersFromRows(rows, input)
	if result.Counts.TotalMatchesAfterFilters != 5 {
		t.Errorf("expected 5 matches for direct English playlist filter, got %d", result.Counts.TotalMatchesAfterFilters)
	}
}

func TestResolveFilters_PlaylistOptions_AreEnglish(t *testing.T) {
	rows := makePlaylistFilterRows()
	result := ResolveFiltersFromRows(rows, domain.FilterContextInput{})
	playlists := result.AvailableOptions.Playlists
	for _, p := range playlists {
		if p.Value == "Unexpected" {
			t.Errorf("playlist option should be FR, got EN value %q", p.Value)
		}
	}
	found := false
	for _, p := range playlists {
		if p.Value == "Ranked Arena" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'Ranked Arena' in playlist options")
	}
}

func TestResolveFilters_MultiplePlaylistsLegacy_AllMigrated(t *testing.T) {
	rows := makePlaylistFilterRows()
	input := domain.FilterContextInput{
		Cascade: domain.CascadeFilter{
			Playlists: []string{"Ranked Arena", "Quick Play"},
		},
	}
	result := ResolveFiltersFromRows(rows, input)
	if result.Counts.TotalMatchesAfterFilters != 8 {
		t.Errorf("expected 8 matches for both playlists migrated, got %d", result.Counts.TotalMatchesAfterFilters)
	}
}
