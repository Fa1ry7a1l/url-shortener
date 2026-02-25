package service

import (
	"crypto/rand"
	"encoding/base64"
)

type IDGenerator interface {
	NewID() (string, error)
}

// 6 bytes -> 8 chars base64url (no padding)
type RandomID struct {
	bytesPerID int
}

func NewRandomID(bytesPerID int) *RandomID {
	if bytesPerID <= 0 {
		bytesPerID = 6
	}
	return &RandomID{bytesPerID: bytesPerID}
}

func (g *RandomID) NewID() (string, error) {
	b := make([]byte, g.bytesPerID)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
