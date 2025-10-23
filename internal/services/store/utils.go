package store

import "crypto/rand"

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
)
