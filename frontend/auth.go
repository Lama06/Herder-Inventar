package frontend

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/Lama06/Herder-Inventar/auth"
	"github.com/Lama06/Herder-Inventar/modell"
)

var (
	//go:embed vorlagen/anmelden.gohtml
	anmeldenVorlageRoh string
	anmeldenVorlage    = template.Must(template.New("anmelden").Parse(anmeldenVorlageRoh))
)

type anmeldenVorlageDaten struct {
	Fehler        bool
	Weiterleitung string
}

func handleAnmeldenGet() http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		err := req.ParseForm()
		if err != nil {
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		var antwort bytes.Buffer
		err = anmeldenVorlage.Execute(&antwort, anmeldenVorlageDaten{
			Fehler:        false,
			Weiterleitung: req.Form.Get("weiterleitung"),
		})
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			log.Println(err)
			return
		}
		_, _ = antwort.WriteTo(res)
	})
}

func handleAnmeldenPost(db *modell.Datenbank) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		err := req.ParseForm()
		if err != nil {
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		benutzername, passwort := req.Form.Get("benutzername"), req.Form.Get("passwort")
		passwortHash := sha256.Sum256([]byte(passwort))
		benutzer, ok := db.Accounts[benutzername]
		if !ok || passwortHash != benutzer.Passwort {
			res.WriteHeader(http.StatusUnauthorized)
			err := anmeldenVorlage.Execute(res, anmeldenVorlageDaten{Fehler: true})
			if err != nil {
				log.Println(err)
				return
			}
			return
		}

		schlüssel, err := auth.GenerateSchlüssel()
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		db.Sitzungen[schlüssel] = &modell.Sitzung{
			Schlüssel:     schlüssel,
			Benutzer:      benutzername,
			LetzerZugriff: time.Now(),
		}

		http.SetCookie(res, &http.Cookie{
			Name:     "schluessel",
			Value:    schlüssel,
			Path:     "/",
			MaxAge:   int(auth.SitzungDauer.Seconds()),
			Secure:   true,
			HttpOnly: false,
			SameSite: http.SameSiteStrictMode,
		})
		if req.Form.Has("weiterleitung") {
			http.Redirect(res, req, req.Form.Get("weiterleitung"), http.StatusFound)
			return
		}
		http.Redirect(res, req, "/objekte/", http.StatusFound)
	})
}

func requireLogin(
	db *modell.Datenbank, admin bool,
	danach http.Handler,
) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		anmeldenSeite := "/anmelden/?weiterleitung=" + req.URL.String()

		schlüsselKeks, err := req.Cookie("schluessel")
		if err != nil {
			http.Redirect(res, req, anmeldenSeite, http.StatusFound)
			return
		}
		sitzung, ok := db.Sitzungen[schlüsselKeks.Value]
		if !ok {
			http.Redirect(res, req, anmeldenSeite, http.StatusFound)
			return
		}
		benutzer, ok := db.Accounts[sitzung.Benutzer]
		if !ok {
			http.Redirect(res, req, anmeldenSeite, http.StatusFound)
			return
		}
		if admin && !benutzer.Admin {
			res.WriteHeader(http.StatusUnauthorized)
			return
		}
		req = req.WithContext(context.WithValue(req.Context(), ctxKeyBenutzer, benutzer))
		danach.ServeHTTP(res, req)
	})
}

func requireLoginSoft(db *modell.Datenbank, danach http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		schlüsselKeks, err := req.Cookie("schluessel")
		if err != nil {
			danach.ServeHTTP(res, req)
			return
		}
		sitzung, ok := db.Sitzungen[schlüsselKeks.Value]
		if !ok {
			danach.ServeHTTP(res, req)
			return
		}
		benutzer, ok := db.Accounts[sitzung.Benutzer]
		if !ok {
			danach.ServeHTTP(res, req)
			return
		}
		req = req.WithContext(context.WithValue(req.Context(), ctxKeyBenutzer, benutzer))
		danach.ServeHTTP(res, req)
	})
}

func requireAccount(db *modell.Datenbank, pfadKomponente string, danach http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		benutzername := req.PathValue(pfadKomponente)
		benutzer, ok := db.Accounts[benutzername]
		if !ok {
			res.WriteHeader(http.StatusNotFound)
			return
		}
		req = req.WithContext(context.WithValue(req.Context(), ctxKeyAccount, benutzer))
		danach.ServeHTTP(res, req)
	})
}

var (
	//go:embed vorlagen/accounts.gohtml
	accountsVorlageRoh string
	accountsVorlage    = template.Must(template.New("accounts").Parse(accountsVorlageRoh))
)

func handleAccounts(db *modell.Datenbank) http.Handler {
	return requireLogin(db, true, http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		var antwort bytes.Buffer
		err := accountsVorlage.Execute(&antwort, db)
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = antwort.WriteTo(res)
	}))
}

func handleAccountRegistrieren(db *modell.Datenbank) http.Handler {
	return requireLogin(db, true, http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		err := req.ParseForm()
		if err != nil || !req.Form.Has("benutzername") || !req.Form.Has("passwort") {
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		benutzername := req.Form.Get("benutzername")
		passwort := req.Form.Get("passwort")
		admin := req.Form.Has("admin")
		if _, schonVorhanden := db.Accounts[benutzername]; schonVorhanden {
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		acc := modell.Benutzer{
			Name:     benutzername,
			Admin:    admin,
			Passwort: sha256.Sum256([]byte(passwort)),
		}
		db.Accounts[benutzername] = &acc
		http.Redirect(res, req, "/accounts/", http.StatusFound)
	}))
}

func handleAccountLöschen(db *modell.Datenbank) http.Handler {
	return requireLogin(
		db, true,
		requireAccount(db, "account", http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			account := req.Context().Value(ctxKeyAccount).(*modell.Benutzer)
			delete(db.Accounts, account.Name)
			http.Redirect(res, req, "/accounts/", http.StatusFound)
		})),
	)
}

func handlePasswortÄndern(db *modell.Datenbank) http.Handler {
	return requireLogin(
		db, true,
		requireAccount(db, "account", http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			err := req.ParseForm()
			if err != nil || !req.Form.Has("passwort") {
				res.WriteHeader(http.StatusBadRequest)
				return
			}
			passwort := req.Form.Get("passwort")

			account := req.Context().Value(ctxKeyAccount).(*modell.Benutzer)
			account.Passwort = sha256.Sum256([]byte(passwort))
			http.Redirect(res, req, "/accounts/", http.StatusFound)
		})),
	)
}
