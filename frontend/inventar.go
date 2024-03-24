package frontend

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

	"github.com/Lama06/Herder-Inventar/modell"
)

func requireObjekt(db *modell.Datenbank, pfadKomponente string, danach http.Handler) http.Handler {
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
		objekt := req.Context().Value(ctxKeyObjekt).(*modell.Objekt)
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
	Objekte       []*modell.Objekt
	Seite, Seiten int
}

func handleInventarListe(db *modell.Datenbank) http.Handler {
	const objekteProSeite = 10

	return requireSeite("seite", requireLogin(db, true, http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		seite := req.Context().Value(ctxKeySeite).(int)
		benutzer := req.Context().Value(ctxKeyBenutzer).(*modell.Benutzer)

		objekte := make([]*modell.Objekt, 0, len(db.Objekte))
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
	Objekte []*modell.Objekt
}

func handleObjekteSuchen(db *modell.Datenbank) http.Handler {
	return requireLogin(db, true, http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		benutzer := req.Context().Value(ctxKeyBenutzer).(*modell.Benutzer)

		err := req.ParseForm()
		if err != nil {
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		suche := req.Form.Get("suche")
		sucheKlein := strings.ToLower(suche)

		var ergebnisse []*modell.Objekt
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

func handleObjektErstellen(db *modell.Datenbank) http.Handler {
	return requireLogin(db, true, http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		err := req.ParseForm()
		if err != nil {
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		id := rand.Int32()
		obj := modell.Objekt{
			Id:       id,
			Name:     req.Form.Get("name"),
			Raum:     req.Form.Get("raum"),
			Probleme: nil,
		}
		db.Objekte[id] = &obj
		http.Redirect(res, req, "/objekte/", http.StatusFound)
	}))
}

func handleObjektLöschen(db *modell.Datenbank) http.Handler {
	return requireLogin(
		db,
		true,
		requireObjekt(db, "objekt", http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			obj := req.Context().Value(ctxKeyObjekt).(*modell.Objekt)
			delete(db.Objekte, obj.Id)
			http.Redirect(res, req, "/objekte/", http.StatusFound)
		})),
	)
}

type objektVorlageDaten struct {
	kopfzeileVorlageDaten
	Obj *modell.Objekt
}

func handleObjekt(db *modell.Datenbank) http.Handler {
	return requireLoginWeich(db, requireObjekt(db, "objekt", http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		benutzer, _ := req.Context().Value(ctxKeyBenutzer).(*modell.Benutzer)
		obj := req.Context().Value(ctxKeyObjekt).(*modell.Objekt)
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

func handleObjektBearbeiten(db *modell.Datenbank) http.Handler {
	return requireLogin(
		db, true,
		requireObjekt(db, "objekt", http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			obj := req.Context().Value(ctxKeyObjekt).(*modell.Objekt)
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

func handleProblemMelden(db *modell.Datenbank) http.Handler {
	return requireLogin(
		db, false,
		requireObjekt(db, "objekt", http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			benutzer := req.Context().Value(ctxKeyBenutzer).(*modell.Benutzer)
			obj := req.Context().Value(ctxKeyObjekt).(*modell.Objekt)

			err := req.ParseForm()
			if err != nil {
				res.WriteHeader(http.StatusBadRequest)
				return
			}
			beschreibung := req.Form.Get("beschreibung")

			problemId := rand.Int32()
			if obj.Probleme == nil {
				obj.Probleme = make(map[int32]*modell.Problem)
			}
			obj.Probleme[problemId] = &modell.Problem{
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

func handleProblemLösen(db *modell.Datenbank) http.Handler {
	return requireLogin(
		db, false,
		requireObjekt(db, "objekt", requireProblem(
			"problem",
			http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
				obj := req.Context().Value(ctxKeyObjekt).(*modell.Objekt)
				problem := req.Context().Value(ctxKeyProblem).(*modell.Problem)
				delete(obj.Probleme, problem.Id)
				http.Redirect(res, req, fmt.Sprintf("/objekt/%v/", obj.Id), http.StatusFound)
			}),
		)),
	)
}
