package main

// watch_wiring.go — THE ONE PIECE OF WATCH THAT NEEDS AN XBOX LIVE TOKEN.
//
// Everything else in this tool runs on the owner's SPARTAN token (ADR 0023). Resolving a
// gamertag to an xuid does not: `/hi/players/{id}/matches` demands the `xuid(N)` form —
// passing a textual gamertag returns a stale frozen response rather than a 404, which is
// the worst possible failure for an unattended job — and the only universal resolver is
// Xbox Live's profile endpoint, on an XSTS chain.
//
// NO NEW AUTH PATH IS INVENTED HERE. The chain is the repo's own, in its canonical order:
// ResolveMSAccessTokenStoreFirst (store-first per ADR 0023, no legacy inputs, no
// re-capture) -> AcquireXSTSForRTA -> XSTSResult.AuthHeader -> XboxProfileResolver. The
// header is memoised by CachedHeaderProvider, the same component cmd/server uses, so a
// watchlist of twenty players costs ONE token acquisition rather than twenty.

import (
	"context"
	"fmt"

	"levelup/go-api/internal/domain/title"
	authpkg "levelup/go-api/internal/platform/auth"
)

// newWatchDeps builds the gamertag resolver.
//
// Built LAZILY — the header is acquired on the first gamertag that actually needs
// resolving. A watchlist whose players are all already resolved (the steady state of an
// hourly job) therefore makes no Xbox Live call at all, and a broken XSTS chain does not
// stop a pass that never needed it.
func newWatchDeps(paths *title.PathResolver, req depsRequest) watchDeps {
	header := authpkg.NewCachedHeaderProvider(0, func(ctx context.Context) (string, error) {
		accessToken, err := authpkg.ResolveMSAccessTokenStoreFirst(ctx,
			authpkg.NewSISUProvider(),
			authpkg.NewMultiUserTokenStore(paths.WatcherTokensDir()),
			req.XUID, req.Gamertag, authpkg.LegacyAuthInputs{})
		if err != nil {
			return "", fmt.Errorf("microsoft access token for xuid %s: %w", req.XUID, err)
		}
		if accessToken == "" {
			return "", fmt.Errorf(
				"no microsoft access token for xuid %s: the watchlist cannot be resolved to xuids "+
					"- diagnose the refresh chain, do NOT re-capture", req.XUID)
		}
		xsts, err := authpkg.AcquireXSTSForRTA(ctx, accessToken)
		if err != nil {
			return "", fmt.Errorf("xsts for the xbox profile endpoint: %w", err)
		}
		return xsts.AuthHeader(), nil
	})
	resolver := authpkg.NewXboxProfileResolver(nil, header.Header)
	return watchDeps{ResolveXUID: resolver.ResolveXUID}
}
