// cmd/mapquant-build — produit le catalogue des BORNES DE QUANTIFICATION PAR CARTE à
// partir des .module du jeu installé, et l'écrit dans
// data/titles/{slug}/reference/map_quant_bounds.json (PathResolver).
//
// Le film ne porte que des indices de quantum ; les bornes (AABB du BSP principal,
// `world bounds x/y/z` du tag sbsp) ne vivent que dans le module de la carte. Sans elles,
// aucune coordonnée monde n'est produite (refus explicite côté décodeur).
//
// Le lien nom de carte affiché -> dossier de module est déclaré ici, EXPLICITEMENT, et
// n'est retenu que lorsqu'il est établi hors de toute mesure (identité du nom, ou preuve
// externe citée). Les cartes dont le module n'est pas établi sont ABSENTES du catalogue :
// l'API préfère refuser que produire une coordonnée fausse.
//
// Usage : CGO_ENABLED=1 go run ./cmd/mapquant-build [--levels DIR] [--title slug] [--out FILE]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/himap"
)

// defaultLevelsDir : arborescence des cartes multijoueur d'une installation Steam.
const defaultLevelsDir = `D:/SteamLibrary/steamapps/common/Halo Infinite/deploy/ds/levels/multi`

// mapModule associe le nom de carte affiché (celui de match_registry.map_name) au dossier
// du module. Chaque entrée porte la RAISON pour laquelle le lien est tenu pour établi.
//
// Toutes les entrées ci-dessous sauf Cliffhanger sont l'identité du nom (le module porte
// le nom de la carte, éventuellement préfixé par sa famille : ctf_, sgh_, va_, btb_).
// Cliffhanger : le module `ridgeline` est le SEUL dont les bornes reproduisent la capture
// live de DAT_14462cbe0[0] sur le film 000d5950 (x[-41.103,72.109] y[-56.607,57.212]
// z[-84.371,53.180], accord à 2,5e-4 unité près) — preuve externe, pas un appariement de
// largeurs.
//
// Vagabond : le module `fo08_wetland` est établi par le `level_id` du .mvar, pas par le
// nom. `vagabond_fo08_wetland.mvar` porte level_id 88891201 (0x054C5F41) ; balayé sur les
// 88 modules de `deploy/any` + `deploy/ds`, il rend EXACTEMENT UNE occurrence,
// `multi/fo08_wetland/fo08_wetland-rtx-new.module` à +0x28, groupe `levl`. Témoin de la
// méthode : le level_id de Catalyst (−1044063363) rend de même une seule occurrence,
// `multi/catalyst`. Preuve externe à toute mesure de largeur (plan maître §J0.2, 2026-07-31).
// Vagabond est une carte Forge : `fo08_wetland` est sa TOILE, et c'est bien la toile qui
// porte les bornes de déquantification.
//
// Recharge, Prism (2026-09-11) : established by the `level_id` of their .mvar
// (map_objectives.json: −687782121, 2068765158), not by name or width. Scanned as raw int32 LE
// over all 132 installed modules (`deploy/{any,ds,pc}`), a 4-byte pattern also hits unrelated
// modules at random (×1 each), so the criterion is the one the three KNOWN links show:
// Streets -> sgh_streets (ds ×6, pc ×5, any ×2), Aquarius -> ctf_aquarius (×8, ×9, ×2),
// Vagabond -> fo08_wetland (×4, ×3, ×2) — the right module is the only level module holding the
// id in all three roots, repeatedly. Both new ids show that signature for exactly one module:
// sgh_blueprint (×7, ×8, ×2), sgh_crystalcaves (×6, ×7, ×2); every other level module holds
// them at most once.
//
// NOT CATALOGUED: Live Fire. Its link is established the same way (level_id 1253388187 ->
// sgh_interlock, ×6, ×7, ×2), but `ds/.../sgh_interlock` is a 216 KB stub with NO sbsp tag, so
// there are no bounds to read. The pc module is NOT a substitute: tried with a witness on every
// map readable from both, and Vagabond's pc main BSP is a different, larger box
// (x −1929..… against −231..231). A pc read would hand Live Fire the wrong box.
//
// Lattice, Origin, Vacancy, Solitude, Argyle, Empyrean (2026-09-13): Forge maps, so the module is
// the CANVAS their variant is built on, as for Vagabond. Their .mvar files (fetched from the public
// UGC blob store by asset/version id, read from the matches' stats) carry level_ids −992358985,
// 88891201, 1437677928, and 426470249 for the last three. The same all-three-roots scan, with
// Origin/Vagabond (88891201 -> fo08_wetland ×2 any, ×4 ds, ×3 pc) and Recharge as controls, gives
// exactly one module each: fo13_frost (×2, ×2, ×3), fo09_academy (×2, ×2, ×6), fo11_blank
// (×2, ×6, ×7); every other level module holds these ids at most in one root. Origin shares
// Vagabond's canvas, so it shares Vagabond's box.
// forgeCanvases are the Forge canvas modules. Each holds the same two sbsp tags, a 463 m box at
// 15/15/17 bits and a 3.9 km box at 18/18/18 bits, and which one is the larger tag differs by
// canvas. Films are quantised on the 463 m box on all of them (2026-09-13,
// replay/map_bounds_plausibility_measure_test.go): with it, Lattice (fo13_frost) and Solitude,
// Argyle, Empyrean (fo11_blank) decode their players into 28-49 m arenas; with the 3.9 km box the
// same films spread 8.35x wider, exactly the ratio of the two extents. Origin (fo08_wetland) and
// Vacancy (fo09_academy), whose larger tag already is the 463 m box, decode into 39x26 m and
// 31x35 m arenas.
var forgeCanvases = map[string]bool{
	"fo08_wetland": true, "fo09_academy": true, "fo11_blank": true, "fo13_frost": true,
}

// mainBSP picks the sbsp whose bounds quantise a map's films: the largest tag, except on a Forge
// canvas, where it is the tag with the smallest XY footprint (see forgeCanvases).
func mainBSP(module string, bsps []himap.BSP) himap.BSP {
	if !forgeCanvases[module] {
		return bsps[0]
	}
	best := bsps[0]
	for _, b := range bsps[1:] {
		if b.Bounds.Extent(0)*b.Bounds.Extent(1) < best.Bounds.Extent(0)*best.Bounds.Extent(1) {
			best = b
		}
	}
	return best
}

var mapModule = map[string]string{
	"Aquarius":      "ctf_aquarius",
	"Argyle":        "fo11_blank",
	"Bazaar":        "ctf_bazaar",
	"Behemoth":      "va_behemoth",
	"Breaker":       "ctf_breaker",
	"Catalyst":      "catalyst",
	"Chasm":         "chasm",
	"Cliffhanger":   "ridgeline",
	"Empyrean":      "fo11_blank",
	"Forbidden":     "ctf_forbidden",
	"Forest":        "forest",
	"Fragmentation": "btb_fragmentation",
	"Highpower":     "btb_highpower",
	"Illusion":      "ctf_illusion",
	"Lattice":       "fo13_frost",
	"Launch Site":   "va_launchsite",
	"Origin":        "fo08_wetland",
	"Prism":         "sgh_crystalcaves",
	"Recharge":      "sgh_blueprint",
	"Solitude":      "fo11_blank",
	"Streets":       "sgh_streets",
	"Vacancy":       "fo09_academy",
	"Vagabond":      "fo08_wetland",
}

func main() {
	levels := flag.String("levels", defaultLevelsDir, "racine des dossiers de cartes (.module)")
	titleSlug := flag.String("title", title.DefaultSlug, "slug du titre")
	out := flag.String("out", "", "fichier de sortie (défaut : PathResolver.MapQuantBoundsPath)")
	flag.Parse()

	outPath := *out
	if outPath == "" {
		root, err := title.FindRepoRoot()
		if err != nil {
			slog.Error("racine repo", "err", err)
			os.Exit(1)
		}
		outPath = title.NewPathResolver(root).MapQuantBoundsPath(*titleSlug)
	}

	cat := filmdec.MapQuantCatalog{
		SchemaVersion: filmdec.MapQuantSchemaVersion,
		Source:        "world bounds x/y/z du tag sbsp principal, lus dans " + *levels,
		Maps:          map[string]filmdec.MapQuantEntry{},
	}
	names := make([]string, 0, len(mapModule))
	for n := range mapModule {
		names = append(names, n)
	}
	sort.Strings(names)
	missing := 0
	for _, name := range names {
		mod := mapModule[name]
		mods, _ := filepath.Glob(filepath.Join(*levels, mod, "*.module"))
		if len(mods) == 0 {
			slog.Error("module absent", "carte", name, "module", mod)
			missing++
			continue
		}
		bsps, err := himap.ReadModuleBSPBounds(mods[0])
		if err != nil {
			slog.Error("lecture des bornes", "err", err, "carte", name, "module", mod)
			missing++
			continue
		}
		main := mainBSP(mod, bsps)
		if !main.Bounds.Valid() {
			slog.Error("AABB dégénérée", "carte", name, "module", mod)
			missing++
			continue
		}
		e := filmdec.MapQuantEntry{Module: mod}
		w := main.Bounds.AxisWidths()
		for ax := 0; ax < 3; ax++ {
			e.Min[ax] = float32(main.Bounds.Min[ax])
			e.Max[ax] = float32(main.Bounds.Max[ax])
			e.AxisWidths[ax] = uint(w[ax])
		}
		cat.Maps[filmdec.NormalizeMapName(name)] = e
		slog.Info("bornes lues", "carte", name, "module", mod,
			"W", fmt.Sprintf("%d/%d/%d", w[0], w[1], w[2]),
			"extent", fmt.Sprintf("%.3f/%.3f/%.3f", main.Bounds.Extent(0), main.Bounds.Extent(1), main.Bounds.Extent(2)))
	}
	if missing > 0 {
		slog.Error("catalogue incomplet — rien écrit", "manquantes", missing)
		os.Exit(1)
	}
	blob, err := json.MarshalIndent(cat, "", "  ")
	if err != nil {
		slog.Error("sérialisation", "err", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		slog.Error("création du répertoire", "err", err, "path", outPath)
		os.Exit(1)
	}
	if err := os.WriteFile(outPath, append(blob, '\n'), 0o644); err != nil {
		slog.Error("écriture", "err", err, "path", outPath)
		os.Exit(1)
	}
	slog.Info("catalogue écrit", "path", outPath, "cartes", len(cat.Maps))
}
