package main

import (
	"crypto/sha256"
	"sync"
	"time"
)

type datenbank struct {
	lock      sync.Mutex
	Accounts  map[string]*benutzer `json:"accounts"`
	Sitzungen map[string]*sitzung  `json:"-"`
	Objekte   map[int32]*objekt    `json:"objekte"`
}

func newLeereDatenbank() *datenbank {
	return &datenbank{
		Accounts:  make(map[string]*benutzer),
		Sitzungen: make(map[string]*sitzung),
		Objekte:   make(map[int32]*objekt),
	}
}

type sitzung struct {
	Schlüssel     string
	Benutzer      string
	LetzerZugriff time.Time
}

type benutzer struct {
	Name     string            `json:"Name"`
	Admin    bool              `json:"admin"`
	Passwort [sha256.Size]byte `json:"passwort"`
}

type objekt struct {
	Id       int32              `json:"id"`
	Name     string             `json:"name"`
	Raum     string             `json:"raum"`
	Probleme map[int32]*problem `json:"probleme"`
}

type problem struct {
	Id           int32     `json:"id"`
	Ersteller    string    `json:"ersteller"`
	Datum        time.Time `json:"datum"`
	Beschreibung string    `json:"beschreibung"`
	Status       string    `json:"status"`
}
