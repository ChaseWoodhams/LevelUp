package analysis

import "testing"

func TestResolvePairName(t *testing.T) {
	modes := map[string]string{
		"CTF":              "Capture du drapeau",
		"Strongholds":      "Bases",
		"Slayer":           "Assassin",
		"Team Slayer":      "Assassin en équipe",
		"Neutral Flag CTF": "Drapeau neutre",
	}

	tests := []struct {
		name        string
		rawPair     string
		currentName string
		assetName   string
		want        string
	}{
		{
			name:        "priorite 1 - mode_name_tr depuis pair_name brut",
			rawPair:     "Arena:CTF on Aquarius",
			currentName: "Arena:CTF on Aquarius", // placeholder COALESCE
			assetName:   "",
			want:        "CTF",
		},
		{
			name:        "priorite 1 - BTB:Strongholds → Bases",
			rawPair:     "BTB:Strongholds on Highpower",
			currentName: "BTB:Strongholds on Highpower",
			assetName:   "",
			want:        "Strongholds",
		},
		{
			name:        "priorite 2 - rawPair vide, asset normalisable → re-lookup",
			rawPair:     "",
			currentName: "",
			assetName:   "Arena:CTF on Shiro",
			want:        "CTF",
		},
		{
			name:        "priorite 2 - rawPair UUID, asset normalisable",
			rawPair:     "bd1457cc-4fd8-4da1-be87-381c142017e8",
			currentName: "bd1457cc-4fd8-4da1-be87-381c142017e8",
			assetName:   "Arena:Team Slayer on Bazaar - Forge",
			want:        "Team Slayer",
		},
		{
			name:        "priorite 3 - asset present mais pas dans mode_name_tr, currentName vide",
			rawPair:     "",
			currentName: "",
			assetName:   "Castle Wars",
			want:        "Castle Wars",
		},
		{
			name:        "priorite 3 - asset present, currentName == EN (placeholder)",
			rawPair:     "Custom:Unknown",
			currentName: "Custom:Unknown",
			assetName:   "Mode Custom Inconnu",
			want:        "Unknown",
		},
		{
			name:        "preservation - currentName vrai FR, pas d'override raw",
			rawPair:     "Custom:Unknown",
			currentName: "Mode Personnalisé",
			assetName:   "Custom:Unknown", // EN raw, ne doit pas écraser le FR existant
			want:        "Unknown",
		},
		{
			name:        "tout vide → string vide",
			rawPair:     "",
			currentName: "",
			assetName:   "",
			want:        "",
		},
		{
			name:        "currentName seul (pas de modeNames ni asset)",
			rawPair:     "Some:Mode",
			currentName: "Mode Connu",
			assetName:   "",
			want:        "Mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolvePairName(tt.rawPair, tt.currentName, tt.assetName, modes)
			if got != tt.want {
				t.Errorf("ResolvePairName(%q, %q, %q) = %q, want %q",
					tt.rawPair, tt.currentName, tt.assetName, got, tt.want)
			}
		})
	}
}

func TestResolvePairName_PreservesIdentityPrefixesViaNormalize(t *testing.T) {
	// Super Fiesta:Slayer n'a pas d'entrée mode_name_tr (intentionnel — c'est
	// un playlist identity prefix). Le helper doit donc préserver le raw EN
	// (currentName), et la normalisation finale (qui extrait "Super Fiesta")
	// est faite par les consumers downstream via NormalizeModeLabel.
	modes := map[string]string{
		"Slayer": "Assassin",
	}
	raw := "Super Fiesta:Slayer on Streets - Forge"
	got := ResolvePairName(raw, raw, "", modes)
	if got != "Super Fiesta" {
		t.Errorf("got %q, want %q (English identity label)", got, "Super Fiesta")
	}
}

func TestNeedsFRTranslationOverride(t *testing.T) {
	tests := []struct {
		name   string
		fr, en string
		want   bool
	}{
		{"FR vide → override", "", "Arena:CTF", true},
		{"FR == EN → override (COALESCE placeholder)", "Arena:CTF", "Arena:CTF", true},
		{"FR == EN ignore case", "ARENA:CTF", "arena:ctf", true},
		{"FR vrai FR → preserve", "Capture du drapeau", "Arena:CTF", false},
		{"FR rempli, EN vide → preserve", "Mode FR", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := needsLabelOverride(tt.fr, tt.en)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
