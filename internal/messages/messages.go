// Package messages is a simple chat between the two users of a match, so they
// can ask questions about the items and arrange the exchange.
package messages

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yeabt/lewe/internal/db"
	"github.com/yeabt/lewe/internal/middleware"
	"github.com/yeabt/lewe/internal/notifications"
	"github.com/yeabt/lewe/internal/request"
	"github.com/yeabt/lewe/internal/response"
	"github.com/yeabt/lewe/internal/validator"
)

const maxBodyLength = 2000

var (
	ErrMatchNotFound = errors.New("match not found")
	ErrMatchClosed   = errors.New("match is closed")
)

// MessageRequest is the payload for sending a message.
type MessageRequest struct {
	Body string `json:"body"`
}

// MessageResponse is a chat message within a match.
type MessageResponse struct {
	ID        string    `json:"id"`
	SenderID  string    `json:"sender_id"`
	FromYou   bool      `json:"from_you"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

// Service contains messaging business logic.
type Service struct {
	pool *pgxpool.Pool
}

// NewService creates a new message service.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// participants returns the other user in the match and the match status,
// or ErrMatchNotFound if userID is not a participant.
func participants(ctx context.Context, q db.DBTX, matchID, userID pgtype.UUID) (other pgtype.UUID, status string, err error) {
	var a, b pgtype.UUID
	err = q.QueryRow(ctx, `SELECT user_a_id, user_b_id, status FROM matches WHERE id = $1`, matchID).
		Scan(&a, &b, &status)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return other, "", ErrMatchNotFound
	case err != nil:
		return other, "", err
	case userID == a:
		return b, status, nil
	case userID == b:
		return a, status, nil
	default:
		return other, "", ErrMatchNotFound
	}
}

// Send posts a message to a match and notifies the other user. Messages can
// be sent while the match is pending, accepted or completed, but not after it
// was declined or cancelled.
func (s *Service) Send(ctx context.Context, matchID, userID pgtype.UUID, req MessageRequest) (*MessageResponse, error) {
	body := strings.TrimSpace(req.Body)
	errs := make(validator.Errors)
	validator.ValidateRequired(errs, "body", body)
	validator.ValidateMaxLength(errs, "body", body, maxBodyLength)
	if errs.HasErrors() {
		return nil, errs
	}

	var msg *MessageResponse
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		other, status, err := participants(ctx, tx, matchID, userID)
		if err != nil {
			return err
		}
		if status == "declined" || status == "cancelled" {
			return ErrMatchClosed
		}

		var (
			id        pgtype.UUID
			createdAt pgtype.Timestamptz
		)
		if err := tx.QueryRow(ctx,
			`INSERT INTO messages (match_id, sender_id, body) VALUES ($1, $2, $3)
			 RETURNING id, created_at`, matchID, userID, body).Scan(&id, &createdAt); err != nil {
			return err
		}
		msg = &MessageResponse{
			ID:        id.String(),
			SenderID:  userID.String(),
			FromYou:   true,
			Body:      body,
			CreatedAt: createdAt.Time,
		}
		return notifications.Create(ctx, tx, other, notifications.TypeMessage, matchID)
	})
	if err != nil {
		return nil, err
	}
	return msg, nil
}

// List returns a match's messages, oldest first. Closed matches keep their
// history readable.
func (s *Service) List(ctx context.Context, matchID, userID pgtype.UUID, page request.Page) ([]MessageResponse, error) {
	if _, _, err := participants(ctx, s.pool, matchID, userID); err != nil {
		return nil, err
	}

	rows, err := s.pool.Query(ctx,
		`SELECT id, sender_id, body, created_at FROM messages
		 WHERE match_id = $1
		 ORDER BY created_at, id
		 LIMIT $2 OFFSET $3`, matchID, page.Limit, page.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []MessageResponse{}
	for rows.Next() {
		var (
			id, sender pgtype.UUID
			m          MessageResponse
			createdAt  pgtype.Timestamptz
		)
		if err := rows.Scan(&id, &sender, &m.Body, &createdAt); err != nil {
			return nil, err
		}
		m.ID, m.SenderID = id.String(), sender.String()
		m.FromYou = sender == userID
		m.CreatedAt = createdAt.Time
		out = append(out, m)
	}
	return out, rows.Err()
}

// --- handler ---

// Handler handles HTTP requests for match messages. All routes require auth.
type Handler struct {
	svc *Service
}

// NewHandler creates a new message handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// List handles GET /api/v1/matches/{id}/messages
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	matchID, err := request.PathUUID(r, "id")
	if err != nil {
		h.handleError(w, ErrMatchNotFound)
		return
	}

	page := request.Pagination(r)
	list, err := h.svc.List(r.Context(), matchID, userID, page)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, response.List[MessageResponse]{Data: list, Limit: page.Limit, Offset: page.Offset})
}

// Send handles POST /api/v1/matches/{id}/messages
func (h *Handler) Send(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	matchID, err := request.PathUUID(r, "id")
	if err != nil {
		h.handleError(w, ErrMatchNotFound)
		return
	}

	var req MessageRequest
	if err := request.DecodeJSON(w, r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	msg, err := h.svc.Send(r.Context(), matchID, userID, req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, msg)
}

// handleError maps domain errors to HTTP status codes.
func (h *Handler) handleError(w http.ResponseWriter, err error) {
	var validationErrs validator.Errors
	if errors.As(err, &validationErrs) {
		response.ValidationError(w, validationErrs)
		return
	}

	switch {
	case errors.Is(err, ErrMatchNotFound):
		response.Error(w, http.StatusNotFound, "Match not found")
	case errors.Is(err, ErrMatchClosed):
		response.Error(w, http.StatusConflict, "This match is closed; messages can no longer be sent")
	default:
		response.InternalError(w, err)
	}
}
