package notify

// labels_test.go — PMT-11: NotifyLabels contract tests.
//   (a) canonical English Halo outcomes;
//   (b) routing through a synthetic title manifest;
//   (c) failsafe fallback when the source or key is absent.

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/games/mappings"
)

// fakeOutcomeSrc satisfait OutcomeSource avec un set injecté (titre fictif).
type fakeOutcomeSrc struct{ set *mappings.OutcomeMappingSet }

func (f fakeOutcomeSrc) Outcomes() *mappings.OutcomeMappingSet { return f.set }

// syntheticOutcomes charge un set divergent (copie du corpus synthetic_title_b).
func syntheticOutcomes(t *testing.T) *mappings.OutcomeMappingSet {
	t.Helper()
	toml := []byte(`
[meta]
title_slug     = "synthetic_title_b"
schema_version = 1

[outcomes.win]
labels = { en = "Victory" }
color_token = "outcome.positive"

[outcomes.loss]
labels = { en = "Defeat" }
color_token = "outcome.negative"

[outcomes.tie]
labels = { en = "Draw" }
color_token = "outcome.neutral"

[outcomes.dnf]
labels = { en = "Forfeit" }
color_token = "outcome.neutral"
`)
	set, err := mappings.LoadOutcomesFromBytes("synth_outcomes.toml", toml)
	if err != nil {
		t.Fatalf("LoadOutcomesFromBytes: %v", err)
	}
	return set
}

func renderOutcome(t *testing.T, outcome int, lang string, labels NotifyLabels) string {
	t.Helper()
	lm := &LastMatchInfo{
		MapName: "M", PlaylistName: "P", VariantName: "V",
		Outcome: outcome, StartTime: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	return strings.Join(lastMatchLines(lm, lang, labels), "\n")
}

// TestNotifyLabels_HaloParity verifies the canonical English Halo outcomes.
func TestNotifyLabels_HaloParity(t *testing.T) {
	cases := []struct {
		outcome int
		en      string
	}{
		{2, "Win"},
		{3, "Loss"},
		{1, "Draw"},
		{4, "Quit"},
	}
	for _, c := range cases {
		if got := HaloLabels().Outcome(outcomeCanonicalKey[c.outcome], "en"); got != c.en {
			t.Errorf("outcome %d = %q, want %q", c.outcome, got, c.en)
		}
	}
}

// TestNotifyLabels_BuildSyncEmbedDefaultEqualsHalo (oracle a) : la valeur produite
// par BuildSyncEmbed est strictement égale à BuildSyncEmbedWithLabels(HaloLabels()).
func TestNotifyLabels_BuildSyncEmbedDefaultEqualsHalo(t *testing.T) {
	players := []PlayerSyncResult{{
		Gamertag: "GT", MatchesSynced: 1,
		LastMatch: &LastMatchInfo{MapName: "Aquarius", PlaylistName: "Ranked", VariantName: "Slayer",
			IsRanked: true, Outcome: 2, Kills: 15, Deaths: 8, Assists: 4,
			StartTime: time.Date(2026, 1, 1, 14, 30, 0, 0, time.UTC)},
	}}
	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	end := start.Add(time.Minute)
	a := BuildSyncEmbed("sync_delta", start, end, players, true, "en")
	b := BuildSyncEmbedWithLabels("sync_delta", start, end, players, true, "en", HaloLabels())
	if !reflect.DeepEqual(a, b) {
		t.Errorf("BuildSyncEmbed != BuildSyncEmbedWithLabels(HaloLabels())")
	}
}

// TestNotifyLabels_SyntheticRouted verifies that a title with a divergent
// English manifest uses its own outcomes rather than the Halo fallback.
func TestNotifyLabels_SyntheticRouted(t *testing.T) {
	labels := LabelsFor(fakeOutcomeSrc{set: syntheticOutcomes(t)}, "")

	win := renderOutcome(t, 2, "en", labels)
	if !strings.Contains(win, "Victory") {
		t.Errorf("win via title B = %q, want Victory", win)
	}
	if strings.Contains(win, "Win") {
		t.Errorf("win via title B contains Halo fallback Win")
	}
	// Integer-to-key bridge: Discord quit (4) maps to dnf.
	if dnf := renderOutcome(t, 4, "en", labels); !strings.Contains(dnf, "Forfeit") {
		t.Errorf("dnf via title B = %q, want Forfeit", dnf)
	}
	if tie := renderOutcome(t, 1, "en", labels); !strings.Contains(tie, "Draw") {
		t.Errorf("tie via title B = %q, want Draw", tie)
	}
}

// TestNotifyLabels_FailsafeDegradation : src nil / Outcomes() nil / clé absente
// dégradent proprement vers les libellés Halo, sans panic.
func TestNotifyLabels_FailsafeDegradation(t *testing.T) {
	if got := LabelsFor(nil, "").Outcome("win", "en"); got != "Win" {
		t.Errorf("LabelsFor(nil).Outcome(win,en) = %q, want Win", got)
	}
	if got := LabelsFor(fakeOutcomeSrc{set: nil}, "").Outcome("win", "en"); got != "Win" {
		t.Errorf("Outcomes()==nil -> %q, want Win", got)
	}

	// Set partiel (seulement win) : la clé présente route, la clé absente dégrade.
	partial, err := mappings.LoadOutcomesFromBytes("partial.toml", []byte(`
[meta]
title_slug     = "x"
schema_version = 1
[outcomes.win]
labels = { en = "Won" }
color_token = "outcome.positive"
`))
	if err != nil {
		t.Fatalf("load partial: %v", err)
	}
	labels := LabelsFor(fakeOutcomeSrc{set: partial}, "")
	if got := labels.Outcome("win", "en"); got != "Won" {
		t.Errorf("win present -> %q, want Won", got)
	}
	if got := labels.Outcome("dnf", "en"); got != "Quit" {
		t.Errorf("dnf absent from title -> %q, want Quit", got)
	}
}
