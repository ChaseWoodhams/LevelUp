package main

// status.go — WHAT IS IN THE ARCHIVE, AND WHAT WENT WRONG (#9).
//
// The health of an unattended hourly job, in one command, without reading logs or opening
// a SQL client. That is the whole point: a `watch` loop that quietly stopped capturing
// looks exactly like a `watch` loop with nothing to capture, and the only thing that tells
// them apart is a report somebody can read at a glance.
//
// READ-ONLY, AND SAFE TO RUN DURING A PASS. The handle comes from
// `duckdb.OpenReadForQuery`, never a forced `OpenReadOnly`: in the same process it reuses
// an existing handle (DuckDB refuses a read-only handle on a file already held read-write
// there), and across processes it opens READ_ONLY beside the archiver's writer. That is
// CLAUDE.md's ART rule 4 and ADR 0013/0016, and it is what makes "run it while a capture is
// in progress" true rather than hopeful.
//
// THE REPORT GOES TO STDOUT, the logs to slog on stderr — the split every other `cmd/`
// tool here uses. Repo rule 3 forbids `fmt.Println` as a LOGGING mechanism; this is the
// tool's output, and rendering an aligned table through a key=value log handler would
// defeat the one criterion this ticket has, which is that it be readable at a glance.

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"levelup/go-api/internal/domain/title"
	ddb "levelup/go-api/internal/platform/duckdb"
)

// countedRow is one line of a breakdown: a label and how many matches carry it.
type countedRow struct {
	Label string
	N     int
}

// watchlistStatus is one tracked player's freshness.
type watchlistStatus struct {
	Gamertag    string
	XUID        string
	LastChecked *time.Time
}

// statusReport is the whole answer, gathered before anything is printed.
//
// A VALUE, rendered separately, so the counts can be asserted in a test without capturing
// stdout and re-parsing a table — the assertions would then be about the formatting rather
// than about the archive.
type statusReport struct {
	Path string
	// Recorded is every match the archive knows; Archived only those that produced an
	// artifact. The gap between them is the interesting part of this report.
	Recorded int
	Archived int
	// Failed and Expired are counted SEPARATELY and never summed into one "errors"
	// number: one of them is waiting for a decoder fix, the other is gone forever.
	Failed  int
	Expired int
	// Captured counts films on disk that produced NO artifact — almost always a map
	// whose quant bounds have not arrived yet. Deliberately not "downloaded", which
	// would also count every successfully archived match and read as a backlog that
	// does not exist.
	Captured int
	Pending  int
	ByMap    []countedRow
	ByMode   []countedRow
	ByPlayer []countedRow
	// UnArchived breaks down, by named reason, every match that produced no artifact.
	UnArchived []countedRow
	Watchlist  []watchlistStatus
}

// readStatus gathers the report from an open read-only handle.
func readStatus(ctx context.Context, db *sql.DB, path string) (statusReport, error) {
	rep := statusReport{Path: path}
	if err := db.QueryRowContext(ctx, `
        SELECT count(*),
               count(artifact_path),
               count(*) FILTER (WHERE film_state = 'failed'),
               count(*) FILTER (WHERE film_state = 'expired'),
               count(*) FILTER (WHERE film_state = 'downloaded' AND artifact_path IS NULL),
               count(*) FILTER (WHERE film_state = 'pending')
        FROM matches`).
		Scan(&rep.Recorded, &rep.Archived, &rep.Failed, &rep.Expired,
			&rep.Captured, &rep.Pending); err != nil {
		return rep, fmt.Errorf("counting matches: %w", err)
	}

	// The three breakdowns count ARCHIVED matches: "what have I actually got to study?".
	// A match that failed to build is reported below, under its reason, not as coverage.
	for _, b := range []struct {
		column string
		into   *[]countedRow
	}{
		{"map_name", &rep.ByMap},
		{"mode", &rep.ByMode},
		{"source_gamertag", &rep.ByPlayer},
	} {
		rows, err := countBy(ctx, db, b.column)
		if err != nil {
			return rep, err
		}
		*b.into = rows
	}

	var err error
	if rep.UnArchived, err = countUnArchived(ctx, db); err != nil {
		return rep, err
	}
	rep.Watchlist, err = readWatchlistStatus(ctx, db)
	return rep, err
}

// countBy groups archived matches by one column. The column name is interpolated because
// SQL cannot parameterise an identifier — and it is safe here because the only values it
// ever takes are the three literals in the caller's table, never anything from outside.
func countBy(ctx context.Context, db *sql.DB, column string) ([]countedRow, error) {
	query := fmt.Sprintf(`
        SELECT coalesce(nullif(%s, ''), '(unrecorded)') AS label, count(*) AS n
        FROM matches WHERE artifact_path IS NOT NULL
        GROUP BY label ORDER BY n DESC, label`, column)
	rows, err := db.QueryContext(ctx, query) //nolint:gosec // identifier from a fixed set, cf. above
	if err != nil {
		return nil, fmt.Errorf("counting by %s: %w", column, err)
	}
	return scanCounted(rows, column)
}

// countUnArchived breaks down what produced no artifact, by its named reason.
func countUnArchived(ctx context.Context, db *sql.DB) ([]countedRow, error) {
	rows, err := db.QueryContext(ctx, `
        SELECT coalesce(nullif(skip_reason, ''), '(no reason recorded)') AS label, count(*) AS n
        FROM matches WHERE artifact_path IS NULL
        GROUP BY label ORDER BY n DESC, label`)
	if err != nil {
		return nil, fmt.Errorf("counting unarchived matches: %w", err)
	}
	return scanCounted(rows, "skip_reason")
}

func scanCounted(rows *sql.Rows, what string) ([]countedRow, error) {
	defer func() { _ = rows.Close() }()
	var out []countedRow
	for rows.Next() {
		var r countedRow
		if err := rows.Scan(&r.Label, &r.N); err != nil {
			return nil, fmt.Errorf("scanning the %s breakdown: %w", what, err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading the %s breakdown: %w", what, err)
	}
	return out, nil
}

func readWatchlistStatus(ctx context.Context, db *sql.DB) ([]watchlistStatus, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT gamertag, xuid, last_checked FROM watchlist ORDER BY lower(gamertag)`)
	if err != nil {
		return nil, fmt.Errorf("reading the watchlist: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []watchlistStatus
	for rows.Next() {
		var (
			w    watchlistStatus
			xuid sql.NullString
			last sql.NullTime
		)
		if err := rows.Scan(&w.Gamertag, &xuid, &last); err != nil {
			return nil, fmt.Errorf("scanning a watchlist row: %w", err)
		}
		w.XUID = xuid.String
		if last.Valid {
			t := last.Time
			w.LastChecked = &t
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// openStatusReport opens the archive for reading and gathers the report.
func openStatusReport(ctx context.Context, paths *title.PathResolver) (statusReport, error) {
	path := paths.StudyArchiveDBPath()
	db, release, err := ddb.OpenReadForQuery(path)
	if err != nil {
		return statusReport{}, fmt.Errorf("opening the archive %s for reading: %w", path, err)
	}
	defer release()
	return readStatus(ctx, db, path)
}
