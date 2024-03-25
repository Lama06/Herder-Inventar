package main

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

func requireObjekt(db *datenbank, pfadKomponente string, danach http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		idText := req.PathValue(pfadKomponente)
		id, err := strconv.ParseInt(idText, 10, 32)
		if err != nil {
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		objekt, ok := db.Objekte[int32(id)]
		if !ok {
			res.WriteHeader(http.StatusNotFound)
			return
		}
		req = req.WithContext(context.WithValue(req.Context(), ctxKeyObjekt, objekt))
		danach.ServeHTTP(res, req)
	})
}

func requireProblem(pfadKomponente string, danach http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		objekt := req.Context().Value(ctxKeyObjekt).(*objekt)
		idText := req.PathValue(pfadKomponente)
		id, err := strconv.ParseInt(idText, 10, 32)
		if err != nil {
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		problem, ok := objekt.Probleme[int32(id)]
		if !ok {
			res.WriteHeader(http.StatusNotFound)
			return
		}
		req = req.WithContext(context.WithValue(req.Context(), ctxKeyProblem, problem))
		danach.ServeHTTP(res, req)
	})
}

func requireSeite(pfadKomponente string, danach http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		seiteText := req.PathValue(pfadKomponente)
		seite, err := strconv.Atoi(seiteText)
		if err != nil {
			seite = 1
		}
		seite = max(1, seite)
		req = req.WithContext(context.WithValue(req.Context(), ctxKeySeite, seite))
		danach.ServeHTTP(res, req)
	})
}

type inventarVorlageDaten struct {
	kopfzeileVorlageDaten
	Objekte       []*objekt
	Seite, Seiten int
}

func handleInventarListe(db *datenbank) http.Handler {
	const objekteProSeite = 10

	return requireSeite("seite", requireLogin(db, true, http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		seite := req.Context().Value(ctxKeySeite).(int)
		benutzer := req.Context().Value(ctxKeyBenutzer).(*benutzer)

		objekte := make([]*objekt, 0, len(db.Objekte))
		for _, objekt := range db.Objekte {
			objekte = append(objekte, objekt)
		}
		sort.Slice(objekte, func(i, j int) bool {
			return objekte[i].Id < objekte[j].Id
		})

		var seiten int
		switch {
		case len(objekte) == 0:
			seiten = 1
		case len(objekte)%objekteProSeite != 0:
			seiten = len(objekte)/objekteProSeite + 1
		default:
			seiten = len(objekte)/objekteProSeite + 1
		}
		seite = min(seiten, seite)

		var antwort bytes.Buffer
		err := vorlage.ExecuteTemplate(&antwort, "inventar", inventarVorlageDaten{
			kopfzeileVorlageDaten: newKopfzeileVorlageDaten(benutzer),
			Objekte:               objekte[(seite-1)*objekteProSeite : min(seite*objekteProSeite, len(objekte))],

			Seite:  seite,
			Seiten: seiten,
		})
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = antwort.WriteTo(res)
	})))
}

type suchenVorlageDaten struct {
	kopfzeileVorlageDaten
	Suche   string
	Objekte []*objekt
}

func handleObjekteSuchen(db *datenbank) http.Handler {
	return requireLogin(db, true, http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		benutzer := req.Context().Value(ctxKeyBenutzer).(*benutzer)

		err := req.ParseForm()
		if err != nil {
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		suche := req.Form.Get("suche")
		sucheKlein := strings.ToLower(suche)

		var ergebnisse []*objekt
		for _, objekt := range db.Objekte {
			nameKlein := strings.ToLower(objekt.Name)
			if strings.Contains(nameKlein, sucheKlein) {
				ergebnisse = append(ergebnisse, objekt)
			}
		}
		sort.Slice(ergebnisse, func(i, j int) bool {
			return ergebnisse[i].Name < ergebnisse[j].Name
		})

		var antwort bytes.Buffer
		err = vorlage.ExecuteTemplate(&antwort, "suche", suchenVorlageDaten{
			kopfzeileVorlageDaten: newKopfzeileVorlageDaten(benutzer),
			Objekte:               ergebnisse,
			Suche:                 suche,
		})
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = antwort.WriteTo(res)
	}))
}

func handleObjektErstellen(db *datenbank) http.Handler {
	return requireLogin(db, true, http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		err := req.ParseForm()
		if err != nil {
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		id := rand.Int32()
		obj := objekt{
			Id:       id,
			Name:     req.Form.Get("name"),
			Raum:     req.Form.Get("raum"),
			Probleme: nil,
		}
		db.Objekte[id] = &obj
		http.Redirect(res, req, "/objekte/", http.StatusFound)
	}))
}

func handleObjektLöschen(db *datenbank) http.Handler {
	return requireLogin(
		db,
		true,
		requireObjekt(db, "objekt", http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			obj := req.Context().Value(ctxKeyObjekt).(*objekt)
			delete(db.Objekte, obj.Id)
			http.Redirect(res, req, "/objekte/", http.StatusFound)
		})),
	)
}

type objektVorlageDaten struct {
	kopfzeileVorlageDaten
	Obj *objekt
}

func handleObjekt(db *datenbank) http.Handler {
	return requireLoginWeich(db, requireObjekt(db, "objekt", http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		benutzer, _ := req.Context().Value(ctxKeyBenutzer).(*benutzer)
		obj := req.Context().Value(ctxKeyObjekt).(*objekt)
		var antwort bytes.Buffer
		err := vorlage.ExecuteTemplate(&antwort, "objekt", objektVorlageDaten{
			kopfzeileVorlageDaten: newKopfzeileVorlageDaten(benutzer),
			Obj:                   obj,
		})
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = antwort.WriteTo(res)
	})))
}

func handleObjektBearbeiten(db *datenbank) http.Handler {
	return requireLogin(
		db, true,
		requireObjekt(db, "objekt", http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			obj := req.Context().Value(ctxKeyObjekt).(*objekt)
			err := req.ParseForm()
			if err != nil || !req.Form.Has("name") || !req.Form.Has("raum") {
				res.WriteHeader(http.StatusBadRequest)
				return
			}
			name, raum := req.Form.Get("name"), req.Form.Get("raum")
			obj.Name = name
			obj.Raum = raum
			http.Redirect(res, req, fmt.Sprintf("/objekt/%v/", obj.Id), http.StatusFound)
		})),
	)
}

func handleProblemMelden(db *datenbank) http.Handler {
	return requireLogin(
		db, false,
		requireObjekt(db, "objekt", http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			benutzer := req.Context().Value(ctxKeyBenutzer).(*benutzer)
			obj := req.Context().Value(ctxKeyObjekt).(*objekt)

			err := req.ParseForm()
			if err != nil {
				res.WriteHeader(http.StatusBadRequest)
				return
			}
			beschreibung := req.Form.Get("beschreibung")

			problemId := rand.Int32()
			if obj.Probleme == nil {
				obj.Probleme = make(map[int32]*problem)
			}
			obj.Probleme[problemId] = &problem{
				Id:           problemId,
				Ersteller:    benutzer.Name,
				Datum:        time.Now(),
				Beschreibung: beschreibung,
				Status:       "offen",
			}

			http.Redirect(res, req, fmt.Sprintf("/objekt/%v/", obj.Id), http.StatusFound)
		})),
	)
}

func handleProblemLösen(db *datenbank) http.Handler {
	return requireLogin(
		db, false,
		requireObjekt(db, "objekt", requireProblem(
			"problem",
			http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
				obj := req.Context().Value(ctxKeyObjekt).(*objekt)
				problem := req.Context().Value(ctxKeyProblem).(*problem)
				delete(obj.Probleme, problem.Id)
				http.Redirect(res, req, fmt.Sprintf("/objekt/%v/", obj.Id), http.StatusFound)
			}),
		)),
	)
}

type problemListeEintrag struct {
	Problem *problem
	Obj     *objekt
}

type problemListeVorlageDaten struct {
	kopfzeileVorlageDaten
	Probleme []problemListeEintrag
}

func handleProblemListe(db *datenbank) http.Handler {
	return requireLogin(db, true, http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		benutzer := req.Context().Value(ctxKeyBenutzer).(*benutzer)

		var probleme []problemListeEintrag
		for _, obj := range db.Objekte {
			for _, problem := range obj.Probleme {
				probleme = append(probleme, problemListeEintrag{Problem: problem, Obj: obj})
			}
		}

		var antwort bytes.Buffer
		err := vorlage.ExecuteTemplate(&antwort, "probleme", problemListeVorlageDaten{
			kopfzeileVorlageDaten: newKopfzeileVorlageDaten(benutzer),
			Probleme:              probleme,
		})
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = antwort.WriteTo(res)
	}))
}
