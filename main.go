package main

import (
	"crypto/sha256"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"
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

func newKopfzeileVorlageDaten(benutzer *benutzer) kopfzeileVorlageDaten {
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

func initRoutes(db *datenbank) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /{$}", http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		http.Redirect(res, req, "/anmelden/", http.StatusFound)
	}))

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

	mux.Handle("GET /probleme/{$}", handleProblemListe(db))

	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		db.lock.Lock()
		defer db.lock.Unlock()
		mux.ServeHTTP(res, req)
	})
}

const saveFile = "daten.json"

type server struct {
	db *datenbank
}

func newServer() (*server, error) {
	s := &server{
		db: newLeereDatenbank(),
	}
	err := s.loadData()
	if err != nil {
		return nil, fmt.Errorf("failed to load data: %w", err)
	}
	return s, nil
}

func (s *server) loadData() error {
	if _, err := os.Stat(saveFile); errors.Is(err, os.ErrNotExist) {
		s.db.Accounts["root"] = &benutzer{
			Name:     "root",
			Admin:    true,
			Passwort: sha256.Sum256([]byte("root")),
		}
		return nil
	}

	data, err := os.ReadFile(saveFile)
	if err != nil {
		return fmt.Errorf("failed to open save file: %w", err)
	}
	err = json.Unmarshal(data, s.db)
	if err != nil {
		return fmt.Errorf("failed to parse save file: %w", err)
	}
	return nil
}

func (s *server) saveData() error {
	data, err := json.MarshalIndent(s.db, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode data: %w", err)
	}
	err = os.WriteFile(saveFile, data, 0700)
	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}
	return nil
}

func (s *server) backupData() {
	for {
		const delay = time.Second

		s.db.lock.Lock()
		func() {
			defer s.db.lock.Unlock()
			err := s.saveData()
			if err != nil {
				log.Println(err)
			}
		}()

		time.Sleep(delay)
	}
}

func (s *server) start() error {
	go s.backupData()
	return http.ListenAndServe(":8080", initRoutes(s.db))
}

func main() {
	s, err := newServer()
	if err != nil {
		panic(err)
	}
	err = s.start()
	if err != nil {
		panic(err)
	}
}
