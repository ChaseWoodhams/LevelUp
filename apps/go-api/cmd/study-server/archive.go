package main

// archive.go — READING THE ARCHIVE, AND ONLY READING IT.
//
// The archiver (cmd/study-archiver) declares the archive's shape and is its only writer.
// This server never creates a table, never writes a row, and holds no lease: it opens the
// file through `duckdb.OpenReadForQuery` — the repo's rule for reading a database somebody
// else may be holding read-write (CLAUDE.md ART rule 4, ADR 0013/0016).
//
// NOT A FORCED `OpenReadOnly`, which is the mistake that reads as safer and is not: DuckDB
// refuses a read-only handle on a file already held read-write in the same process, and
// across processes OpenReadForQuery opens READ_ONLY anyway. That is what makes "browse the
// archive while an hourly capture is running" true rather than hopeful — the property the
// ticket asks for, and the same one `study-archiver status` already relies on.
//
// EVERY IDENTIFIER THE SERVER PUTS ON DISK COMES BACK OUT OF THIS FILE. A path is only ever
// built from a `short_id` read off a row (cf. artifact.go), never from the URL, so the
// artifact route cannot be talked into resolving a path of the caller's choosing.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	ddb "levelup/go-api/internal/platform/duckdb"
)

// errNoArchive is "nothing has been captured on this machine yet" — an operator state, not a
// failure of this run, and the first thing a new user sees.
var errNoArchive = errors.New("no study archive on this machine")

// errMatchUnknown is "this archive holds no servable match under that identifier". It covers
// both an id nobody ever captured and a match recorded without an artifact: neither can be
// replayed, and the caller has nothing different to do about them.
var errMatchUnknown = errors.New("unknown match")

// archive is an open read handle on the archive database.
type archive struct {
	db      *sql.DB
	release func()
	path    string
}

// openArchive opens the archive for reading.
func openArchive(path string) (*archive, error) {
	// Checked BEFORE opening: DuckDB's own failure on an absent file names the driver and
	// the path and nothing an operator can act on, and this is the state every machine is
	// in until the first capture runs.
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("%w at %s: run `study-archiver watch` or `fetch-one` to create it",
			errNoArchive, path)
	}
	db, release, err := ddb.OpenReadForQuery(path)
	if err != nil {
		return nil, fmt.Errorf("opening the archive %s for reading: %w", path, err)
	}
	return &archive{db: db, release: release, path: path}, nil
}

// Close releases the handle. It closes nothing if the handle was borrowed from a writer
// already open in this process, which is OpenReadForQuery's contract.
func (a *archive) Close() {
	if a != nil && a.release != nil {
		a.release()
	}
}

// matchSummary is one row of the archive browser: what was played, and what the decoder got
// out of it. Everything here is a fact the archiver recorded.
type matchSummary struct {
	MatchID   string     `json:"match_id"`
	ShortID   string     `json:"short_id"`
	PlayedAt  *time.Time `json:"played_at,omitempty"`
	MapName   string     `json:"map_name,omitempty"`
	MapModule string     `json:"map_module,omitempty"`
	Mode      string     `json:"mode,omitempty"`
	Playlist  string     `json:"playlist,omitempty"`
	// DurationMS is milliseconds, the replay document's own clock, so the two can be
	// compared without a unit change.
	DurationMS *int64 `json:"duration_ms,omitempty"`
	// SourceGamertag is whose archiving pass DISCOVERED the match. It is NOT "whose match
	// this is": when two tracked players meet, the first pass takes credit for the game.
	// Filter by player instead; this field is provenance, not ownership.
	SourceGamertag string     `json:"source_gamertag,omitempty"`
	BuiltAt        *time.Time `json:"built_at,omitempty"`
	// Coverage is named lives over total lives, 0..1 — ABSENT when the artifact reported no
	// lives at all, which is "unknown", not "zero" (cf. coverageRatio).
	Coverage   *float64 `json:"coverage,omitempty"`
	NamedLives int      `json:"named_lives"`
	TotalLives int      `json:"total_lives"`
	Tracks     int      `json:"tracks"`
	Points     int      `json:"points"`
	Shots      int      `json:"shots"`
}

// matchPage is one page of the browser, and how many rows the filter matched in total —
// the second number being the one a pager needs and the first cannot give.
type matchPage struct {
	Matches []matchSummary `json:"matches"`
	Total   int            `json:"total"`
}

// matchIdentity is a servable match: both forms of its identifier, read off the row.
type matchIdentity struct {
	MatchID string
	ShortID string
}

// listMatches runs a filter and returns one page of archived matches, newest first.
func (a *archive) listMatches(ctx context.Context, f matchFilter) (matchPage, error) {
	where, args := f.where()

	page := matchPage{Matches: []matchSummary{}}
	if err := a.db.QueryRowContext(ctx,
		`SELECT count(*) FROM matches m WHERE `+where, args...).Scan(&page.Total); err != nil {
		return matchPage{}, fmt.Errorf("counting the archived matches: %w", err)
	}

	// NULLS LAST so a match whose stats never named a start time sinks to the bottom
	// rather than heading a list sorted by "most recent". match_id breaks ties, so two
	// matches played in the same second keep a stable order across pages.
	rows, err := a.db.QueryContext(ctx, `
        SELECT m.match_id, m.short_id, m.played_at, m.map_name, m.map_module, m.mode,
               m.playlist, m.duration_ms, m.source_gamertag, m.built_at,
               m.tracks, m.points, m.shots, m.named_lives, m.total_lives,
               `+coverageRatio+` AS coverage
        FROM matches m
        WHERE `+where+`
        ORDER BY m.played_at DESC NULLS LAST, m.match_id
        LIMIT ? OFFSET ?`, append(args, f.Limit, f.Offset)...)
	if err != nil {
		return matchPage{}, fmt.Errorf("listing the archived matches: %w", err)
	}
	defer closeRows(ctx, rows, "archived matches")

	for rows.Next() {
		m, err := scanMatch(rows)
		if err != nil {
			return matchPage{}, err
		}
		page.Matches = append(page.Matches, m)
	}
	if err := rows.Err(); err != nil {
		return matchPage{}, fmt.Errorf("reading the archived matches: %w", err)
	}
	return page, nil
}

// scanMatch reads one row. Every text column is nullable in the archive — a match whose
// stats named no map really does store NULL — so all of them go through sql.NullString
// rather than crashing the scan on exactly the degraded row worth looking at.
func scanMatch(rows *sql.Rows) (matchSummary, error) {
	var (
		m                                  matchSummary
		mapName, mapModule, mode, playlist sql.NullString
		sourceGT                           sql.NullString
		playedAt, builtAt                  sql.NullTime
		durationMS                         sql.NullInt64
		coverage                           sql.NullFloat64
	)
	if err := rows.Scan(&m.MatchID, &m.ShortID, &playedAt, &mapName, &mapModule, &mode,
		&playlist, &durationMS, &sourceGT, &builtAt,
		&m.Tracks, &m.Points, &m.Shots, &m.NamedLives, &m.TotalLives, &coverage); err != nil {
		return matchSummary{}, fmt.Errorf("scanning an archived match: %w", err)
	}
	m.MapName, m.MapModule = mapName.String, mapModule.String
	m.Mode, m.Playlist, m.SourceGamertag = mode.String, playlist.String, sourceGT.String
	m.PlayedAt, m.BuiltAt = nullTime(playedAt), nullTime(builtAt)
	if durationMS.Valid {
		ms := durationMS.Int64
		m.DurationMS = &ms
	}
	if coverage.Valid {
		c := coverage.Float64
		m.Coverage = &c
	}
	return m, nil
}

// lookupMatch resolves either form of a match identifier to a SERVABLE match.
//
// A match recorded without an artifact answers errMatchUnknown, deliberately: to a caller
// asking for a replay, "the archiver saw this match but could never build it" and "no such
// match" lead to the same place, and the archive's own `status` command is where the
// difference is reported.
func (a *archive) lookupMatch(ctx context.Context, id string) (matchIdentity, error) {
	var got matchIdentity
	err := a.db.QueryRowContext(ctx, `
        SELECT match_id, short_id FROM matches
        WHERE (match_id = ? OR short_id = ?) AND artifact_path IS NOT NULL`, id, id).
		Scan(&got.MatchID, &got.ShortID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return matchIdentity{}, fmt.Errorf("%w: %s", errMatchUnknown, id)
	case err != nil:
		return matchIdentity{}, fmt.Errorf("looking up match %s: %w", id, err)
	}
	return got, nil
}

// closeRows logs a cursor that refuses to close rather than dropping it (repo rule 3: no
// error goes by in silence, even one the caller can do nothing about).
func closeRows(ctx context.Context, rows *sql.Rows, what string) {
	if err := rows.Close(); err != nil {
		slog.ErrorContext(ctx, "study-server: closing a cursor", "err", err, "query", what)
	}
}

func nullTime(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}
