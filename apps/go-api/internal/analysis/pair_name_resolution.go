package analysis

import "strings"

// ResolvePairName returns the canonical English label for a mode/map pair.
// The raw registry label is preferred; the resolved asset label covers UUID
// records, and the stored label is the final compatibility fallback.
func ResolvePairName(rawPairName, currentName, pairAssetName string, _ map[string]string) string {
	for _, value := range []string{rawPairName, pairAssetName, currentName} {
		value = strings.TrimSpace(value)
		if value == "" || looksLikeAssetID(value) {
			continue
		}
		if normalized := NormalizeModeLabel(value); normalized != "" {
			return normalized
		}
	}
	return ""
}

func looksLikeAssetID(value string) bool {
	return len(value) == 36 && strings.Count(value, "-") == 4
}

// needsLabelOverride reports whether a stored label is empty or still mirrors
// the raw English label and can therefore be refreshed from the asset catalog.
func needsLabelOverride(storedLabel, rawLabel string) bool {
	stored := strings.TrimSpace(storedLabel)
	if stored == "" {
		return true
	}
	raw := strings.TrimSpace(rawLabel)
	return raw != "" && strings.EqualFold(stored, raw)
}
