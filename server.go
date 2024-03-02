package main

import (
	"crypto/sha256"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

const saveFile = "daten.json"

//go:embed frontend/*
var frontend embed.FS

type server struct {
	lock     sync.Mutex
	Accounts map[string]*account `json:"accounts"`
	Objekte  map[int32]*objekt   `json:"objekte"`
}

func newServer() (*server, error) {
	s := &server{
		Accounts: make(map[string]*account),
		Objekte:  make(map[int32]*objekt),
	}
	err := s.loadData()
	if err != nil {
		return nil, fmt.Errorf("failed to load data: %w", err)
	}
	return s, nil
}

func (s *server) loadData() error {
	if _, err := os.Stat(saveFile); errors.Is(err, os.ErrNotExist) {
		s.Accounts["root"] = &account{
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
	err = json.Unmarshal(data, s)
	if err != nil {
		return fmt.Errorf("failed to parse save file: %w", err)
	}
	return nil
}

func (s *server) saveData() error {
	data, err := json.MarshalIndent(s, "", "  ")
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

		s.lock.Lock()
		func() {
			defer s.lock.Unlock()
			err := s.saveData()
			if err != nil {
				log.Println(err)
			}
		}()

		time.Sleep(delay)
	}
}

func (s *server) lockHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		s.lock.Lock()
		defer s.lock.Unlock()
		handler.ServeHTTP(res, req)
	})
}

func (s *server) initApiRoutes() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/api/auth/login/", http.HandlerFunc(s.handleLogin))
	mux.Handle("/api/auth/logout/", s.requireLogin(false, s.handleLogout))
	mux.Handle("/api/auth/register/", s.requireLogin(true, s.handleRegister))
	mux.Handle("/api/auth/list_accounts/", s.requireLogin(true, s.handleListAccounts))
	mux.Handle("/api/auth/change_password/", s.requireLogin(false, s.handleChangePassword))
	mux.Handle("/api/auth/delete/", s.requireLogin(true, s.handleDeleteAccount))

	mux.HandleFunc("/api/objekte/lesen/", s.handleObjekteLesen)
	mux.Handle("/api/objekte/auflisten/", s.requireLogin(true, s.handleObjekteAuflisten))
	mux.Handle("/api/objekte/erstellen/", s.requireLogin(true, s.handleObjektErstellen))
	mux.Handle("/api/objekte/loeschen/", s.requireLogin(true, s.handleObjektLöschen))
	mux.Handle("/api/objekte/aendern/", s.requireLogin(true, s.handleObjektÄndern))
	mux.Handle("/api/probleme/melden/", s.requireLogin(false, s.handleProblemMelden))
	mux.Handle("/api/probleme/loesen/", s.requireLogin(true, s.handleProblemLösen))

	return s.lockHandler(mux)
}

func (s *server) initRoutes() (http.Handler, error) {
	frontendSub, err := fs.Sub(frontend, "frontend")
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.Handle("/api/", s.initApiRoutes())
	mux.Handle("/frontend/", http.StripPrefix("/frontend/", http.FileServerFS(frontendSub)))
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
