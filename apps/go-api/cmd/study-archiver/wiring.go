package main

// wiring.go — TURNING THE REPO ON DISK INTO A `deps`.
//
// Everything the archiver reads from the repository is resolved ONCE here: the path
// resolver, the title's quant-bounds catalogue, the title's label catalogue, and the
// Halo client. `fetchOne` itself touches no configuration — which is what lets it be
// driven from a test against a throwaway root, and what will let `watch` (#8) archive a
// hundred matches without re-reading a catalogue a hundred times.
//
// NO `internal/config` HERE, AND THAT IS THE REASON THE CREDENTIAL IS AN XUID AND NOT A
// GAMERTAG. `config` is the only package that reads db_profiles.json (gamertag -> xuid),
// and it pulls DuckDB — hence cgo — into whatever imports it. cmd/replay-build, the
// offline tool this one extends, is deliberately cgo-free for the same reason, and so is
// this one: `CGO_ENABLED=0 go test ./cmd/study-archiver/` runs anywhere, with no C
// toolchain. Ticket #6 brings the archive database in and with it cgo; that is when
// `--player <Gamertag>` becomes free and can replace `--xuid`.

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/replaylabels"
	authpkg "levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/sync/haloclient"
)

// depsRequest is what the CLI knows before anything is read from disk.
type depsRequest struct {
	// XUID identifies the token that authenticates the run, in
	// data/auth/watcher_tokens/{xuid}.json. Unused when SPARTAN_TOKEN is set.
	XUID string
	// Gamertag is optional and only labels the store's re-auth flag if the refresh
	// chain turns out to be revoked.
	Gamertag        string
	Title           string
	FrameIntervalMS int
	RequestsPerSec  int
}

// newOfflineDeps is newDeps WITHOUT the Halo client, for the subcommands that make no
// network call: `rebuild` (#10) and `status` (#9).
//
// NO CREDENTIAL IS REQUIRED, and that is the point rather than a shortcut. `rebuild` exists
// for the day the decoder improves, when every match worth rebuilding has a CDN link that
// died months ago — a rebuild that needed a token would fail on machines and in situations
// where the bytes on disk are perfectly sufficient. deps.Client stays nil: any path that
// reaches for it panics loudly in a test rather than quietly downloading.
func newOfflineDeps(ctx context.Context, req depsRequest) (deps, error) {
	return newDepsWith(ctx, req, false)
}

// newDeps loads the catalogues and authenticates the Halo client.
func newDeps(ctx context.Context, req depsRequest) (deps, error) {
	return newDepsWith(ctx, req, true)
}

func newDepsWith(ctx context.Context, req depsRequest, withClient bool) (deps, error) {
	repoRoot, err := title.FindRepoRoot()
	if err != nil {
		return deps{}, fmt.Errorf("repo root: %w", err)
	}
	paths := title.NewPathResolver(repoRoot)

	// The map catalogue is MANDATORY: without it no match can be judged supported, and
	// the archiver would have no way to refuse a build rather than produce one at
	// another map's scale.
	catalog, err := filmdec.LoadMapQuantCatalog(paths.MapQuantBoundsPath(req.Title))
	if err != nil {
		return deps{}, fmt.Errorf("map bounds catalogue: %w", err)
	}
	// The label catalogue is mandatory too, for the reason cmd/replay-build already
	// gives: a replay with no labels is indistinguishable on screen from a replay whose
	// weapons are unknown. Refuse to archive rather than publish a mute document.
	labels, err := replaylabels.Load(repoRoot, req.Title)
	if err != nil {
		return deps{}, fmt.Errorf("title label catalogue (%s): %w", req.Title, err)
	}

	var client filmAPI
	if withClient {
		tokens, tErr := resolveTokens(ctx, paths, req)
		if tErr != nil {
			return deps{}, tErr
		}
		client = haloclient.NewHaloAPIClient(
			tokens.SpartanToken, tokens.ClearanceToken, req.RequestsPerSec)
	}
	// Opened LAST, and only once every read-only prerequisite has succeeded: an archive
	// handle taken before a failing catalogue load would leave a DuckDB file locked by a
	// process that is about to exit with an error.
	store, err := openArchiveAt(paths)
	if err != nil {
		return deps{}, err
	}
	slog.InfoContext(ctx, "study-archiver: ready",
		"titleSlug", req.Title, "maps", len(catalog.Maps), "repo_root", repoRoot,
		"archive", paths.StudyArchiveDBPath(), "network", withClient)

	return deps{
		Client:  client,
		Paths:   paths,
		Title:   req.Title,
		Catalog: catalog,
		Labels:  labels,
		Archive: store,
		// Handed over unwrapped: the build lock lives at the call site (runBuild), so a
		// wiring cannot forget it.
		Build:           replay.BuildFromFilm,
		SourceGamertag:  req.Gamertag,
		FrameIntervalMS: req.FrameIntervalMS,
	}, nil
}

// resolveTokens obtains the owner's Halo tokens by the project's canonical path
// (ADR 0023): MultiUserTokenStore, no legacy refresh-token environment variable, and NO
// re-capture — a dead refresh token fails loudly and is diagnosed.
//
// The store is the single source (ADR 0023); the legacy inputs are left empty
// deliberately, as cmd/mapobj-build already records: a new tool must not become a new
// reader of the deprecated environment variable.
func resolveTokens(ctx context.Context, paths *title.PathResolver, req depsRequest) (*domain.HaloTokens, error) {
	if envToken := os.Getenv("SPARTAN_TOKEN"); envToken != "" {
		slog.InfoContext(ctx, "study-archiver: Spartan token supplied by the environment")
		return &domain.HaloTokens{
			SpartanToken:   envToken,
			ClearanceToken: os.Getenv("CLEARANCE_TOKEN"),
		}, nil
	}
	if req.XUID == "" {
		return nil, fmt.Errorf("no credential: pass --xuid <xuid> (the owner's own token, " +
			"data/auth/watcher_tokens/{xuid}.json), or set SPARTAN_TOKEN")
	}
	store := authpkg.NewMultiUserTokenStore(paths.WatcherTokensDir())
	result, err := authpkg.RefreshHaloTokensViaStoreFirst(
		ctx, store, authpkg.NewSISUProvider(), req.XUID, req.Gamertag, authpkg.LegacyAuthInputs{})
	if err != nil {
		return nil, err
	}
	tokens := authpkg.HaloTokensFromExchange(result)
	if tokens == nil || tokens.SpartanToken == "" {
		return nil, fmt.Errorf(
			"no usable token for xuid %s - diagnose the refresh chain, do NOT re-capture",
			req.XUID)
	}
	slog.InfoContext(ctx, "study-archiver: tokens obtained",
		"xuid", req.XUID, "clearance", tokens.ClearanceToken != "")
	return tokens, nil
}
