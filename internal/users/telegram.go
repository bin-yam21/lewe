package users

import (
	"context"
	"errors"
	"time"

	"github.com/yeabt/lewe/internal/telegram"
)

// telegramInitDataMaxAge bounds how old Mini App launch data may be. Telegram
// re-signs it each time the app is opened, so a day is generous.
const telegramInitDataMaxAge = 24 * time.Hour

var ErrTelegramDisabled = errors.New("telegram sign-in is not configured")

// TelegramLogin signs in (or signs up) the Telegram user who opened the Mini
// App, after verifying that Telegram signed the launch data.
func (s *Service) TelegramLogin(ctx context.Context, initData string) (*AuthResponse, error) {
	if s.botToken == "" {
		return nil, ErrTelegramDisabled
	}
	data, err := telegram.ParseInitData(initData, s.botToken, telegramInitDataMaxAge, time.Now())
	if err != nil {
		return nil, err
	}

	tg := data.User
	name := tg.FullName()
	if name == "" {
		name = tg.Username
	}
	if name == "" {
		name = "Telegram user"
	}
	if len(name) > 100 {
		name = name[:100]
	}

	user, err := s.repo.UpsertTelegram(ctx, tg.ID, emptyToNil(tg.Username), &name, emptyToNil(tg.PhotoURL))
	if err != nil {
		return nil, err
	}
	return s.generateAuthResponse(ctx, user)
}

func emptyToNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
