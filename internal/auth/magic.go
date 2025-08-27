package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type MagicLink struct {
	Secret  []byte
	BaseURL string
}

var (
	ErrBadToken   = errors.New("bad token")
	ErrBadSig     = errors.New("invalid signature")
	ErrExpired    = errors.New("expired")
	ErrBadPayload = errors.New("bad payload")
)

// Sign: use URL-safe base64 WITH padding (clearer in URLs)
func (m MagicLink) Sign(email string, exp time.Time) string {
	msg := email + "|" + strconv.FormatInt(exp.Unix(), 10)
	mac := hmac.New(sha256.New, m.Secret)
	mac.Write([]byte(msg))
	sig := base64.URLEncoding.EncodeToString(mac.Sum(nil))    // padded
	payload := base64.URLEncoding.EncodeToString([]byte(msg)) // padded
	return payload + "." + sig
}

// decodeURLB64 tries raw (no padding) then padded
func decodeURLB64(s string) ([]byte, error) {
	if b, err := base64.RawURLEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	return base64.URLEncoding.DecodeString(s)
}

func (m MagicLink) Verify(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return "", ErrBadToken
	}
	payload, sig := parts[0], parts[1]

	raw, err := decodeURLB64(payload)
	if err != nil {
		return "", ErrBadToken
	}

	mac := hmac.New(sha256.New, m.Secret)
	mac.Write(raw)
	expectedRaw := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	expectedPad := base64.URLEncoding.EncodeToString(mac.Sum(nil))

	if sig != expectedRaw && sig != expectedPad {
		return "", ErrBadSig
	}

	fields := strings.SplitN(string(raw), "|", 2)
	if len(fields) != 2 {
		return "", ErrBadPayload
	}
	email := strings.TrimSpace(fields[0])
	ts, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil || email == "" {
		return "", ErrBadPayload
	}
	if time.Now().After(time.Unix(ts, 0)) {
		return "", ErrExpired
	}
	return email, nil
}

func (m MagicLink) URL(email string, ttl time.Duration) string {
	exp := time.Now().Add(ttl)
	tok := m.Sign(email, exp)
	u, _ := url.Parse(m.BaseURL)
	u.Path = "/auth/callback"
	q := u.Query()
	q.Set("token", tok)
	u.RawQuery = q.Encode()
	return u.String()
}
