package main

// archive_watchlist.go — WHAT THE ARCHIVE REMEMBERS ABOUT THE PLAYERS IT FOLLOWS.
//
// The `watchlist` table has existed since #6, deliberately created empty so that filling
// it here would not be a schema migration. It carries two facts, and they are recorded for
// two different reasons:
//
//   - the XUID a gamertag resolves to, because resolving costs an Xbox Live round trip on
//     a token chain the archiver otherwise never needs, and the answer does not change;
//   - LAST CHECKED, because that is how an operator (and #9's `status`) tells a watch loop
//     that is finding nothing from one that has not run since Tuesday. Those look
//     identical in an archive that only records what it captured.
//
// Writes follow the same SELECT-then-UPDATE-or-INSERT discipline as the rest of the
// archive (archive_write.go): no row is removed to be written again.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// watchedPlayer is one row of `watchlist`.
type watchedPlayer struct {
	Gamertag string
	// XUID is empty until the gamertag has been resolved once.
	XUID string
	// LastChecked is nil until a pass has completed for this player.
	LastChecked *time.Time
}

// watched reads back what the archive knows about one tracked gamertag.
func (a *archive) watched(ctx context.Context, gamertag string) (watchedPlayer, bool, error) {
	var (
		p      = watchedPlayer{Gamertag: gamertag}
		xuid   sql.NullString
		last   sql.NullTime
		lookup = `SELECT xuid, last_checked FROM watchlist WHERE lower(gamertag) = lower(?)`
	)
	// Matched case-insensitively because Xbox gamertags are: a watchlist re-typed with
	// different capitalisation must find the resolution the archive already holds rather
	// than resolve the same player again under a second row.
	err := a.db.QueryRow(ctx, lookup, gamertag).Scan(&xuid, &last)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return watchedPlayer{}, false, nil
	case err != nil:
		return watchedPlayer{}, false, fmt.Errorf("reading the watchlist row of %s: %w", gamertag, err)
	}
	p.XUID = xuid.String
	if last.Valid {
		t := last.Time
		p.LastChecked = &t
	}
	return p, true, nil
}

// rememberWatched records a tracked gamertag and, once known, its xuid.
//
// added_at is written ONLY on insert: it says when the archive first followed this player,
// and refreshing it on every pass would turn a durable fact into "now", every hour.
func (a *archive) rememberWatched(ctx context.Context, gamertag, xuid string) error {
	exists, err := a.rowExistsDB(ctx,
		`SELECT 1 FROM watchlist WHERE lower(gamertag) = lower(?)`, gamertag)
	if err != nil {
		return fmt.Errorf("looking up the watchlist row of %s: %w", gamertag, err)
	}
	if exists {
		// COALESCE keeps a known xuid when the caller has none to offer: a resolution that
		// failed this run must not erase the one an earlier run succeeded at.
		_, err = a.db.Exec(ctx, `
            UPDATE watchlist SET gamertag = ?, xuid = COALESCE(?, xuid)
            WHERE lower(gamertag) = lower(?)`, gamertag, nullString(xuid), gamertag)
	} else {
		_, err = a.db.Exec(ctx, `
            INSERT INTO watchlist (gamertag, xuid, added_at, last_checked)
            VALUES (?, ?, now(), NULL)`, gamertag, nullString(xuid))
	}
	if err != nil {
		return fmt.Errorf("recording the watchlist entry of %s: %w", gamertag, err)
	}
	return nil
}

// markChecked stamps a completed pass on a tracked player.
func (a *archive) markChecked(ctx context.Context, gamertag string) error {
	if _, err := a.db.Exec(ctx,
		`UPDATE watchlist SET last_checked = now() WHERE lower(gamertag) = lower(?)`,
		gamertag); err != nil {
		return fmt.Errorf("stamping the watchlist check of %s: %w", gamertag, err)
	}
	return nil
}

// rowExistsDB is rowExists outside a transaction: the watchlist writes are single
// statements, so they do not need one.
func (a *archive) rowExistsDB(ctx context.Context, query string, args ...any) (bool, error) {
	var one int
	err := a.db.QueryRow(ctx, query, args...).Scan(&one)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return false, nil
	case err != nil:
		return false, err
	}
	return true, nil
}
