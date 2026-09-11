package filmdec

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// component_probe_test.go — LE BANC DE LA SONDE DE PRÉSENCE.
//
// Il répond à UNE question : parmi les 130 composants dont `traverse.go` connaît déjà la
// grammaire binaire mais dont il jette la valeur, lesquels sont réellement RÉPLIQUÉS dans
// nos films, et sur quel type d'entité. Un composant dispatché par son nom vient de la table
// d'archétypes du jeu — cela ne dit rien de sa présence dans une capture donnée.
//
// POURQUOI IL PARCOURT LES KEYFRAMES ET NON LES DELTAS. `ScanFilmBipedPositions` ne descend
// que dans les paquets DELTA, et n'y décode que les enregistrements de BIPED. Les composants
// visés (équipe ti=9, joueur ti=5, statborg ti=6, moteur de jeu ti=0) vivent sur d'autres
// types d'entité, que ce chemin ne touche jamais — d'où une première version de cette sonde
// qui n'a rien vu du tout. Les keyframes portent l'état COMPLET du monde.
//
// CE QU'IL NE PEUT PAS LIRE. Traverser un enregistrement non-biped exige le déserialiseur de
// son default-state (`defaultStateDeserByTI`). Les archétypes absents de cette table sont
// comptés à part, comme NON TRAVERSABLES : c'est une limite du décodeur, pas une absence du
// composant, et confondre les deux orienterait le chantier au mauvais endroit.
//
// LANCEMENT (les films ne sont pas dans le dépôt) :
//
//	REPLAY_FILMS=<...>/data/cache/film_chunks go test ./internal/analysis/filmdec/ \
//	  -run TestComponentPresenceProbe -v -timeout 60m

// probeShortlist : les composants dont la capture changerait quelque chose au rejeu.
var probeShortlist = []struct{ name, buys string }{
	{compManagedPlayerTeamDesignator, "equipe par joueur (Track.Team vaut -1 aujourd'hui)"},
	{"player-respawn-timer-component", "mort/reapparition par joueur, sans fenetre de 150 ms"},
	{"object-dead-state-component", "etat mort par objet"},
	{"player-lives-remaining-component", "vies restantes par joueur"},
	{"player-early-respawn-requested-component", "demande de reapparition anticipee"},
	{"game-engine-round-timer-component", "horloge de manche REELLE (aujourd'hui inferee)"},
	{"game-engine-current-state-component", "machine a etats du match (debut jouable)"},
	{"game-engine-current-round-component", "numero de manche"},
	{"game-engine-grace-period-time-left-component", "periode de grace avant coup d'envoi"},
	{"game-engine-sudden-death-time-left-component", "mort subite"},
	{"game-engine-game-finished-component", "fin de partie"},
	{"statborg-entry-index-and-type-component", "pont entite film <-> monde statborg"},
	{"statborg-current-round-value-stat-component", "stats d'objectif de la manche"},
	{"statborg-round-outcomes-component", "issues de manche"},
	{"player-representation-component", "lien joueur <-> representation"},
}

type compKey struct {
	name string
	ti   int
}

type probeTally struct {
	total         map[string]int
	byTI          map[compKey]int
	films         map[string]map[string]bool
	sample        map[string]string
	recs          map[int]int
	comps         map[int]int
	empty         map[int]int
	desync        map[int]int
	untraversable map[int]int
	blockers      map[string]int
}

func newProbeTally() *probeTally {
	return &probeTally{
		total: map[string]int{}, byTI: map[compKey]int{},
		films: map[string]map[string]bool{}, sample: map[string]string{},
		recs: map[int]int{}, desync: map[int]int{}, untraversable: map[int]int{},
		comps: map[int]int{}, empty: map[int]int{},
		blockers: map[string]int{},
	}
}

func TestComponentPresenceProbe(t *testing.T) {
	root := os.Getenv("REPLAY_FILMS")
	if root == "" {
		t.Skip("REPLAY_FILMS non défini : banc de sonde hors CI")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("REPLAY_FILMS illisible : %v", err)
	}
	tal := newProbeTally()
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if probeFilm(t, filepath.Join(root, e.Name()), e.Name(), tal) {
			t.Logf("film %s parcouru", e.Name())
		}
	}
	tal.report(t)
}

// probeFilm parcourt les keyframes d'un film et alimente le relevé.
func probeFilm(t *testing.T, dir, film string, tal *probeTally) bool {
	t.Helper()
	n := CountFilmChunks(dir)
	if n == 0 {
		return false
	}
	raw, err := os.ReadFile(filepath.Join(dir, "chunk_00.bin"))
	if err != nil {
		t.Logf("%s : chunk_00 (registre) illisible (%v)", film, err)
		return false
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Logf("%s : registre illisible (%v)", film, err)
		return false
	}
	compProbeHook = func(ti uint32, name string, payload any, _ bool) {
		tal.total[name]++
		tal.byTI[compKey{name, int(ti)}]++
		if tal.films[name] == nil {
			tal.films[name] = map[string]bool{}
		}
		tal.films[name][film] = true
		if payload != nil && tal.sample[name] == "" {
			tal.sample[name] = fmt.Sprintf("%+v", payload)
		}
	}
	defer func() { compProbeHook = nil }()
	for c := 1; c <= n; c++ {
		chunk, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range WalkPackets(chunk) {
			if p.Type != PacketTypeKeyframe {
				continue
			}
			pay := p.Payload(chunk)
			for _, r := range WalkKeyframeWorld(pay) {
				probeKeyframeRecord(pay, r, reg, tal)
			}
		}
	}
	return true
}

// probeKeyframeRecord traverse UN enregistrement de la table keyframe. Même grammaire que
// TraverseKeyframeBipedAt, étendue aux archétypes non-biped par leur deser de default-state.
func probeKeyframeRecord(pay []byte, r KeyframeRec, reg *Registry, tal *probeTally) {
	ti := uint32(r.TI)
	br := NewBitReader(pay)
	br.SetBitPos(r.Bit + keyframeHeaderBits(ti))
	switch {
	case ti == bipedDefaultStateTypeIndex:
		consumeBipedDefaultState(br)
		consumeBipedDefaultStateTail(br)
	case defaultStateBitsByTI[ti] != 0:
		br.Skip(defaultStateBitsByTI[ti])
	default:
		// Absent de la table = default-state de ZERO bit, et non « illisible » : le stub
		// FUN_1408d8220 a ete mesure statiquement pour ti0, 1, 2, 4, 7, 15, 16, 18, 19, 22,
		// 25, 26, 27, 30..34, 45 et 46 (cf. default_state_arch.go). La premiere version de
		// cette sonde comptait ces types comme NON TRAVERSABLES et sautait leurs records —
		// dont les 35 de ti=0, celui qui porte l horloge de match. C etait un defaut de la
		// sonde, pas du decodeur : TraverseEntity, lui, retombe deja sur un skip de 0 bit.
		if fn, ok := defaultStateDeserByTI[ti]; ok {
			fn(br)
		}
	}
	br.ReadBit() // porte has-components
	arch, ok := reg.Archetype(r.TI)
	if !ok {
		tal.untraversable[r.TI]++
		return
	}
	tr := decodeDeltaWithArch(br, arch, ti)
	tal.recs[r.TI]++
	tal.comps[r.TI] += len(tr.Comps)
	if len(tr.Comps) == 0 {
		tal.empty[r.TI]++
	}
	if tr.DesyncAt >= 0 {
		tal.desync[r.TI]++
		// QUEL composant a arrete la traversee. Un composant non porte fait sortir la
		// boucle : c est donc lui, et non le type d entite, qui borne ce qu on sait lire.
		// Le compter par nom classe le travail de portage par ce qu il debloque vraiment.
		if tr.DesyncAt < len(arch.Components) {
			tal.blockers[arch.Components[tr.DesyncAt]]++
		}
	}
}

func (tal *probeTally) report(t *testing.T) {
	t.Helper()
	t.Logf("")
	t.Logf("=== ENREGISTREMENTS KEYFRAME, par type d'entite ===")
	t.Logf("%6s %10s %10s %12s %10s %10s", "ti", "traverses", "desync", "nonTravers", "comps", "recVides")
	seen := map[int]bool{}
	for k := range tal.recs {
		seen[k] = true
	}
	for k := range tal.untraversable {
		seen[k] = true
	}
	order := make([]int, 0, len(seen))
	for k := range seen {
		order = append(order, k)
	}
	sort.Ints(order)
	for _, ti := range order {
		t.Logf("%6d %10d %10d %12d %10d %10d", ti, tal.recs[ti], tal.desync[ti], tal.untraversable[ti], tal.comps[ti], tal.empty[ti])
	}

	t.Logf("")
	t.Logf("=== COMPOSANTS QUI ARRETENT LA TRAVERSEE (non portes) ===")
	{
		type bk struct {
			n string
			c int
		}
		var bs []bk
		for n, c := range tal.blockers {
			bs = append(bs, bk{n, c})
		}
		sort.Slice(bs, func(a, b int) bool { return bs[a].c > bs[b].c })
		for i, e := range bs {
			if i >= 15 {
				break
			}
			t.Logf("%9d  %s", e.c, e.n)
		}
	}
	t.Logf("")
	t.Logf("=== LISTE COURTE : ce que la capture debloquerait ===")
	t.Logf("%-46s %9s %6s  %s", "composant", "occur.", "films", "ce qu'il debloque")
	for _, s := range probeShortlist {
		mark := "ABSENT"
		if tal.total[s.name] > 0 {
			mark = fmt.Sprintf("%d", tal.total[s.name])
		}
		t.Logf("%-46s %9s %6d  %s", trunc(s.name, 46), mark, len(tal.films[s.name]), s.buys)
	}

	t.Logf("")
	t.Logf("=== PORTEURS ET EXEMPLES (liste courte, presents) ===")
	for _, s := range probeShortlist {
		if tal.total[s.name] == 0 {
			continue
		}
		var tis []string
		for k, v := range tal.byTI {
			if k.name == s.name {
				tis = append(tis, fmt.Sprintf("ti=%d:%d", k.ti, v))
			}
		}
		sort.Strings(tis)
		line := fmt.Sprintf("%-46s %v", trunc(s.name, 46), tis)
		if tal.sample[s.name] != "" {
			line += "  ex=" + tal.sample[s.name]
		}
		t.Logf("%s", line)
	}

	t.Logf("")
	t.Logf("=== TOUS COMPOSANTS OBSERVES, par frequence ===")
	names := make([]string, 0, len(tal.total))
	for n := range tal.total {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool { return tal.total[names[i]] > tal.total[names[j]] })
	for i, n := range names {
		if i >= 40 {
			t.Logf("... et %d autres", len(names)-40)
			break
		}
		t.Logf("%9d  %s", tal.total[n], n)
	}
	if len(tal.total) == 0 {
		t.Fatal("sonde muette : aucun composant observé, le branchement est à revoir")
	}
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "~"
}

// keyframeHeaderBits rend la taille de l'en-tete d'un enregistrement keyframe, EN BITS,
// pour un type d'entite.
//
// ELLE N'EST PAS CONSTANTE, ET C'EST LA DECOUVERTE. Le decodeur supposait 64 bits partout,
// valeur tiree d'un utilitaire ecrit pour le BIPED (probe_export.go) et sans aucun appui
// dans le binaire — verifie au desassemblage. Deux mesures independantes la contredisent :
//
//	ti=9  47 bits. Le prefixe total mesure vaut 61 (SEUL, sur 300 essayes, a rendre huit
//	      entites joueur d'equipe stable partagees 4-4 dans les SIX films) et son
//	      default-state en consomme 14 (V+6+6+1). 61-14 = 47.
//	ti=35 64 bits. A 47, le biped passe de 19 736 composants decodes a ZERO.
//
// Les deux ne peuvent pas etre vrais avec un en-tete unique : il est PROPRE AU TYPE. Les
// types encore absents de cette table restent a calibrer un par un (cf. prefix_sweep_test.go),
// et leur valeur par defaut de 64 n'est qu'un heritage, pas une mesure.
func keyframeHeaderBits(ti uint32) int {
	if bits, ok := keyframeHeaderByTI[ti]; ok {
		return bits
	}
	return 64 // heritage biped : correct pour ti=35, non verifie ailleurs
}

// keyframeHeaderByTI : en-tetes MESURES, avec la preuve qui les fonde en commentaire.
var keyframeHeaderByTI = map[uint32]int{
	9: 47, // partage 4-4 des equipes, six films, unique sur 300 prefixes essayes
}
