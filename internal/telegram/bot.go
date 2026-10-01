package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// ErrBlocked means the user blocked the bot or never started a chat with it,
// so messages to them will keep failing.
var ErrBlocked = errors.New("telegram: bot cannot message this user")

// Bot is a minimal Telegram Bot API client.
type Bot struct {
	token   string
	baseURL string
	client  *http.Client
}

// NewBot creates a client. baseURL defaults to https://api.telegram.org.
func NewBot(token, baseURL string) *Bot {
	if baseURL == "" {
		baseURL = "https://api.telegram.org"
	}
	return &Bot{token: token, baseURL: baseURL, client: &http.Client{Timeout: 15 * time.Second}}
}

// WebAppButton is an inline keyboard button that opens the Mini App at URL.
type WebAppButton struct {
	Text string
	URL  string
}

// SendMessage sends a text message, optionally with one button that opens
// the Mini App.
func (b *Bot) SendMessage(ctx context.Context, chatID int64, text string, button *WebAppButton) error {
	payload := map[string]any{"chat_id": chatID, "text": text}
	if button != nil {
		payload["reply_markup"] = map[string]any{
			"inline_keyboard": [][]map[string]any{{
				{"text": button.Text, "web_app": map[string]string{"url": button.URL}},
			}},
		}
	}
	return b.call(ctx, "sendMessage", payload)
}

// SetMenuButton makes the bot's chat menu button open the Mini App.
func (b *Bot) SetMenuButton(ctx context.Context, text, url string) error {
	return b.call(ctx, "setChatMenuButton", map[string]any{
		"menu_button": map[string]any{"type": "web_app", "text": text, "web_app": map[string]string{"url": url}},
	})
}

func (b *Bot) call(ctx context.Context, method string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.baseURL+"/bot"+b.token+"/"+method, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		// Strip the URL (it contains the token) from transport errors.
		var uerr interface{ Unwrap() error }
		if errors.As(err, &uerr) {
			return fmt.Errorf("telegram %s: %w", method, uerr.Unwrap())
		}
		return fmt.Errorf("telegram %s: request failed", method)
	}
	defer resp.Body.Close()

	var result struct {
		OK          bool   `json:"ok"`
		ErrorCode   int    `json:"error_code"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("telegram %s: status %d", method, resp.StatusCode)
	}
	if !result.OK {
		if result.ErrorCode == http.StatusForbidden {
			return ErrBlocked
		}
		return fmt.Errorf("telegram %s: %d %s", method, result.ErrorCode, result.Description)
	}
	return nil
}
