package telegram

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

const token = "123456:TEST-token"

func initData(t *testing.T, authDate time.Time, user string) string {
	t.Helper()
	v := url.Values{}
	v.Set("query_id", "AAH")
	v.Set("user", user)
	v.Set("auth_date", strconv.FormatInt(authDate.Unix(), 10))
	v.Set("hash", Sign(v, token))
	return v.Encode()
}

func TestParseInitData(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	user := `{"id":4242,"first_name":"Selam","last_name":"Tadesse","username":"selamt"}`
	data := initData(t, now.Add(-time.Minute), user)

	got, err := ParseInitData(data, token, time.Hour, now)
	if err != nil {
		t.Fatal(err)
	}
	if got.User.ID != 4242 || got.User.FullName() != "Selam Tadesse" || got.User.Username != "selamt" {
		t.Errorf("user = %+v", got.User)
	}

	if _, err := ParseInitData(data, "other:token", time.Hour, now); err != ErrInvalidInitData {
		t.Errorf("wrong token: err = %v", err)
	}
	tampered := strings.Replace(data, "4242", "4243", 1)
	if _, err := ParseInitData(tampered, token, time.Hour, now); err != ErrInvalidInitData {
		t.Errorf("tampered: err = %v", err)
	}
	old := initData(t, now.Add(-2*time.Hour), user)
	if _, err := ParseInitData(old, token, time.Hour, now); err != ErrExpiredInitData {
		t.Errorf("expired: err = %v", err)
	}
	for _, bad := range []string{"", "hash=abc", "%zz"} {
		if _, err := ParseInitData(bad, token, time.Hour, now); err != ErrInvalidInitData {
			t.Errorf("%q: err = %v", bad, err)
		}
	}
	noUser := initData(t, now, `{}`)
	if _, err := ParseInitData(noUser, token, time.Hour, now); err != ErrInvalidInitData {
		t.Errorf("no user: err = %v", err)
	}
}

func TestBot(t *testing.T) {
	var got map[string]any
	var path string
	blocked := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		json.NewDecoder(r.Body).Decode(&got)
		if blocked {
			w.Write([]byte(`{"ok":false,"error_code":403,"description":"Forbidden: bot was blocked by the user"}`))
			return
		}
		w.Write([]byte(`{"ok":true,"result":{}}`))
	}))
	defer srv.Close()

	bot := NewBot(token, srv.URL)
	err := bot.SendMessage(context.Background(), 4242, "hello", &WebAppButton{Text: "Open", URL: "https://app.test/#/m/1"})
	if err != nil {
		t.Fatal(err)
	}
	if path != "/bot"+token+"/sendMessage" || got["chat_id"].(float64) != 4242 || got["text"] != "hello" {
		t.Errorf("path %s body %v", path, got)
	}
	button := got["reply_markup"].(map[string]any)["inline_keyboard"].([]any)[0].([]any)[0].(map[string]any)
	if button["web_app"].(map[string]any)["url"] != "https://app.test/#/m/1" {
		t.Errorf("button = %v", button)
	}

	blocked = true
	if err := bot.SendMessage(context.Background(), 1, "x", nil); err != ErrBlocked {
		t.Errorf("blocked: err = %v", err)
	}
}
