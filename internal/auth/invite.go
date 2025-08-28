package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type InviteLink struct {
	Secret  []byte
	BaseURL string // eg., http://localhost:8080
}

func (i InviteLink) Sign(coachID, athleteID string, exp time.Time) string {
	msg := strings.Join([]string{coachID, athleteID, strconv.FormatInt(exp.Unix(), 10)}, "|")
	mac := hmac.New(sha256.New, i.Secret)
	mac.Write([]byte(msg))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	payload := base64.RawURLEncoding.EncodeToString([]byte(msg))
	return payload + "." + sig
}

func (i InviteLink) Verify(token string) (coachID, athleteID string, err error) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		err = ErrBadToken
		return
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		err = ErrBadToken
		return
	}

	mac := hmac.New(sha256.New, i.Secret)
	mac.Write(payload)

	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		err = ErrBadSig
		return
	}

	fields := strings.SplitN(string(payload), "|", 3)
	if len(fields) != 3 {
		err = ErrBadPayload
		return
	}

	coachID = fields[0]
	athleteID = fields[1]
	expUnix, err := strconv.ParseInt(fields[2], 10, 64)
	if err != nil {
		err = ErrBadPayload
		return
	}

	if time.Now().After(time.Unix(expUnix, 0)) {
		err = ErrExpired
		return
	}

	return
}

func (i InviteLink) URL(coachID, athleteID string, ttl time.Duration) string {
	exp := time.Now().Add(ttl)
	tok := i.Sign(coachID, athleteID, exp)
	u, _ := url.Parse(i.BaseURL)
	u.Path = "/invite"
	q := u.Query()
	q.Set("token", tok)
	u.RawQuery = q.Encode()
	return u.String()
}
