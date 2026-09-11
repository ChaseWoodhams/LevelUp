package main

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/assetnames"
	"levelup/go-api/internal/domain/title"
)

// stubAssetNames answers the discovery API's name lookup with a fixed name or error.
type stubAssetNames struct {
	name        string
	err         error
	calls       int
	gotID       string
	gotVersion  string
	gotAssetTyp string
}

func (s *stubAssetNames) FetchName(_ context.Context, assetType, _, assetID, versionID, _ string) (string, error) {
	s.calls++
	s.gotAssetTyp, s.gotID, s.gotVersion = assetType, assetID, versionID
	return s.name, s.err
}

func onlineDeps(f assetnames.Fetcher) deps {
	return deps{Catalog: testCatalog(), Title: title.DefaultSlug, AssetNames: f}
}

// The case the fallback exists for: stats carrying only the asset id, no local metadata.
func TestResolveMatchMapOnline_NamesAnAssetIDThroughDiscovery(t *testing.T) {
	stub := &stubAssetNames{name: "Cliffhanger"}
	facts := matchFacts{MapName: cliffhangerAssetID, MapID: cliffhangerAssetID, MapVersionID: "v7"}

	got, err := resolveMatchMapOnline(context.Background(), onlineDeps(stub), facts)
	if err != nil {
		t.Fatalf("resolveMatchMapOnline: %v", err)
	}
	if got.Name != "Cliffhanger" || got.Module != "olympus" {
		t.Errorf("name/module = %q/%q, want Cliffhanger/olympus", got.Name, got.Module)
	}
	if stub.calls != 1 || stub.gotID != cliffhangerAssetID || stub.gotVersion != "v7" || stub.gotAssetTyp != "map" {
		t.Errorf("fetch = %d call(s) for %s %s@%s, want one map lookup of the asset id and version",
			stub.calls, stub.gotAssetTyp, stub.gotID, stub.gotVersion)
	}
}

// A supported display name never reaches the network.
func TestResolveMatchMapOnline_ACatalogueHitMakesNoCall(t *testing.T) {
	stub := &stubAssetNames{name: "should not be asked"}
	facts := matchFacts{MapName: "Cliffhanger", MapID: cliffhangerAssetID, MapVersionID: "v7"}

	if _, err := resolveMatchMapOnline(context.Background(), onlineDeps(stub), facts); err != nil {
		t.Fatalf("resolveMatchMapOnline: %v", err)
	}
	if stub.calls != 0 {
		t.Errorf("discovery called %d time(s) for a map the catalogue already knew", stub.calls)
	}
}

func TestResolveMatchMapOnline_AFailedLookupKeepsTheLocalVerdict(t *testing.T) {
	stub := &stubAssetNames{err: errors.New("discovery 404")}
	facts := matchFacts{MapName: cliffhangerAssetID, MapID: cliffhangerAssetID, MapVersionID: "v7"}

	got, err := resolveMatchMapOnline(context.Background(), onlineDeps(stub), facts)
	var skip skipError
	if !errors.As(err, &skip) || skip.Reason != skipUnsupportedMap {
		t.Fatalf("err = %v, want skipError(%s)", err, skipUnsupportedMap)
	}
	if got.Name != cliffhangerAssetID {
		t.Errorf("name = %q, want the recorded asset id left as it was", got.Name)
	}
}

// A real name without bounds is still a skip, but the row gains the readable name.
func TestResolveMatchMapOnline_ResolvedButNoBoundsKeepsTheName(t *testing.T) {
	stub := &stubAssetNames{name: "Forbidden Sands"}
	facts := matchFacts{MapName: cliffhangerAssetID, MapID: cliffhangerAssetID, MapVersionID: "v7"}

	got, err := resolveMatchMapOnline(context.Background(), onlineDeps(stub), facts)
	var skip skipError
	if !errors.As(err, &skip) || skip.Reason != skipUnsupportedMap {
		t.Fatalf("err = %v, want skipError(%s)", err, skipUnsupportedMap)
	}
	if got.Name != "Forbidden Sands" {
		t.Errorf("name = %q, want the resolved name recorded", got.Name)
	}
}

// Discovery requires the version id: without it (or without a fetcher) nothing is asked.
func TestResolveMatchMapOnline_NoVersionOrNoFetcherMakesNoCall(t *testing.T) {
	stub := &stubAssetNames{name: "Cliffhanger"}
	noVersion := matchFacts{MapName: cliffhangerAssetID, MapID: cliffhangerAssetID}
	if _, err := resolveMatchMapOnline(context.Background(), onlineDeps(stub), noVersion); err == nil {
		t.Error("resolved a map with no version id")
	}
	if stub.calls != 0 {
		t.Errorf("discovery called %d time(s) without a version id", stub.calls)
	}

	offline := deps{Catalog: testCatalog(), Title: title.DefaultSlug}
	facts := matchFacts{MapName: cliffhangerAssetID, MapID: cliffhangerAssetID, MapVersionID: "v7"}
	if _, err := resolveMatchMapOnline(context.Background(), offline, facts); err == nil {
		t.Error("resolved a map offline with no metadata and no fetcher")
	}
}
