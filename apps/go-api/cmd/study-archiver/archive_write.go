package main

// archive_write.go — RECORDING ONE MATCH.
//
// SELECT-THEN-UPDATE-OR-INSERT, ROW BY ROW. Not `ON CONFLICT DO UPDATE`, not
// `INSERT OR REPLACE`, and NOT delete-then-reinsert.
//
// This was first written as delete-then-reinsert, justified by "single writer, no
// concurrency, so the ART index bug cannot bite". THAT JUSTIFICATION IS FALSE, and the
// repo had already written down why: `internal/sync/no_art_patterns_test.go` records that
// `compactMatchSkillRankSuperseded` "déclenchait le bug ART #23046 malgré mono-writer +
// PK BIGINT (crash JGtm 2026-06-20)", and admits a raw DELETE into its allowlist only
// with "PK BIGINT, pas VARCHAR". Both of this archive's keys are VARCHAR. Being the only
// writer is not the mitigation it looks like — DuckDB issue #23046 is about index
// maintenance during row removal, and it does not care how many processes are watching.
//
// What the same file calls SAFE is the shape used here: "leurs UPDATE bitmask /
// row-by-row sérialisés ... sont sûrs". CLAUDE.md names it directly as the pattern for a
// database whose rows must be refreshed in place. So no row is ever removed to be written
// again; an existing row is updated in place, and a new one inserted.
//
// The one place a row IS removed is the roster prune below, and it is deliberately
// narrowed to the rows that must actually go — normally none.
//
// The transaction is what makes the pair atomic: a crash between the match row and its
// roster would otherwise leave a match recorded with somebody else's players.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// recordMatch writes a match and its roster, refreshing anything previously recorded.
func (a *archive) recordMatch(ctx context.Context, rec matchRecord, roster []participantRecord) error {
	tx, err := a.db.SQLDb().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("archive transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op once committed

	if err := writeMatchRow(ctx, tx, rec); err != nil {
		return err
	}
	if err := writeRoster(ctx, tx, rec.MatchID, roster); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("archive commit (%s): %w", rec.MatchID, err)
	}
	return nil
}

// matchColumnValues is the column order shared by the INSERT and the UPDATE, so the two
// statements cannot drift apart into recording different things.
func matchColumnValues(rec matchRecord) []any {
	return []any{
		rec.ShortID, rec.PlayedAt,
		nullString(rec.MapName), nullString(rec.MapModule),
		nullString(rec.Mode), nullString(rec.Playlist),
		rec.DurationMS, nullString(rec.SourceGT),
		string(rec.State), nullString(string(rec.SkipReason)),
		nullString(rec.ArtifactPath), rec.BuiltAt, nullString(rec.DecoderRev),
		rec.Tracks, rec.Points, rec.Shots, rec.NamedLives, rec.TotalLives,
	}
}

func writeMatchRow(ctx context.Context, tx *sql.Tx, rec matchRecord) error {
	exists, err := rowExists(ctx, tx, `SELECT 1 FROM matches WHERE match_id = ?`, rec.MatchID)
	if err != nil {
		return fmt.Errorf("looking up match %s: %w", rec.MatchID, err)
	}
	args := matchColumnValues(rec)
	if exists {
		// Every column is refreshed: the archiver re-derives all of them from the same
		// two sources, so a partial update would leave a mix of two readings.
		args = append(args, rec.MatchID)
		_, err = tx.ExecContext(ctx, `
            UPDATE matches SET
                short_id = ?, played_at = ?, map_name = ?, map_module = ?, mode = ?,
                playlist = ?, duration_ms = ?, source_gamertag = ?, film_state = ?,
                skip_reason = ?, artifact_path = ?, built_at = ?, decoder_rev = ?,
                tracks = ?, points = ?, shots = ?, named_lives = ?, total_lives = ?,
                recorded_at = now()
            WHERE match_id = ?`, args...)
	} else {
		args = append([]any{rec.MatchID}, args...)
		_, err = tx.ExecContext(ctx, `
            INSERT INTO matches (
                match_id, short_id, played_at, map_name, map_module, mode, playlist,
                duration_ms, source_gamertag, film_state, skip_reason, artifact_path,
                built_at, decoder_rev, tracks, points, shots, named_lives, total_lives,
                recorded_at
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, now())`, args...)
	}
	if err != nil {
		return fmt.Errorf("recording match %s: %w", rec.MatchID, err)
	}
	return nil
}

// updateBuild refreshes ONLY the columns a build produced, on a match already recorded.
//
// WHY NOT recordMatch. That one writes every column, from a matchRecord assembled out of a
// fresh reading of the match stats. A rebuild (#10) has no such reading — it never touches
// the network — and `archive.recorded` is a PARTIAL reader by design, so round-tripping
// through recordMatch would blank mode, playlist, played-at and source_gamertag on every
// rebuild. Those facts belong to the match, not to the build, and a rebuild has nothing new
// to say about them.
//
// The roster is untouched for the same reason: a rebuild cannot have changed who played.
func (a *archive) updateBuild(ctx context.Context, out outcome, builtAt *time.Time, decoderRev string) error {
	res, err := a.db.Exec(ctx, `
        UPDATE matches SET
            film_state = ?, skip_reason = ?, artifact_path = ?, built_at = ?, decoder_rev = ?,
            tracks = ?, points = ?, shots = ?, named_lives = ?, total_lives = ?,
            recorded_at = now()
        WHERE match_id = ?`,
		string(filmStateOf(out)), nullString(string(out.SkipReason)),
		nullString(out.ArtifactPath), builtAt, nullString(decoderRev),
		out.Tracks, out.Points, out.Shots, out.NamedLives, out.TotalLives,
		out.MatchID)
	if err != nil {
		return fmt.Errorf("recording the rebuild of %s: %w", out.MatchID, err)
	}
	// A rebuild that matched no row would otherwise report success having changed nothing:
	// the caller checked the match was recorded, so zero rows here means it went away
	// underneath us, which the operator has to be told about rather than left to infer.
	if n, rErr := res.RowsAffected(); rErr == nil && n == 0 {
		return fmt.Errorf("recording the rebuild of %s: the archive row disappeared mid-run", out.MatchID)
	}
	return nil
}

// writeRoster refreshes each player in place and removes only those a re-read no longer
// reports.
func writeRoster(ctx context.Context, tx *sql.Tx, matchID string, roster []participantRecord) error {
	keep := make(map[string]bool, len(roster))
	for _, p := range roster {
		keep[p.XUID] = true
		if err := writeParticipant(ctx, tx, matchID, p); err != nil {
			return err
		}
	}
	return pruneRoster(ctx, tx, matchID, keep)
}

func writeParticipant(ctx context.Context, tx *sql.Tx, matchID string, p participantRecord) error {
	exists, err := rowExists(ctx, tx,
		`SELECT 1 FROM participants WHERE match_id = ? AND xuid = ?`, matchID, p.XUID)
	if err != nil {
		return fmt.Errorf("looking up participant %s of %s: %w", p.XUID, matchID, err)
	}
	if exists {
		_, err = tx.ExecContext(ctx, `
            UPDATE participants SET gamertag = ?, team = ?, outcome = ?,
                   kills = ?, deaths = ?, assists = ?
            WHERE match_id = ? AND xuid = ?`,
			nullString(p.Gamertag), p.Team, p.Outcome, p.Kills, p.Deaths, p.Assists,
			matchID, p.XUID)
	} else {
		_, err = tx.ExecContext(ctx, `
            INSERT INTO participants (match_id, xuid, gamertag, team, outcome, kills, deaths, assists)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			matchID, p.XUID, nullString(p.Gamertag),
			p.Team, p.Outcome, p.Kills, p.Deaths, p.Assists)
	}
	if err != nil {
		return fmt.Errorf("recording participant %s of %s: %w", p.XUID, matchID, err)
	}
	return nil
}

// pruneRoster removes players a re-read no longer reports.
//
// NORMALLY A NO-OP: a match's roster is fixed by the match, so the set only shrinks when
// an earlier, wrong read recorded somebody who was not there. The rows to remove are
// therefore identified FIRST and deleted one by one, rather than clearing the roster and
// writing it back — the whole point of the pattern in this file's header is that rows are
// not removed just to be written again.
func pruneRoster(ctx context.Context, tx *sql.Tx, matchID string, keep map[string]bool) error {
	rows, err := tx.QueryContext(ctx, `SELECT xuid FROM participants WHERE match_id = ?`, matchID)
	if err != nil {
		return fmt.Errorf("listing the roster of %s: %w", matchID, err)
	}
	var stale []string
	for rows.Next() {
		var xuid string
		if err := rows.Scan(&xuid); err != nil {
			_ = rows.Close()
			return fmt.Errorf("scanning the roster of %s: %w", matchID, err)
		}
		if !keep[xuid] {
			stale = append(stale, xuid)
		}
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("reading the roster of %s: %w", matchID, err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("closing the roster cursor of %s: %w", matchID, err)
	}
	for _, xuid := range stale {
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM participants WHERE match_id = ? AND xuid = ?`, matchID, xuid); err != nil {
			return fmt.Errorf("removing stale participant %s of %s: %w", xuid, matchID, err)
		}
	}
	return nil
}

// rowExists reports whether a single-row lookup found anything.
func rowExists(ctx context.Context, tx *sql.Tx, query string, args ...any) (bool, error) {
	var one int
	err := tx.QueryRowContext(ctx, query, args...).Scan(&one)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return false, nil
	case err != nil:
		return false, err
	}
	return true, nil
}

// nullString maps an empty string to SQL NULL. An absent map module and a map module
// that is the empty string are the same fact, and only one of them is queryable.
func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
