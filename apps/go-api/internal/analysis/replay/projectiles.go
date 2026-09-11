package replay

import (
	"math"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

// projectiles.go — TRAJECTOIRES DE PROJECTILE projetées sur la grille du rejeu.
//
// SOURCE : filmdec.ScanFilmProjectiles — l'archétype ti=41, que le registre du film NOMME
// lui-même (`projectile-at-rest-state`, `projectile-tether-state`, `projectile-command_tick`).
// Position répliquée à ~60 Hz du départ à l'immobilisation.
//
// TÉMOIN : 65 des 70 lancers de grenade connus voient naître une trajectoire dans les 200 ms,
// contre 11 à 13 pour les mêmes lancers décalés en bloc.
//
// CE QUE CE CALQUE NE DIT PAS : l'IMPACT. Il n'existe aucun événement de détonation dans le
// film. Le dernier point est la DERNIÈRE POSITION RÉPLIQUÉE — pour une grenade à fragmentation
// la réplication cesse ~1,4 s après le lancer alors que la mèche court jusqu'à ~3 s. Le dernier
// point approche l'explosion parce que l'objet ne bouge plus, pas parce qu'on la lit. Le client
// doit écrire « dernière position connue », jamais « impact ».

// Projectile est une trajectoire de projectile, échantillonnée sur la grille du rejeu.
type Projectile struct {
	// T0 est l'index de frame du premier point, sur le même axe que Point.T.
	T0 int `json:"t0"`
	// P est la suite des points [dt, x, y] où dt est le décalage en frames depuis T0.
	// Format compact : une trajectoire porte des dizaines de points sur une seconde, et
	// répéter l'index absolu à chaque point doublerait le poids du document pour rien.
	P [][3]float32 `json:"p"`
	// Rest signale que le dernier point porte `projectile-at-rest-state` — le seul champ qui
	// CERTIFIE une fin de vol (78 fois sur 79 sur le film de référence). Sans lui, le vol
	// s'arrête parce que la réplication s'arrête, ce qui n'est pas la même chose.
	Rest bool `json:"rest,omitempty"`
}

// projectileMaxStepM borne le déplacement d'un projectile entre deux points de la grille
// (100 ms). Au-delà, ce n'est plus une lecture : c'est un artefact de déquantification.
//
// LE DÉFAUT MESURÉ, ET POURQUOI CE GARDE-FOU EXISTE. Sur les quatre films Streets archivés,
// 27 à 35 % des trajectoires portent au moins un pas impossible, et la signature est nette :
// le saut vaut EXACTEMENT l'étendue Y de la carte (52,88 m pour `sgh_streets`, dont les bornes
// de quantification sont Y ∈ [-23,018 ; 29,867]), l'autre axe ne bougeant pas d'un centimètre —
// par exemple (14,37 ; -23,00) -> (14,97 ; 29,84) en un seul pas de 100 ms, soit 528 m/s.
// Jamais sur X, toujours sur Y : le quantum Y repasse d'un bord à l'autre.
//
// FIXED UPSTREAM (2026-09-11): filmdec.unwrapLife now follows the wrap, so the decoded flight
// continues past the box edge. This guard stays as a backstop against a mis-decoded record,
// which the unwrap cannot tell apart from a real position.
//
// 10 m par pas, soit 100 m/s, laisse passer tout projectile du jeu (une grenade tient sous
// 20 m/s, une roquette sous 30) et ne coupe que l'impossible.
const projectileMaxStepM = 10

// buildProjectiles projette les trajectoires décodées sur la grille de frames du rejeu.
//
// DÉCIMATION : le film réplique à ~60 Hz, la grille du rejeu est à 10 Hz. On garde UN point
// par frame — le premier — plutôt que de moyenner : un projectile suit une parabole, et
// moyenner deux positions distantes de 100 ms couperait le sommet de l'arc.
//
// LE VOL S'ARRÊTE AU PREMIER PAS IMPOSSIBLE, il n'est pas recousu. The quantum wrap is already
// undone by filmdec, so an impossible step here is a mis-decoded record, and nothing says where
// the projectile really went. Same rule as the end of a flight: publish what is read, stop where
// the film stops being readable.
func buildProjectiles(tracks []filmdec.ProjectileTrack, origin, step uint64) ([]Projectile, int) {
	if len(tracks) == 0 {
		return nil, 0
	}
	truncated := 0
	out := make([]Projectile, 0, len(tracks))
	for _, tr := range tracks {
		if len(tr.Pts) < 3 || tr.Pts[0].TimestampUS < origin {
			continue
		}
		t0 := int((tr.Pts[0].TimestampUS - origin) / step)
		var pts [][3]float32
		last := -1
		cut := false
		for _, p := range tr.Pts {
			if p.TimestampUS < origin {
				continue
			}
			f := int((p.TimestampUS - origin) / step)
			if f == last {
				continue // un seul point par frame de la grille
			}
			if n := len(pts); n > 0 {
				dx, dy := float64(round2(p.X)-pts[n-1][1]), float64(round2(p.Y)-pts[n-1][2])
				if math.Hypot(dx, dy) > projectileMaxStepM {
					cut = true
					break
				}
			}
			last = f
			pts = append(pts, [3]float32{float32(f - t0), round2(p.X), round2(p.Y)})
		}
		if cut {
			truncated++
		}
		if len(pts) < 2 { // une trajectoire d'un seul point de grille ne se dessine pas
			continue
		}
		// `Rest` CERTIFIE une fin de vol : un vol coupé n'a pas la sienne, et le dire
		// serait affirmer qu'on a vu le projectile s'immobiliser là.
		out = append(out, Projectile{T0: t0, P: pts, Rest: !cut && tr.Pts[len(tr.Pts)-1].AtRest})
	}
	// Tri TOTAL : T0 est un index de frame de la grille 10 Hz, donc les ex æquo sont la règle,
	// pas l'exception. Départager par la première position publiée puis par la longueur rend
	// l'ordre indépendant de celui des `tracks` reçues — deux projectiles que ces trois clés ne
	// séparent pas produisent les mêmes octets, quel que soit leur rang.
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		switch {
		case a.T0 != b.T0:
			return a.T0 < b.T0
		case a.P[0][1] != b.P[0][1]:
			return a.P[0][1] < b.P[0][1]
		case a.P[0][2] != b.P[0][2]:
			return a.P[0][2] < b.P[0][2]
		default:
			return len(a.P) < len(b.P)
		}
	})
	return out, truncated
}
