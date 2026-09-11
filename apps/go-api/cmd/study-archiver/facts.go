package main

// facts.go — WHAT THE MATCH STATS SAY ABOUT A MATCH.
//
// THE EXTRACTION IS THE REPO'S OWN, NOT A COPY OF IT. `sync.ExtractRegistry` and
// `sync.ExtractParticipants` already read this payload for the warehouse, and they are
// exported. Re-deriving map, playlist, duration, xuid, team, outcome and K/D/A here
// would be a second implementation of a well-tested one — and worse than duplication, it
// would let the archive DISAGREE with the warehouse about the same match: two numbers
// for one match, with nothing to say which is right.
//
// The cost is honest and worth naming: importing `internal/sync` pulls its whole
// dependency tree, DuckDB included, into this binary. That is acceptable because this
// ticket makes the archiver a DuckDB writer anyway (the archive database), so the
// toolchain requirement is already paid for. Before #6 the binary was deliberately
// cgo-free; from here it is not, and cmd/study-archiver must be built with the UCRT
// toolchain like the rest of the repo (cf. CLAUDE.md).

import (
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/sync"
)

// matchFacts is the archive's reading of one match-stats payload.
type matchFacts struct {
	Team0Score *int
	Team1Score *int
	// MapName is the DISPLAY name; it is also what the quant-bounds catalogue is keyed
	// by, so map resolution reads it from here rather than digging into the payload
	// a second time.
	MapName string
	// MapID / MapVersionID identify the map asset: the discovery API needs both to name a map
	// whose stats carried no display name (mapresolve_online.go).
	MapID        string
	MapVersionID string
	PlayedAt     *time.Time
	Mode         string
	Playlist     string
	DurationMS   *int64
	Roster       []participantRecord
}

// readMatchFacts extracts everything the archive records from a raw match-stats payload.
//
// sourceGT is the gamertag whose archiving pass produced this match; it is what
// `ExtractRegistry` calls the sync-by, and what the archive records as source_gamertag.
func readMatchFacts(stats map[string]any, sourceGT string) (matchFacts, error) {
	reg, err := sync.ExtractRegistry(stats, sourceGT)
	if err != nil {
		return matchFacts{}, fmt.Errorf("unreadable match stats: %w", err)
	}
	facts := matchFacts{
		Team0Score: reg.Team0Score, Team1Score: reg.Team1Score,
		MapName:      derefStr(reg.MapName),
		MapID:        derefStr(reg.MapID),
		MapVersionID: derefStr(reg.MapVersionID),
		Mode:         derefStr(reg.GameVariantName),
		Playlist:     derefStr(reg.PlaylistName),
	}
	if !reg.StartTime.IsZero() {
		played := reg.StartTime
		facts.PlayedAt = &played
	}
	// The registry carries seconds; the archive stores milliseconds, matching the
	// replay document's own clock so the two can be compared without a unit change.
	if reg.DurationSeconds != nil {
		ms := int64(*reg.DurationSeconds) * 1000
		facts.DurationMS = &ms
	}
	rounds := roundsByXUID(stats)
	for _, p := range sync.ExtractParticipants(stats) {
		facts.Roster = append(facts.Roster, participantRecord{
			XUID:     p.XUID,
			Gamertag: derefStr(p.Gamertag),
			Team:     p.TeamID,
			Outcome:  p.Outcome,
			Kills:    p.Kills,
			Deaths:   p.Deaths,
			Assists:  p.Assists,
			Rounds:   rounds[p.XUID],
		})
	}
	return facts, nil
}

// roundsByXUID reads how many rounds each player played: RoundsWon + RoundsLost + RoundsTied in
// the same PlayerTeamStats[0] CoreStats block sync.ExtractParticipants reads K/D/A from.
//
// NOT A SECOND EXTRACTION (cf. this file's header): the warehouse does not extract rounds at all,
// so this is the only reading of them and nothing can disagree with it. A player whose stats carry
// none of the three is absent from the map.
//
// Measured on six archived matches (rounds_measure_test.go): Oddball reported 3 and 2, Zones and
// CTF reported 1 — a single-round mode says 1, not 0.
func roundsByXUID(stats map[string]any) map[string]*int {
	out := map[string]*int{}
	players, _ := stats["Players"].([]any)
	for _, p := range players {
		pm, _ := p.(map[string]any)
		id, _ := pm["PlayerId"].(string)
		xuid := strings.TrimSuffix(strings.TrimPrefix(id, "xuid("), ")")
		if xuid == "" || xuid == id {
			continue // a bot ("bid(...)") or no id: no roster row to attach rounds to
		}
		teams, _ := pm["PlayerTeamStats"].([]any)
		if len(teams) == 0 {
			continue
		}
		team, _ := teams[0].(map[string]any)
		st, _ := team["Stats"].(map[string]any)
		core, _ := st["CoreStats"].(map[string]any)
		total, seen := 0, false
		for _, k := range []string{"RoundsWon", "RoundsLost", "RoundsTied"} {
			if v, ok := core[k].(float64); ok {
				total, seen = total+int(v), true
			}
		}
		if seen {
			n := total
			out[xuid] = &n
		}
	}
	return out
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// countsOf reads the decoded totals off a built replay document.
//
// NamedLives / TotalLives come from the document's own coverage report rather than being
// recomputed: the ratio between them is how a later coverage drop is spotted, and it has
// to be the SAME number the artifact published, not a second opinion.
//
// Coverage is a POINTER, and `omitempty`: a document that attached nothing carries none
// at all, and an artifact read back from disk carries none either. Dereferencing it
// blindly crashes the archiver on precisely the degraded match it most needs to record —
// so a missing report reads as zero lives, which is what it means.
func countsOf(doc replay.ReplayDocument) (tracks, points, shots, namedLives, totalLives int) {
	tracks, points, shots = len(doc.Tracks), totalPoints(doc), len(doc.Shots)
	if doc.Coverage == nil {
		return tracks, points, shots, 0, 0
	}
	return tracks, points, shots, doc.Coverage.Bridge.LivesNamed, doc.Coverage.Bridge.LivesTotal
}
