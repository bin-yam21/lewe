package users

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// telegramAuthMaxAge is how long signed initData stays valid.
const telegramAuthMaxAge = 24 * time.Hour

var (
	ErrTelegramDisabled    = errors.New("telegram login is not configured")
	ErrInvalidTelegramData = errors.New("invalid or expired telegram login data")
)

// TelegramRequest carries the raw Telegram.WebApp.initData string.
type TelegramRequest struct {
	InitData string `json:"init_data"`
}

type telegramUser struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	PhotoURL  string `json:"photo_url"`
}

// validateTelegramInitData checks the HMAC signature Telegram attaches to
// Mini App launch data and returns the user it describes.
// See https://core.telegram.org/bots/webapps#validating-data-received-via-the-mini-app
func validateTelegramInitData(initData, botToken string, now time.Time) (*telegramUser, error) {
	values, err := url.ParseQuery(initData)
	if err != nil {
		return nil, ErrInvalidTelegramData
	}
	hash := values.Get("hash")
	if hash == "" {
		return nil, ErrInvalidTelegramData
	}
	values.Del("hash")

	pairs := make([]string, 0, len(values))
	for k, v := range values {
		pairs = append(pairs, k+"="+v[0])
	}
	sort.Strings(pairs)

	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(botToken))
	mac := hmac.New(sha256.New, secret.Sum(nil))
	mac.Write([]byte(strings.Join(pairs, "\n")))
	got, err := hex.DecodeString(hash)
	if err != nil || !hmac.Equal(got, mac.Sum(nil)) {
		return nil, ErrInvalidTelegramData
	}

	authDate, err := strconv.ParseInt(values.Get("auth_date"), 10, 64)
	if err != nil || now.Sub(time.Unix(authDate, 0)) > telegramAuthMaxAge {
		return nil, ErrInvalidTelegramData
	}

	var u telegramUser
	if err := json.Unmarshal([]byte(values.Get("user")), &u); err != nil || u.ID == 0 {
		return nil, ErrInvalidTelegramData
	}
	return &u, nil
}

// TelegramLogin signs in (or creates) the account linked to a Telegram user.
func (s *Service) TelegramLogin(ctx context.Context, initData string) (*AuthResponse, error) {
	if s.telegramBotToken == "" {
		return nil, ErrTelegramDisabled
	}
	tu, err := validateTelegramInitData(initData, s.telegramBotToken, time.Now())
	if err != nil {
		return nil, err
	}

	user, err := s.repo.GetByTelegramID(ctx, tu.ID)
	if errors.Is(err, ErrUserNotFound) {
		user, err = s.createTelegramUser(ctx, tu)
	}
	if err != nil {
		return nil, err
	}
	return s.generateAuthResponse(ctx, user)
}

func (s *Service) createTelegramUser(ctx context.Context, tu *telegramUser) (*UserRow, error) {
	name := strings.TrimSpace(tu.FirstName + " " + tu.LastName)
	if name == "" {
		name = tu.Username
	}
	if name == "" {
		name = "Telegram user"
	}
	if len(name) > 100 {
		name = name[:100]
	}

	// Telegram accounts have no usable password; store a hash of random bytes.
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(hex.EncodeToString(random)), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	email := fmt.Sprintf("tg%d@telegram.invalid", tu.ID)
	return s.repo.CreateTelegram(ctx, tu.ID, email, string(hash), name)
}
