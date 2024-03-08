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
			res.WriteHeader(http.StatusFound)
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
