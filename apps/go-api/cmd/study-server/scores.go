package main

import (
	"context"
	"fmt"
)

// The archiver owns migrations. Older archives remain readable until the next writer open
// adds the nullable score columns; the server must never upgrade a database while browsing it.
func (a *archive) matchColumns(ctx context.Context) (string, error) {
	var count int
	err := a.db.QueryRowContext(ctx, `
  SELECT count(*) FROM information_schema.columns
  WHERE table_schema = 'main' AND table_name = 'matches'
    AND column_name IN ('team0_score', 'team1_score')`).Scan(&count)
	if err != nil {
		return "", fmt.Errorf("reading archive score schema: %w", err)
	}
	if count == 2 {
		return summaryColumns + `, m.team0_score, m.team1_score`, nil
	}
	return summaryColumns + `, NULL, NULL`, nil
}
