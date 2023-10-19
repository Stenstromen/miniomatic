package rnd

import (
	"math/rand"
	"time"
)

func RandomString(key bool, length int) string {
	var charset string
	if key {
		charset = "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	} else {
		charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	}
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}
