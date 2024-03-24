package modell

import (
	"crypto/sha256"
	"sync"
	"time"
)

type Datenbank struct {
	Lock      sync.Mutex           `json:"-"`
	Accounts  map[string]*Benutzer `json:"accounts"`
	Sitzungen map[string]*Sitzung  `json:"-"`
	Objekte   map[int32]*Objekt    `json:"objekte"`
}

func NewLeereDatenbank() *Datenbank {
	return &Datenbank{
		Accounts:  make(map[string]*Benutzer),
		Sitzungen: make(map[string]*Sitzung),
		Objekte:   make(map[int32]*Objekt),
	}
}

type Sitzung struct {
	Schlüssel     string
	Benutzer      string
	LetzerZugriff time.Time
}

type Benutzer struct {
	Name     string            `json:"Name"`
	Admin    bool              `json:"admin"`
	Passwort [sha256.Size]byte `json:"passwort"`
}

type Objekt struct {
	Id       int32              `json:"id"`
	Name     string             `json:"name"`
	Raum     string             `json:"raum"`
	Probleme map[int32]*Problem `json:"probleme"`
}

type Problem struct {
	Id           int32     `json:"id"`
	Ersteller    string    `json:"ersteller"`
	Datum        time.Time `json:"datum"`
	Beschreibung string    `json:"beschreibung"`
	Status       string    `json:"status"`
}
