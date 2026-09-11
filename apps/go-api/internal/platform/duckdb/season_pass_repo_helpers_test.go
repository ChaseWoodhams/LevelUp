package duckdb

import (
	"context"
	"strings"
	"testing"
)

func TestLocalizedText_LocaleOrdering(t *testing.T) {
	multi := map[string]any{"fr": "Français", "en": "English", "default": "Default"}
	if got := localizedText(multi, false); got != "Français" {
		t.Errorf("preferEN=false: got %q", got)
	}
	if got := localizedText(multi, true); got != "English" {
		t.Errorf("preferEN=true: got %q", got)
	}
	resolved := map[string]any{"value": "Resolved", "fr": "Français", "en": "English"}
	if got := localizedText(resolved, true); got != "Resolved" {
		t.Errorf("value prime (EN): got %q", got)
	}
	if got := localizedText(resolved, false); got != "Resolved" {
		t.Errorf("value prime (fallback): got %q", got)
	}
	frOnly := map[string]any{"fr": "SeulementFR"}
	if got := localizedText(frOnly, true); got != "SeulementFR" {
		t.Errorf("fallback: got %q", got)
	}
	enOnly := map[string]any{"en": "OnlyEN"}
	if got := localizedText(enOnly, false); got != "OnlyEN" {
		t.Errorf("fallback: got %q", got)
	}
	if got := localizedText("  plain  ", true); got != "plain" {
		t.Errorf("plain: got %q", got)
	}
}

func TestBPItemFieldCoalesce_ValueIsEnglishSource(t *testing.T) {
	sql := bpItemFieldCoalesce(context.Background(), "Title", "title")
	posValue := strings.Index(sql, "Title.value")
	posEnglish := strings.Index(sql, "Title.translations.en-US")
	if posValue < 0 || posEnglish < 0 {
		t.Fatalf("English expressions absent from %q", sql)
	}
	if posEnglish > posValue {
		t.Errorf("English translation should precede the raw value fallback, got %q", sql)
	}
}
