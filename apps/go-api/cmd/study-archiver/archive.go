package main

// archive.go — THE ARCHIVE'S MEMORY.
//
// A DuckDB file at data/study/archive.duckdb (PathResolver.StudyArchiveDBPath, added by
// #4) recording what has been captured, so the tool can answer "do I already have this
// match?" and so the study viewer can later browse by map, mode and player.
//
// WHY THE DDL LIVES HERE AND NOT IN internal/migration. That package is the app's boot
// migration chain, applied by cmd/server. This database belongs to the archiver alone:
// nothing else creates it, nothing else writes it, and it must not appear or disappear
// with the server's schema version. Its shape is therefore declared where its only
// writer is.
//
// SINGLE WRITER BY CONSTRUCTION: the archiver is a serial batch job and the only process
// that writes here, which is why the ticket exempts it from the shared-DB BatchBuilder
// machinery (ADR 0019/0030).
//
// BUT SINGLE-WRITER IS NOT A SAFETY ARGUMENT ABOUT THE ART BUG, and it must not be read
// as one. `internal/sync/no_art_patterns_test.go` records that a compaction DELETE
// "déclenchait le bug ART #23046 malgré mono-writer + PK BIGINT (crash JGtm 2026-06-20)".
// DuckDB #23046 is about index maintenance during row removal; it does not care how many
// processes are watching, and both keys here are VARCHAR — the case that file's allowlist
// explicitly refuses. The write discipline in archive_write.go is what keeps this
// database safe, not the writer count.
//
// READERS (cmd/study-server) must open through `duckdb.OpenReadForQuery`, NOT a forced
// `OpenReadOnly`: DuckDB refuses a read-only handle on a file already held read-write in the
// same process, which is the rule ADR 0013/0016 and CLAUDE.md's ART rule 4 already state for
// every other database here.
//
// AND THEY MUST NOT HOLD IT. DuckDB is single-instance-per-file ACROSS PROCESSES too, in both
// directions (measured 2026-09-04, `cmd/study-server/crossprocess_test.go`): a reader that
// keeps the handle open stops THIS tool from writing, and a pass that cannot write is films
// lost to expiry. A long-lived reader borrows the archive per query and gives it back; this
// binary, being a batch job that opens and exits, needs no such discipline.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"levelup/go-api/internal/domain/title"
	ddb "levelup/go-api/internal/platform/duckdb"
)

// archiveSchema is the whole shape of the archive, applied on every open.
//
// The primary keys are declared HERE, in the initial CREATE, and that is deliberate:
// `CREATE TABLE IF NOT EXISTS` never adds a key to a table that already exists (a trap
// the repo has already paid for), so a key added later would silently not exist on any
// database created before it.
const archiveSchema = `
CREATE TABLE IF NOT EXISTS matches (
    match_id        VARCHAR PRIMARY KEY,
    short_id        VARCHAR NOT NULL,
    played_at       TIMESTAMPTZ,
    map_name        VARCHAR,
    map_module      VARCHAR,
    mode            VARCHAR,
    playlist        VARCHAR,
    duration_ms     BIGINT,
    source_gamertag VARCHAR,
    -- film_state: the life cycle of filmstate.go - pending / downloaded / failed /
    -- expired. Only expired is terminal: it stops every later run, the others do not.
    film_state      VARCHAR NOT NULL,
    skip_reason     VARCHAR,
    artifact_path   VARCHAR,
    built_at        TIMESTAMPTZ,
    decoder_rev     VARCHAR,
    tracks          INTEGER,
    points          INTEGER,
    shots           INTEGER,
    named_lives     INTEGER,
    total_lives     INTEGER,
    recorded_at     TIMESTAMPTZ NOT NULL
);
CREATE TABLE IF NOT EXISTS participants (
    match_id VARCHAR NOT NULL,
    xuid     VARCHAR NOT NULL,
    gamertag VARCHAR,
    -- team: 0 = Eagle, 1 = Cobra. outcome: Halo's own encoding, 1 = tie, 2 = win,
    -- 3 = loss, 4 = did-not-finish. Both are stored raw, as the API reports them, so a
    -- reader never has to guess which side of a re-mapping a row was written on.
    team     INTEGER,
    outcome  INTEGER,
    kills    INTEGER,
    deaths   INTEGER,
    assists  INTEGER,
    PRIMARY KEY (match_id, xuid)
);
CREATE TABLE IF NOT EXISTS watchlist (
    gamertag     VARCHAR PRIMARY KEY,
    xuid         VARCHAR,
    added_at     TIMESTAMPTZ NOT NULL,
    last_checked TIMESTAMPTZ
);
ALTER TABLE matches ADD COLUMN IF NOT EXISTS team0_score INTEGER;
ALTER TABLE matches ADD COLUMN IF NOT EXISTS team1_score INTEGER;
-- The replay checked against the official match stats (groundtruth.go). NULL when the match
-- was not compared, never 0: zero over-named lives is a result, not an absence.
ALTER TABLE matches ADD COLUMN IF NOT EXISTS gt_players INTEGER;
ALTER TABLE matches ADD COLUMN IF NOT EXISTS gt_expected_lives INTEGER;
ALTER TABLE matches ADD COLUMN IF NOT EXISTS gt_named_lives INTEGER;
ALTER TABLE matches ADD COLUMN IF NOT EXISTS gt_over_named INTEGER;
ALTER TABLE matches ADD COLUMN IF NOT EXISTS gt_missing_lives INTEGER;
ALTER TABLE matches ADD COLUMN IF NOT EXISTS gt_unknown_named INTEGER;
ALTER TABLE matches ADD COLUMN IF NOT EXISTS gt_lives_gap INTEGER;
ALTER TABLE participants ADD COLUMN IF NOT EXISTS replay_named_lives INTEGER;`

// archive is an open handle on the archive database.
type archive struct {
	db *ddb.DB
}

// openArchive opens (creating if needed) the archive database and applies its schema.
func openArchive(path string) (*archive, error) {
	if err := os.MkdirAll(filepath.Dir(path), cacheDirPerm); err != nil {
		return nil, fmt.Errorf("study data directory: %w", err)
	}
	db, err := ddb.OpenReadWrite(path)
	if err != nil {
		return nil, fmt.Errorf("archive database %s: %w", path, err)
	}
	if _, err := db.Exec(context.Background(), archiveSchema); err != nil {
		// Logged, not swallowed: a close that also fails hides nothing behind the schema
		// error the caller is about to see, but it does mean a leaked DuckDB handle.
		if cErr := db.Close(); cErr != nil {
			slog.Error("study-archiver: closing the archive after a failed schema apply",
				"err", cErr, "path", path)
		}
		return nil, fmt.Errorf("archive schema: %w", err)
	}
	return &archive{db: db}, nil
}

// openArchiveAt is the PathResolver-rooted form, the one production uses.
func openArchiveAt(paths *title.PathResolver) (*archive, error) {
	return openArchive(paths.StudyArchiveDBPath())
}

func (a *archive) Close() error {
	if a == nil || a.db == nil {
		return nil
	}
	return a.db.Close()
}

// matchRecord is one row of `matches`. It is the archive's view of an outcome: what was
// played, what state its film is in, and what the decoder got out of it.
type matchRecord struct {
	Team0Score *int
	Team1Score *int
	MatchID    string
	ShortID    string
	PlayedAt   *time.Time
	MapName    string
	MapModule  string
	Mode       string
	Playlist   string
	DurationMS *int64
	SourceGT   string
	State      filmState
	SkipReason reason
	// ArtifactPath, BuiltAt and DecoderRev are set only when an artifact was written.
	ArtifactPath string
	BuiltAt      *time.Time
	DecoderRev   string
	Tracks       int
	Points       int
	Shots        int
	NamedLives   int
	TotalLives   int
	// GroundTruth is the build's replay-versus-stats comparison (groundtruth.go).
	GroundTruth groundTruthRecord
}

// participantRecord is one row of `participants`. Team and outcome come from MATCH
// STATS and never from the film: the film carries no team information at all, and the
// replay artifact's team field is unset by design.
type participantRecord struct {
	XUID     string
	Gamertag string
	Team     *int
	Outcome  *int
	Kills    *int
	Deaths   *int
	Assists  *int
	// ReplayNamedLives is how many lives the replay named for this player, beside the official
	// Deaths it is checked against. NULL when the build did not compare this player.
	ReplayNamedLives *int
}

// recorded reads back what the archive already knows about a match.
//
// ONE reader rather than one per question. The idempotency check needs the state, the
// artifact path AND the counts (to report what a skipped re-run already holds), and
// three separate small queries would each be a place for the answers to disagree.
//
// PARTIAL BY DESIGN: it selects what the idempotency check uses, so Mode, Playlist,
// PlayedAt, SourceGT, BuiltAt and DecoderRev come back ZERO whatever the row holds.
// Do not hand the result to anything that reports those fields without widening the
// SELECT first.
func (a *archive) recorded(ctx context.Context, matchID string) (matchRecord, bool, error) {
	// Every nullable column is scanned through sql.NullString: map_name and map_module
	// are NULL for a match whose stats named no map, and scanning that into a string
	// fails at runtime on exactly the row this check exists to read.
	var (
		rec                                matchRecord
		state                              string
		mapName, mapModule, skip, artifact sql.NullString
	)
	err := a.db.QueryRow(ctx, `
        SELECT short_id, map_name, map_module, film_state, skip_reason, artifact_path,
               tracks, points, shots, named_lives, total_lives, team0_score, team1_score
        FROM matches WHERE match_id = ?`, matchID).
		Scan(&rec.ShortID, &mapName, &mapModule, &state, &skip, &artifact,
			&rec.Tracks, &rec.Points, &rec.Shots, &rec.NamedLives, &rec.TotalLives,
			&rec.Team0Score, &rec.Team1Score)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return matchRecord{}, false, nil
	case err != nil:
		return matchRecord{}, false, fmt.Errorf("reading archive row of %s: %w", matchID, err)
	}
	rec.MatchID = matchID
	rec.State = filmState(state)
	rec.MapName, rec.MapModule = mapName.String, mapModule.String
	rec.SkipReason = reason(skip.String)
	rec.ArtifactPath = artifact.String
	return rec, true, nil
}

// roster reads back the official roster recorded for a match. A rebuild never reads the match
// stats again, so this is what it checks the rebuilt replay against.
func (a *archive) roster(ctx context.Context, matchID string) ([]participantRecord, error) {
	rows, err := a.db.SQLDb().QueryContext(ctx, `
        SELECT xuid, gamertag, team, outcome, kills, deaths, assists
        FROM participants WHERE match_id = ? ORDER BY xuid`, matchID)
	if err != nil {
		return nil, fmt.Errorf("reading the roster of %s: %w", matchID, err)
	}
	defer func() { _ = rows.Close() }()
	var out []participantRecord
	for rows.Next() {
		var (
			p                                     participantRecord
			gamertag                              sql.NullString
			team, outcome, kills, deaths, assists sql.NullInt64
		)
		if err := rows.Scan(&p.XUID, &gamertag, &team, &outcome, &kills, &deaths, &assists); err != nil {
			return nil, fmt.Errorf("scanning the roster of %s: %w", matchID, err)
		}
		p.Gamertag = gamertag.String
		p.Team, p.Outcome = nullIntPtr(team), nullIntPtr(outcome)
		p.Kills, p.Deaths, p.Assists = nullIntPtr(kills), nullIntPtr(deaths), nullIntPtr(assists)
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading the roster of %s: %w", matchID, err)
	}
	return out, nil
}

// nullIntPtr maps a nullable integer column to *int: NULL stays nil, never 0.
func nullIntPtr(v sql.NullInt64) *int {
	if !v.Valid {
		return nil
	}
	n := int(v.Int64)
	return &n
}
