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

import (
	"fmt"

	"levelup/go-api/internal/analysis/filmdec"
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
// Both failure modes come back as a skipError with a NAMED reason: the caller keeps the
// downloaded film either way, and only the BUILD is skipped.
func resolveMatchMap(name string, cat *filmdec.MapQuantCatalog) (matchMap, error) {
	if name == "" {
		return matchMap{}, skipError{
			Reason: skipNoMapInStats,
			Detail: "match stats carry no MatchInfo.MapVariant.PublicName",
		}
	}
	entry, err := cat.Lookup(name)
	if err != nil {
		return matchMap{Name: name}, skipError{
			Reason: skipUnsupportedMap,
			Cause:  err,
			Detail: fmt.Sprintf("map %q absent from the title's quant-bounds catalogue", name),
		}
	}
	return matchMap{Name: name, Module: entry.Module, Range: entry.Range()}, nil
}
