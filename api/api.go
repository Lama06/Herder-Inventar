package api

import (
	"net/http"

	"github.com/Lama06/Herder-Inventar/modell"
)

func New(db *modell.Datenbank) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("POST /auth/anmelden/", handleAnmelden(db))
	mux.Handle("POST /auth/abmelden/", handleAbmelden(db))
	mux.Handle("POST /auth/passwort_aendern/", handlePasswortÄndern(db))

	mux.Handle("POST /accounts/registrieren/", handleRegistrieren(db))
	mux.Handle("POST /accounts/loeschen/", handleAccountLöschen(db))
	mux.Handle("POST /accounts/liste/", handleAccountsAuflisten(db))

	mux.Handle("POST /objekte/lesen/", handleObjekteLesen(db))
	mux.Handle("POST /objekte/auflisten/", handleObjekteAuflisten(db))
	mux.Handle("POST /objekte/erstellen", handleObjektErstellen(db))
	mux.Handle("POST /objekte/loeschen/", handleObjektLöschen(db))
	mux.Handle("POST /objekte/aendern/", handleObjektÄndern(db))

	mux.Handle("POST /probleme/melden/", handleProblemMelden(db))
	mux.Handle("POST /probleme/loesen/", handleProblemLösen(db))

	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		db.Lock.Lock()
		defer db.Lock.Unlock()
		mux.ServeHTTP(res, req)
	})
}
