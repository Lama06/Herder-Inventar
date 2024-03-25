package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	_ "embed"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"time"
)

const sitzungDauer = 30 * time.Minute

func generateSchlüssel() (string, error) {
	const länge = 64
	const buchstaben = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	var schlüssel [länge]byte
	for i := range schlüssel {
		index, err := rand.Int(rand.Reader, big.NewInt(int64(len(buchstaben))))
		if err != nil {
			return "", fmt.Errorf("failed to generate random number: %w", err)
		}
		schlüssel[i] = buchstaben[index.Int64()]
	}
	return string(schlüssel[:]), nil
}

func sitzungSanduhr(db *datenbank, schlüssel string) {
	for {
		time.Sleep(sitzungDauer)
		db.lock.Lock()
		sitzung, ok := db.Sitzungen[schlüssel]
		if !ok {
			db.lock.Unlock()
			return
		}
		if time.Now().Sub(sitzung.LetzerZugriff) > sitzungDauer {
			delete(db.Sitzungen, schlüssel)
			db.lock.Unlock()
			return
		}
		db.lock.Unlock()
	}
}

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
		err = vorlage.ExecuteTemplate(&antwort, "anmelden", anmeldenVorlageDaten{
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

func handleAnmeldenPost(db *datenbank) http.Handler {
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
			err := vorlage.ExecuteTemplate(res, "anmelden", anmeldenVorlageDaten{Fehler: true})
			if err != nil {
				log.Println(err)
				return
			}
			return
		}

		schlüssel, err := generateSchlüssel()
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		db.Sitzungen[schlüssel] = &sitzung{
			Schlüssel:     schlüssel,
			Benutzer:      benutzername,
			LetzerZugriff: time.Now(),
		}
		go sitzungSanduhr(db, schlüssel)

		http.SetCookie(res, &http.Cookie{
			Name:     "schluessel",
			Value:    schlüssel,
			Path:     "/",
			MaxAge:   int(sitzungDauer.Seconds()),
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		})
		if req.Form.Has("weiterleitung") {
			http.Redirect(res, req, req.Form.Get("weiterleitung"), http.StatusFound)
			return
		}
		http.Redirect(res, req, "/objekte/", http.StatusFound)
	})
}

func handleAbmelden(db *datenbank) http.Handler {
	return requireLogin(db, false, http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		benutzer := req.Context().Value(ctxKeyBenutzer).(*benutzer)
		for schlüssel, sitzung := range db.Sitzungen {
			if sitzung.Benutzer == benutzer.Name {
				delete(db.Sitzungen, schlüssel)
			}
		}
		http.Redirect(res, req, "/anmelden/", http.StatusFound)
	}))
}

func requireLogin(
	db *datenbank, admin bool,
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

func requireLoginWeich(db *datenbank, danach http.Handler) http.Handler {
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

func requireAccount(db *datenbank, pfadKomponente string, danach http.Handler) http.Handler {
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

type accountsVorlageDaten struct {
	kopfzeileVorlageDaten
	Accounts map[string]*benutzer
}

func handleAccounts(db *datenbank) http.Handler {
	return requireLogin(db, true, http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		benutzer := req.Context().Value(ctxKeyBenutzer).(*benutzer)
		var antwort bytes.Buffer
		err := vorlage.ExecuteTemplate(&antwort, "accounts", accountsVorlageDaten{
			kopfzeileVorlageDaten: newKopfzeileVorlageDaten(benutzer),
			Accounts:              db.Accounts,
		})
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = antwort.WriteTo(res)
	}))
}

func handleAccountRegistrieren(db *datenbank) http.Handler {
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
		acc := benutzer{
			Name:     benutzername,
			Admin:    admin,
			Passwort: sha256.Sum256([]byte(passwort)),
		}
		db.Accounts[benutzername] = &acc
		http.Redirect(res, req, "/accounts/", http.StatusFound)
	}))
}

func handleAccountLöschen(db *datenbank) http.Handler {
	return requireLogin(
		db, true,
		requireAccount(db, "account", http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			account := req.Context().Value(ctxKeyAccount).(*benutzer)
			delete(db.Accounts, account.Name)
			http.Redirect(res, req, "/accounts/", http.StatusFound)
		})),
	)
}

func handlePasswortÄndern(db *datenbank) http.Handler {
	return requireLogin(
		db, true,
		requireAccount(db, "account", http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			err := req.ParseForm()
			if err != nil || !req.Form.Has("passwort") {
				res.WriteHeader(http.StatusBadRequest)
				return
			}
			passwort := req.Form.Get("passwort")

			account := req.Context().Value(ctxKeyAccount).(*benutzer)
			account.Passwort = sha256.Sum256([]byte(passwort))
			http.Redirect(res, req, "/accounts/", http.StatusFound)
		})),
	)
}
