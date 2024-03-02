package main

import (
	"encoding/json"
	"log"
	"math/rand/v2"
	"net/http"
	"sort"
	"time"
)

type objekt struct {
	Id       int32     `json:"id"`
	Name     string    `json:"name"`
	Raum     string    `json:"raum"`
	Probleme []problem `json:"probleme"`
}

type problem struct {
	Ersteller    string    `json:"ersteller"`
	Datum        time.Time `json:"datum"`
	Beschreibung string    `json:"beschreibung"`
	Status       string    `json:"status"`
}

func (s *server) handleObjekteLesen(res http.ResponseWriter, req *http.Request) {
	type anfrage struct {
		Id int32
	}

	var anfrageDaten anfrage
	err := json.NewDecoder(req.Body).Decode(&anfrageDaten)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	obj, ok := s.Objekte[anfrageDaten.Id]
	if !ok {
		res.WriteHeader(http.StatusNotFound)
		return
	}
	err = json.NewEncoder(res).Encode(obj)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}
}

func (s *server) handleObjekteAuflisten(res http.ResponseWriter, req *http.Request, admin *account) {
	type antwort struct {
		Objekte []*objekt `json:"objekte"`
	}

	antwortDaten := antwort{
		Objekte: make([]*objekt, 0, len(s.Objekte)),
	}
	for _, obj := range s.Objekte {
		antwortDaten.Objekte = append(antwortDaten.Objekte, obj)
	}
	sort.Slice(antwortDaten.Objekte, func(i, j int) bool {
		return antwortDaten.Objekte[i].Name < antwortDaten.Objekte[i].Name
	})
	err := json.NewEncoder(res).Encode(antwortDaten)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}
}

func (s *server) handleObjektErstellen(res http.ResponseWriter, req *http.Request, admin *account) {
	type antwort struct {
		Id int32 `json:"id"`
	}

	neueId := rand.Int32()
	s.Objekte[neueId] = &objekt{
		Id:   neueId,
		Name: "Neu Erstellt",
	}
	err := json.NewEncoder(res).Encode(antwort{
		Id: neueId,
	})
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}
}

func (s *server) handleObjektLöschen(res http.ResponseWriter, req *http.Request, admin *account) {
	type anfrage struct {
		Id int32
	}

	var anfrageDaten anfrage
	err := json.NewDecoder(req.Body).Decode(&anfrageDaten)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	if _, ok := s.Objekte[anfrageDaten.Id]; !ok {
		res.WriteHeader(http.StatusNotFound)
		return
	}
	delete(s.Objekte, anfrageDaten.Id)
}

func (s *server) handleObjektÄndern(res http.ResponseWriter, req *http.Request, admin *account) {
	type anfrage struct {
		Id   int32
		Name string
		Raum string
	}

	var anfrageDaten anfrage
	err := json.NewDecoder(req.Body).Decode(&anfrageDaten)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	obj, ok := s.Objekte[anfrageDaten.Id]
	if !ok {
		res.WriteHeader(http.StatusNotFound)
		return
	}
	obj.Name = anfrageDaten.Name
	obj.Raum = anfrageDaten.Raum
}

func (s *server) handleProblemMelden(res http.ResponseWriter, req *http.Request, acc *account) {
	type anfrage struct {
		Id           int32
		Beschreibung string
	}

	var anfrageDaten anfrage
	err := json.NewDecoder(req.Body).Decode(&anfrageDaten)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	obj, ok := s.Objekte[anfrageDaten.Id]
	if !ok {
		res.WriteHeader(http.StatusNotFound)
		return
	}
	obj.Probleme = append(obj.Probleme, problem{
		Ersteller:    acc.Name,
		Datum:        time.Now(),
		Beschreibung: anfrageDaten.Beschreibung,
		Status:       "offen",
	})
}

func (s *server) handleProblemLösen(res http.ResponseWriter, req *http.Request, admin *account) {
	type anfrage struct {
		Id      int32
		Problem int64
	}

	var anfrageDaten anfrage
	err := json.NewDecoder(req.Body).Decode(&anfrageDaten)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	obj, ok := s.Objekte[anfrageDaten.Id]
	if !ok {
		res.WriteHeader(http.StatusNotFound)
		return
	}
	obj.Probleme = append(obj.Probleme[:anfrageDaten.Problem], obj.Probleme[anfrageDaten.Problem+1:]...)
}
