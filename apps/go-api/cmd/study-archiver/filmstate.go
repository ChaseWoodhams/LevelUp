package main

// filmstate.go — WHAT A LATER RUN MAY DO WITH A MATCH IT HAS ALREADY SEEN (#7).
//
// The archive records a film state per match, and the whole point of having FOUR values
// rather than a success flag is that they prescribe four different answers to the
// unattended hourly run:
//
//	pending     nothing was fetched yet             -> fetch it
//	downloaded  the chunks are on disk              -> build it (the map catalogue grows)
//	failed      the film arrived, the decoder did not accept it -> retry after a fix
//	expired     the CDN answered 404/410            -> NEVER again
//
// EXPIRED AND FAILED ARE OPPOSITES, which is why they are not one "it did not work" flag.
// An expired film is gone forever: retrying it every hour spends a request against a link
// that cannot come back, and it does so silently, in the run nobody watches. A failed one
// is a decoder problem with its raw material still on disk — the case a future decoder fix
// is meant to rescue, so nothing here may bar it.
//
// TRANSIENT FAILURES ARE NOT A STATE AT ALL. A 5xx, a timeout, a dropped connection: the
// archiver records NOTHING and returns an error, so the next run starts over. Writing
// `expired` on a bad afternoon would bury a perfectly good film permanently — and this is
// the failure mode with no alarm attached, because a buried match simply stops appearing.

// filmState is the life-cycle of a match's film, as stored in `matches.film_state`.
type filmState string

const (
	statePending    filmState = "pending"
	stateDownloaded filmState = "downloaded"
	stateExpired    filmState = "expired"
	stateFailed     filmState = "failed"
)

// terminal reports whether a recorded state is FINAL: no later run may re-attempt the
// match, whatever else changes.
//
// `expired` is the only one. `failed` deliberately is not — its chunks are on disk and a
// decoder fix rescues it — and neither is a `downloaded` match skipped for an unsupported
// map, which the quant-bounds catalogue can make buildable at any time.
func (s filmState) terminal() bool { return s == stateExpired }

// filmCaptured reports whether the recorded state means the WHOLE film reached the chunk
// cache. Both states that say so are written only after `writeFilmChunks` returned without
// error — a partial write is an error, and an error records nothing — so the claim is not
// a guess about what a crashed run left behind.
//
// It is what makes "a failed match remains eligible for a later rebuild once its chunks
// are on disk" true rather than aspirational: a captured film is rebuilt FROM DISK, so the
// rebuild neither costs a CDN request nor depends on a link that may already be dead.
func (s filmState) filmCaptured() bool { return s == stateDownloaded || s == stateFailed }

// stateAfterRebuild is filmStateOf for a rebuild, which starts from a state already
// recorded rather than from nothing.
//
// A REBUILD THAT PRODUCED NO ARTIFACT NEVER CLEARS A TERMINAL VERDICT. Rebuilding an
// `expired` match from chunks another tool left in the cache is exactly what `rebuild` is
// for — but if it does not succeed, the film is still gone from the CDN, and writing
// `downloaded` (which "the chunks are on disk" would otherwise imply) would quietly make
// the match retryable again. The hourly loop would then chase a link known to be dead, the
// precise waste #7 exists to prevent. Only an ARTIFACT settles the match, and that path
// short-circuits on the artifact rather than on the state.
//
// The reason is carried over with the state for the same reason: a row saying `expired`
// with a reason of `unsupported_map` contradicts itself, and the two are read together.
func stateAfterRebuild(prior matchRecord, out outcome) (filmState, reason) {
	if out.ArtifactPath == "" && prior.State.terminal() {
		return prior.State, prior.SkipReason
	}
	return filmStateOf(out), out.SkipReason
}

// filmStateOf maps a finished outcome onto the state the archive records.
func filmStateOf(out outcome) filmState {
	switch out.SkipReason {
	case skipFilmAbsent:
		return stateExpired
	case skipNoTracks, skipBuildFailed:
		return stateFailed
	}
	if out.ChunksWritten > 0 {
		return stateDownloaded
	}
	return statePending
}
