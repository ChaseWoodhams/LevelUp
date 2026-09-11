// coach.go — Rich Embed Discord pour le relais externe des notifications coach
// (proposals de progression les plus fortes). Consommé par le package
// internal/notifications/external, qui décide QUAND relayer (opt-in strict) ;
// ce fichier ne fait que CONSTRUIRE l'embed en réutilisant le client webhook,
// les couleurs et le footer title-aware du package notify.
//
// Contenu TECHNIQUE volontairement sobre : les libellés utilisateur des
// notifications coach (titre/corps) vivent côté front (clés i18n title_key /
// body_key résolues dans le navigateur, cf. internal/notifications/types.go) et
// ne sont PAS disponibles côté Go. L'embed rend donc : catégorie humanisée (FR/EN),
// joueur, params clés triés, et un lien app optionnel si une base URL publique
// est configurée.
package notify

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// maxCoachParamFields borne le nombre de params rendus dans l'embed (garde-fou
// anti-bloat : un payload coach ne doit jamais produire un embed géant).
const maxCoachParamFields = 6

// maxCoachParamValueLen tronque une valeur de param trop longue dans l'embed.
const maxCoachParamValueLen = 200

// CoachEmbedInput porte les données title-agnostic nécessaires à l'embed coach.
// Aucun libellé FR/EN en dur : la catégorie est humanisée via coachCategoryLabel,
// les params sont rendus bruts (clé technique = valeur).
type CoachEmbedInput struct {
	Category string         // catégorie notification (ex. "milestone_unlocked")
	Severity string         // "info" | "success" | "warn" | "error"
	Player   string         // gamertag du joueur concerné
	Params   map[string]any // params métier de la notification (metric, value, ...)
	AppURL   string         // lien profil/app optionnel ("" = pas de champ lien)
	Lang     string         // "fr" | "en"
}

// BuildCoachEmbed construit le Rich Embed Discord d'un signal coach.
// labels fournit le footer title-aware (nil → libellés Halo, byte-identique).
func BuildCoachEmbed(in CoachEmbedInput, labels NotifyLabels) Embed {
	lang := "en"
	embed := Embed{
		Title:       T("discord_coach_title", lang),
		Description: coachCategoryLabel(in.Category),
		Color:       coachColor(in.Severity),
		Footer:      &EmbedFooter{Text: discordFooterText(labels)},
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Fields:      coachEmbedFields(in, lang),
	}
	return embed
}

// coachEmbedFields assemble les champs de l'embed : joueur, catégorie, params
// clés triés (déterministe), puis lien app optionnel.
func coachEmbedFields(in CoachEmbedInput, lang string) []EmbedField {
	fields := make([]EmbedField, 0, 4)
	if in.Player != "" {
		fields = append(fields, EmbedField{
			Name:   T("discord_coach_player", lang),
			Value:  in.Player,
			Inline: true,
		})
	}
	fields = append(fields, EmbedField{
		Name:   T("discord_coach_category", lang),
		Value:  in.CategoryOrRaw(),
		Inline: true,
	})
	if details := coachParamsValue(in.Params); details != "" {
		fields = append(fields, EmbedField{
			Name:  T("discord_coach_details", lang),
			Value: details,
		})
	}
	if in.AppURL != "" {
		fields = append(fields, EmbedField{
			Name:  T("discord_coach_link", lang),
			Value: in.AppURL,
		})
	}
	return fields
}

// CategoryOrRaw retourne la clé de catégorie brute (utile pour le champ « code »
// technique, distinct du libellé humanisé de la description).
func (in CoachEmbedInput) CategoryOrRaw() string {
	if in.Category == "" {
		return "-"
	}
	return in.Category
}

// coachParamsValue rend les params en une liste `clé = valeur` triée par clé
// (déterministe pour les tests), plafonnée à maxCoachParamFields entrées.
func coachParamsValue(params map[string]any) string {
	if len(params) == 0 {
		return ""
	}
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if len(keys) > maxCoachParamFields {
		keys = keys[:maxCoachParamFields]
	}
	lines := make([]string, 0, len(keys))
	for _, k := range keys {
		v := fmt.Sprintf("%v", params[k])
		if len(v) > maxCoachParamValueLen {
			v = v[:maxCoachParamValueLen] + "…"
		}
		lines = append(lines, fmt.Sprintf("`%s` = %s", k, v))
	}
	return strings.Join(lines, "\n")
}

// coachColor mappe la severity de notification sur une couleur Discord.
func coachColor(severity string) int {
	switch severity {
	case "success":
		return colorSuccess
	case "warn":
		return colorWarning
	case "error":
		return colorError
	default:
		return colorBlurple
	}
}

// coachCategoryLabel returns the English label of a coach category. Fallback: the raw
// key when the category is unknown (clean degradation, never a panic).
func coachCategoryLabel(category string) string {
	if lbl, ok := coachCategoryLabels[category]; ok && lbl != "" {
		return lbl
	}
	if category == "" {
		return "-"
	}
	return category
}

// Category key and label also asserted by this package's tests.
const (
	coachCategoryPatternLever   = "pattern_lever"
	coachLabelMilestoneUnlocked = "Milestone unlocked"
)

// coachCategoryLabels: English labels of the relayed coach categories (keys come from
// internal/progression/coach/emitter.go via AlertType.NotificationCategory; consistency
// guard-rail in internal/notifications/external). Neutral, non-judgemental wording,
// in the coach's spirit (positive signals / things to consolidate).
var coachCategoryLabels = map[string]string{
	"personal_record":         "Personal record",
	"record_near_miss":        "Record within reach",
	"milestone_unlocked":      coachLabelMilestoneUnlocked,
	"milestone_near_miss":     "Milestone within reach",
	"lusr_tier_approach":      "LUSR tier approach",
	"streak_milestone":        "Streak milestone",
	"comeback_welcome":        "Comeback",
	"threshold_crossed":       "Threshold crossed",
	"trend_consolidate":       "Axis to consolidate",
	"pattern_strength":        "Strength detected",
	"pattern_weakness":        "Weakness detected",
	"pattern_behavior":        "Play pattern",
	coachCategoryPatternLever: "Priority lever",
	"combat_pattern":          "Combat profile",
}
