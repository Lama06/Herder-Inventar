package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"sort"
	"time"
)

type account struct {
	Name     string            `json:"Name"`
	Admin    bool              `json:"admin"`
	Passwort [sha256.Size]byte `json:"passwort"`

	schlüssel     string
	letzteAnfrage time.Time
}

func (a *account) checkSchlüssel() string {
	const timeout = 20 * time.Minute
	if time.Now().Sub(a.letzteAnfrage) > timeout {
		a.schlüssel = ""
	}
	return a.schlüssel
}

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

func (s *server) accountForSchlüssel(schlüssel string) *account {
	for _, acc := range s.Accounts {
		accSchlüssel := acc.checkSchlüssel()
		if accSchlüssel == schlüssel {
			return acc
		}
	}
	return nil
}

func (s *server) handleLogin(res http.ResponseWriter, req *http.Request) {
	type anfrage struct {
		Benutzername string
		Passwort     string
	}
	type antwort struct {
		Schlüssel string `json:"schlüssel"`
	}

	var anfrageDaten anfrage
	err := json.NewDecoder(req.Body).Decode(&anfrageDaten)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	benutzer, ok := s.Accounts[anfrageDaten.Benutzername]
	if !ok {
		res.WriteHeader(http.StatusUnauthorized)
		return
	}
	passwortHash := sha256.Sum256([]byte(anfrageDaten.Passwort))
	if benutzer.Passwort != passwortHash {
		res.WriteHeader(http.StatusUnauthorized)
		return
	}
	schlüssel, err := generateSchlüssel()
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
	benutzer.letzteAnfrage = time.Now()
}

func (s *server) requireLogin(
	admin bool,
	danach func(res http.ResponseWriter, req *http.Request, acc *account),
) http.Handler {
	type anfrage struct {
		Schlüssel string
	}

	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		var anfrageDaten anfrage
		err := json.NewDecoder(req.Body).Decode(&anfrageDaten)
		if err != nil {
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		acc := s.accountForSchlüssel(anfrageDaten.Schlüssel)
		if acc == nil || (admin && !acc.Admin) {
			res.WriteHeader(http.StatusUnauthorized)
			return
		}
		acc.letzteAnfrage = time.Now()
		danach(res, req, acc)
	})
}

func (s *server) handleLogout(res http.ResponseWriter, req *http.Request, acc *account) {
	acc.schlüssel = ""
}

func (s *server) handleRegister(res http.ResponseWriter, req *http.Request, admin *account) {
	type anfrage struct {
		Name     string
		Admin    bool
		Passwort string
	}

	var anfrageDaten anfrage
	err := json.NewDecoder(req.Body).Decode(&anfrageDaten)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	acc := &account{
		Name:     anfrageDaten.Name,
		Admin:    anfrageDaten.Admin,
		Passwort: sha256.Sum256([]byte(anfrageDaten.Passwort)),
	}
	s.Accounts[acc.Name] = acc
}

func (s *server) handleDeleteAccount(res http.ResponseWriter, req *http.Request, admin *account) {
	type anfrage struct {
		Name string
	}

	var anfrageDaten anfrage
	err := json.NewDecoder(req.Body).Decode(&anfrageDaten)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	_, ok := s.Accounts[anfrageDaten.Name]
	if !ok {
		res.WriteHeader(http.StatusNotFound)
		return
	}
	delete(s.Accounts, anfrageDaten.Name)
}

func (s *server) handleChangePassword(res http.ResponseWriter, req *http.Request, benutzer *account) {
	type anfrage struct {
		NeuesPasswort string
	}

	var anfrageDaten anfrage
	err := json.NewDecoder(req.Body).Decode(&anfrageDaten)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	benutzer.Passwort = sha256.Sum256([]byte(anfrageDaten.NeuesPasswort))
}

func (s *server) handleListAccounts(res http.ResponseWriter, req *http.Request, admin *account) {
	type accountInfo struct {
		Name  string `json:"Name"`
		Admin bool   `json:"Admin"`
	}
	type antwort struct {
		Accounts []accountInfo `json:"Accounts"`
	}

	antwortDaten := antwort{
		Accounts: make([]accountInfo, 0, len(s.Accounts)),
	}
	for _, acc := range s.Accounts {
		antwortDaten.Accounts = append(antwortDaten.Accounts, accountInfo{
			Name:  acc.Name,
			Admin: acc.Admin,
		})
	}
	sort.Slice(antwortDaten.Accounts, func(i, j int) bool {
		return antwortDaten.Accounts[i].Name < antwortDaten.Accounts[i].Name
	})
	err := json.NewEncoder(res).Encode(antwortDaten)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}
}
