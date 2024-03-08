package auth

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/Lama06/Herder-Inventar/modell"
)

const SitzungDauer = 30 * time.Minute

func GenerateSchlüssel() (string, error) {
	const länge = 64
	const buchstaben = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	var schlüssel [länge]byte
	for i := range schlüssel {
		index, err := rand.Int(rand.Reader, big.NewInt(int64(len(buchstaben))))
		if err != nil {
			return "", fmt.Errorf("failed to generate random number: %w", err)
		}
		schlüssel[i] = buchstaben[index.Int64()]
	}
	return string(schlüssel[:]), nil
}

func SitzungSanduhr(db *modell.Datenbank, schlüssel string) {
	for {
		time.Sleep(SitzungDauer)
		db.Lock.Lock()
		sitzung, ok := db.Sitzungen[schlüssel]
		if !ok {
			db.Lock.Unlock()
			return
		}
		if time.Now().Sub(sitzung.LetzerZugriff) > SitzungDauer {
			delete(db.Sitzungen, schlüssel)
			db.Lock.Unlock()
			return
		}
		db.Lock.Unlock()
	}
}
