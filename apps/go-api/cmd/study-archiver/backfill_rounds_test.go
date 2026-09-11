package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/domain/title"
)

const backfillMatchID = "60000000-0000-4000-8000-000000000006"

// withRounds sets a stats player's round counts where the API puts them.
func withRounds(player map[string]any, won, lost, tied int) map[string]any {
	teams, _ := player["PlayerTeamStats"].([]any)
	stats, _ := teams[0].(map[string]any)["Stats"].(map[string]any)
	core, _ := stats["CoreStats"].(map[string]any)
	core["RoundsWon"], core["RoundsLost"], core["RoundsTied"] = float64(won), float64(lost), float64(tied)
	return player
}

// backfillFixture archives one Oddball-like match recorded before rounds were: an artifact on disk
// naming deaths + 3 lives per player, a roster with no round counts, and stats that carry them.
func backfillFixture(t *testing.T) (*fakeHalo, deps) {
	t.Helper()
	f := newFakeHalo(t, map[int][]byte{0: []byte("header"), 1: []byte("replication")}, nil)
	f.statsByMatch = map[string]map[string]any{backfillMatchID: statsFor(backfillMatchID, "Cliffhanger", []any{
		withRounds(statsPlayer("xuid(1)", "JGtm", 0, 2, 15, 2, 4), 2, 1, 0),
		withRounds(statsPlayer("xuid(2)", "Rival", 1, 3, 9, 4, 2), 1, 2, 0),
	})}
	d := f.deps(t, stubBuild(nil))

	artifact := filepath.Join(t.TempDir(), "backfill.json")
	blob, err := json.Marshal(livesDoc(map[string]int{"1": 5, "2": 7}, 0, 12))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifact, blob, 0o600); err != nil {
		t.Fatal(err)
	}
	rec, _ := sampleRecord()
	played := time.Date(2026, 5, 19, 20, 15, 0, 0, time.UTC)
	rec.MatchID, rec.ShortID, rec.PlayedAt = backfillMatchID, title.FilmShortMatchID(backfillMatchID), &played
	rec.State, rec.SkipReason, rec.ArtifactPath = stateDownloaded, "", artifact
	roster := []participantRecord{officialPlayer("1", intPtr(2)), officialPlayer("2", intPtr(4))}
	if err := d.Archive.recordMatch(context.Background(), rec, roster); err != nil {
		t.Fatalf("seeding: %v", err)
	}
	return f, d
}

// Rounds are recorded from the stats, and the comparison is re-graded from the artifact on disk:
// the replay that read as 4 over-named lives under deaths + 1 is exact under deaths + rounds.
func TestBackfillRounds_RecordsRoundsAndRegradesFromTheArtifact(t *testing.T) {
	_, d := backfillFixture(t)
	ctx := context.Background()

	sum := backfillRoundsPass(ctx, d, 0)

	if sum.Candidates != 1 || sum.Updated != 1 || sum.Failed != 0 {
		t.Fatalf("summary = %+v, want one match updated", sum)
	}
	roster, err := d.Archive.roster(ctx, backfillMatchID)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range roster {
		if p.Rounds == nil || *p.Rounds != 3 {
			t.Errorf("player %s rounds = %v, want 3", p.XUID, p.Rounds)
		}
	}
	var expected, over int
	if err := d.Archive.db.QueryRow(ctx,
		`SELECT gt_expected_lives, gt_over_named FROM matches WHERE match_id = ?`, backfillMatchID).
		Scan(&expected, &over); err != nil {
		t.Fatal(err)
	}
	if expected != 12 || over != 0 {
		t.Errorf("re-graded expected/over-named = %d/%d, want 12/0", expected, over)
	}

	if again := backfillRoundsPass(ctx, d, 0); again.Candidates != 0 {
		t.Errorf("second run candidates = %d, want 0 - every round count is recorded", again.Candidates)
	}
}

// A player whose stats carry no round field gets no count: absent, never a guessed 1.
func TestRoundsByXUID_AbsentFieldsStayAbsent(t *testing.T) {
	stats := statsFor(backfillMatchID, "Cliffhanger", []any{
		withRounds(statsPlayer("xuid(1)", "JGtm", 0, 2, 15, 2, 4), 0, 1, 0),
		statsPlayer("xuid(2)", "Rival", 1, 3, 9, 4, 2),
	})
	got := roundsByXUID(stats)
	if r := got["1"]; r == nil || *r != 1 {
		t.Errorf("player 1 rounds = %v, want 1", r)
	}
	if r, ok := got["2"]; ok {
		t.Errorf("player 2 rounds = %v, want absent", *r)
	}
}
