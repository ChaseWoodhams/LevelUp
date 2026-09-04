package main

// watchlist.go — WHO THE ARCHIVER FOLLOWS.
//
// A TOML file at the repository root, git-ignored, beside `db_profiles.json` and
// `app_settings.json` — the two other pieces of local configuration the repo keeps there.
// It is git-ignored and shipped as `watchlist.example.toml` for the reason those are: it
// names real people, and whose games somebody studies is not something to publish in a
// public repository by accident.
//
// THE FILE HOLDS GAMERTAGS AND NOTHING ELSE. The xuid each one resolves to is recorded in
// the archive database, not written back here: a resolution is a fact the tool learned,
// not a setting the operator chose, and a tool that rewrites the human's configuration
// file eventually rewrites it wrongly.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// watchlistFileName is the file's name at the repo root, and watchlistExampleName the
// committed sample. Named here rather than at three call sites: the loader, the error
// message that tells the operator what to create, and the test that writes one.
const (
	watchlistFileName    = "watchlist.toml"
	watchlistExampleName = "watchlist.example.toml"
)

// watchlist is the parsed file.
type watchlist struct {
	// Gamertags is the tracked roster, in file order. Order is preserved because it is
	// the order the pass works in, and an operator reading the log should be able to
	// follow it down their own list.
	Gamertags []string `toml:"gamertags"`
}

// loadWatchlist reads and validates the watchlist at the repo root.
//
// An ABSENT file is an error, not an empty run. A `watch` that quietly archives nothing
// looks exactly like a `watch` with nothing new to archive, and the hourly job would
// report success forever while capturing not one film.
func loadWatchlist(repoRoot string) (watchlist, error) {
	path := filepath.Join(repoRoot, watchlistFileName)
	blob, err := os.ReadFile(path) //nolint:gosec // a fixed name under the repo root
	if err != nil {
		if os.IsNotExist(err) {
			return watchlist{}, fmt.Errorf(
				"no watchlist at %s: copy %s to %s and list the gamertags to follow",
				path, watchlistExampleName, watchlistFileName)
		}
		return watchlist{}, fmt.Errorf("reading %s: %w", path, err)
	}
	var wl watchlist
	if err := toml.Unmarshal(blob, &wl); err != nil {
		return watchlist{}, fmt.Errorf("parsing %s: %w", path, err)
	}
	return wl, wl.validate(path)
}

// validate rejects a file that would make the pass silently pointless, and normalises
// what it keeps.
func (w *watchlist) validate(path string) error {
	seen := make(map[string]bool, len(w.Gamertags))
	kept := w.Gamertags[:0]
	for _, gt := range w.Gamertags {
		gt = strings.TrimSpace(gt)
		if gt == "" {
			continue
		}
		// Xbox gamertags are case-insensitive, so two spellings of one player would
		// otherwise be resolved twice, pull the same history twice, and appear as two
		// rows in a `status` report that counts tracked players.
		key := strings.ToLower(gt)
		if seen[key] {
			continue
		}
		seen[key] = true
		kept = append(kept, gt)
	}
	w.Gamertags = kept
	if len(w.Gamertags) == 0 {
		return fmt.Errorf("%s lists no gamertag: a watch pass would archive nothing", path)
	}
	return nil
}
