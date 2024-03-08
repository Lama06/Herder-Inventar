package api

import (
	"encoding/json"
	"log"
	"math/rand/v2"
	"net/http"
	"sort"
	"time"

	"github.com/Lama06/Herder-Inventar/modell"
)

type objektApi struct {
	Id       int32        `json:"id"`
	Name     string       `json:"name"`
	Raum     string       `json:"raum"`
	Probleme []problemApi `json:"probleme"`
}

func newApiObjek(o *modell.Objekt) objektApi {
	probleme := make([]problemApi, 0, len(o.Probleme))
	for _, problem := range o.Probleme {
		probleme = append(probleme, newApiProblem(problem))
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

type problemApi struct {
	Id           int32  `json:"id"`
	Ersteller    string `json:"ersteller"`
	Datum        int64  `json:"datum"`
	Beschreibung string `json:"beschreibung"`
	Status       string `json:"status"`
}

func newApiProblem(p *modell.Problem) problemApi {
	return problemApi{
		Id:           p.Id,
		Ersteller:    p.Ersteller,
		Datum:        p.Datum.Unix(),
		Beschreibung: p.Beschreibung,
		Status:       p.Status,
	}
}

func handleObjekteLesen(db *modell.Datenbank) http.Handler {
	type anfrage struct {
		Id int32
	}

	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		var anfrageDaten anfrage
		err := json.NewDecoder(req.Body).Decode(&anfrageDaten)
		if err != nil {
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		obj, ok := db.Objekte[anfrageDaten.Id]
		if !ok {
			res.WriteHeader(http.StatusNotFound)
			return
		}
		err = json.NewEncoder(res).Encode(newApiObjek(obj))
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			log.Println(err)
			return
		}
	})
}

func handleObjekteAuflisten(db *modell.Datenbank) http.Handler {
	type antwort struct {
		Objekte []objektApi `json:"objekte"`
	}

	return requireLogin(db, true, func(res http.ResponseWriter, req *http.Request, benutzer *modell.Benutzer) {
		antwortDaten := antwort{
			Objekte: make([]objektApi, 0, len(db.Objekte)),
		}
		for _, obj := range db.Objekte {
			antwortDaten.Objekte = append(antwortDaten.Objekte, newApiObjek(obj))
		}
		sort.Slice(antwortDaten.Objekte, func(i, j int) bool {
			return antwortDaten.Objekte[i].Name < antwortDaten.Objekte[j].Name
		})
		err := json.NewEncoder(res).Encode(antwortDaten)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			log.Println(err)
			return
		}
	})
}

func handleObjektErstellen(db *modell.Datenbank) http.Handler {
	type antwort struct {
		Id int32 `json:"id"`
	}

	return requireLogin(db, true, func(res http.ResponseWriter, req *http.Request, benutzer *modell.Benutzer) {
		neueId := rand.Int32()
		db.Objekte[neueId] = &modell.Objekt{
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
	})
}

func handleObjektLöschen(db *modell.Datenbank) http.Handler {
	type anfrage struct {
		Id int32
	}

	return requireLogin(db, true, func(res http.ResponseWriter, req *http.Request, benutzer *modell.Benutzer) {
		var anfrageDaten anfrage
		err := json.NewDecoder(req.Body).Decode(&anfrageDaten)
		if err != nil {
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		if _, ok := db.Objekte[anfrageDaten.Id]; !ok {
			res.WriteHeader(http.StatusNotFound)
			return
		}
		delete(db.Objekte, anfrageDaten.Id)
	})
}

func handleObjektÄndern(db *modell.Datenbank) http.Handler {
	type anfrage struct {
		Id   int32
		Name string
		Raum string
	}

	return requireLogin(db, true, func(res http.ResponseWriter, req *http.Request, benutzer *modell.Benutzer) {
		var anfrageDaten anfrage
		err := json.NewDecoder(req.Body).Decode(&anfrageDaten)
		if err != nil {
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		obj, ok := db.Objekte[anfrageDaten.Id]
		if !ok {
			res.WriteHeader(http.StatusNotFound)
			return
		}
		obj.Name = anfrageDaten.Name
		obj.Raum = anfrageDaten.Raum
	})
}

func handleProblemMelden(db *modell.Datenbank) http.Handler {
	type anfrage struct {
		Id           int32
		Beschreibung string
	}
	type antwort struct {
		Id int32 `json:"id"`
	}

	return requireLogin(db, false, func(res http.ResponseWriter, req *http.Request, benutzer *modell.Benutzer) {
		var anfrageDaten anfrage
		err := json.NewDecoder(req.Body).Decode(&anfrageDaten)
		if err != nil {
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		obj, ok := db.Objekte[anfrageDaten.Id]
		if !ok {
			res.WriteHeader(http.StatusNotFound)
			return
		}
		neueId := rand.Int32()
		obj.Probleme[neueId] = &modell.Problem{
			Id:           neueId,
			Ersteller:    benutzer.Name,
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
	})
}

func handleProblemLösen(db *modell.Datenbank) http.Handler {
	type anfrage struct {
		Id      int32
		Problem int32
	}

	return requireLogin(db, true, func(res http.ResponseWriter, req *http.Request, benutzer *modell.Benutzer) {
		var anfrageDaten anfrage
		err := json.NewDecoder(req.Body).Decode(&anfrageDaten)
		if err != nil {
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		obj, ok := db.Objekte[anfrageDaten.Id]
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
	})
}
