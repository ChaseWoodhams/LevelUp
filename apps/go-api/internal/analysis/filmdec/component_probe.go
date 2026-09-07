package filmdec

// component_probe.go — SONDE DE PRÉSENCE DES COMPOSANTS.
//
// CE QU'ELLE RÉPOND. `traverseComponentLoop` dispatche 130 composants nommés dont il connaît
// la grammaire binaire, mais n'en CAPTURE que quatre (cf. capture.go) : pour tous les autres
// il avance le curseur et jette la valeur. Décider lequel vaut d'être capturé demande de
// savoir lesquels sont réellement RÉPLIQUÉS dans les films qu'on possède — un composant
// dispatché par son nom vient de la table d'archétypes du jeu, ce qui ne dit rien de sa
// présence dans une capture donnée.
//
// POURQUOI ELLE SE BRANCHE SUR posCaptureHook. Le décodage de trame procède par ESSAIS : il
// tente un enregistrement à chaque bit, et ne re-décode « pour de vrai » que ceux qu'il a
// confirmés (cf. frame_records.go, « re-decode with hooks live -> real samples »). Les
// milliers d'essais infructueux traversent le même code et compteraient comme des présences.
// Les neuf sites qui suspendent les crochets pendant un essai mettent tous `posCaptureHook`
// à nil ; s'y adosser donne exactement la bonne suppression SANS toucher à ces neuf sites,
// et sans risquer d'en oublier un.
//
// HORS PRODUCTION. Le crochet est nil par défaut et n'est posé que par la sonde.

// compProbeHook reçoit chaque composant consommé d'un enregistrement ACCEPTÉ : le type
// d'entité porteur, le nom du composant, et sa valeur quand il en porte une (les quatre de
// captureNames ; nil partout ailleurs — la sonde ne mesure alors que la présence).
// `live` distingue un enregistrement ACCEPTÉ (crochet de position posé) d'un ESSAI
// spéculatif. La sonde reçoit les deux et fait le tri elle-même : compter uniquement les
// vivants rendrait indiscernables « composant absent du film » et « traversée jamais
// atteinte », qui appellent des conclusions opposées.
var compProbeHook func(typeIndex uint32, name string, payload any, live bool)

// observeComponent notifie la sonde, si elle est posée.
func observeComponent(typeIndex uint32, name string, payload any) {
	if compProbeHook != nil {
		compProbeHook(typeIndex, name, payload, posCaptureHook != nil)
	}
}
