// Package notifications stores in-app notifications about match activity and
// exposes them to the user they are addressed to.
package notifications

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yeabt/lewe/internal/db"
	"github.com/yeabt/lewe/internal/middleware"
	"github.com/yeabt/lewe/internal/request"
	"github.com/yeabt/lewe/internal/response"
)

// Notification types.
const (
	TypeMatchFound      = "match_found"      // the worker found a new swap for you
	TypeMatchAccepted   = "match_accepted"   // the other user accepted; your turn
	TypeMatchConfirmed  = "match_confirmed"  // both accepted; items are reserved
	TypeMatchDeclined   = "match_declined"   // the other user declined
	TypeMatchCancelled  = "match_cancelled"  // the match was called off
	TypeExchangeUpdated = "exchange_updated" // the other user set the exchange method
	TypeMatchCompleted  = "match_completed"  // both confirmed the hand-over
	TypeMessage         = "message_received" // new chat message in a match
	TypeRatingReceived  = "rating_received"  // the other user rated you
)

var messages = map[string]string{
	TypeMatchFound:      "We found a new swap for you",
	TypeMatchAccepted:   "The other user accepted your match",
	TypeMatchConfirmed:  "Your match is confirmed — agree how to exchange",
	TypeMatchDeclined:   "The other user declined the match",
	TypeMatchCancelled:  "A match was cancelled",
	TypeExchangeUpdated: "The exchange plan for your match changed",
	TypeMatchCompleted:  "Your swap is complete — rate the other user",
	TypeMessage:         "You have a new message",
	TypeRatingReceived:  "You received a new rating",
}

var ErrNotificationNotFound = errors.New("notification not found")

// Create adds a notification for userID about a match. It takes a DBTX so it
// can run inside the transaction that caused the event.
func Create(ctx context.Context, q db.DBTX, userID pgtype.UUID, typ string, matchID pgtype.UUID) error {
	_, err := q.Exec(ctx,
		`INSERT INTO notifications (user_id, type, match_id) VALUES ($1, $2, $3)`,
		userID, typ, matchID)
	return err
}

// CancelPendingMatches cancels the pending matches selected by where (a SQL
// condition on the matches table using $1…$n for args) and notifies both
// users of each one.
func CancelPendingMatches(ctx context.Context, q db.DBTX, where string, args ...any) error {
	_, err := q.Exec(ctx,
		`WITH cancelled AS (
		     UPDATE matches SET status = 'cancelled', updated_at = now()
		     WHERE status = 'pending' AND (`+where+`)
		     RETURNING id, user_a_id, user_b_id
		 )
		 INSERT INTO notifications (user_id, type, match_id)
		 SELECT user_a_id, '`+TypeMatchCancelled+`', id FROM cancelled
		 UNION ALL
		 SELECT user_b_id, '`+TypeMatchCancelled+`', id FROM cancelled`,
		args...)
	return err
}

// NotificationResponse is a notification as returned to its recipient.
type NotificationResponse struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	MatchID   *string   `json:"match_id,omitempty"`
	Read      bool      `json:"read"`
	CreatedAt time.Time `json:"created_at"`
}

// --- repository + service ---

// Service reads and updates a user's notifications.
type Service struct {
	pool *pgxpool.Pool
}

// NewService creates a new notification service.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// List returns the user's notifications, newest first.
func (s *Service) List(ctx context.Context, userID pgtype.UUID, unreadOnly bool, page request.Page) ([]NotificationResponse, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, type, match_id, read_at, created_at FROM notifications
		 WHERE user_id = $1 AND (NOT $2 OR read_at IS NULL)
		 ORDER BY created_at DESC, id
		 LIMIT $3 OFFSET $4`,
		userID, unreadOnly, page.Limit, page.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []NotificationResponse{}
	for rows.Next() {
		var (
			id, matchID pgtype.UUID
			typ         string
			readAt      pgtype.Timestamptz
			createdAt   pgtype.Timestamptz
		)
		if err := rows.Scan(&id, &typ, &matchID, &readAt, &createdAt); err != nil {
			return nil, err
		}
		n := NotificationResponse{
			ID:        id.String(),
			Type:      typ,
			Message:   messages[typ],
			Read:      readAt.Valid,
			CreatedAt: createdAt.Time,
		}
		if matchID.Valid {
			m := matchID.String()
			n.MatchID = &m
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// UnreadCount returns how many unread notifications the user has.
func (s *Service) UnreadCount(ctx context.Context, userID pgtype.UUID) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&n)
	return n, err
}

// MarkRead marks one of the user's notifications as read.
func (s *Service) MarkRead(ctx context.Context, id, userID pgtype.UUID) error {
	var exists bool
	err := s.pool.QueryRow(ctx,
		`WITH upd AS (
		     UPDATE notifications SET read_at = COALESCE(read_at, now())
		     WHERE id = $1 AND user_id = $2
		     RETURNING 1
		 )
		 SELECT EXISTS (SELECT 1 FROM upd)`, id, userID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return ErrNotificationNotFound
	}
	return nil
}

// MarkAllRead marks all of the user's notifications as read.
func (s *Service) MarkAllRead(ctx context.Context, userID pgtype.UUID) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE notifications SET read_at = now() WHERE user_id = $1 AND read_at IS NULL`, userID)
	return err
}

// --- handler ---

// Handler handles HTTP requests for notifications. All routes require auth.
type Handler struct {
	svc *Service
}

// NewHandler creates a new notification handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// List handles GET /api/v1/notifications?unread=true
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	page := request.Pagination(r)
	unread := r.URL.Query().Get("unread") == "true"

	list, err := h.svc.List(r.Context(), userID, unread, page)
	if err != nil {
		response.InternalError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, response.List[NotificationResponse]{Data: list, Limit: page.Limit, Offset: page.Offset})
}

// UnreadCount handles GET /api/v1/notifications/unread-count
func (h *Handler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	n, err := h.svc.UnreadCount(r.Context(), userID)
	if err != nil {
		response.InternalError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]int{"unread": n})
}

// MarkRead handles POST /api/v1/notifications/{id}/read
func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	id, err := request.PathUUID(r, "id")
	if err == nil {
		err = h.svc.MarkRead(r.Context(), id, userID)
	} else {
		err = ErrNotificationNotFound
	}
	switch {
	case errors.Is(err, ErrNotificationNotFound):
		response.Error(w, http.StatusNotFound, "Notification not found")
	case err != nil:
		response.InternalError(w, err)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

// MarkAllRead handles POST /api/v1/notifications/read-all
func (h *Handler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	if err := h.svc.MarkAllRead(r.Context(), userID); err != nil {
		response.InternalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
