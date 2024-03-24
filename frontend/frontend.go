package frontend

import (
	"embed"
	"html/template"
	"net/http"

	"github.com/Lama06/Herder-Inventar/modell"
)

var (
	//go:embed vorlagen/***
	vorlagenDateien embed.FS
	vorlage         = template.Must(template.New("vorlagen").Funcs(map[string]any{
		"inc": func(i int) int { return i + 1 },
		"dec": func(i int) int { return i - 1 },
	}).ParseFS(vorlagenDateien, "vorlagen/***"))
)

type kopfzeileVorlageDaten struct {
	Admin, Angemeldet bool
	Benutzername      string
}

func newKopfzeileVorlageDaten(benutzer *modell.Benutzer) kopfzeileVorlageDaten {
	if benutzer == nil {
		return kopfzeileVorlageDaten{}
	}
	return kopfzeileVorlageDaten{
		Admin:        benutzer.Admin,
		Angemeldet:   true,
		Benutzername: benutzer.Name,
	}
}

type contextKey int

const (
	ctxKeyBenutzer contextKey = iota
	ctxKeyObjekt
	ctxKeyProblem
	ctxKeyAccount
	ctxKeySeite
)

func New(db *modell.Datenbank) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /anmelden/{$}", handleAnmeldenGet())
	mux.Handle("POST /anmelden/{$}", handleAnmeldenPost(db))
	mux.Handle("GET /abmelden/{$}", handleAbmelden(db))

	mux.Handle("GET /accounts/{$}", handleAccounts(db))
	mux.Handle("POST /accounts/registrieren/{$}", handleAccountRegistrieren(db))
	mux.Handle("POST /accounts/{account}/passwort_aendern/{$}", handlePasswortÄndern(db))
	mux.Handle("GET /accounts/{account}/loeschen/{$}", handleAccountLöschen(db))

	mux.Handle("GET /objekte/{$}", handleInventarListe(db))
	mux.Handle("GET /objekte/{seite}/{$}", handleInventarListe(db))
	mux.Handle("POST /objekte/suche/{$}", handleObjekteSuchen(db))
	mux.Handle("POST /objekte/erstellen/{$}", handleObjektErstellen(db))

	mux.Handle("GET /objekt/{objekt}/{$}", handleObjekt(db))
	mux.Handle("GET /objekt/{objekt}/loeschen/{$}", handleObjektLöschen(db))
	mux.Handle("POST /objekt/{objekt}/bearbeiten/", handleObjektBearbeiten(db))
	mux.Handle("POST /objekt/{objekt}/probleme/melden/{$}", handleProblemMelden(db))
	mux.Handle("GET /objekt/{objekt}/probleme/{problem}/loesen/{$}", handleProblemLösen(db))

	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		db.Lock.Lock()
		defer db.Lock.Unlock()
		mux.ServeHTTP(res, req)
	})
}
