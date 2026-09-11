package main

// status_groundtruth.go — THE REPLAY AGAINST THE MATCH STATS, ACROSS THE ARCHIVE.
//
// groundtruth.go compares each build with Halo's own death counts, and the archive keeps the
// result. This is where it becomes a check on the DECODER rather than on one match: rebuild the
// archive after a decoder change, run `status`, and these lines say whether the change named
// more lives, split or merged lives, or - the figure that must never move - named a life after
// the wrong player.
//
// Only ARCHIVED matches count, like every other breakdown in this report: a match whose
// artifact is gone has no replay left to be right or wrong about.

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"
)

// maxGroundTruthRows caps each per-match list: the worst few are what an operator acts on.
const maxGroundTruthRows = 5

// groundTruthMatch is one archived match in a ground-truth list.
type groundTruthMatch struct {
	ShortID   string
	MapName   string
	OverNamed int
	LivesGap  int
}

// groundTruthSummary totals the comparison over every archived match that was compared.
type groundTruthSummary struct {
	Compared      int
	ExpectedLives int
	NamedLives    int
	OverNamed     int
	MissingLives  int
	UnknownNamed  int
	GapMin        int
	GapMax        int
	// OverNamedMatches lists the matches with any over-named life, worst first.
	OverNamedMatches []groundTruthMatch
	// WidestGaps lists the matches whose life count strays furthest from the stats.
	WidestGaps []groundTruthMatch
}

// readGroundTruthSummary gathers the ground-truth section of the report.
//
// The sums are CAST to BIGINT: DuckDB widens sum() over an INTEGER column to HUGEINT, which the
// driver hands back as a big integer that does not scan into an int.
func readGroundTruthSummary(ctx context.Context, db *sql.DB) (groundTruthSummary, error) {
	var s groundTruthSummary
	if err := db.QueryRowContext(ctx, `
        SELECT count(*),
               CAST(coalesce(sum(gt_expected_lives), 0) AS BIGINT),
               CAST(coalesce(sum(gt_named_lives), 0) AS BIGINT),
               CAST(coalesce(sum(gt_over_named), 0) AS BIGINT),
               CAST(coalesce(sum(gt_missing_lives), 0) AS BIGINT),
               CAST(coalesce(sum(gt_unknown_named), 0) AS BIGINT),
               coalesce(min(gt_lives_gap), 0),
               coalesce(max(gt_lives_gap), 0)
        FROM matches
        WHERE artifact_path IS NOT NULL AND gt_expected_lives IS NOT NULL`).
		Scan(&s.Compared, &s.ExpectedLives, &s.NamedLives, &s.OverNamed, &s.MissingLives,
			&s.UnknownNamed, &s.GapMin, &s.GapMax); err != nil {
		return s, fmt.Errorf("summarising the ground-truth comparison: %w", err)
	}
	var err error
	if s.OverNamedMatches, err = groundTruthMatches(ctx, db,
		"gt_over_named > 0", "gt_over_named DESC"); err != nil {
		return s, err
	}
	s.WidestGaps, err = groundTruthMatches(ctx, db, "gt_lives_gap <> 0", "abs(gt_lives_gap) DESC")
	return s, err
}

// groundTruthMatches lists compared archived matches matching one clause, in one order. Both
// are interpolated because SQL cannot parameterise them, and it is safe because the only values
// they take are the literals in readGroundTruthSummary, never anything from outside.
func groundTruthMatches(ctx context.Context, db *sql.DB, where, order string) ([]groundTruthMatch, error) {
	query := fmt.Sprintf(`
        SELECT short_id, coalesce(map_name, ''), gt_over_named, gt_lives_gap
        FROM matches
        WHERE artifact_path IS NOT NULL AND gt_expected_lives IS NOT NULL AND %s
        ORDER BY %s, short_id
        LIMIT %d`, where, order, maxGroundTruthRows)
	rows, err := db.QueryContext(ctx, query) //nolint:gosec // clauses from a fixed set, cf. above
	if err != nil {
		return nil, fmt.Errorf("listing ground-truth matches: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []groundTruthMatch
	for rows.Next() {
		var m groundTruthMatch
		if err := rows.Scan(&m.ShortID, &m.MapName, &m.OverNamed, &m.LivesGap); err != nil {
			return nil, fmt.Errorf("scanning a ground-truth match: %w", err)
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading ground-truth matches: %w", err)
	}
	return out, nil
}

// renderGroundTruth writes the ground-truth section of the report.
func renderGroundTruth(w io.Writer, s groundTruthSummary) {
	fmt.Fprint(w, "Replay vs official match stats\n")
	if s.Compared == 0 {
		fmt.Fprint(w, "  (no archived match compared yet - rebuild to compute)\n\n")
		return
	}
	fmt.Fprintf(w, "  matches compared\t%d\n", s.Compared)
	fmt.Fprintf(w, "  over-named lives\t%d\t(must stay 0)\n", s.OverNamed)
	fmt.Fprintf(w, "  named / expected lives\t%d / %d\t(%s)\n",
		s.NamedLives, s.ExpectedLives, percentOf(s.NamedLives, s.ExpectedLives))
	fmt.Fprintf(w, "  missing lives\t%d\n", s.MissingLives)
	fmt.Fprintf(w, "  lives on unlisted players\t%d\n", s.UnknownNamed)
	fmt.Fprintf(w, "  lives gap (replay - stats)\t%+d .. %+d\n", s.GapMin, s.GapMax)
	renderGroundTruthMatches(w, "over-named in", s.OverNamedMatches,
		func(m groundTruthMatch) string { return fmt.Sprintf("%d", m.OverNamed) })
	renderGroundTruthMatches(w, "widest gaps", s.WidestGaps,
		func(m groundTruthMatch) string { return fmt.Sprintf("%+d", m.LivesGap) })
	fmt.Fprint(w, "\n")
}

// renderGroundTruthMatches writes one per-match list on a single line, or nothing when empty.
func renderGroundTruthMatches(w io.Writer, heading string, matches []groundTruthMatch,
	value func(groundTruthMatch) string) {
	if len(matches) == 0 {
		return
	}
	parts := make([]string, 0, len(matches))
	for _, m := range matches {
		parts = append(parts, strings.TrimSpace(fmt.Sprintf("%s %s %s", m.ShortID, m.MapName, value(m))))
	}
	fmt.Fprintf(w, "  %s\t%s\n", heading, strings.Join(parts, ", "))
}

// percentOf formats part/whole as a percentage, and a dash when there is no whole to divide by.
func percentOf(part, whole int) string {
	if whole == 0 {
		return "-"
	}
	return fmt.Sprintf("%.1f%%", 100*float64(part)/float64(whole))
}
