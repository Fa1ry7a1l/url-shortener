package service

import (
	"crypto/rand"
	"encoding/base64"
	"math"
)

type IDGenerator interface {
	NewID() (string, error)
}

// RandomID — криптостойкий генератор коротких URL-safe ID.
type RandomID struct {
	length int // длина в символах (например 8)
}

// Оставляем имя NewRandomID
func NewRandomID(length int) *RandomID {
	if length <= 0 {
		length = 8
	}
	return &RandomID{length: length}
}

func (g *RandomID) NewID() (string, error) {
	// base64: 3 bytes -> 4 chars
	// нужно минимум ceil(length * 3 / 4) байт
	nBytes := int(math.Ceil(float64(g.length) * 3.0 / 4.0))
	if nBytes < 1 {
		nBytes = 1
	}

	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	s := base64.RawURLEncoding.EncodeToString(b)

	if len(s) >= g.length {
		return s[:g.length], nil
	}

	// теоретическая защита (практически не потребуется)
	for len(s) < g.length {
		extra := make([]byte, 1)
		if _, err := rand.Read(extra); err != nil {
			return "", err
		}
		s += base64.RawURLEncoding.EncodeToString(extra)
	}
	return s[:g.length], nil
}
