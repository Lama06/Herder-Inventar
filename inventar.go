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
	Id       int32              `json:"id"`
	Name     string             `json:"name"`
	Raum     string             `json:"raum"`
	Probleme map[int32]*problem `json:"probleme"`
}

func (o *objekt) api() objektApi {
	probleme := make([]problemApi, 0, len(o.Probleme))
	for _, problem := range o.Probleme {
		probleme = append(probleme, problem.api())
	}
	sort.Slice(probleme, func(i, j int) bool {
		return probleme[i].Datum > probleme[j].Datum
	})
	return objektApi{
		Id:       o.Id,
		Name:     o.Name,
		Raum:     o.Raum,
		Probleme: probleme,
	}
}

type objektApi struct {
	Id       int32        `json:"id"`
	Name     string       `json:"name"`
	Raum     string       `json:"raum"`
	Probleme []problemApi `json:"probleme"`
}

type problem struct {
	Id           int32     `json:"id"`
	Ersteller    string    `json:"ersteller"`
	Datum        time.Time `json:"datum"`
	Beschreibung string    `json:"beschreibung"`
	Status       string    `json:"status"`
}

func (p *problem) api() problemApi {
	return problemApi{
		Id:           p.Id,
		Ersteller:    p.Ersteller,
		Datum:        p.Datum.Unix(),
		Beschreibung: p.Beschreibung,
		Status:       p.Status,
	}
}

type problemApi struct {
	Id           int32  `json:"id"`
	Ersteller    string `json:"ersteller"`
	Datum        int64  `json:"datum"`
	Beschreibung string `json:"beschreibung"`
	Status       string `json:"status"`
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
	err = json.NewEncoder(res).Encode(obj.api())
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}
}

func (s *server) handleObjekteAuflisten(res http.ResponseWriter, req *http.Request, admin *account) {
	type antwort struct {
		Objekte []objektApi `json:"objekte"`
	}

	antwortDaten := antwort{
		Objekte: make([]objektApi, 0, len(s.Objekte)),
	}
	for _, obj := range s.Objekte {
		antwortDaten.Objekte = append(antwortDaten.Objekte, obj.api())
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
	type antwort struct {
		Id int32 `json:"id"`
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
	neueId := rand.Int32()
	obj.Probleme[neueId] = &problem{
		Id:           neueId,
		Ersteller:    acc.Name,
		Datum:        time.Now(),
		Beschreibung: anfrageDaten.Beschreibung,
		Status:       "offen",
	}
	err = json.NewEncoder(res).Encode(antwort{
		Id: neueId,
	})
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}
}

func (s *server) handleProblemLösen(res http.ResponseWriter, req *http.Request, admin *account) {
	type anfrage struct {
		Id      int32
		Problem int32
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
	_, ok = obj.Probleme[anfrageDaten.Problem]
	if !ok {
		res.WriteHeader(http.StatusNotFound)
		return
	}
	delete(obj.Probleme, anfrageDaten.Problem)
}
