package notifications

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yeabt/lewe/internal/db"
	"github.com/yeabt/lewe/internal/telegram"
)

// Sender delivers a chat message; *telegram.Bot implements it.
type Sender interface {
	SendMessage(ctx context.Context, chatID int64, text string, button *telegram.WebAppButton) error
}

// Dispatcher delivers notifications to users' Telegram chats. The
// notifications table is the outbox: rows are written in the same transaction
// as the event, and the dispatcher sends whatever has delivered_at unset.
// Rows are claimed with SKIP LOCKED, so several API instances can run it.
type Dispatcher struct {
	pool     *pgxpool.Pool
	sender   Sender
	appURL   string
	interval time.Duration
	batch    int
}

// NewDispatcher creates a dispatcher. Buttons in messages open the Mini App
// at appURL.
func NewDispatcher(pool *pgxpool.Pool, sender Sender, appURL string, interval time.Duration) *Dispatcher {
	return &Dispatcher{pool: pool, sender: sender, appURL: appURL, interval: interval, batch: 50}
}

// RunOnce delivers up to one batch of pending notifications and returns how
// many were sent. Notifications for users without Telegram are marked
// delivered without sending; so are ones for users who blocked the bot.
func (d *Dispatcher) RunOnce(ctx context.Context) (int, error) {
	sent := 0
	err := db.WithTx(ctx, d.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx,
			`UPDATE notifications n SET delivered_at = now()
			 FROM users u
			 WHERE u.id = n.user_id AND n.delivered_at IS NULL AND u.telegram_id IS NULL`); err != nil {
			return err
		}

		rows, err := tx.Query(ctx,
			`SELECT n.id, n.type, n.match_id, u.telegram_id
			 FROM notifications n JOIN users u ON u.id = n.user_id
			 WHERE n.delivered_at IS NULL
			 ORDER BY n.created_at
			 LIMIT $1
			 FOR UPDATE OF n SKIP LOCKED`, d.batch)
		if err != nil {
			return err
		}
		type pending struct {
			id, matchID pgtype.UUID
			typ         string
			chatID      int64
		}
		var batch []pending
		for rows.Next() {
			var p pending
			if err := rows.Scan(&p.id, &p.typ, &p.matchID, &p.chatID); err != nil {
				rows.Close()
				return err
			}
			batch = append(batch, p)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}

		var sendErr error
		for _, p := range batch {
			err := d.sender.SendMessage(ctx, p.chatID, "Lewe: "+messageFor(p.typ), d.button(p.matchID))
			if err != nil && !errors.Is(err, telegram.ErrBlocked) {
				// Keep it pending and retry on the next pass; commit what was sent.
				sendErr = err
				break
			}
			if _, err := tx.Exec(ctx, `UPDATE notifications SET delivered_at = now() WHERE id = $1`, p.id); err != nil {
				return err
			}
			sent++
		}
		if sendErr != nil {
			log.Printf("Telegram notification delivery paused: %v", sendErr)
		}
		return nil
	})
	return sent, err
}

func (d *Dispatcher) button(matchID pgtype.UUID) *telegram.WebAppButton {
	if d.appURL == "" {
		return nil
	}
	url := d.appURL
	text := "Open Lewe"
	if matchID.Valid {
		url += "#/matches/" + matchID.String()
		text = "Open match"
	}
	return &telegram.WebAppButton{Text: text, URL: url}
}

func messageFor(typ string) string {
	if m, ok := messages[typ]; ok {
		return m
	}
	return "You have a new notification"
}

// Run delivers notifications until ctx is cancelled.
func (d *Dispatcher) Run(ctx context.Context) {
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()
	log.Printf("Telegram notification dispatcher started (interval %s)", d.interval)
	for {
		select {
		case <-ctx.Done():
			log.Println("Telegram notification dispatcher stopped")
			return
		case <-ticker.C:
			passCtx, cancel := context.WithTimeout(ctx, time.Minute)
			if _, err := d.RunOnce(passCtx); err != nil && ctx.Err() == nil {
				log.Printf("Telegram notification pass failed: %v", err)
			}
			cancel()
		}
	}
}
