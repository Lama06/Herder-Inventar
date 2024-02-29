# Schemata
## Objekt
```
{
  id: int // Interne ID
  name: string // Dem Benutzer angezeigter Name
  trakt: string // Name des Traktes
  probleme: Problem[] // Vorliegende Probleme
}
```
## Problem
```
{
  ersteller: string // Benutzername des Erstellers
  datum: int // Erstellungsdatum
  beschreibung: string // Text des Erstellers
  status: ProblemStatus
}
```
## ProblemStatus
```"offen" | "gelöst" | "kenntnisnahme"```

# Authentifikation

## Anmelden
Anfrage:
```
POST /anmelden/
{
  benutzer: string
  passwort: string
}
```
Antwort:
- 200 Anmeldedaten richtig
  ```
  {
    schlüssel: string
  }
  ```
- 400 Anmeldedaten falsch

## Abmelden
Anfrage:
```
POST /abmelden/
```

# Inventar

## Ein Objekt abfragen
Anfrage: 
```
POST /objekte/lesen/
{
  id: int
}
```
Antwort: 
- 200: Objekt gefunden: `Objekt`
- 400: Falsche ID

## Mehrere Objekte Abfragen
Alle Objekte abrufen
Anfrage:
```
POST /objekte/lesen/
{
  schlüssel: string
}
```
Antwort:
- 401: Falscher Schlüssel
- 200:
  ```
  {
    objekte: Objekt[]
  }
  ```

## Objekt erstellen
```
POST /objekte/erstellen/
{
  schlüssel: string
}
```
Antwort:
- 200: Erstellt
  ```
  {
    id: int // ID des neuen Objektes
  }
  ```
- 401: Falscher Schlüssel

## Objekt löschen
```
POST /objekte/löschen/
{
  schlüssel: string
  id: int
}
```
Antwort:
- 200: Gelöscht
- 400: Falsche ID
- 401: Falscher Schlüssel
