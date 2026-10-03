package users

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

func signInitData(token string, fields map[string]string) string {
	pairs := make([]string, 0, len(fields))
	for k, v := range fields {
		pairs = append(pairs, k+"="+v)
	}
	sort.Strings(pairs)
	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(token))
	mac := hmac.New(sha256.New, secret.Sum(nil))
	mac.Write([]byte(strings.Join(pairs, "\n")))

	v := url.Values{}
	for k, val := range fields {
		v.Set(k, val)
	}
	v.Set("hash", hex.EncodeToString(mac.Sum(nil)))
	return v.Encode()
}

func TestValidateTelegramInitData(t *testing.T) {
	now := time.Now()
	fields := func(age time.Duration) map[string]string {
		return map[string]string{
			"auth_date": strconv.FormatInt(now.Add(-age).Unix(), 10),
			"user":      `{"id":42,"first_name":"Abebe","last_name":"K"}`,
		}
	}

	u, err := validateTelegramInitData(signInitData("tok", fields(time.Minute)), "tok", now)
	if err != nil || u.ID != 42 || u.FirstName != "Abebe" {
		t.Fatalf("valid data rejected: %v %+v", err, u)
	}
	if _, err := validateTelegramInitData(signInitData("tok", fields(time.Minute)), "other", now); err == nil {
		t.Error("wrong bot token accepted")
	}
	if _, err := validateTelegramInitData(signInitData("tok", fields(48*time.Hour)), "tok", now); err == nil {
		t.Error("expired data accepted")
	}
	tampered := strings.Replace(signInitData("tok", fields(time.Minute)), "42", "43", 1)
	if _, err := validateTelegramInitData(tampered, "tok", now); err == nil {
		t.Error("tampered data accepted")
	}
	if _, err := validateTelegramInitData("user=x", "tok", now); err == nil {
		t.Error("missing hash accepted")
	}
}
