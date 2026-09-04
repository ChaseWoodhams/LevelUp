// cmd/study-archiver — builds a durable local archive of match replay artifacts,
// captured BEFORE their film CDN links expire (study-tool epic, issue #1).
//
// Subcommands:
//
//	study-archiver fetch-one --xuid <xuid> [--title slug] [--interval MS] [--rps N] <matchId>
//
// `fetch-one` downloads a single match's whole film — header, replication data and
// highlight footer — into the film-chunk cache, resolves the map the match was played
// on, and assembles the 2D replay artifact the existing viewer already reads. The raw
// chunks are KEPT: the artifact is regenerable, the expired CDN link is not.
//
// Example (from apps/go-api):
//
//	LEVELUP_REPO_ROOT=<repo> CC=/c/msys64/ucrt64/bin/gcc.exe \
//	  go run ./cmd/study-archiver fetch-one --xuid 2533274823110022 <matchId>
//
// This binary LINKS DUCKDB (the archive database), so cgo is required — the UCRT
// toolchain, never mingw64; cf. CLAUDE.md. It was cgo-free until the archive landed.
//
// PREFER `go build` OVER `go run` FOR REAL ARCHIVING RUNS. Go stamps the VCS revision
// into a built binary but not into `go run`, and that stamp is what the archive records
// as decoder_rev — the only way a later coverage drop can be traced back to the build
// that caused it. A `go run` archive is still correct; it just cannot answer that
// question, and the tool says so once per run.
//
// Exit codes: 0 archived, 3 skipped for a named reason (expired film, unsupported map,
// nothing decoded — all normal outcomes, all logged with their reason), 1 failure, 2
// usage.
//
// WHAT A SECOND RUN DOES depends on what the first one recorded, and the rule is in
// filmstate.go: a match whose film EXPIRED is never touched again, while one that failed
// to build, or whose map had no bounds yet, is re-attempted every time — its chunks are on
// disk, and a decoder fix or a catalogue update is exactly what rescues it. A transient
// failure (5xx, a timeout) records nothing at all, so it never becomes either verdict.
//
// Authentication follows ADR 0023: MultiUserTokenStore via
// auth.RefreshHaloTokensViaStoreFirst, the owner's own token. NO token re-capture — a
// dead refresh token is diagnosed, not worked around. No per-player credential is
// needed: match history, film and stats for any xuid are readable with the owner's
// token. Alternatively set SPARTAN_TOKEN (and CLEARANCE_TOKEN) in the environment.
//
// `watch` (#8) turns a watchlist into an unattended hourly pass over the same fetchOne
// seam. `status` (#9) and `rebuild` (#10) make NO network call and need no credential at
// all: one reads the archive, the other re-assembles an artifact from chunks already on
// disk, which is what keeps the archive useful long after every CDN link in it has died.
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"time"

	"levelup/go-api/internal/domain/title"
)

// Exit codes. `skipped` is distinct from `failed` on purpose: a scheduled run must be
// able to tell "this match will never build" from "this run went wrong".
const (
	exitOK      = 0
	exitFailure = 1
	exitUsage   = 2
	exitSkipped = 3
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	if len(os.Args) < 2 {
		usage()
		os.Exit(exitUsage)
	}
	switch os.Args[1] {
	case "fetch-one":
		os.Exit(runFetchOne(context.Background(), os.Args[2:]))
	case "watch":
		os.Exit(runWatch(context.Background(), os.Args[2:]))
	case "status":
		os.Exit(runStatus(context.Background(), os.Args[2:]))
	case "rebuild":
		os.Exit(runRebuild(context.Background(), os.Args[2:]))
	case "-h", "--help", "help":
		usage()
		os.Exit(exitOK)
	default:
		slog.Error("study-archiver: unknown subcommand", "subcommand", os.Args[1])
		usage()
		os.Exit(exitUsage)
	}
}

// usage goes through slog like every other line this tool emits (repo rule 3, and the
// shape cmd/replay-build already uses for its own usage line): one stream, one format,
// nothing that a log collector has to be told to read separately.
func usage() {
	slog.Info("usage: study-archiver fetch-one --xuid <xuid> [--gamertag GT] [--title slug] " +
		"[--interval MS] [--rps N] <matchId> - archive one match: download its whole film " +
		"into the chunk cache and build the 2D replay artifact. Options must precede " +
		"<matchId>. SPARTAN_TOKEN in the environment replaces --xuid.")
	slog.Info("usage: study-archiver watch --xuid <xuid> [--gamertag GT] [--title slug] " +
		"[--interval MS] [--rps N] - one pass over " + watchlistFileName + " at the repo " +
		"root: pull each tracked player's recent history and archive the 4v4 matches not " +
		"already known. Exits when done; run it from the OS scheduler, hourly.")
	slog.Info("usage: study-archiver status - what is in the archive and what went wrong, " +
		"on stdout. Read-only: safe to run while a capture is in progress. No credential " +
		"needed.")
	slog.Info("usage: study-archiver rebuild [--title slug] [--interval MS] <matchId> - " +
		"re-assemble a match's replay artifact from the film chunks already on disk. Makes " +
		"NO network call and needs no credential; a match whose chunks are gone fails rather " +
		"than re-downloading.")
}

// commonFlags are the options EVERY archiving subcommand takes: they all authenticate the
// same way, against the same title, and write the same archive.
//
// Registered in one place rather than copied per subcommand, because the failure of a copy
// is silent: a flag added to fetch-one and forgotten in watch does not break a build, it
// just makes the unattended pass quietly ignore an option the operator passed it.
type commonFlags struct {
	xuid, gamertag, titleSlug *string
	interval, rps             *int
}

func registerCommonFlags(fs *flag.FlagSet) commonFlags {
	return commonFlags{
		xuid:      fs.String("xuid", "", "xuid whose stored token authenticates the run (ADR 0023)"),
		gamertag:  fs.String("gamertag", "", "gamertag of --xuid; recorded as the archive row's source_gamertag"),
		titleSlug: fs.String("title", title.DefaultSlug, "title slug"),
		interval:  fs.Int("interval", 0, "replay grid step in ms (0 = the replay package's default)"),
		rps:       fs.Int("rps", 0, "outgoing requests per second (0 = the Halo client's default)"),
	}
}

func (c commonFlags) request() depsRequest {
	return depsRequest{
		XUID:            *c.xuid,
		Gamertag:        *c.gamertag,
		Title:           *c.titleSlug,
		FrameIntervalMS: *c.interval,
		RequestsPerSec:  *c.rps,
	}
}

// runWatch runs ONE pass over the watchlist and exits. The OS scheduler owns the clock.
//
// EXIT CODES SAY WHAT THE SCHEDULER NEEDS TO KNOW: 0 when the pass completed, whether or
// not it found anything new — a quiet hour is a success, not a skip — and 1 when any
// player or match failed. A pass never aborts on a single failure: the films of every
// other player are expiring while it would be giving up.
func runWatch(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("watch", flag.ContinueOnError)
	common := registerCommonFlags(fs)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if fs.NArg() != 0 {
		slog.ErrorContext(ctx, "study-archiver: watch takes no positional argument "+
			"(the players come from "+watchlistFileName+")", "args", fs.Args())
		return exitUsage
	}

	// The watchlist is read BEFORE anything is authenticated or opened: a missing or empty
	// one is an operator mistake, and failing on it should cost no token and leave no
	// DuckDB handle behind.
	repoRoot, err := title.FindRepoRoot()
	if err != nil {
		slog.ErrorContext(ctx, "study-archiver: repo root", "err", err)
		return exitFailure
	}
	wl, err := loadWatchlist(repoRoot)
	if err != nil {
		slog.ErrorContext(ctx, "study-archiver: watchlist", "err", err)
		return exitUsage
	}

	d, err := newDeps(ctx, common.request())
	if err != nil {
		slog.ErrorContext(ctx, "study-archiver: setup failed", "err", err)
		return exitFailure
	}
	defer func() {
		if cErr := d.Archive.Close(); cErr != nil {
			slog.ErrorContext(ctx, "study-archiver: closing the archive", "err", cErr)
		}
	}()

	if sum := watchPass(ctx, d, newWatchDeps(d.Paths, common.request()), wl); sum.Failed > 0 {
		return exitFailure
	}
	return exitOK
}

// runStatus prints the archive's health. It opens NOTHING read-write and authenticates
// nothing: it is the command an operator runs at any moment, including mid-capture.
func runStatus(ctx context.Context, args []string) int {
	// No --title: there is ONE archive, at data/study/, and it holds every title's
	// matches. Accepting a flag it would have to ignore would be a lie in the help text.
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	repoRoot, err := title.FindRepoRoot()
	if err != nil {
		slog.ErrorContext(ctx, "study-archiver: repo root", "err", err)
		return exitFailure
	}
	report, err := openStatusReport(ctx, title.NewPathResolver(repoRoot))
	if err != nil {
		slog.ErrorContext(ctx, "study-archiver: status failed", "err", err)
		return exitFailure
	}
	if err := report.render(os.Stdout, time.Now()); err != nil {
		slog.ErrorContext(ctx, "study-archiver: writing the report", "err", err)
		return exitFailure
	}
	return exitOK
}

// runRebuild re-assembles one match's artifact from cached chunks, offline.
func runRebuild(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("rebuild", flag.ContinueOnError)
	common := registerCommonFlags(fs)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if fs.NArg() != 1 {
		slog.ErrorContext(ctx, "study-archiver: rebuild takes exactly one match id "+
			"(options must precede it)", "args", fs.Args())
		return exitUsage
	}
	matchID := fs.Arg(0)

	d, err := newOfflineDeps(ctx, common.request())
	if err != nil {
		slog.ErrorContext(ctx, "study-archiver: setup failed", "err", err)
		return exitFailure
	}
	defer func() {
		if cErr := d.Archive.Close(); cErr != nil {
			slog.ErrorContext(ctx, "study-archiver: closing the archive", "err", cErr)
		}
	}()

	out, err := rebuildOne(ctx, d, matchID)
	if err != nil {
		slog.ErrorContext(ctx, "study-archiver: rebuild failed", "err", err, "match_id", matchID)
		return exitFailure
	}
	if out.SkipReason != "" {
		return exitSkipped
	}
	return exitOK
}

// runFetchOne parses the subcommand's flags, wires the archiver and reports what it did.
func runFetchOne(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("fetch-one", flag.ContinueOnError)
	common := registerCommonFlags(fs)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if fs.NArg() != 1 {
		slog.ErrorContext(ctx, "study-archiver: fetch-one takes exactly one match id "+
			"(options must precede it)", "args", fs.Args())
		return exitUsage
	}
	matchID := fs.Arg(0)

	d, err := newDeps(ctx, common.request())
	if err != nil {
		slog.ErrorContext(ctx, "study-archiver: setup failed", "err", err)
		return exitFailure
	}
	// The archive is a DuckDB file and this process is its only writer: leaving the
	// handle open past the run would keep it locked against the study server.
	defer func() {
		if cErr := d.Archive.Close(); cErr != nil {
			slog.ErrorContext(ctx, "study-archiver: closing the archive", "err", cErr)
		}
	}()

	out, err := fetchOne(ctx, d, matchID)
	if err != nil {
		slog.ErrorContext(ctx, "study-archiver: fetch-one failed",
			"err", err, "match_id", matchID, "chunks", out.ChunksWritten)
		return exitFailure
	}
	if out.SkipReason != "" {
		// Already logged with its reason by fetchOne; the exit code is the machine-
		// readable half of the same statement.
		return exitSkipped
	}
	return exitOK
}
