package main

// skip.go — THE NAMED REASONS A MATCH YIELDS NO ARTIFACT.
//
// A match can end without a replay artifact for reasons that are NOT failures of this
// tool: the film has already expired, the map has no bounds in the catalogue yet, the
// decoder read the film but found no trajectory. Each is a normal outcome of archiving,
// and each needs a DIFFERENT answer later — retry never (expired), retry once the
// catalogue grows (unsupported map), retry after a decoder fix (no tracks).
//
// So the reason is a NAME, not a log sentence. Ticket #6 records it in the archive
// database and #7 branches on it; a free-text message would have to be re-parsed, and a
// bare error would collapse the three into one.

// reason is the name of a skip. A defined type rather than a bare string so a new reason
// has to be declared below to exist at all: the set is the archiver's contract with its
// own database (#6) and with the watch loop's retry policy (#7), and an undeclared
// literal slipping into it would be a row nothing knows how to interpret.
type reason string

// Skip reasons. Renaming one is a data migration, not a cosmetic edit.
const (
	// skipUnsupportedMap : the map is absent from the title's quant-bounds catalogue.
	// Building anyway would apply ANOTHER map's bounds, which is wrong by an arbitrary
	// scale factor and invisible on screen. The film chunks are kept: the catalogue can
	// grow later, the CDN link cannot come back.
	skipUnsupportedMap reason = "unsupported_map"
	// skipNoMapInStats : the match-stats payload carries no map name at all. Distinct
	// from unsupported_map — nothing here says a rebuild would ever succeed.
	skipNoMapInStats = "no_map_in_stats"
	// skipFilmAbsent : the film manifest or its blobs answered 404/410. This is the
	// expiry this whole tool exists to race. Ticket #7 owns turning it into a permanent
	// `expired` state that watch never retries.
	skipFilmAbsent = "film_absent"
	// skipNoTracks : the film downloaded and decoded but yielded no trajectory. The
	// chunks are on disk, so a later decoder fix can rebuild (#10) without the CDN.
	skipNoTracks = "no_tracks_decoded"
	// skipUnknown : a skip that reached the recorder without a named reason. Declared
	// rather than written as a literal at the guard that produces it, so that a reason
	// added later without a name still lands inside this set instead of beside it.
	skipUnknown = "unknown"
)

// skipError is a named, NON-FATAL reason a match produced no artifact. It wraps the
// underlying cause so `errors.Is` still reaches sentinels such as
// filmdec.ErrUnknownMapBounds — the name is added context, not a replacement.
type skipError struct {
	// Reason is one of the constants above.
	Reason reason
	// Cause is the underlying error, if any. Nil when the skip is a plain observation
	// ("the stats carry no map name") rather than a failed operation.
	Cause error
	// Detail is a short human-readable complement for the log line.
	Detail string
}

func (e skipError) Error() string {
	msg := string(e.Reason)
	if e.Detail != "" {
		msg += " (" + e.Detail + ")"
	}
	if e.Cause != nil {
		msg += ": " + e.Cause.Error()
	}
	return msg
}

func (e skipError) Unwrap() error { return e.Cause }
