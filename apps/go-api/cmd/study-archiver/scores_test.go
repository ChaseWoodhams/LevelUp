package main

import (
	"context"
	"testing"
)

func TestFetchOne_RecordsFinalScoresFromMatchStats(t *testing.T) {
	stats := statsWithMap("Cliffhanger")
	stats["Teams"] = []any{
		map[string]any{"TeamId": float64(0), "Stats": map[string]any{"CoreStats": map[string]any{"Score": float64(3)}}},
		map[string]any{"TeamId": float64(1), "Stats": map[string]any{"CoreStats": map[string]any{"Score": float64(0)}}},
	}
	srv := newFakeHalo(t, map[int][]byte{0: []byte("header"), 1: []byte("replication")}, stats)
	_, rec := archived(t, srv.deps(t, stubBuild(nil)))
	if rec.Team0Score == nil || *rec.Team0Score != 3 || rec.Team1Score == nil || *rec.Team1Score != 0 {
		t.Fatalf("scores = %v / %v; want 3 / 0, not participant kills", rec.Team0Score, rec.Team1Score)
	}
}

func TestArchive_RoundTripsFinalScores(t *testing.T) {
	a := testArchive(t)
	rec, roster := sampleRecord()
	rec.Team0Score, rec.Team1Score = intPtr(3), intPtr(0)
	if err := a.recordMatch(context.Background(), rec, roster); err != nil {
		t.Fatal(err)
	}
	got, found, err := a.recorded(context.Background(), rec.MatchID)
	if err != nil || !found {
		t.Fatalf("recorded: found=%v err=%v", found, err)
	}
	if got.Team0Score == nil || *got.Team0Score != 3 || got.Team1Score == nil || *got.Team1Score != 0 {
		t.Fatalf("scores = %v / %v; want 3 / 0", got.Team0Score, got.Team1Score)
	}
}
