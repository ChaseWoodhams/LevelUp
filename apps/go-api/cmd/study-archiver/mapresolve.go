package main

// mapresolve.go — WHICH MAP WAS THIS MATCH PLAYED ON.
//
// The film carries only quantum indices; turning them into world coordinates needs the
// AABB of the map's BSP, which lives in the title's versioned quant-bounds catalogue
// (cf. filmdec.MapQuantCatalog). The catalogue is keyed by the map's DISPLAY name, and
// the only place that name exists for a match is its stats payload.
//
// Same chain as cmd/replay-build's resolveMapEntry, and deliberately the same failure
// rule: no bounds -> NO artifact. A replay built with another map's bounds is wrong by
// an arbitrary scale factor and nothing on screen would say so.
//
// THE NAME IS OFTEN NOT A NAME AT ALL. Halo's match-stats payload has never reliably
// embedded MatchInfo.MapVariant's PublicName — confirmed against real captures, where it
// was absent on every single match seen — and `sync.ExtractRegistry` falls back to the
// raw asset GUID when that happens (its own documented behaviour, not a bug there). A
// GUID never matches the catalogue's display-name keys, so every match would refuse a
// build for a reason that has nothing to do with whether its map is actually supported.
//
// THE FALLBACK IS A LOCAL METADATA LOOKUP, NOT A RE-FETCH. `metadata.duckdb`'s
// `asset_translations` (built by `levelup populate-assets`, or unpacked from the repo's
// prebuilt snapshot) already maps an asset id to its canonical English name — the same
// table `sync.EnrichRegistryFromMetadata` reads for the main app's sync pipeline. Trying
// it here, on a bare catalogue miss, is what lets `rebuild` (#10) recover a match that was
// captured before this fallback existed: its stored `map_name` IS the raw GUID, frozen in
// the archive row, and rebuild is not allowed to make a network call to re-derive it.
// Reading a local DuckDB file is not a network call.

import (
	"context"
	"database/sql"
	"fmt"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/sync"
)

// matchMap is everything the build needs to know about the match's map: the display
// name (for logs and, later, the archive row), the module (which keys the map's frozen
// structure file) and the dequantisation bounds.
type matchMap struct {
	Name   string
	Module string
	Range  filmdec.Vec3Range
}

// resolveMatchMap looks a match's map name up in the title's quant-bounds catalogue.
//
// It takes the NAME rather than the stats payload: the payload is read once, by
// readMatchFacts, through the repo's own `sync.ExtractRegistry`. This function used to
// dig the name out itself, which made it a second reader of
// MatchInfo.MapVariant.PublicName — the duplication #5 recorded as a finding, retired
// here now that the archiver imports that extractor anyway.
//
// `metadataDB` is best-effort and may be nil (an install with no `metadata.duckdb` yet):
// the catalogue lookup is retried with a name resolved from it ONLY when the first
// lookup, on whatever `name` actually is, fails — a match whose stats genuinely carried
// a display name never touches the fallback at all.
//
// Both failure modes come back as a skipError with a NAMED reason: the caller keeps the
// downloaded film either way, and only the BUILD is skipped.
func resolveMatchMap(
	ctx context.Context, name string, cat *filmdec.MapQuantCatalog, metadataDB *sql.DB,
) (matchMap, error) {
	if name == "" {
		return matchMap{}, skipError{
			Reason: skipNoMapInStats,
			Detail: "match stats carry no MatchInfo.MapVariant.PublicName",
		}
	}
	entry, err := cat.Lookup(name)
	if err == nil {
		return matchMap{Name: name, Module: entry.Module, Range: entry.Range()}, nil
	}
	if resolved, ok := resolveViaMetadata(ctx, name, metadataDB); ok {
		if entry2, err2 := cat.Lookup(resolved); err2 == nil {
			return matchMap{Name: resolved, Module: entry2.Module, Range: entry2.Range()}, nil
		}
	}
	return matchMap{Name: name}, skipError{
		Reason: skipUnsupportedMap,
		Cause:  err,
		Detail: fmt.Sprintf("map %q absent from the title's quant-bounds catalogue", name),
	}
}

// resolveViaMetadata treats `name` AS an asset id — which is exactly what it is, on the
// path that reaches here — and asks the local metadata catalogue for its canonical
// English name. `false` covers every reason it cannot: no catalogue, no row, a lookup
// error (logged by LookupAssetCanonicalEN itself); the caller's own catalogue miss is
// the error that actually gets reported.
func resolveViaMetadata(ctx context.Context, name string, metadataDB *sql.DB) (string, bool) {
	if metadataDB == nil {
		return "", false
	}
	canonical, err := sync.LookupAssetCanonicalEN(ctx, metadataDB, games.AssetKindMap, name)
	if err != nil || canonical == "" {
		return "", false
	}
	return canonical, true
}
