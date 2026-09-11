// Package duckdb provides the canonical English medal label expressions.

package duckdb

// medalLabelDescCoalesceSQL returns the English label and description
// expressions. The aliases `md` and `mt_en` must exist in the caller.
func medalLabelDescCoalesceSQL() (labelExpr, descExpr string) {
	labelExpr = `COALESCE(
		NULLIF(TRIM(mt_en.name),''),
		NULLIF(TRIM(md.name_en),''),
		''
	)`
	descExpr = `COALESCE(
		NULLIF(TRIM(mt_en.description),''),
		NULLIF(TRIM(md.description_en),''),
		''
	)`
	return labelExpr, descExpr
}

// medalTranslationJoinsSQL adds the English translation fallback after
// `FROM medal_definitions md`. An empty translation table is valid.
func medalTranslationJoinsSQL() string {
	return `LEFT JOIN medal_translations mt_en
			    ON mt_en.medal_name_id = md.medal_name_id AND mt_en.lang = '` + LangCodeEN + `'`
}
