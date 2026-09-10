// Outil ops : peuple la metadata.duckdb Halo 5 (médailles, cartes, désignations CSR)
// depuis l'API Metadata OFFICIELLE Halo 5 (www.haloapi.com). Ces référentiels
// canoniques ne sont PAS sur les endpoints internes SpartanToken (cf.
// .ai/PLAN_H5_ASSETS.md) — seule l'API officielle les expose. Auth = clé
// d'abonnement Azure APIM (Ocp-Apim-Subscription-Key), lue dans l'env
// LEVELUP_HALOAPI_KEY — JAMAIS committée. Données Halo 5 figées → seed one-shot
// idempotent (INSERT OR REPLACE).
//
// Usage : LEVELUP_HALOAPI_KEY=<clé> LEVELUP_REPO_ROOT=<repo> go run ./cmd/h5-metadata-fetch
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/canonical"
	halo5 "levelup/go-api/internal/games/halo_5"
	halo5migrations "levelup/go-api/internal/games/halo_5/migrations"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
)

const officialMetaBase = "https://www.haloapi.com/metadata/h5/metadata/"

func main() {
	key := os.Getenv("LEVELUP_HALOAPI_KEY")
	if key == "" {
		fatal("LEVELUP_HALOAPI_KEY is required")
	}
	cfg, err := config.Load()
	if err != nil {
		fatal("config.Load: %v", err)
	}
	pr := titlePkg.NewPathResolver(cfg.RepoRoot)
	metaPath := pr.MetadataDBPath(halo5.TitleSlug)

	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	halo5migrations.Register()
	db, err := sql.Open("duckdb", metaPath)
	if err != nil {
		fatal("open metadata %s: %v", metaPath, err)
	}
	defer db.Close()
	if err := migration.RunForTitleDB(db, halo5.TitleSlug, migration.TargetMetadata); err != nil {
		fatal("provision metadata h5: %v", err)
	}
	fmt.Printf("metadata h5: %s\n", metaPath)

	seedMedals(db, key)
	seedMaps(db, key)
	seedWeapons(db, key)
	seedCSRDesignations(db, key)
	seedTeamColors(db, key)
	seedCommendations(db, key)
	seedPlaylists(db, key)
	seedAssetTranslations(db, key)

	logUnresolvedMaps(db, pr.SharedDBPath(halo5.TitleSlug))
}

// logUnresolvedMaps ouvre le registre H5 en lecture seule et WARN (slog) la liste
// des map_id référencés par match_registry qui n'ont PAS d'entrée asset_translations
// (map, en-US) dans la metadata — le cas d'origine du bug « carte vide » (Tidal).
// Best-effort : registre indisponible (tenu RW par un writer) → skip loggé.
func logUnresolvedMaps(metaDB *sql.DB, registryPath string) {
	regDB, err := sql.Open("duckdb", registryPath+"?access_mode=read_only")
	if err != nil {
		slog.Warn("h5_metadata: garde-fou maps non résolues sauté (open registre)", "err", err)
		return
	}
	defer regDB.Close()

	resolved := readAssetNames(metaDB, "map") // asset_id → nom EN seedé
	rows, err := regDB.Query(
		`SELECT DISTINCT map_id FROM match_registry WHERE TRIM(COALESCE(map_id,'')) != ''`)
	if err != nil {
		slog.Warn("h5_metadata: garde-fou maps non résolues sauté (query registre)", "err", err)
		return
	}
	defer rows.Close()

	var unresolved []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		if name, ok := resolved[id]; !ok || strings.TrimSpace(name) == "" {
			unresolved = append(unresolved, id)
		}
	}
	if len(unresolved) == 0 {
		slog.Info("h5_metadata: toutes les maps du registre sont résolues", "count_registry_maps", len(resolved))
		return
	}
	slog.Warn("h5_metadata: maps référencées par le registre SANS nom résolu (ajouter un override maps_by_id)",
		"count", len(unresolved), "map_ids", unresolved)
}

// apiPlaylist — élément de l'API Metadata officielle /playlists. `isRanked` fait foi
// pour classer match_registry.is_ranked (par playlist_id = UUID). Source autoritative
// du ranked H5 (cf. .ai, ne PAS dériver des parties).
type apiPlaylist struct {
	Name     string `json:"name"`
	ID       string `json:"id"`
	IsRanked bool   `json:"isRanked"`
}

// seedPlaylists peuple la table `playlists` (id, name, is_ranked) depuis l'API
// Metadata officielle. Table de référence (catalogue) créée à la volée si absente.
// Sert à classer is_ranked des matchs H5 (jointure offline sur playlist_id).
func seedPlaylists(db *sql.DB, key string) {
	body, err := fetchMeta(key, "playlists")
	if err != nil {
		fmt.Printf("playlists: SKIP (%v)\n", err)
		return
	}
	var pls []apiPlaylist
	if err := json.Unmarshal(body, &pls); err != nil {
		fmt.Printf("playlists: parse %v\n", err)
		return
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS playlists (
		id VARCHAR PRIMARY KEY, name VARCHAR, is_ranked BOOLEAN)`); err != nil {
		fmt.Printf("playlists: create table %v\n", err)
		return
	}
	n, ranked := 0, 0
	for _, p := range pls {
		if p.ID == "" {
			continue
		}
		if _, err := db.Exec(`INSERT OR REPLACE INTO playlists (id, name, is_ranked) VALUES (?,?,?)`,
			p.ID, strings.TrimSpace(p.Name), p.IsRanked); err != nil {
			fmt.Printf("playlists: insert %s: %v\n", p.ID, err)
			continue
		}
		n++
		if p.IsRanked {
			ranked++
		}
	}
	fmt.Printf("playlists: %d seedées (%d ranked) sur %d retournées\n", n, ranked, len(pls))
}

// fetchMeta récupère un type de métadonnée officiel (corps JSON brut) en EN
// (langue par défaut de l'API). Conserve la signature historique des seeders EN.
func fetchMeta(key, typ string) ([]byte, error) {

	req, err := http.NewRequest(http.MethodGet, officialMetaBase+typ, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Ocp-Apim-Subscription-Key", key)
	req.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d for %s: %.200s", resp.StatusCode, typ, body)
	}
	return body, nil
}

type apiMedal struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	Classification string `json:"classification"`
	Difficulty     int    `json:"difficulty"`
	ID             string `json:"id"`
	SpriteLocation struct {
		SpriteSheetURI string `json:"spriteSheetUri"`
		Left           int    `json:"left"`
		Top            int    `json:"top"`
		Width          int    `json:"width"`
		Height         int    `json:"height"`
	} `json:"spriteLocation"`
}

func seedMedals(db *sql.DB, key string) {
	body, err := fetchMeta(key, "medals")
	if err != nil {
		fmt.Printf("medals: SKIP (%v)\n", err)
		return
	}
	var medals []apiMedal
	if err := json.Unmarshal(body, &medals); err != nil {
		fmt.Printf("medals: parse %v\n", err)
		return
	}
	n := 0
	for _, m := range medals {
		id, perr := strconv.ParseInt(m.ID, 10, 64)
		if perr != nil {
			continue // id non numérique → ignoré (medal_name_id = BIGINT)
		}
		// h5 : `difficulty` (0..245) n'est PAS l'enum 0-3 d'Infinite → stocké brut en
		// VARCHAR (difficulty), difficulty_index laissé à 0 (non applicable). medal_type
		// = classification normalisée vers les clés canoniques inter-titres
		// (multikill/spree/skill/style/mode/proficiency/other) via
		// canonical.NormalizeMedalCategory — sinon l'enum brut H5 (MultiKill, Style,
		// CaptureTheFlag…) court-circuite la traduction frontend (categoryLabels).
		// Icône = sprite (feuille + offset).
		_, err := db.Exec(`INSERT OR REPLACE INTO medal_definitions
			(medal_name_id, name_en, name_fr, description_en, description_fr,
			 difficulty_index, difficulty, medal_type,
			 sprite_sheet_url, sprite_left, sprite_top, sprite_width, sprite_height)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			id, m.Name, m.Name, m.Description, m.Description,
			0, strconv.Itoa(m.Difficulty), canonical.NormalizeMedalCategory(m.Classification),
			m.SpriteLocation.SpriteSheetURI, m.SpriteLocation.Left, m.SpriteLocation.Top,
			m.SpriteLocation.Width, m.SpriteLocation.Height)
		if err != nil {
			fmt.Printf("medals: insert %s: %v\n", m.ID, err)
			continue
		}
		n++
	}
	fmt.Printf("medals: %d seedées (sur %d)\n", n, len(medals))
}

type apiMap struct {
	Name               string   `json:"name"`
	ImageURL           string   `json:"imageUrl"`
	SupportedGameModes []string `json:"supportedGameModes"`
	ID                 string   `json:"id"`
}

func seedMaps(db *sql.DB, key string) {
	body, err := fetchMeta(key, "maps")
	if err != nil {
		fmt.Printf("maps: SKIP (%v)\n", err)
		return
	}
	var maps []apiMap
	if err := json.Unmarshal(body, &maps); err != nil {
		fmt.Printf("maps: parse %v\n", err)
		return
	}
	n := 0
	now := time.Now()
	for _, m := range maps {
		if m.ID == "" {
			continue
		}
		_, err := db.Exec(`INSERT OR REPLACE INTO maps_catalog
			(title_slug, map_asset_id, name_canonical, image_url, last_fetched_at)
			VALUES (?,?,?,?,?)`,
			halo5.TitleSlug, m.ID, m.Name, m.ImageURL, now)
		if err != nil {
			fmt.Printf("maps: insert %s: %v\n", m.ID, err)
			continue
		}
		n++
	}
	fmt.Printf("maps: %d seedées (sur %d)\n", n, len(maps))
}

type apiCSRDesignation struct {
	Name           string `json:"name"`
	BannerImageURL string `json:"bannerImageUrl"`
	Tiers          []struct {
		IconImageURL string `json:"iconImageUrl"`
		ID           string `json:"id"`
	} `json:"tiers"`
}

type apiWeapon struct {
	Name              string `json:"name"`
	Type              string `json:"type"`
	LargeIconImageURL string `json:"largeIconImageUrl"`
	ID                string `json:"id"`
}

// seedWeapons
// seedWeapons loads the official English weapon catalog into the metadata database.

func seedWeapons(db *sql.DB, key string) {
	body, err := fetchMeta(key, "weapons")
	if err != nil {
		fmt.Printf("weapons: SKIP (%v)\n", err)
		return
	}
	var weapons []apiWeapon
	if err := json.Unmarshal(body, &weapons); err != nil {
		fmt.Printf("weapons: parse %v\n", err)
		return
	}
	n := 0
	for _, w := range weapons {
		// id officiel = numérique (tient dans weapon_labels.weapon_id UBIGINT).
		id, perr := strconv.ParseInt(w.ID, 10, 64)
		if perr != nil {
			continue
		}
		_, err := db.Exec(`INSERT OR REPLACE INTO weapon_labels
			(weapon_id, name_en, name_fr, icon_url, weapon_type) VALUES (?,?,?,?,?)`,
			id, w.Name, w.Name, w.LargeIconImageURL, w.Type)
		if err != nil {
			fmt.Printf("weapons: insert %s: %v\n", w.ID, err)
			continue
		}
		n++
	}
	fmt.Printf("weapons: %d seeded (of %d)\n", n, len(weapons))
}

func seedCSRDesignations(db *sql.DB, key string) {
	body, err := fetchMeta(key, "csr-designations")
	if err != nil {
		fmt.Printf("csr-designations: SKIP (%v)\n", err)
		return
	}
	var desigs []apiCSRDesignation
	if err := json.Unmarshal(body, &desigs); err != nil {
		fmt.Printf("csr-designations: parse %v\n", err)
		return
	}
	n := 0
	for _, d := range desigs {
		for _, t := range d.Tiers {
			tierID, perr := strconv.Atoi(t.ID)
			if perr != nil {
				continue
			}
			_, err := db.Exec(`INSERT OR REPLACE INTO csr_designations
				(designation_name, tier_id, icon_url, banner_url) VALUES (?,?,?,?)`,
				d.Name, tierID, t.IconImageURL, d.BannerImageURL)
			if err != nil {
				fmt.Printf("csr: insert %s/%s: %v\n", d.Name, t.ID, err)
				continue
			}
			n++
		}
	}
	fmt.Printf("csr-designations: %d tiers seedés (sur %d désignations)\n", n, len(desigs))
}

// apiTeamColor is the official Halo 5 team-color catalog shape.

type apiTeamColor struct {
	Name    string `json:"name"`
	Color   string `json:"color"`
	IconURL string `json:"iconUrl"`
	ID      string `json:"id"`
}

// seedTeamColors
// seedTeamColors stores official team names, colors, and icons.

func seedTeamColors(db *sql.DB, key string) {
	body, err := fetchMeta(key, "team-colors")
	if err != nil {
		fmt.Printf("team-colors: SKIP (%v)\n", err)
		return
	}
	var colors []apiTeamColor
	if err := json.Unmarshal(body, &colors); err != nil {
		fmt.Printf("team-colors: parse %v\n", err)
		return
	}
	n := persistTeamColors(db, colors)
	fmt.Printf("team-colors: %d seedées (sur %d)\n", n, len(colors))
}

// persistTeamColors writes team colors idempotently and mirrors the English name into legacy columns.

func persistTeamColors(db *sql.DB, colors []apiTeamColor) int {
	n := 0
	for _, c := range colors {
		// id officiel = entier sérialisé en string (spec API /team-colors) → team_id INTEGER.
		teamID, perr := strconv.Atoi(strings.TrimSpace(c.ID))
		if perr != nil {
			fmt.Printf("team-colors: id non numérique %q: %v\n", c.ID, perr)
			continue
		}
		nameEN := strings.TrimSpace(c.Name)
		if _, err := db.Exec(`INSERT OR REPLACE INTO team_colors
			(team_id, name_en, name_fr, color, icon_url) VALUES (?,?,?,?,?)`,
			teamID, nameEN, nameEN, strings.TrimSpace(c.Color), strings.TrimSpace(c.IconURL)); err != nil {
			fmt.Printf("team-colors: insert %d: %v\n", teamID, err)
			continue
		}
		n++
	}
	return n
}

// apiCommendation — élément de l'API Metadata officielle /commendations. `id` (UUID)
// = la clé naturelle référencée par carnage ProgressiveCommendationDeltas[].Id.
//
// `levels[]` (Progressive uniquement) porte les PALIERS : chaque level a un
// `threshold` = score cumulé cible du palier (≈ 5 levels/commendation, p.ex.
// "Spartan Slayer" → [1, 41, …]). On extrait la suite croissante des thresholds
// (CSV) pour réutiliser la mécanique de progression des citations Infinite
// (parseTierTargets/ComputeTierProgression).
type apiCommendation struct {
	Type         string `json:"type"` // Progressive | Meta | Daily
	Name         string `json:"name"`
	Description  string `json:"description"`
	IconImageURL string `json:"iconImageUrl"`
	ID           string `json:"id"`
	Category     struct {
		Name string `json:"name"`
	} `json:"category"`
	Levels []struct {
		Threshold int `json:"threshold"`
	} `json:"levels"`
}

// tierTargetsCSV projette les paliers d'une commendation en CSV croissant
// IDENTIQUE au format citation_mappings.tier_targets d'Infinite (réutilise
// analysis.ParseTierTargets / ComputeTierProgression côté lecture). Seuils <= 0
// ignorés ; tri croissant ; chaîne vide si aucun palier exploitable (commendations
// Meta/Daily sans levels → dégradation propre : anneau vide côté front).
func tierTargetsCSV(levels []struct {
	Threshold int `json:"threshold"`
}) string {
	thresholds := make([]int, 0, len(levels))
	for _, l := range levels {
		if l.Threshold > 0 {
			thresholds = append(thresholds, l.Threshold)
		}
	}
	if len(thresholds) == 0 {
		return ""
	}
	sort.Ints(thresholds)
	parts := make([]string, len(thresholds))
	for i, t := range thresholds {
		parts[i] = strconv.Itoa(t)
	}
	return strings.Join(parts, ",")
}

// seedCommendations
// seedCommendations stores the official English commendation catalog and tier thresholds.

func seedCommendations(db *sql.DB, key string) {
	body, err := fetchMeta(key, "commendations")
	if err != nil {
		fmt.Printf("commendations: SKIP (%v)\n", err)
		return
	}
	var comms []apiCommendation
	if err := json.Unmarshal(body, &comms); err != nil {
		fmt.Printf("commendations: parse %v\n", err)
		return
	}
	// Idempotent : garantit la colonne tier_targets même si la DB a été provisionnée
	// avant l'ajout de la migration (parité ALTER idempotent côté schéma).
	if _, err := db.Exec(`ALTER TABLE commendation_definitions ADD COLUMN IF NOT EXISTS tier_targets VARCHAR`); err != nil {
		fmt.Printf("commendations: ensure tier_targets column: %v\n", err)
	}
	n, withTiers := 0, 0
	for _, c := range comms {
		if c.ID == "" {
			continue // pas de clé naturelle → ignoré
		}
		nameEN := strings.TrimSpace(c.Name)
		tierTargets := tierTargetsCSV(c.Levels)
		_, err := db.Exec(`INSERT OR REPLACE INTO commendation_definitions
			(commendation_id, name_en, name_fr, description_en, description_fr,
			 commendation_type, category, icon_url, tier_targets)
			VALUES (?,?,?,?,?,?,?,?,?)`,
			c.ID, nameEN, nameEN, strings.TrimSpace(c.Description), strings.TrimSpace(c.Description),
			c.Type, c.Category.Name, c.IconImageURL, tierTargets)
		if err != nil {
			fmt.Printf("commendations: insert %s: %v\n", c.ID, err)
			continue
		}
		n++
		if tierTargets != "" {
			withTiers++
		}
	}
	fmt.Printf("commendations: %d seeded (of %d, %d with tiers)\n", n, len(comms), withTiers)
}

// langEN is the language key used by the asset resolver.

const (
	langEN = "en-US"
)

// apiGameBaseVariant — élément de l'API Metadata officielle /game-base-variants.
// `id` = le GameBaseVariantId porté par chaque match Halo 5 (cf.
// internal/games/halo_5/mapping.go : GameVariant = assetRef("game_variant",
// r.GameBaseVariantId)). Le NOM du mode (EN) vit ici, pas dans la donnée de match.
// Shape minimale alignée sur apiPlaylist/apiMap (tous les types officiels exposent
// `name` + `id`).
type apiGameBaseVariant struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

// seedAssetTranslations populates the English asset catalog consumed by the resolver.

func seedAssetTranslations(db *sql.DB, key string) {
	n := 0
	n += seedPlaylistTranslations(db)
	n += seedMapTranslations(db)
	n += seedModeTranslations(db, key)
	fmt.Printf("asset_translations[en-US]: %d seedées\n", n)
}

func readAssetNames(db *sql.DB, assetType string) map[string]string {
	names := map[string]string{}
	rows, err := db.Query(
		`SELECT asset_id, name FROM asset_translations WHERE asset_type = ? AND lang = ?`,
		assetType, langEN)
	if err != nil {
		return names
	}
	defer rows.Close()
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err == nil {
			names[id] = name
		}
	}
	return names
}

// upsertAssetTranslation insère/réécrit une ligne asset_translations. Best-effort.
func upsertAssetTranslation(db *sql.DB, assetID, assetType, lang, name string) bool {
	if strings.TrimSpace(assetID) == "" || strings.TrimSpace(name) == "" {
		return false
	}
	if _, err := db.Exec(
		`INSERT OR REPLACE INTO asset_translations (asset_id, asset_type, lang, name)
		 VALUES (?,?,?,?)`,
		assetID, assetType, lang, strings.TrimSpace(name)); err != nil {
		fmt.Printf("asset_translations: insert %s/%s/%s: %v\n", assetType, assetID, lang, err)
		return false
	}
	return true
}

// seedPlaylistTranslations mirrors playlist names into the English asset catalog.

func seedPlaylistTranslations(db *sql.DB) int {
	rows, err := db.Query(`SELECT id, name FROM playlists WHERE TRIM(COALESCE(name,'')) != ''`)
	if err != nil {
		fmt.Printf("asset_translations[playlist]: read playlists %v\n", err)
		return 0
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			continue
		}
		if upsertAssetTranslation(db, id, "playlist", langEN, name) {
			n++
		}
	}
	return n
}

// seedMapTranslations mirrors canonical map names into the English asset catalog.

func seedMapTranslations(db *sql.DB) int {
	rows, err := db.Query(`SELECT map_asset_id, name_canonical FROM maps_catalog
		WHERE title_slug = ? AND TRIM(COALESCE(name_canonical,'')) != ''`, halo5.TitleSlug)
	if err != nil {
		fmt.Printf("asset_translations[map]: read maps_catalog %v\n", err)
		return 0
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var id, nameEN string
		if err := rows.Scan(&id, &nameEN); err != nil {
			continue
		}
		if upsertAssetTranslation(db, id, "map", langEN, nameEN) {
			n++
		}
	}
	return n
}

// seedModeTranslations fetche l'endpoint officiel /game-base-variants et écrit
// asset_type='game_variant' lang='en-US'. C'est la SOURCE de nom de mode pour Halo 5
// (le match ne porte que le GameBaseVariantId, jamais le nom). EN seul (l'API ne
// localise pas ; aucune section modes dans le TOML).
func seedModeTranslations(db *sql.DB, key string) int {
	body, err := fetchMeta(key, "game-base-variants")
	if err != nil {
		fmt.Printf("asset_translations[game_variant]: SKIP (%v)\n", err)
		return 0
	}
	var variants []apiGameBaseVariant
	if err := json.Unmarshal(body, &variants); err != nil {
		fmt.Printf("asset_translations[game_variant]: parse %v\n", err)
		return 0
	}
	n := 0
	for _, v := range variants {
		if upsertAssetTranslation(db, v.ID, "game_variant", langEN, v.Name) {
			n++
		}
	}
	return n
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
