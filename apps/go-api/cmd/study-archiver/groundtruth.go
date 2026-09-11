package main

// groundtruth.go — THE REPLAY, CHECKED AGAINST HALO'S OWN MATCH STATS.
//
// The replay names each life after the player the film's death feed says died. Halo's match
// stats say, independently, how many times each player died. The two come from different
// systems reading different data, so comparing them checks the replay in a way no golden film
// can: it runs on EVERY archived match, on every build, and needs nothing the archive does not
// already hold.
//
// WHAT THE COMPARISON CAN AND CANNOT SAY. A player who died N times over R rounds had N + R lives:
// each death ends one, and each round STARTS one with no death behind it — at a round reset every
// player's life ends at the same instant and all respawn together (measured in the films of three
// Oddball matches, and matched to their official round counts: 3, 2, 2). A single-round mode
// reports R = 1, which is the old N + 1. So:
//   - OverNamed: lives the replay named for a player BEYOND their N + R. Those lives belong to
//     somebody else, so this is a naming DEFECT and must stay at zero, whatever the coverage
//     figures say.
//   - MissingLives: lives the replay did not name for a player SHORT of their N + R. That is
//     the replay refusing to guess (an ambiguous tie, a life the feed never closed): the work
//     left, not an error.
//   - UnknownNamed: lives named after an xuid the stats do not list at all.
//   - LivesGap: how many lives the replay SEGMENTED minus how many the stats imply. Positive
//     means lives split where nobody died; negative means lives merged or missed.
//
// A player the stats give no death count for is left out rather than guessed at. A match with
// no death count at all, or a document without a coverage report (and so without its own life
// total), is not compared.
//
// WHERE IT GOES. The archive keeps the totals on the match row and each player's named lives
// beside their official deaths in `participants`; `status` sums them over the archive, which is
// what turns a rebuild after a decoder change into a measured one.

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/analysis/replay"
)

// groundTruth is the replay-versus-stats comparison of one match.
type groundTruth struct {
	// Compared is false when there was nothing to compare against; every other field is then
	// zero and the archive stores NULL rather than a comparison that never happened.
	Compared bool
	// Players is how many roster players carried an official death count.
	Players int
	// ExpectedLives is the sum of (deaths + rounds) over those players.
	ExpectedLives int
	// NamedLives is how many lives the replay named for those players.
	NamedLives   int
	OverNamed    int
	MissingLives int
	UnknownNamed int
	LivesGap     int
	// NamedByPlayer is how many lives the replay named for each compared player, keyed by
	// xuid: what the archive stores beside that player's official deaths.
	NamedByPlayer map[string]int
}

// compareGroundTruth compares a built replay document with the official roster of its match.
func compareGroundTruth(doc replay.ReplayDocument, roster []participantRecord) groundTruth {
	if doc.Coverage == nil {
		return groundTruth{}
	}
	named := make(map[string]int)
	for _, t := range doc.Tracks {
		if t.XUID != "" {
			named[t.XUID]++
		}
	}
	gt := groundTruth{NamedByPlayer: make(map[string]int)}
	listed := make(map[string]bool, len(roster))
	for _, p := range roster {
		listed[p.XUID] = true
		if p.Deaths == nil {
			continue
		}
		expected, n := *p.Deaths+roundsOf(p), named[p.XUID]
		gt.Players++
		gt.ExpectedLives += expected
		gt.NamedLives += n
		gt.NamedByPlayer[p.XUID] = n
		if n > expected {
			gt.OverNamed += n - expected
		} else {
			gt.MissingLives += expected - n
		}
	}
	if gt.Players == 0 {
		return groundTruth{}
	}
	for xuid, n := range named {
		if !listed[xuid] {
			gt.UnknownNamed += n
		}
	}
	gt.Compared = true
	gt.LivesGap = doc.Coverage.Bridge.LivesTotal - gt.ExpectedLives
	return gt
}

// roundsOf is how many lives a player started without dying first: one per round played. A row
// with no round count (recorded before rounds were) counts one, the single-round reading.
func roundsOf(p participantRecord) int {
	if p.Rounds == nil || *p.Rounds < 1 {
		return 1
	}
	return *p.Rounds
}

// groundTruthRecord is a groundTruth as the archive stores it: every column NULL when the
// match was not compared, so "not compared" never reads as "compared, and all zero".
type groundTruthRecord struct {
	Players       *int
	ExpectedLives *int
	NamedLives    *int
	OverNamed     *int
	MissingLives  *int
	UnknownNamed  *int
	LivesGap      *int
}

// groundTruthColumns is the archive form of a comparison.
func groundTruthColumns(gt groundTruth) groundTruthRecord {
	if !gt.Compared {
		return groundTruthRecord{}
	}
	v := func(n int) *int { return &n }
	return groundTruthRecord{
		Players: v(gt.Players), ExpectedLives: v(gt.ExpectedLives), NamedLives: v(gt.NamedLives),
		OverNamed: v(gt.OverNamed), MissingLives: v(gt.MissingLives),
		UnknownNamed: v(gt.UnknownNamed), LivesGap: v(gt.LivesGap),
	}
}

// replayLivesOf is the replay's named-lives count for one player, or nil when the build did
// not compare that player.
func replayLivesOf(gt groundTruth, xuid string) *int {
	if !gt.Compared {
		return nil
	}
	n, ok := gt.NamedByPlayer[xuid]
	if !ok {
		return nil
	}
	return &n
}

// withReplayLives copies a roster with each player's replay-named lives filled in, so the
// archive records them beside that player's official deaths. The input is left untouched.
func withReplayLives(roster []participantRecord, gt groundTruth) []participantRecord {
	out := make([]participantRecord, len(roster))
	for i, p := range roster {
		p.ReplayNamedLives = replayLivesOf(gt, p.XUID)
		out[i] = p
	}
	return out
}

// logGroundTruth reports the comparison of one build. An over-named life is a naming defect,
// so it is a warning rather than an informational line.
func logGroundTruth(ctx context.Context, out outcome) {
	gt := out.GroundTruth
	if !gt.Compared {
		slog.InfoContext(ctx, "study-archiver: replay not compared with the match stats - "+
			"no official death counts or no coverage report",
			"match_id", out.MatchID, "short_id", out.ShortID)
		return
	}
	attrs := []any{
		"match_id", out.MatchID, "short_id", out.ShortID, "players", gt.Players,
		"expected_lives", gt.ExpectedLives, "named_lives", gt.NamedLives,
		"over_named", gt.OverNamed, "missing_lives", gt.MissingLives,
		"unknown_named", gt.UnknownNamed, "lives_gap", gt.LivesGap,
	}
	if gt.OverNamed > 0 {
		slog.WarnContext(ctx, "study-archiver: the replay names more lives than the match stats "+
			"allow - a naming defect", attrs...)
		return
	}
	slog.InfoContext(ctx, "study-archiver: replay checked against the match stats", attrs...)
}
