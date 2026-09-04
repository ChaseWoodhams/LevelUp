package main

// status_render.go — THE REPORT, AS A HUMAN READS IT.
//
// Kept apart from the queries so that the numbers can be tested without asserting on
// column widths, and so the layout can change without touching a single SQL statement.
//
// The one criterion this ticket has is "readable at a glance", which decided the shape:
// the headline first (what is in there), then the gap (what is not, and why), then the
// watchlist's freshness — the three questions an operator opens this for, in the order
// they ask them.

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"
)

// maxBreakdownRows caps each breakdown. An archive of ten thousand matches spans every map
// and every playlist Halo has, and a report that scrolls off the screen answers nothing at
// a glance. The tail is summarised rather than dropped silently.
const maxBreakdownRows = 12

// render writes the report.
func (r statusReport) render(w io.Writer, now time.Time) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "STUDY ARCHIVE\t%s\n\n", r.Path)
	fmt.Fprintf(tw, "Matches\t%d recorded, %d with a replay artifact\n", r.Recorded, r.Archived)
	// Failed and expired are printed side by side but never added together: the pair is
	// the whole point of the film-state distinction.
	fmt.Fprintf(tw, "Not built\t%d failed (rebuildable), %d expired (gone for good), "+
		"%d captured but unbuilt\n\n", r.Failed, r.Expired, r.Captured)

	renderBreakdown(tw, "Archived by map", r.ByMap)
	renderBreakdown(tw, "Archived by mode", r.ByMode)
	renderBreakdown(tw, "Archived by tracked player", r.ByPlayer)
	renderBreakdown(tw, "Not archived, by reason", r.UnArchived)
	renderWatchlist(tw, r.Watchlist, now)
	return tw.Flush()
}

func renderBreakdown(w io.Writer, heading string, rows []countedRow) {
	fmt.Fprintf(w, "%s\n", heading)
	if len(rows) == 0 {
		fmt.Fprint(w, "  (nothing yet)\n\n")
		return
	}
	shown, rest, restRows := rows, 0, 0
	if len(rows) > maxBreakdownRows {
		shown = rows[:maxBreakdownRows]
		for _, r := range rows[maxBreakdownRows:] {
			rest += r.N
			restRows++
		}
	}
	for _, row := range shown {
		fmt.Fprintf(w, "  %s\t%d\n", row.Label, row.N)
	}
	if restRows > 0 {
		fmt.Fprintf(w, "  and %d more\t%d\n", restRows, rest)
	}
	fmt.Fprint(w, "\n")
}

func renderWatchlist(w io.Writer, players []watchlistStatus, now time.Time) {
	fmt.Fprint(w, "Watchlist\n")
	if len(players) == 0 {
		fmt.Fprint(w, "  (no player followed yet - run `watch`)\n")
		return
	}
	for _, p := range players {
		fmt.Fprintf(w, "  %s\t%s\tlast checked %s\n", p.Gamertag, xuidLabel(p.XUID),
			lastPassAge(watchedPlayer{LastChecked: p.LastChecked}, now))
	}
}

// xuidLabel spells out an unresolved player rather than printing an empty column, which
// reads as a rendering bug rather than as the fact it is.
func xuidLabel(xuid string) string {
	if xuid == "" {
		return "(unresolved)"
	}
	return "xuid " + xuid
}
