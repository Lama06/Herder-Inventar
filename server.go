package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

const saveFile = "daten.json"

type server struct {
	lock     sync.RWMutex
	Accounts map[string]*account `json:"accounts"`
	Objekte  map[int64]*objekt   `json:"objekte"`
}

func newServer() (*server, error) {
	s := &server{}
	err := s.loadData()
	if err != nil {
		return nil, fmt.Errorf("failed to load data: %w", err)
	}
	return s, nil
}

func (s *server) loadData() error {
	if _, err := os.Stat(saveFile); errors.Is(err, os.ErrNotExist) {
		file, err := os.Create(saveFile)
		if err != nil {
			return fmt.Errorf("failed to create save file: %w", err)
		}
		err = file.Close()
		if err != nil {
			return fmt.Errorf("failed to close file: %w", err)
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
	file, err := os.Open(saveFile)
	if err != nil {
		return fmt.Errorf("failed to open save file: %w", err)
	}
	defer func() {
		err := file.Close()
		if err != nil {
			log.Println(err)
		}
	}()

	return nil
}

func (s *server) backupData() {
	for {
		const delay = 30 * time.Minute
		time.Sleep(delay)

		s.lock.Lock()
		func() {
			defer s.lock.Unlock()
			err := s.saveData()
			if err != nil {
				log.Println(err)
			}
		}()
	}
}

func (s *server) lockHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		s.lock.Lock()
		defer s.lock.Unlock()
		handler.ServeHTTP(res, req)
	})
}

func (s *server) initRoutes() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("POST /auth/login/", http.HandlerFunc(s.handleLogin))
	mux.Handle("POST /auth/logout/", s.requireLogin(false, s.handleLogout))
	mux.Handle("POST /auth/register/", s.requireLogin(true, s.handleRegister))
	mux.Handle("POST /auth/list_accounts/", s.requireLogin(true, s.handleListAccounts))
	mux.Handle("POST /auth/change_password/", s.requireLogin(false, s.handleChangePassword))
	mux.Handle("POST /auth/delete/", s.requireLogin(true, s.handleDeleteAccount))

	mux.HandleFunc("GET /objekte/lesen/", s.handleObjekteLesen)
	mux.Handle("POST /objekte/auflisten/", s.requireLogin(true, s.handleObjekteAuflisten))
	mux.Handle("POST /objekte/erstellen/", s.requireLogin(true, s.handleObjektErstellen))
	mux.Handle("POST /objekte/löschen/", s.requireLogin(true, s.handleObjektLöschen))
	mux.Handle("POST /objekte/ändern/", s.requireLogin(true, s.handleObjektÄndern))
	mux.Handle("POST /probleme/melden/", s.requireLogin(false, s.handleProblemMelden))
	mux.Handle("POST /probleme/lösen/", s.requireLogin(true, s.handleProblemLösen))

	return mux
}

func (s *server) start() error {
	go s.backupData()
	return http.ListenAndServe(":8080", s.lockHandler(s.initRoutes()))
}