package main

// filter.go — WHAT A QUERY STRING MEANS, AND THE SQL IT BECOMES.
//
// Kept apart from the handler and from the database on purpose: the rules here (what a bare
// date means, which unit a coverage floor is in, what an unknown coverage does to a floor)
// are the only real decisions this server makes, and they are the ones worth testing without
// a row on disk. The handler decodes, this file decides, and archive.go executes.
//
// EVERY VALUE IS BOUND, NEVER INTERPOLATED. The clause is assembled from fixed fragments and
// `?` placeholders; nothing from the query string reaches the SQL text itself.
//
// THE REFUSALS ARE IN FRENCH because they are the CLIENT'S message, not a log line: the
// handler hands `err.Error()` straight to the caller as the body of a 400, exactly as
// `internal/api/handlers/replay.go` does with "match_id est requis" (CLAUDE.md rule 1). The
// comments around them stay in the language of the rest of this package.

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Paging bounds. A list endpoint with no ceiling hands back the whole archive on a typo, so
// the default is generous and the cap is explicit.
const (
	defaultLimit = 200
	maxLimit     = 1000
)

// dateOnlyLayout is the shorthand an operator actually types. Read in UTC, which is what
// `played_at` is stored in.
const dateOnlyLayout = "2006-01-02"

// coverageRatio is the archive's coverage as a fraction of 1 (ADR 0006's unit for every
// canonical indicator): the lives the decoder could NAME over the lives the match had.
//
// NULL — not zero — when the artifact reported no lives at all. The two are different facts:
// zero means "nothing was attached", NULL means "there was nothing to attach it to", and
// only the first one is a coverage figure. SQL's three-valued logic then does the right
// thing on its own: `NULL >= 0.8` is unknown, so a match whose coverage cannot be computed
// never satisfies a floor, whatever the floor is.
//
// ONE definition, used by both the SELECT and the WHERE, so a filter can never disagree with
// the number the caller is shown.
const coverageRatio = `CASE WHEN m.total_lives > 0 THEN m.named_lives::DOUBLE / m.total_lives END`

// rawFilter is the query string as it arrives: strings that have been decoded and nothing
// more. Parsing it is a separate step so that "2026-13-45" fails with a message naming the
// parameter rather than as a zero time nobody notices.
type rawFilter struct {
	Map         string
	Mode        string
	Player      string
	From        string
	To          string
	MinCoverage string
	Limit       int
	Offset      int
}

// matchFilter is a validated query: every field either absent or usable.
type matchFilter struct {
	Map    string
	Mode   string
	Player string
	// From is inclusive, To EXCLUSIVE — a half-open range. That is what makes a bare
	// `from=D&to=D` mean the whole of day D (cf. parseBound), and what lets two consecutive
	// ranges tile a month without a match falling into both.
	From        *time.Time
	To          *time.Time
	MinCoverage *float64
	Limit       int
	Offset      int
}

// parseFilter validates a query string and reports the first thing wrong with it.
func parseFilter(raw rawFilter) (matchFilter, error) {
	f := matchFilter{
		Map: strings.TrimSpace(raw.Map), Mode: strings.TrimSpace(raw.Mode),
		Player: strings.TrimSpace(raw.Player),
	}
	var err error
	if f.From, err = parseBound("from", raw.From, false); err != nil {
		return matchFilter{}, err
	}
	if f.To, err = parseBound("to", raw.To, true); err != nil {
		return matchFilter{}, err
	}
	if f.From != nil && f.To != nil && !f.From.Before(*f.To) {
		return matchFilter{}, fmt.Errorf("from (%s) doit précéder to (%s)",
			f.From.Format(time.RFC3339), f.To.Format(time.RFC3339))
	}
	if f.MinCoverage, err = parseCoverage(raw.MinCoverage); err != nil {
		return matchFilter{}, err
	}
	if f.Limit, err = parseLimit(raw.Limit); err != nil {
		return matchFilter{}, err
	}
	if raw.Offset < 0 {
		return matchFilter{}, fmt.Errorf("offset ne peut pas être négatif (reçu %d)", raw.Offset)
	}
	f.Offset = raw.Offset
	return f, nil
}

// parseBound reads one end of the date range, in either of the two forms a caller writes it.
//
// A BARE DATE IS A DAY, NOT AN INSTANT. `to=2026-05-19` means "through the 19th", so as an
// upper bound it becomes the following midnight — the range being half-open, that includes
// every match played on the 19th and excludes the first of the 20th. Read as a plain instant
// it would have meant the empty span at the start of the day, and `from=D&to=D` — the most
// natural thing anyone types — would have returned nothing at all.
func parseBound(name, raw string, upper bool) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		t = t.UTC()
		return &t, nil
	}
	t, err := time.ParseInLocation(dateOnlyLayout, raw, time.UTC)
	if err != nil {
		return nil, fmt.Errorf("%s : %q n'est ni un instant RFC 3339 ni une date %s",
			name, raw, dateOnlyLayout)
	}
	if upper {
		t = t.AddDate(0, 0, 1)
	}
	return &t, nil
}

// parseCoverage reads the coverage floor as a FRACTION of 1, the repo's canonical unit for
// every indicator (ADR 0006). "0.85", never "85" — and a value above 1 is refused rather
// than clamped, because it is far more likely to be a percentage than an intent.
func parseCoverage(raw string) (*float64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return nil, fmt.Errorf("min_coverage : %q n'est pas un nombre", raw)
	}
	if v < 0 || v > 1 {
		return nil, fmt.Errorf(
			"min_coverage doit être une fraction entre 0 et 1 (0.85, pas 85) ; reçu %v", v)
	}
	return &v, nil
}

// parseLimit applies the default and the cap. A limit past the cap is REFUSED, not clamped:
// silently returning fewer rows than asked reads, at the other end, like "that is all there
// is" — the one answer a browsing tool must never give by accident.
func parseLimit(limit int) (int, error) {
	switch {
	case limit == 0:
		return defaultLimit, nil
	case limit < 0:
		return 0, fmt.Errorf("limit ne peut pas être négatif (reçu %d)", limit)
	case limit > maxLimit:
		return 0, fmt.Errorf("limit ne peut pas dépasser %d (reçu %d)", maxLimit, limit)
	}
	return limit, nil
}

// where builds the predicate and its bound arguments, in one order so the two cannot drift.
//
// The base clause is not a filter and is not optional: `artifact_path IS NOT NULL` is what
// makes this a list of ARCHIVED matches. A match the archiver recorded but could not build
// has nothing to replay, and listing it would offer the operator a row that leads nowhere.
func (f matchFilter) where() (string, []any) {
	clauses := []string{"m.artifact_path IS NOT NULL"}
	var args []any

	if f.Map != "" {
		clauses = append(clauses, "lower(m.map_name) = lower(?)")
		args = append(args, f.Map)
	}
	if f.Mode != "" {
		clauses = append(clauses, "lower(m.mode) = lower(?)")
		args = append(args, f.Mode)
	}
	if f.Player != "" {
		// Matched on EITHER key: the archive identifies players by xuid, and a gamertag is
		// the only one of the two anybody types. Case-insensitive on the gamertag alone —
		// an xuid has no case to fold.
		clauses = append(clauses, `EXISTS (SELECT 1 FROM participants p
             WHERE p.match_id = m.match_id AND (p.xuid = ? OR lower(p.gamertag) = lower(?)))`)
		args = append(args, f.Player, f.Player)
	}
	if f.From != nil {
		clauses = append(clauses, "m.played_at >= ?")
		args = append(args, *f.From)
	}
	if f.To != nil {
		clauses = append(clauses, "m.played_at < ?")
		args = append(args, *f.To)
	}
	if f.MinCoverage != nil {
		clauses = append(clauses, coverageRatio+" >= ?")
		args = append(args, *f.MinCoverage)
	}
	return strings.Join(clauses, "\n          AND "), args
}
