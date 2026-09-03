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

// resolveMatchMap reads the match's map out of its raw stats payload and looks it up in
// the title's quant-bounds catalogue.
//
// Both failure modes come back as a skipError with a NAMED reason: the caller keeps the
// downloaded film either way, and only the BUILD is skipped.
func resolveMatchMap(stats map[string]any, cat *filmdec.MapQuantCatalog) (matchMap, error) {
	name := mapNameFromStats(stats)
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

// mapNameFromStats reads MatchInfo.MapVariant.PublicName out of the raw stats payload.
//
// A second reader of that path (internal/sync has its own, unexported, serving the
// canonical row builder). Two copies is the repo's limit, not an invitation to a third:
// a third caller centralises this into a shared extractor.
func mapNameFromStats(stats map[string]any) string {
	info, _ := stats["MatchInfo"].(map[string]any)
	if info == nil {
		return ""
	}
	variant, _ := info["MapVariant"].(map[string]any)
	if variant == nil {
		return ""
	}
	name, _ := variant["PublicName"].(string)
	return name
}
