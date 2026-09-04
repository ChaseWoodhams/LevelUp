// Package sync — halo_client_isfilmgone_test.go : garde-fou IsFilmGoneErr (verdict
// DÉFINITIF vs transitoire sur un film, consommé par cmd/study-archiver pour graver
// l'état terminal `expired`).
//
// Ce prédicat décide le seul état qu'aucun run ultérieur ne retente : un faux positif
// enterre un film vivant pour toujours, un faux négatif fait retaper un lien mort toutes
// les heures. Les deux moitiés sont couvertes — le manifeste (statut TYPÉ via *HTTPError)
// et les blobs (message formaté par downloadBlob, cf. le commentaire du prédicat).
package haloclient

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestIsFilmGoneErr(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		// Manifeste : doGet rend un *HTTPError, le statut est une donnée.
		{"manifeste 404", &HTTPError{StatusCode: 404, URL: "u"}, true},
		{"manifeste 410", &HTTPError{StatusCode: 410, URL: "u"}, true},
		{"manifeste 404 emballé", fmt.Errorf("GetFilmChunks: %w", &HTTPError{StatusCode: 404, URL: "u"}), true},
		// Transitoires : le film peut très bien être là au run suivant.
		{"503", &HTTPError{StatusCode: 503, URL: "u"}, false},
		{"429", &HTTPError{StatusCode: 429, URL: "u"}, false},
		// 401/403 : refus d'auth, pas une disparition. Un token remis d'aplomb rend le film.
		{"403", &HTTPError{StatusCode: 403, URL: "u"}, false},
		{"timeout", context.DeadlineExceeded, false},
		{"coupure réseau", errors.New("dial tcp: connection refused"), false},
		// Blobs : downloadBlob formate « downloadBlob HTTP %d », d'où le repli textuel.
		{"blob 404", errors.New("downloadBlob HTTP 404"), true},
		{"blob 410", errors.New("downloadBlob HTTP 410"), true},
		{"blob 503", errors.New("downloadBlob HTTP 503"), false},
		{"blob 404 emballé", fmt.Errorf("GetFilmChunks chunk 3(m): %w",
			errors.New("downloadBlob HTTP 404")), true},
	}
	for _, c := range cases {
		if got := IsFilmGoneErr(c.err); got != c.want {
			t.Errorf("%s: IsFilmGoneErr = %v, want %v", c.name, got, c.want)
		}
	}
}
