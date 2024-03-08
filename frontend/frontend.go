package frontend

import (
	"net/http"

	"github.com/Lama06/Herder-Inventar/modell"
)

type contextKey int

const (
	ctxKeyBenutzer contextKey = iota
	ctxKeyObjekt
	ctxKeyProblem
)

func New(db *modell.Datenbank) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /anmelden/{$}", handleAnmeldenGet())
	mux.Handle("POST /anmelden/{$}", handleAnmeldenPost(db))

	mux.Handle("GET /objekte/{$}", handleInventarListe(db))
	mux.Handle("POST /objekte/erstellen/{$}", handleObjektErstellen(db))
	mux.Handle("GET /objekte/{objekt}/{$}", handleObjekt(db))
	mux.Handle("GET /objekte/{objekt}/loeschen/{$}", handleObjektLöschen(db))
	mux.Handle("POST /objekte/{objekt}/probleme/melden/{$}", handleProblemMelden(db))
	mux.Handle("GET /objekte/{objekt}/probleme/{problem}/loesen/", handleProblemLösen(db))

	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		db.Lock.Lock()
		defer db.Lock.Unlock()
		mux.ServeHTTP(res, req)
	})
}
