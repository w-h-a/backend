package store

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
)

const (
	charset  = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	idLength = 12
)

var (
	GenerateId = func() string {
		bs := make([]byte, idLength)
		if _, err := rand.Read(bs); err != nil {
			panic("failed to generate random id: " + err.Error())
		}
		for i := range bs {
			bs[i] = charset[int(bs[i])%len(charset)]
		}
		return string(bs)
	}

	HashPassword = func(password string, salt string) string {
		sum := sha256.Sum256([]byte(salt + password))
		return base32.StdEncoding.EncodeToString(sum[:])
	}
)
