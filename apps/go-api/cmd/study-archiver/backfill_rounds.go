package main

// backfill_rounds.go — ROUND COUNTS FOR MATCHES ARCHIVED BEFORE THE ARCHIVE RECORDED THEM.
//
// The ground truth expects deaths + rounds lives per player (groundtruth.go). A roster recorded
// before `participants.rounds` existed holds NULL there, which reads as one round: right for
// Slayer, CTF and Zones, wrong for every Oddball match, where each round starts a life with no
// death behind it. Neither existing path can repair it: `rebuild` never touches the network, and
// fetchOne returns early on a match whose artifact is already on disk.
//
// `backfill-rounds` reads the match stats ONCE per such match, records each player's rounds, and
// re-grades the ground truth from the artifact already on disk. Nothing is decoded again, so the
// comparison stays the one for the build that is published.

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"levelup/go-api/internal/analysis/replay"
)

// maxConsecutiveBackfillFailures stops a run whose failures are no longer about one match, for
// the reason recapture.go gives.
const maxConsecutiveBackfillFailures = 5

// backfillSummary is what one backfill-rounds run did.
type backfillSummary struct {
	Candidates int
	Updated    int
	Failed     int
	// Stopped is true when the run gave up after maxConsecutiveBackfillFailures errors.
	Stopped bool
}

// matchesMissingRounds lists the archived matches with at least one player whose round count is
// not recorded.
func (a *archive) matchesMissingRounds(ctx context.Context) ([]string, error) {
	rows, err := a.db.SQLDb().QueryContext(ctx, `
        SELECT DISTINCT m.match_id FROM matches m
        JOIN participants p ON p.match_id = m.match_id
        WHERE m.artifact_path IS NOT NULL AND p.rounds IS NULL
        ORDER BY m.match_id`)
	if err != nil {
		return nil, fmt.Errorf("listing matches without round counts: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning a match without round counts: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// recordRounds writes the round count of every roster player the stats gave one for, and returns
// how many rows it wrote.
func (a *archive) recordRounds(ctx context.Context, matchID string, roster []participantRecord) (int, error) {
	tx, err := a.db.SQLDb().BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("archive transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op once committed
	n := 0
	for _, p := range roster {
		if p.Rounds == nil {
			continue
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE participants SET rounds = ? WHERE match_id = ? AND xuid = ?`,
			*p.Rounds, matchID, p.XUID); err != nil {
			return 0, fmt.Errorf("recording the rounds of %s in %s: %w", p.XUID, matchID, err)
		}
		n++
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("archive commit (%s): %w", matchID, err)
	}
	return n, nil
}

// updateGroundTruth refreshes only a match's comparison columns and its players' replay lives.
func (a *archive) updateGroundTruth(ctx context.Context, matchID string, gt groundTruth) error {
	cols := groundTruthColumns(gt)
	tx, err := a.db.SQLDb().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("archive transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op once committed
	if _, err := tx.ExecContext(ctx, `
        UPDATE matches SET
            gt_players = ?, gt_expected_lives = ?, gt_named_lives = ?, gt_over_named = ?,
            gt_missing_lives = ?, gt_unknown_named = ?, gt_lives_gap = ?
        WHERE match_id = ?`,
		cols.Players, cols.ExpectedLives, cols.NamedLives, cols.OverNamed,
		cols.MissingLives, cols.UnknownNamed, cols.LivesGap, matchID); err != nil {
		return fmt.Errorf("re-grading %s: %w", matchID, err)
	}
	if err := writeReplayLives(ctx, tx, matchID, gt); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("archive commit (%s): %w", matchID, err)
	}
	return nil
}

// backfillRoundsPass records rounds and re-grades every archived match missing them.
func backfillRoundsPass(ctx context.Context, d deps, limit int) backfillSummary {
	var sum backfillSummary
	ids, err := d.Archive.matchesMissingRounds(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "study-archiver: backfill-rounds could not list its matches", "err", err)
		sum.Failed++
		return sum
	}
	if limit > 0 && len(ids) > limit {
		ids = ids[:limit]
	}
	sum.Candidates = len(ids)
	consecutive := 0
	for _, id := range ids {
		if err := backfillOne(ctx, d, id); err != nil {
			sum.Failed++
			consecutive++
			slog.ErrorContext(ctx, "study-archiver: backfill-rounds failed on a match", "match_id", id, "err", err)
			if consecutive >= maxConsecutiveBackfillFailures {
				sum.Stopped = true
				slog.ErrorContext(ctx, "study-archiver: backfill-rounds stopped - consecutive failures",
					"failures", consecutive)
				break
			}
			continue
		}
		consecutive = 0
		sum.Updated++
	}
	slog.InfoContext(ctx, "study-archiver: backfill-rounds done", "candidates", sum.Candidates,
		"updated", sum.Updated, "failed", sum.Failed, "stopped", sum.Stopped)
	return sum
}

// backfillOne reads one match's stats, records its rounds and re-grades it from its artifact.
func backfillOne(ctx context.Context, d deps, matchID string) error {
	stats, err := d.Client.GetMatchStats(ctx, matchID)
	if err != nil {
		return fmt.Errorf("match stats: %w", err)
	}
	facts, err := readMatchFacts(stats, d.SourceGamertag)
	if err != nil {
		return err
	}
	written, err := d.Archive.recordRounds(ctx, matchID, facts.Roster)
	if err != nil {
		return err
	}
	rec, found, err := d.Archive.recorded(ctx, matchID)
	if err != nil {
		return err
	}
	if !found || rec.ArtifactPath == "" {
		return fmt.Errorf("match %s has no artifact to re-grade", matchID)
	}
	raw, err := os.ReadFile(rec.ArtifactPath)
	if err != nil {
		return fmt.Errorf("reading the artifact of %s: %w", matchID, err)
	}
	var doc replay.ReplayDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("decoding the artifact of %s: %w", matchID, err)
	}
	roster, err := d.Archive.roster(ctx, matchID)
	if err != nil {
		return err
	}
	gt := compareGroundTruth(doc, roster)
	if err := d.Archive.updateGroundTruth(ctx, matchID, gt); err != nil {
		return err
	}
	slog.InfoContext(ctx, "study-archiver: rounds recorded and ground truth re-graded",
		"match_id", matchID, "players_with_rounds", written, "expected_lives", gt.ExpectedLives,
		"named_lives", gt.NamedLives, "over_named", gt.OverNamed, "lives_gap", gt.LivesGap)
	return nil
}
