package main

// mapresolve_online.go — THE MAP'S NAME FROM HALO ITSELF, WHEN NOTHING LOCAL HAS IT.
//
// mapresolve.go recovers a map whose stats carried only an asset id through the local metadata
// catalogue. That fallback fails SILENTLY whenever the main app is running: the server holds
// metadata.duckdb read-write, DuckDB admits one process per file (measured, cf.
// docs/RUNBOOK_OPS_DUCKDB_CLI_TOOLS.md), so the archiver opens no handle at all and every such
// match is recorded unsupported_map - 139 of the first 145 matches archived.
//
// The subcommands that fetch (fetch-one, watch, recapture) already hold the owner's token, so
// they ask Halo's discovery API for the asset's English name: the same authenticated call the
// app's sync makes to fill asset_translations (halo.NewAssetNameFetcher). The resolved name is
// what the archive row records, so a later offline `rebuild` finds a real name in the row and
// needs neither the network nor the metadata catalogue.

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"levelup/go-api/internal/assetnames"
	"levelup/go-api/internal/games"
)

// resolveMatchMapOnline is resolveMatchMap with one last fallback, on the discovery API.
//
// It runs ONLY on a miss: a match whose stats named a supported map, or whose asset id the local
// catalogue resolved, never makes the call. It needs the map's asset id AND version id, which
// discovery requires; without either, or without a fetcher (offline subcommands, tests), the
// local verdict stands.
func resolveMatchMapOnline(ctx context.Context, d deps, facts matchFacts) (matchMap, error) {
	m, err := resolveMatchMap(ctx, facts.MapName, d.Catalog, d.MetadataDB)
	var skip skipError
	if err == nil || !errors.As(err, &skip) ||
		(skip.Reason != skipUnsupportedMap && skip.Reason != skipNoMapInStats) {
		return m, err
	}
	if d.AssetNames == nil || facts.MapID == "" || facts.MapVersionID == "" {
		return m, err
	}
	name, fErr := d.AssetNames.FetchName(ctx, games.AssetKindMap, d.Title, facts.MapID,
		facts.MapVersionID, assetnames.DefaultLangs[0])
	name = strings.TrimSpace(name)
	if fErr != nil || name == "" {
		slog.WarnContext(ctx, "study-archiver: map name not resolved by the discovery API",
			"map_id", facts.MapID, "map_version_id", facts.MapVersionID, "err", fErr)
		return m, err
	}
	entry, lErr := d.Catalog.Lookup(name)
	if lErr != nil {
		// A real name whose bounds have not arrived yet: still a skip, but the row now carries
		// a readable name instead of a GUID.
		return matchMap{Name: name}, skipError{
			Reason: skipUnsupportedMap,
			Cause:  lErr,
			Detail: fmt.Sprintf("map %q absent from the title's quant-bounds catalogue", name),
		}
	}
	slog.InfoContext(ctx, "study-archiver: map name resolved by the discovery API",
		"map_id", facts.MapID, "map", name)
	return matchMap{Name: name, Module: entry.Module, Range: entry.Range()}, nil
}
