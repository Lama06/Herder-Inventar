package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Lama06/Herder-Inventar/api"
	"github.com/Lama06/Herder-Inventar/frontend"
	"github.com/Lama06/Herder-Inventar/modell"
)

const saveFile = "daten.json"

type server struct {
	db *modell.Datenbank
}

func newServer() (*server, error) {
	s := &server{
		db: modell.NewLeereDatenbank(),
	}
	err := s.loadData()
	if err != nil {
		return nil, fmt.Errorf("failed to load data: %w", err)
	}
	return s, nil
}

func (s *server) loadData() error {
	if _, err := os.Stat(saveFile); errors.Is(err, os.ErrNotExist) {
		s.db.Accounts["root"] = &modell.Benutzer{
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

		s.db.Lock.Lock()
		func() {
			defer s.db.Lock.Unlock()
			err := s.saveData()
			if err != nil {
				log.Println(err)
			}
		}()

		time.Sleep(delay)
	}
}

func (s *server) initRoutes() (http.Handler, error) {
	mux := http.NewServeMux()
	mux.Handle("/api/", http.StripPrefix("/api", api.New(s.db)))
	mux.Handle("/", frontend.New(s.db))
	return mux, nil
}

func (s *server) start() error {
	routes, err := s.initRoutes()
	if err != nil {
		return err
	}

	go s.backupData()
	return http.ListenAndServe(":8080", routes)
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
