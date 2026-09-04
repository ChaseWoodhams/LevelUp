package main

// watch.go — THE UNATTENDED PASS THAT GROWS THE ARCHIVE (#8).
//
// One invocation = one pass over the watchlist, then exit. NOT a daemon: the OS scheduler
// owns the clock (hourly), which means a crashed pass is retried by the scheduler rather
// than leaving a dead process that looks alive, and the operator can run one by hand
// without fighting a running instance for the archive's single writer slot.
//
// WHY DISCOVERY IS A PLAIN HISTORY PULL, and not the app's presence-driven watcher. That
// one follows players signed into this app and reacts when they start a game. The players
// worth studying never sign in here at all, so there is no presence to react to: the only
// thing that works is asking the API what they have played lately, with the owner's own
// token.
//
// WHAT A PASS REFUSES TO DO, AND WHY EACH REFUSAL IS DIFFERENT:
//
//   - a match already ARCHIVED is skipped — it is the idempotency #6 built;
//   - a match recorded EXPIRED is skipped, permanently, because its film cannot come back;
//   - a match recorded FAILED is skipped BY THIS LOOP ONLY. Its chunks are on disk and a
//     decoder fix will rescue it — but re-running a decoder that is known broken against
//     the same bytes every hour is pure waste, and it would drown the log this run is
//     judged by. Retrying a failed match is a deliberate act, and `rebuild` (#10) is it;
//   - a match that is not the 4v4 the archive exists for is skipped and logged.
//
// The distinction matters: `expired` means nobody may ever retry, `failed` means not HERE.

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/sync/haloclient"
)

// matchTypes are the two histories a pass reads, in order. Matchmaking is where the games
// worth studying are; customs are read too because scrims and tournament matches are
// played there, and they are exactly the games a player would most want to review.
var matchTypes = []string{"matchmaking", "custom"}

// historyPageSize is how far back one pass looks per history. 25 is the API's maximum page
// and about a day of heavy play, so an hourly pass overlaps itself many times over — which
// is the point: a missed pass (a reboot, a laptop closed) costs nothing as long as the next
// one still sees the match, and the already-archived check makes the overlap free.
const historyPageSize = 25

// arenaTeamSize and arenaTeams define the shape the archive is for: 4v4.
//
// THE SHAPE IS THE FILTER, not the playlist's name. A playlist gets renamed between
// seasons and its label is localised, and CLAUDE.md forbids hard-coded labels in Go for
// exactly that reason. Two teams of four is what "4v4 Arena" MEANS, it is what the heat
// maps of epic #3 aggregate over, and it is stable across every rename Halo has done.
const (
	arenaTeamSize = 4
	arenaTeams    = 2
)

// watchDeps is what a pass needs on top of the archiving deps.
type watchDeps struct {
	// ResolveXUID turns a gamertag into the numeric xuid the history endpoint demands.
	// A seam rather than a concrete resolver: it is the only part of this tool that needs
	// an Xbox Live token chain rather than a Halo one, and a test must not.
	ResolveXUID func(ctx context.Context, gamertag string) (string, error)
}

// watchSummary is what one pass did, for the log line and the exit code.
type watchSummary struct {
	Players  int
	Seen     int
	Archived int
	Skipped  int
	// Failed counts matches whose archiving returned an error. A pass does NOT stop on
	// one: a single decoder failure must not cost the whole hour's capture, and the films
	// still expiring behind it are the reason.
	Failed int
}

// watchPass runs one pass over the watchlist.
//
// A player whose pass fails does not stop the others, for the same reason: the run exists
// to beat an expiry clock, and the next player's films are expiring too.
func watchPass(ctx context.Context, d deps, w watchDeps, wl watchlist) watchSummary {
	var sum watchSummary
	for _, gamertag := range wl.Gamertags {
		sum.Players++
		if err := watchPlayer(ctx, d, w, gamertag, &sum); err != nil {
			sum.Failed++
			slog.ErrorContext(ctx, "study-archiver: watch pass failed for a player - continuing",
				"err", err, "gamertag", gamertag)
			continue
		}
		if err := d.Archive.markChecked(ctx, gamertag); err != nil {
			// Logged, not fatal: the capture succeeded, and only the freshness stamp #9
			// reports is lost. Swallowing it silently would make `status` quietly wrong.
			slog.ErrorContext(ctx, "study-archiver: could not stamp the watchlist check",
				"err", err, "gamertag", gamertag)
		}
	}
	slog.InfoContext(ctx, "study-archiver: watch pass complete",
		"players", sum.Players, "seen", sum.Seen, "archived", sum.Archived,
		"skipped", sum.Skipped, "failed", sum.Failed)
	return sum
}

// watchPlayer archives what one tracked player has played since the last pass.
func watchPlayer(ctx context.Context, d deps, w watchDeps, gamertag string, sum *watchSummary) error {
	player, _, err := d.Archive.watched(ctx, gamertag)
	if err != nil {
		return err
	}
	slog.InfoContext(ctx, "study-archiver: watching a player",
		"gamertag", gamertag, "last_checked", lastPassAge(player, time.Now()))
	xuid, err := resolveWatched(ctx, d, w, gamertag, player)
	if err != nil {
		return err
	}
	seen := make(map[string]bool)
	for _, matchType := range matchTypes {
		entries, hErr := d.Client.GetMatchHistory(
			ctx, fmt.Sprintf("xuid(%s)", xuid), matchType, 0, historyPageSize)
		if hErr != nil {
			return fmt.Errorf("%s history of %s: %w", matchType, gamertag, hErr)
		}
		slog.InfoContext(ctx, "study-archiver: history read",
			"gamertag", gamertag, "xuid", xuid, "type", matchType, "matches", len(entries))
		for _, entry := range entries {
			// The same match appears in both histories only for customs read twice, but a
			// player can also appear twice in one watchlist pass through two spellings —
			// the dedup is cheap and the alternative is a duplicated archiving attempt.
			if seen[entry.MatchID] {
				continue
			}
			seen[entry.MatchID] = true
			sum.Seen++
			considerMatch(ctx, d, gamertag, entry, sum)
		}
	}
	return nil
}

// considerMatch decides what to do with one discovered match and does it. Errors are
// counted and logged rather than returned: one unarchivable match must not end the pass.
func considerMatch(ctx context.Context, d deps, gamertag string,
	entry haloclient.MatchHistoryEntry, sum *watchSummary) {
	known, reason, err := alreadySettled(ctx, d, entry.MatchID)
	if err != nil {
		sum.Failed++
		slog.ErrorContext(ctx, "study-archiver: could not read the archive row of a discovered match",
			"err", err, "match_id", entry.MatchID)
		return
	}
	if known {
		sum.Skipped++
		slog.DebugContext(ctx, "study-archiver: match skipped by the watch loop",
			"match_id", entry.MatchID, "gamertag", gamertag, "reason", reason)
		return
	}

	// The stats payload is read HERE, to judge the shape, and handed to the archiver so
	// the pass does not pay for it twice.
	stats, err := d.Client.GetMatchStats(ctx, entry.MatchID)
	if err != nil {
		sum.Failed++
		slog.ErrorContext(ctx, "study-archiver: match stats unreadable - not archived this pass",
			"err", err, "match_id", entry.MatchID)
		return
	}
	facts, err := readMatchFacts(stats, gamertag)
	if err != nil {
		sum.Failed++
		slog.ErrorContext(ctx, "study-archiver: match stats unusable - not archived this pass",
			"err", err, "match_id", entry.MatchID)
		return
	}
	if !isArenaFourVFour(facts.Roster) {
		sum.Skipped++
		slog.InfoContext(ctx, "study-archiver: not a 4v4 - skipped",
			"match_id", entry.MatchID, "gamertag", gamertag, "map", facts.MapName,
			"mode", facts.Mode, "playlist", facts.Playlist, "players", len(facts.Roster))
		return
	}

	// SourceGamertag is per-match here: it records WHOSE pass surfaced the match, which is
	// what makes "archived matches by tracked player" answerable in #9.
	perMatch := d
	perMatch.SourceGamertag = gamertag
	out, err := fetchOneWithStats(ctx, perMatch, entry.MatchID, stats)
	switch {
	case err != nil:
		sum.Failed++
		slog.ErrorContext(ctx, "study-archiver: archiving failed - continuing the pass",
			"err", err, "match_id", entry.MatchID, "gamertag", gamertag)
	case out.SkipReason != "":
		sum.Skipped++
	default:
		sum.Archived++
	}
}

// alreadySettled reports whether the watch loop should leave a match alone, and why.
//
// The reason is returned for the log rather than derived at the call site: "skipped" with
// no reason is the log line that makes an operator open a SQL client.
func alreadySettled(ctx context.Context, d deps, matchID string) (bool, string, error) {
	rec, found, err := d.Archive.recorded(ctx, matchID)
	if err != nil || !found {
		return false, "", err
	}
	switch {
	case rec.ArtifactPath != "":
		return true, "already archived", nil
	case rec.State.terminal():
		return true, "film expired - never retried", nil
	case rec.State == stateFailed:
		// See the file header: skipped HERE, not everywhere. `rebuild` (#10) is the retry.
		return true, "build failed - rebuild it deliberately, not hourly", nil
	}
	// `downloaded` with no artifact: an unsupported map whose bounds may have arrived, or
	// stats that named no map. Both are worth another try.
	return false, "", nil
}

// isArenaFourVFour reports whether a roster is two teams of four.
//
// Read off the ROSTER rather than the playlist because that is the fact that matters to
// everything downstream: a replay of a 12v12 is unreadable at this scale, and a heat map
// that mixes team sizes compares nothing to nothing.
func isArenaFourVFour(roster []participantRecord) bool {
	if len(roster) != arenaTeams*arenaTeamSize {
		return false
	}
	perTeam := make(map[int]int, arenaTeams)
	for _, p := range roster {
		if p.Team == nil {
			// A roster entry with no team is not a 4v4 that can be reasoned about: the
			// team is what every downstream reading is sliced by.
			return false
		}
		perTeam[*p.Team]++
	}
	if len(perTeam) != arenaTeams {
		return false
	}
	for _, n := range perTeam {
		if n != arenaTeamSize {
			return false
		}
	}
	return true
}

// resolveWatched returns the xuid of a tracked gamertag, resolving it at most once ever.
//
// The resolution is recorded BEFORE the pass proceeds, so that a run interrupted halfway
// does not throw away the one Xbox Live round trip it made.
func resolveWatched(ctx context.Context, d deps, w watchDeps,
	gamertag string, player watchedPlayer) (string, error) {
	if player.XUID != "" {
		return player.XUID, nil
	}
	if w.ResolveXUID == nil {
		return "", fmt.Errorf("no xuid recorded for %s and no resolver wired", gamertag)
	}
	xuid, err := w.ResolveXUID(ctx, gamertag)
	if err != nil {
		return "", fmt.Errorf("resolving %s to an xuid: %w", gamertag, err)
	}
	if xuid == "" {
		return "", fmt.Errorf("resolving %s to an xuid: the profile carries none", gamertag)
	}
	if err := d.Archive.rememberWatched(ctx, gamertag, xuid); err != nil {
		return "", err
	}
	slog.InfoContext(ctx, "study-archiver: gamertag resolved and recorded - not resolved again",
		"gamertag", gamertag, "xuid", xuid)
	return xuid, nil
}

// lastPassAge says how long ago a tracked player was last checked.
//
// A STRING, and "never" spelled out, because the alternative — a zero duration — reads in
// the log exactly like "checked a moment ago". Telling a quiet pass ("nothing new") from a
// job that has not run since Tuesday is the whole reason last_checked is recorded.
func lastPassAge(p watchedPlayer, now time.Time) string {
	if p.LastChecked == nil {
		return "never"
	}
	return now.Sub(*p.LastChecked).Round(time.Second).String() + " ago"
}
