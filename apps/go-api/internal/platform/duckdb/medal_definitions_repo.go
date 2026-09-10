// Package duckdb provides metadata repositories for the English-only API.

package duckdb

import (
	"context"
	"fmt"
	"time"

	"levelup/go-api/internal/port"
)

// Codes BCP-47 utilisés dans les tables de traductions DuckDB
// (medal_translations, asset_translations, mode_translations). Externalisés
// pour goconst (chacun apparaît à >10 endroits dans les repos).
const LangCodeEN = "en-US"

// MedalDefinitionsRepo implémente port.MedalDefinitionsRepository.
// Requête sur pdb.Metadata (medal_definitions + medal_translations).
type MedalDefinitionsRepo struct {
	pdb *PlayerDB
}

// NewMedalDefinitionsRepo crée un MedalDefinitionsRepo lié à un PlayerDB.
func NewMedalDefinitionsRepo(pdb *PlayerDB) *MedalDefinitionsRepo {
	return &MedalDefinitionsRepo{pdb: pdb}
}

func (r *MedalDefinitionsRepo) LookupByIDs(
	ctx context.Context,
	ids []int64,
	_ string,
) (map[int64]port.MedalDefinitionRow, error) {
	result := make(map[int64]port.MedalDefinitionRow, len(ids))
	if len(ids) == 0 || r.pdb == nil || r.pdb.Metadata == nil {
		return result, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Source unique de la chaîne label/description (locale-aware) : helper partagé
	// medal_label_resolve.go. Corrige le COALESCE divergent qui omettait md.name_fr /
	// md.description_fr (médailles non traduites sur Escouade/Explorer).
	labelExpr, descExpr := medalLabelDescCoalesceSQL()
	q, args, ok := buildLookupQuery(
		`SELECT md.medal_name_id,
		        `+labelExpr+` AS label,
		        `+descExpr+` AS description,
		        COALESCE(NULLIF(TRIM(md.difficulty),''), 'Normal') AS difficulty,
		        COALESCE(NULLIF(TRIM(md.medal_type),''), '') AS medal_type,
		        COALESCE(md.personal_score, 0) AS personal_score
		 FROM medal_definitions md
			 `+medalTranslationJoinsSQL()+`
		 WHERE md.medal_name_id IN (%s)`,
		ids,
	)
	if !ok {
		return result, nil
	}

	rows, err := r.pdb.Metadata.Query(ctx, q, args...)
	if err != nil {
		return result, fmt.Errorf("MedalDefinitionsRepo.LookupByIDs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var label, description, difficulty, medalType string
		var personalScore int
		if err := rows.Scan(&id, &label, &description, &difficulty, &medalType, &personalScore); err != nil {
			return result, fmt.Errorf("MedalDefinitionsRepo.LookupByIDs: scan: %w", err)
		}
		result[id] = port.MedalDefinitionRow{
			MedalID:       id,
			Label:         label,
			Description:   description,
			Difficulty:    difficulty,
			MedalType:     medalType,
			PersonalScore: personalScore,
		}
	}
	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("MedalDefinitionsRepo.LookupByIDs: rows: %w", err)
	}

	// Fallback citation_mappings pour les IDs absents de medal_definitions OU
	// sans libellé exploitable (parité avec la vue Match — corrige Explorer
	// top_medals + Squad qui affichaient des libellés vides). Source unique :
	// medal_citation_fallback.go.
	missing := make([]int64, 0)
	for _, id := range ids {
		if row, ok := result[id]; !ok || row.Label == "" {
			missing = append(missing, id)
		}
	}
	for id, label := range lookupMedalCitationLabels(ctx, r.pdb.Metadata, missing) {
		result[id] = port.MedalDefinitionRow{MedalID: id, Label: label, Difficulty: "Normal"}
	}
	return result, nil
}
