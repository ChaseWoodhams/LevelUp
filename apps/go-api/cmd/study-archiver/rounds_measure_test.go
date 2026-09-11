package main

import (
	"context"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/sync/haloclient"
)

// rounds_measure_test.go — WHAT THE OFFICIAL STATS SAY ABOUT ROUNDS.
//
// The ground-truth check expects deaths + 1 lives per player. A round-based match gives every
// player a fresh life at each new round with no death behind it (measured in the films: every
// player's life ends at the same instant, and all respawn ~3.5 s later). Before the check reads
// rounds from CoreStats, this bench shows what RoundsWon / RoundsLost / RoundsTied actually hold,
// per mode, on real matches. It reads the network with the stored token and writes nothing.
//
//	STUDY_ROUNDS_XUID=<owner xuid> STUDY_ROUNDS_MATCHES=<matchId>,<matchId> \
//	  go test ./cmd/study-archiver/ -run TestRoundsInOfficialStats -v
func TestRoundsInOfficialStats(t *testing.T) {
	xuid, ids := os.Getenv("STUDY_ROUNDS_XUID"), os.Getenv("STUDY_ROUNDS_MATCHES")
	if xuid == "" || ids == "" {
		t.Skip("STUDY_ROUNDS_XUID / STUDY_ROUNDS_MATCHES not set: network bench, not CI")
	}
	ctx := context.Background()
	root, err := title.FindRepoRoot()
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	tokens, err := resolveTokens(ctx, title.NewPathResolver(root), depsRequest{XUID: xuid})
	if err != nil {
		t.Fatalf("tokens: %v", err)
	}
	client := haloclient.NewHaloAPIClient(tokens.SpartanToken, tokens.ClearanceToken, 0)
	for _, id := range strings.Split(ids, ",") {
		stats, err := client.GetMatchStats(ctx, strings.TrimSpace(id))
		if err != nil {
			t.Errorf("%s: %v", id, err)
			continue
		}
		facts, _ := readMatchFacts(stats, "")
		players, _ := stats["Players"].([]any)
		var lines []string
		blocks := map[string]bool{}
		for _, p := range players {
			pm, _ := p.(map[string]any)
			pts, _ := pm["PlayerTeamStats"].([]any)
			if len(pts) == 0 {
				continue
			}
			st, _ := pts[0].(map[string]any)["Stats"].(map[string]any)
			for k := range st {
				if k != "CoreStats" {
					blocks[k] = true
				}
			}
			core, _ := st["CoreStats"].(map[string]any)
			lines = append(lines, "won "+num(core["RoundsWon"])+" lost "+num(core["RoundsLost"])+
				" tied "+num(core["RoundsTied"])+" deaths "+num(core["Deaths"])+" "+strings.TrimPrefix(
				asStr(pm["PlayerId"]), "xuid"))
		}
		names := make([]string, 0, len(blocks))
		for k := range blocks {
			names = append(names, k)
		}
		sort.Strings(names)
		t.Logf("%s mode %q map %q stat blocks %v team scores %v/%v", id[:8], facts.Mode, facts.MapName,
			names, derefInt(facts.Team0Score), derefInt(facts.Team1Score))
		for _, l := range lines {
			t.Logf("  %s", l)
		}
	}
}

func num(v any) string {
	if f, ok := v.(float64); ok {
		return strconv.Itoa(int(f))
	}
	return "-"
}

func asStr(v any) string {
	s, _ := v.(string)
	return s
}

func derefInt(p *int) any {
	if p == nil {
		return "-"
	}
	return *p
}
