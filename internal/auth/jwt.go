package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

const (
	// CookieName is the HTTP cookie name used to store the signed auth token.
	CookieName = "auth_token"
	// DefaultTTL is used when a Manager is created without a positive TTL.
	DefaultTTL = 30 * 24 * time.Hour
)

type contextKey struct{}

// Claims contains the user identity and expiration time stored in a token.
type Claims struct {
	// UserID identifies the authenticated anonymous user.
	UserID string `json:"user_id,omitempty"`
	// Exp is the optional Unix expiration timestamp.
	Exp int64 `json:"exp,omitempty"`
}

// Manager signs, parses, and issues authentication cookies for anonymous users.
type Manager struct {
	secret []byte
	ttl    time.Duration
}

// NewManager creates a token manager with the provided signing secret and TTL.
func NewManager(secret string, ttl time.Duration) *Manager {
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	return &Manager{
		secret: []byte(secret),
		ttl:    ttl,
	}
}

// UserIDFromContext returns the authenticated user ID stored in ctx.
func UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(contextKey{}).(string)
	return userID, ok && userID != ""
}

// ContextWithUserID stores a user ID in ctx.
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, contextKey{}, userID)
}

// Middleware ensures every request has an auth cookie and a user ID in context.
func (m *Manager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := m.userIDFromRequest(r)
		if !ok {
			userID = newUserID()
			token, err := m.NewToken(userID)
			if err == nil {
				http.SetCookie(w, &http.Cookie{
					Name:     CookieName,
					Value:    token,
					Path:     "/",
					Expires:  time.Now().Add(m.ttl),
					HttpOnly: true,
					SameSite: http.SameSiteLaxMode,
				})
			}
		}

		if userID != "" {
			r = r.WithContext(ContextWithUserID(r.Context(), userID))
		}

		next.ServeHTTP(w, r)
	})
}

func (m *Manager) userIDFromRequest(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(CookieName)
	if err != nil {
		return "", false
	}

	claims, err := m.Parse(cookie.Value)
	if err != nil {
		return "", false
	}

	return claims.UserID, true
}

// NewToken signs a new token for userID.
func (m *Manager) NewToken(userID string) (string, error) {
	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}
	claims := Claims{
		UserID: userID,
		Exp:    time.Now().Add(m.ttl).Unix(),
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	signingString := encodeSegment(headerJSON) + "." + encodeSegment(claimsJSON)
	return signingString + "." + m.sign(signingString), nil
}

// Parse verifies token integrity, validates expiration, and returns claims.
func (m *Manager) Parse(token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, errors.New("invalid token")
	}

	signingString := parts[0] + "." + parts[1]
	if !hmac.Equal([]byte(parts[2]), []byte(m.sign(signingString))) {
		return Claims{}, errors.New("invalid signature")
	}

	headerJSON, err := decodeSegment(parts[0])
	if err != nil {
		return Claims{}, err
	}
	var header struct {
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return Claims{}, err
	}
	if header.Alg != "HS256" {
		return Claims{}, errors.New("invalid algorithm")
	}

	claimsJSON, err := decodeSegment(parts[1])
	if err != nil {
		return Claims{}, err
	}
	var claims Claims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return Claims{}, err
	}
	if claims.Exp != 0 && time.Now().Unix() > claims.Exp {
		return Claims{}, errors.New("token expired")
	}

	return claims, nil
}

func (m *Manager) sign(signingString string) string {
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(signingString))
	return encodeSegment(mac.Sum(nil))
}

func encodeSegment(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func decodeSegment(data string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(data)
}

func newUserID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return encodeSegment(b)
}
