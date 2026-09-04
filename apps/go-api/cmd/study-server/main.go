// cmd/study-server — serves the study archive to the study app (study-tool epic, issue #12).
//
// Three read-only endpoints over what cmd/study-archiver captured: the list of archived
// matches, one match's replay artifact, and that match's roster.
//
//	study-server [--addr 127.0.0.1:8100] [--title halo_infinite]
//
//	GET /matches?map=&mode=&player=&from=&to=&min_coverage=&limit=&offset=
//	GET /matches/{match_id}/replay
//	GET /matches/{match_id}/participants
//
// WHY THIS EXISTS AT ALL, WHEN THE APP ALREADY SERVES A REPLAY. The app's own replay route
// hangs off a player: it resolves a ReplayService per `player_slug` and joins the roster
// against the scoreboard of a player declared in `db_profiles.json`. The archive is full of
// OTHER PEOPLE'S matches — that is the point of it — so the app's route can serve neither the
// artifact (no such player) nor the roster (nobody in it is declared). The archiver already
// recorded team and K/D/A for every participant of every match it captured; this server is
// what puts those rows in front of the viewer.
//
// WHY GO AND NOT NODE. Two rules would otherwise have to be re-implemented in a second
// language and kept in step by hand: `PathResolver.ReplayArtifactPath` and the short-film-ID
// convention it applies (`title.FilmShortMatchID`). Both are reused here directly.
//
// READ-ONLY, AND SAFE TO RUN DURING A CAPTURE. The archive is opened through
// `duckdb.OpenReadForQuery` (cf. archive.go), so this server can serve while the hourly
// `study-archiver watch` pass is writing.
//
// IT LISTENS ON THE LOOPBACK ADDRESS. The archive holds films and rosters of matches played
// by people who never published them; the tool is a local study aid, not a service. `--addr`
// can be pointed elsewhere by an operator who means it, and the default will not do it for
// them. This is also why there is no local-only middleware of the kind the app's replay route
// carries: the binding is the boundary, and one rule is better than two that can disagree.
//
// This binary LINKS DUCKDB, so cgo is required — the UCRT toolchain, never mingw64
// (cf. CLAUDE.md).
//
// Exit codes: 0 a clean shutdown, 1 a failure, 2 usage.
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain/title"
)

const (
	exitOK      = 0
	exitFailure = 1
	exitUsage   = 2
)

// defaultAddr is the loopback, deliberately. Cf. the package comment.
const defaultAddr = "127.0.0.1:8100"

// HTTP timeouts. The write budget is generous because a replay artifact runs to megabytes and
// the client is a browser on the same machine; the header budget is not, because a stuck
// header is never legitimate.
const (
	readHeaderTimeout = 10 * time.Second
	writeTimeout      = 60 * time.Second
	idleTimeout       = 120 * time.Second
	shutdownGrace     = 10 * time.Second
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	os.Exit(run(context.Background(), os.Args[1:]))
}

func run(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("study-server", flag.ContinueOnError)
	addr := fs.String("addr", defaultAddr, "listen address (loopback by default; the archive is local data)")
	// One archive holds every title's matches, but an artifact is filed under its title, so
	// the path resolution needs a slug. The archiver defaults the same way.
	titleSlug := fs.String("title", title.DefaultSlug, "title slug the replay artifacts are filed under")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if fs.NArg() != 0 {
		slog.ErrorContext(ctx, "study-server: takes no positional argument", "args", fs.Args())
		return exitUsage
	}

	h, closeArchive, err := newHandler(*titleSlug)
	if err != nil {
		slog.ErrorContext(ctx, "study-server: setup failed", "err", err)
		return exitFailure
	}
	defer closeArchive()

	return serve(ctx, *addr, h)
}

// newHandler opens the archive and wires the routes over it.
func newHandler(titleSlug string) (*studyHandler, func(), error) {
	repoRoot, err := title.FindRepoRoot()
	if err != nil {
		return nil, nil, err
	}
	paths := title.NewPathResolver(repoRoot)
	a, err := openArchive(paths.StudyArchiveDBPath())
	if err != nil {
		return nil, nil, err
	}
	h := &studyHandler{archive: a, artifacts: artifacts{paths: paths, titleSlug: titleSlug}}
	return h, a.Close, nil
}

// serve runs the HTTP server until the process is asked to stop.
//
// GRACEFUL, because the one long response this server produces is a multi-megabyte artifact:
// dropping the listener mid-transfer would leave the viewer holding a truncated JSON document,
// which fails as a parse error naming nothing that happened.
func serve(ctx context.Context, addr string, h *studyHandler) int {
	r := chi.NewRouter()
	h.mount(r)

	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: readHeaderTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	slog.InfoContext(ctx, "study-server: listening",
		"addr", addr, "archive", h.archive.path, "title", h.artifacts.titleSlug)

	select {
	case err := <-errc:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.ErrorContext(ctx, "study-server: listener stopped", "err", err, "addr", addr)
			return exitFailure
		}
	case <-ctx.Done():
		slog.Info("study-server: shutting down")
		// A fresh context: the one that carries the signal is already cancelled, and passing
		// it would turn the grace period into an immediate close.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("study-server: shutdown did not complete", "err", err)
			return exitFailure
		}
	}
	return exitOK
}
