// Package telegram integrates Lewe with Telegram: it verifies Mini App launch
// data so users can sign in with their Telegram account, and talks to the Bot
// API to deliver notifications as chat messages.
package telegram

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalidInitData = errors.New("invalid Telegram init data")
	ErrExpiredInitData = errors.New("Telegram init data has expired")
)

// User is the Telegram account a Mini App was opened by.
type User struct {
	ID              int64  `json:"id"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	Username        string `json:"username"`
	PhotoURL        string `json:"photo_url"`
	LanguageCode    string `json:"language_code"`
	AllowsWriteToPM bool   `json:"allows_write_to_pm"`
}

// FullName joins the first and last name.
func (u User) FullName() string {
	return strings.TrimSpace(u.FirstName + " " + u.LastName)
}

// InitData is the verified launch data of a Mini App session.
type InitData struct {
	User       User
	AuthDate   time.Time
	StartParam string
}

// ParseInitData verifies Telegram.WebApp.initData against the bot token, as
// described in https://core.telegram.org/bots/webapps#validating-data-received-via-the-mini-app,
// and rejects data older than maxAge.
//
// The check: secret = HMAC-SHA256(key "WebAppData", bot token); the "hash"
// field must equal hex(HMAC-SHA256(key secret, data-check-string)), where the
// data-check-string is every other field as "key=value", sorted by key and
// joined with newlines.
func ParseInitData(initData, botToken string, maxAge time.Duration, now time.Time) (*InitData, error) {
	values, err := url.ParseQuery(initData)
	if err != nil || botToken == "" {
		return nil, ErrInvalidInitData
	}
	hash := values.Get("hash")
	if hash == "" {
		return nil, ErrInvalidInitData
	}

	expected := Sign(values, botToken)
	if !hmac.Equal([]byte(hash), []byte(expected)) {
		return nil, ErrInvalidInitData
	}

	authUnix, err := strconv.ParseInt(values.Get("auth_date"), 10, 64)
	if err != nil {
		return nil, ErrInvalidInitData
	}
	authDate := time.Unix(authUnix, 0)
	if maxAge > 0 && now.Sub(authDate) > maxAge {
		return nil, ErrExpiredInitData
	}

	var user User
	if err := json.Unmarshal([]byte(values.Get("user")), &user); err != nil || user.ID == 0 {
		return nil, ErrInvalidInitData
	}

	return &InitData{User: user, AuthDate: authDate, StartParam: values.Get("start_param")}, nil
}

// Sign computes the hash Telegram puts in init data. It ignores any existing
// "hash" field. Exported so tests (and local tools) can produce valid data.
func Sign(values url.Values, botToken string) string {
	keys := make([]string, 0, len(values))
	for k := range values {
		if k != "hash" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	lines := make([]string, len(keys))
	for i, k := range keys {
		lines[i] = k + "=" + values.Get(k)
	}

	secret := hmacSHA256([]byte("WebAppData"), []byte(botToken))
	return hex.EncodeToString(hmacSHA256(secret, []byte(strings.Join(lines, "\n"))))
}

func hmacSHA256(key, msg []byte) []byte {
	m := hmac.New(sha256.New, key)
	m.Write(msg)
	return m.Sum(nil)
}
