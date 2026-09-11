package duckdb

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// The metadata mode catalog is read through this file so all callers use the
// same English-only query and the same best-effort error handling.
const selectModeName = `SELECT mode_en, name FROM mode_name_tr WHERE lang = 'en' AND mode_en IN (%s)`
const selectDistinctModeEN = `SELECT DISTINCT mode_en FROM mode_name_tr`
const modeNameQueryTimeout = 3 * time.Second

func queryModeName(ctx context.Context, meta *DB, modeNames []string) (map[string]string, error) {
	if meta == nil || len(modeNames) == 0 {
		return nil, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(modeNames)), ",")
	args := make([]any, len(modeNames))
	for i, name := range modeNames {
		args[i] = name
	}
	rows, err := meta.QueryRecovered(ctx, fmt.Sprintf(selectModeName, placeholders), args...)
	if err != nil {
		if isTableNotFoundErr(err) {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]string, len(modeNames))
	for rows.Next() {
		var modeName, label string
		if scanErr := rows.Scan(&modeName, &label); scanErr != nil {
			continue
		}
		if strings.TrimSpace(label) != "" {
			out[modeName] = label
		}
	}
	return out, rows.Err()
}

func loadModeNamesForKeys(ctx context.Context, meta *DB, modeNames []string) map[string]string {
	if meta == nil || len(modeNames) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, modeNameQueryTimeout)
	defer cancel()
	out, err := queryModeName(ctx, meta, modeNames)
	if err != nil {
		slog.WarnContext(ctx, "metadata_labels: load mode names failed", "err", err)
		return nil
	}
	return out
}

func loadKnownModesEN(ctx context.Context, meta *DB) []string {
	if meta == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, modeNameQueryTimeout)
	defer cancel()
	rows, err := meta.QueryRecovered(ctx, selectDistinctModeEN)
	if err != nil {
		if !isTableNotFoundErr(err) {
			slog.WarnContext(ctx, "metadata_labels: load known modes failed", "err", err)
		}
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var name string
		if rows.Scan(&name) == nil && strings.TrimSpace(name) != "" {
			out = append(out, name)
		}
	}
	return out
}
