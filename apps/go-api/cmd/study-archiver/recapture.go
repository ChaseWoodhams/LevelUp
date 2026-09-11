package main

// recapture.go — FETCHING AGAIN WHAT THE ARCHIVE RECORDED BUT NEVER BUILT.
//
// `watch` reads only the recent history (maxHistoryPages), so a match it recorded without an
// artifact - a map with no bounds yet, a map name the archiver could not resolve at the time, a
// film cache emptied since - falls out of its window and is never looked at again, while its film
// keeps ageing towards expiry. `recapture` walks the ARCHIVE instead of the history: every
// recorded match with no artifact whose film is neither EXPIRED (gone for good) nor FAILED (its
// chunks are on disk waiting for a decoder fix; `rebuild` is the command for those) goes back
// through fetchOne.
//
// fetchOne already does the right thing with each one: it re-reads the stats (so a map name
// resolved today replaces the GUID recorded then), reuses a film still in the chunk cache and
// re-downloads one that is not, and records an expired film as expired rather than retrying it
// forever.
//
// OLDEST FIRST: films expire by age, so a limited or interrupted run spends its time on the
// matches closest to being lost.

import (
	"context"
	"fmt"
	"log/slog"
)

// maxConsecutiveRecaptureFailures stops a run whose failures are no longer about one match. Five
// errors in a row is a dead token or a network outage, and carrying on would only turn the rest
// of the list into the same error.
const maxConsecutiveRecaptureFailures = 5

// recaptureSummary is what one recapture run did.
type recaptureSummary struct {
	Candidates int
	Archived   int
	Skipped    int
	Failed     int
	// Stopped is true when the run gave up after maxConsecutiveRecaptureFailures errors.
	Stopped bool
	// Reasons counts the skips by named reason: what still blocks a match.
	Reasons map[reason]int
}

// unbuiltMatchIDs lists the recorded matches a recapture fetches again, oldest first.
func (a *archive) unbuiltMatchIDs(ctx context.Context) ([]string, error) {
	rows, err := a.db.SQLDb().QueryContext(ctx, `
        SELECT match_id FROM matches
        WHERE artifact_path IS NULL AND film_state NOT IN (?, ?)
        ORDER BY played_at ASC NULLS LAST, match_id`,
		string(stateExpired), string(stateFailed))
	if err != nil {
		return nil, fmt.Errorf("listing unbuilt matches: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning an unbuilt match: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading unbuilt matches: %w", err)
	}
	return ids, nil
}

// recapturePass sends every unbuilt, non-terminal match back through fetchOne. limit > 0 caps
// the run to that many matches, oldest first.
func recapturePass(ctx context.Context, d deps, limit int) recaptureSummary {
	sum := recaptureSummary{Reasons: map[reason]int{}}
	ids, err := d.Archive.unbuiltMatchIDs(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "study-archiver: recapture could not list its matches", "err", err)
		sum.Failed++
		return sum
	}
	if limit > 0 && len(ids) > limit {
		ids = ids[:limit]
	}
	sum.Candidates = len(ids)
	consecutive := 0
	for i, id := range ids {
		if ctx.Err() != nil {
			break
		}
		out, err := fetchOne(ctx, d, id)
		switch {
		case err != nil:
			sum.Failed++
			consecutive++
			slog.ErrorContext(ctx, "study-archiver: recapture of one match failed - continuing",
				"err", err, "match_id", id, "done", i+1, "of", len(ids))
		case out.SkipReason != "":
			sum.Skipped++
			sum.Reasons[out.SkipReason]++
			consecutive = 0
		default:
			sum.Archived++
			consecutive = 0
		}
		if consecutive >= maxConsecutiveRecaptureFailures {
			sum.Stopped = true
			slog.ErrorContext(ctx, "study-archiver: recapture stopped - consecutive failures are "+
				"no longer about one match (token or network)", "failures", consecutive,
				"done", i+1, "of", len(ids))
			break
		}
	}
	slog.InfoContext(ctx, "study-archiver: recapture done",
		"candidates", sum.Candidates, "archived", sum.Archived, "skipped", sum.Skipped,
		"failed", sum.Failed, "stopped", sum.Stopped, "skip_reasons", fmt.Sprint(sum.Reasons))
	return sum
}
