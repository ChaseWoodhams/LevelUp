package main

// filter_test.go — the filter is the one piece of this server with real rules of its own,
// so it is tested WITHOUT a database: every case here is about what a query string means,
// and none of them needs a row to prove it.

import (
	"strings"
	"testing"
	"time"
)

func TestParseFilter_Defaults(t *testing.T) {
	f, err := parseFilter(rawFilter{})
	if err != nil {
		t.Fatalf("an empty query string must be valid: %v", err)
	}
	if f.Limit != defaultLimit || f.Offset != 0 {
		t.Errorf("limit/offset = %d/%d, want %d/0", f.Limit, f.Offset, defaultLimit)
	}
	if f.From != nil || f.To != nil || f.MinCoverage != nil {
		t.Errorf("an unset bound must stay nil, got from=%v to=%v cov=%v", f.From, f.To, f.MinCoverage)
	}
}

// TestParseFilter_DateOnlyRangeCoversTheWholeDay is the case a half-open range exists for:
// from=D&to=D has to mean "the whole of day D", not "the empty instant at its start".
func TestParseFilter_DateOnlyRangeCoversTheWholeDay(t *testing.T) {
	f, err := parseFilter(rawFilter{From: "2026-05-19", To: "2026-05-19"})
	if err != nil {
		t.Fatalf("parseFilter: %v", err)
	}
	wantFrom := time.Date(2026, 5, 19, 0, 0, 0, 0, time.UTC)
	wantTo := time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC)
	if !f.From.Equal(wantFrom) {
		t.Errorf("from = %s, want %s", f.From, wantFrom)
	}
	if !f.To.Equal(wantTo) {
		t.Errorf("to = %s, want %s (exclusive upper bound)", f.To, wantTo)
	}
}

func TestParseFilter_RFC3339Instants(t *testing.T) {
	f, err := parseFilter(rawFilter{From: "2026-05-19T20:15:00Z", To: "2026-05-19T21:00:00Z"})
	if err != nil {
		t.Fatalf("parseFilter: %v", err)
	}
	if !f.From.Equal(time.Date(2026, 5, 19, 20, 15, 0, 0, time.UTC)) {
		t.Errorf("from = %s", f.From)
	}
	if !f.To.Equal(time.Date(2026, 5, 19, 21, 0, 0, 0, time.UTC)) {
		t.Errorf("to = %s", f.To)
	}
}

func TestParseFilter_Rejections(t *testing.T) {
	cases := []struct {
		name string
		raw  rawFilter
		want string
	}{
		{"unparsable from", rawFilter{From: "hier"}, "from"},
		{"unparsable to", rawFilter{To: "2026-13-45"}, "to"},
		{"reversed range", rawFilter{From: "2026-05-20", To: "2026-05-19"}, "from"},
		{"coverage not a number", rawFilter{MinCoverage: "beaucoup"}, "min_coverage"},
		{"coverage above one", rawFilter{MinCoverage: "85"}, "min_coverage"},
		{"coverage below zero", rawFilter{MinCoverage: "-0.1"}, "min_coverage"},
		{"negative limit", rawFilter{Limit: -1}, "limit"},
		{"limit past the cap", rawFilter{Limit: maxLimit + 1}, "limit"},
		{"negative offset", rawFilter{Offset: -1}, "offset"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := parseFilter(c.raw)
			if err == nil {
				t.Fatalf("%+v must be refused", c.raw)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("the message must name %q, got %q", c.want, err)
			}
		})
	}
}

// TestParseFilter_CoverageIsAFraction pins the unit. ADR 0006 fixes canonical indicators at
// 0..1, and the archive's coverage is one of them: 0.85, never 85.
func TestParseFilter_CoverageIsAFraction(t *testing.T) {
	f, err := parseFilter(rawFilter{MinCoverage: "0.85"})
	if err != nil {
		t.Fatalf("parseFilter: %v", err)
	}
	if f.MinCoverage == nil || *f.MinCoverage != 0.85 {
		t.Fatalf("min_coverage = %v, want 0.85", f.MinCoverage)
	}
}

// TestFilterWhere_OnlyArchivedMatches — the base predicate is not optional. A match with no
// artifact cannot be replayed, so it has no business in a list of things to study.
func TestFilterWhere_OnlyArchivedMatches(t *testing.T) {
	where, args := matchFilter{}.where()
	if !strings.Contains(where, "artifact_path IS NOT NULL") {
		t.Errorf("the base predicate must exclude unbuilt matches, got %q", where)
	}
	if len(args) != 0 {
		t.Errorf("an empty filter must bind no argument, got %v", args)
	}
}

func TestFilterWhere_BindsEveryValue(t *testing.T) {
	from := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	cov := 0.8
	f := matchFilter{Map: "Cliffhanger", Mode: "Slayer", Player: "JGtm",
		From: &from, To: &to, MinCoverage: &cov}

	where, args := f.where()

	// Six values for six criteria: the player is matched on TWO columns (xuid or gamertag)
	// and therefore binds twice.
	if len(args) != 7 {
		t.Fatalf("args = %v (%d), want 7", args, len(args))
	}
	for _, fragment := range []string{"map_name", "mode", "EXISTS", "played_at >=", "played_at <"} {
		if !strings.Contains(where, fragment) {
			t.Errorf("clause is missing %q: %s", fragment, where)
		}
	}
	// A single `?` per bound value, in the order the args are given.
	if n := strings.Count(where, "?"); n != len(args) {
		t.Errorf("%d placeholders for %d arguments: %s", n, len(args), where)
	}
}

// TestFilterWhere_MinCoverageExcludesTheUnknown. total_lives = 0 means the artifact reported
// no lives at all, so its coverage is UNKNOWN, not zero. A caller asking for at least 80 %
// is asking to be shown matches that meet the bar; one that cannot be shown to meet it does
// not belong in the answer.
func TestFilterWhere_MinCoverageExcludesTheUnknown(t *testing.T) {
	cov := 0.0
	where, args := matchFilter{MinCoverage: &cov}.where()
	if !strings.Contains(where, "total_lives > 0") {
		t.Errorf("a coverage floor must exclude matches with no lives recorded: %s", where)
	}
	if len(args) != 1 {
		t.Errorf("args = %v, want the coverage floor alone", args)
	}
}

// TestFilterWhere_PlayerMatchesXUIDOrGamertag — the archive keys players by xuid, but the
// only thing an operator knows how to type is a gamertag. Both reach the same rows.
func TestFilterWhere_PlayerMatchesXUIDOrGamertag(t *testing.T) {
	where, args := matchFilter{Player: "JGtm"}.where()
	if !strings.Contains(where, "p.xuid") || !strings.Contains(where, "p.gamertag") {
		t.Errorf("the player predicate must try both columns: %s", where)
	}
	if len(args) != 2 || args[0] != "JGtm" || args[1] != "JGtm" {
		t.Errorf("args = %v, want the player bound to both columns", args)
	}
}
