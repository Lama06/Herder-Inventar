package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/Lama06/Herder-Inventar/auth"
	"github.com/Lama06/Herder-Inventar/modell"
)

func handleAnmelden(db *modell.Datenbank) http.Handler {
	type anfrage struct {
		Benutzername string
		Passwort     string
	}
	type antwort struct {
		Schlüssel string `json:"schlüssel"`
	}

	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		var anfrageDaten anfrage
		err := json.NewDecoder(req.Body).Decode(&anfrageDaten)
		if err != nil {
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		passwortHash := sha256.Sum256([]byte(anfrageDaten.Passwort))
		benutzer, ok := db.Accounts[anfrageDaten.Benutzername]
		if !ok || benutzer.Passwort != passwortHash {
			res.WriteHeader(http.StatusUnauthorized)
			return
		}

		schlüssel, err := auth.GenerateSchlüssel()
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			log.Println(err)
			return
		}
		err = json.NewEncoder(res).Encode(antwort{
			Schlüssel: schlüssel,
		})
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			log.Println(err)
			return
		}

		db.Sitzungen[schlüssel] = &modell.Sitzung{
			Schlüssel:     schlüssel,
			Benutzer:      anfrageDaten.Benutzername,
			LetzerZugriff: time.Now(),
		}

		go auth.SitzungSanduhr(db, schlüssel)
	})
}

func requireLogin(
	db *modell.Datenbank, admin bool,
	danach func(res http.ResponseWriter, req *http.Request, benutzer *modell.Benutzer),
) http.Handler {
	type anfrage struct {
		Schlüssel string
	}

	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		rumpf, err := io.ReadAll(req.Body)
		if err != nil {
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		req.Body = io.NopCloser(bytes.NewBuffer(rumpf))

		var anfrageDaten anfrage
		err = json.Unmarshal(rumpf, &anfrageDaten)
		if err != nil {
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		sitzung, ok := db.Sitzungen[anfrageDaten.Schlüssel]
		if !ok {
			res.WriteHeader(http.StatusUnauthorized)
			return
		}
		sitzung.LetzerZugriff = time.Now()
		benutzer, ok := db.Accounts[sitzung.Benutzer]
		if !ok || (admin && !benutzer.Admin) {
			res.WriteHeader(http.StatusUnauthorized)
			return
		}

		danach(res, req, benutzer)
	})
}

func handleAbmelden(db *modell.Datenbank) http.Handler {
	return requireLogin(db, false, func(res http.ResponseWriter, req *http.Request, benutzer *modell.Benutzer) {
		for schlüssel, sitzung := range db.Sitzungen {
			if sitzung.Benutzer != benutzer.Name {
				continue
			}
			delete(db.Sitzungen, schlüssel)
		}
	})
}

func handleRegistrieren(db *modell.Datenbank) http.Handler {
	type anfrage struct {
		Name     string
		Admin    bool
		Passwort string
	}

	return requireLogin(db, true, func(res http.ResponseWriter, req *http.Request, benutzer *modell.Benutzer) {
		var anfrageDaten anfrage
		err := json.NewDecoder(req.Body).Decode(&anfrageDaten)
		if err != nil {
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		acc := modell.Benutzer{
			Name:     anfrageDaten.Name,
			Admin:    anfrageDaten.Admin,
			Passwort: sha256.Sum256([]byte(anfrageDaten.Passwort)),
		}
		db.Accounts[acc.Name] = &acc
	})
}

func handleAccountLöschen(db *modell.Datenbank) http.Handler {
	type anfrage struct {
		Name string
	}

	return requireLogin(db, true, func(res http.ResponseWriter, req *http.Request, benutzer *modell.Benutzer) {
		var anfrageDaten anfrage
		err := json.NewDecoder(req.Body).Decode(&anfrageDaten)
		if err != nil {
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		_, ok := db.Accounts[anfrageDaten.Name]
		if !ok {
			res.WriteHeader(http.StatusNotFound)
			return
		}
		delete(db.Accounts, anfrageDaten.Name)
		for schlüssel, sitzung := range db.Sitzungen {
			if sitzung.Benutzer == anfrageDaten.Name {
				delete(db.Sitzungen, schlüssel)
			}
		}
	})
}

func handlePasswortÄndern(db *modell.Datenbank) http.Handler {
	type anfrage struct {
		NeuesPasswort string
	}

	return requireLogin(db, true, func(res http.ResponseWriter, req *http.Request, benutzer *modell.Benutzer) {
		var anfrageDaten anfrage
		err := json.NewDecoder(req.Body).Decode(&anfrageDaten)
		if err != nil {
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		benutzer.Passwort = sha256.Sum256([]byte(anfrageDaten.NeuesPasswort))
	})
}

func handleAccountsAuflisten(db *modell.Datenbank) http.Handler {
	type accountInfo struct {
		Name  string `json:"Name"`
		Admin bool   `json:"Admin"`
	}
	type antwort struct {
		Accounts []accountInfo `json:"Accounts"`
	}

	return requireLogin(db, true, func(res http.ResponseWriter, req *http.Request, benutzer *modell.Benutzer) {
		antwortDaten := antwort{
			Accounts: make([]accountInfo, 0, len(db.Accounts)),
		}
		for _, acc := range db.Accounts {
			antwortDaten.Accounts = append(antwortDaten.Accounts, accountInfo{
				Name:  acc.Name,
				Admin: acc.Admin,
			})
		}
		sort.Slice(antwortDaten.Accounts, func(i, j int) bool {
			return antwortDaten.Accounts[i].Name < antwortDaten.Accounts[j].Name
		})

		err := json.NewEncoder(res).Encode(antwortDaten)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			log.Println(err)
			return
		}
	})
}
