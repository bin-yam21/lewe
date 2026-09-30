package matches

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/yeabt/lewe/internal/catalog"
	"github.com/yeabt/lewe/internal/db"
	"github.com/yeabt/lewe/internal/notifications"
	"github.com/yeabt/lewe/internal/request"
	"github.com/yeabt/lewe/internal/validator"
)

var (
	ErrInvalidState    = errors.New("action not allowed in the match's current status")
	ErrItemUnavailable = errors.New("one of the items is no longer available")
	ErrExchangeNotSet  = errors.New("an exchange method must be chosen first")
)

// Service implements the match lifecycle:
//
//	pending ──both accept──▶ accepted ──exchange chosen, both complete──▶ completed
//	   │                        │
//	   └──decline──▶ declined   └──cancel──▶ cancelled
//
// Accepting reserves both items and cancels competing pending matches;
// cancelling releases them; completing marks them exchanged.
type Service struct {
	repo *Repository
}

// NewService creates a new match service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// List returns the user's matches.
func (s *Service) List(ctx context.Context, userID pgtype.UUID, status string, page request.Page) ([]MatchResponse, error) {
	switch status {
	case "", StatusPending, StatusAccepted, StatusCompleted, StatusDeclined, StatusCancelled:
	default:
		return nil, validator.Errors{"status": "must be one of: pending, accepted, completed, declined, cancelled"}
	}
	rows, err := s.repo.ListForUser(ctx, userID, status, page)
	if err != nil {
		return nil, err
	}
	out := make([]MatchResponse, 0, len(rows))
	for _, d := range rows {
		out = append(out, toResponse(d, userID))
	}
	return out, nil
}

// Get returns a single match the user participates in.
func (s *Service) Get(ctx context.Context, id, userID pgtype.UUID) (*MatchResponse, error) {
	d, err := s.repo.GetDetailForUser(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	resp := toResponse(d, userID)
	return &resp, nil
}

// Accept records the user's acceptance. When both users have accepted, the
// items are reserved and the match becomes accepted.
func (s *Service) Accept(ctx context.Context, id, userID pgtype.UUID) (*MatchResponse, error) {
	return s.mutate(ctx, id, userID, func(tx pgx.Tx, m *Row) (pgtype.UUID, error) {
		if m.Status != StatusPending {
			return pgtype.UUID{}, ErrInvalidState
		}

		now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
		alreadyAccepted := m.AAcceptedAt.Valid
		if m.isUserA(userID) {
			m.AAcceptedAt = firstTime(m.AAcceptedAt, now)
		} else {
			alreadyAccepted = m.BAcceptedAt.Valid
			m.BAcceptedAt = firstTime(m.BAcceptedAt, now)
		}
		if alreadyAccepted {
			return pgtype.UUID{}, nil
		}

		if !(m.AAcceptedAt.Valid && m.BAcceptedAt.Valid) {
			return pgtype.UUID{}, notify(ctx, tx, m, userID, notifications.TypeMatchAccepted)
		}
		// Both have accepted: reserve the items. If either is gone the swap
		// can no longer happen, so close it and tell the caller.
		ok, err := lockAvailableItems(ctx, tx, m.ItemAID, m.ItemBID)
		if err != nil {
			return pgtype.UUID{}, err
		}
		if !ok {
			m.Status = StatusCancelled
			if err := notify(ctx, tx, m, userID, notifications.TypeMatchCancelled); err != nil {
				return pgtype.UUID{}, err
			}
			return pgtype.UUID{}, errCommitThen(ErrItemUnavailable)
		}
		if err := setItemsStatus(ctx, tx, m.ItemAID, m.ItemBID, "available", "reserved"); err != nil {
			return pgtype.UUID{}, err
		}
		if err := cancelOtherPendingForItems(ctx, tx, m.ID, m.ItemAID, m.ItemBID); err != nil {
			return pgtype.UUID{}, err
		}
		m.Status = StatusAccepted
		return pgtype.UUID{}, notify(ctx, tx, m, userID, notifications.TypeMatchConfirmed)
	})
}

// Decline rejects a pending match.
func (s *Service) Decline(ctx context.Context, id, userID pgtype.UUID) (*MatchResponse, error) {
	return s.mutate(ctx, id, userID, func(tx pgx.Tx, m *Row) (pgtype.UUID, error) {
		if m.Status != StatusPending {
			return pgtype.UUID{}, ErrInvalidState
		}
		m.Status = StatusDeclined
		return userID, notify(ctx, tx, m, userID, notifications.TypeMatchDeclined)
	})
}

// Cancel backs out of an accepted match and releases both items.
func (s *Service) Cancel(ctx context.Context, id, userID pgtype.UUID) (*MatchResponse, error) {
	return s.mutate(ctx, id, userID, func(tx pgx.Tx, m *Row) (pgtype.UUID, error) {
		if m.Status != StatusAccepted {
			return pgtype.UUID{}, ErrInvalidState
		}
		if err := setItemsStatus(ctx, tx, m.ItemAID, m.ItemBID, "reserved", "available"); err != nil {
			return pgtype.UUID{}, err
		}
		m.Status = StatusCancelled
		return userID, notify(ctx, tx, m, userID, notifications.TypeMatchCancelled)
	})
}

// SetExchange chooses how the items will be exchanged. Changing the method
// resets any completion confirmations, since they applied to the old plan.
func (s *Service) SetExchange(ctx context.Context, id, userID pgtype.UUID, req ExchangeRequest) (*MatchResponse, error) {
	req.Method = strings.ToLower(strings.TrimSpace(req.Method))
	if req.Details != nil {
		d := strings.TrimSpace(*req.Details)
		req.Details = &d
		if d == "" {
			req.Details = nil
		}
	}
	errs := make(validator.Errors)
	validator.ValidateOneOf(errs, "method", req.Method, catalog.ExchangeMethods)
	if req.Details != nil {
		validator.ValidateMaxLength(errs, "details", *req.Details, 1000)
	}
	if errs.HasErrors() {
		return nil, errs
	}

	return s.mutate(ctx, id, userID, func(tx pgx.Tx, m *Row) (pgtype.UUID, error) {
		if m.Status != StatusAccepted {
			return pgtype.UUID{}, ErrInvalidState
		}
		details := pgtype.Text{}
		if req.Details != nil {
			details = pgtype.Text{String: *req.Details, Valid: true}
		}
		if m.ExchangeMethod.String != req.Method || m.ExchangeDetails != details {
			m.ACompletedAt = pgtype.Timestamptz{}
			m.BCompletedAt = pgtype.Timestamptz{}
		}
		m.ExchangeMethod = pgtype.Text{String: req.Method, Valid: true}
		m.ExchangeDetails = details
		m.ExchangeProposedBy = userID
		return pgtype.UUID{}, notify(ctx, tx, m, userID, notifications.TypeExchangeUpdated)
	})
}

// Complete confirms the user has handed over their item. When both users
// confirm, the match is completed, the items are marked exchanged and the
// wants behind the match are fulfilled.
func (s *Service) Complete(ctx context.Context, id, userID pgtype.UUID) (*MatchResponse, error) {
	return s.mutate(ctx, id, userID, func(tx pgx.Tx, m *Row) (pgtype.UUID, error) {
		if m.Status != StatusAccepted {
			return pgtype.UUID{}, ErrInvalidState
		}
		if !m.ExchangeMethod.Valid {
			return pgtype.UUID{}, ErrExchangeNotSet
		}

		now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
		if m.isUserA(userID) {
			m.ACompletedAt = firstTime(m.ACompletedAt, now)
		} else {
			m.BCompletedAt = firstTime(m.BCompletedAt, now)
		}

		if m.ACompletedAt.Valid && m.BCompletedAt.Valid {
			if err := setItemsStatus(ctx, tx, m.ItemAID, m.ItemBID, "reserved", "exchanged"); err != nil {
				return pgtype.UUID{}, err
			}
			if err := fulfillWants(ctx, tx, m.ID, m.WantAID, m.WantBID); err != nil {
				return pgtype.UUID{}, err
			}
			m.Status = StatusCompleted
			m.CompletedAt = now
			return pgtype.UUID{}, notify(ctx, tx, m, userID, notifications.TypeMatchCompleted)
		}
		return pgtype.UUID{}, nil
	})
}

// firstTime keeps an existing timestamp, or uses now if it is unset.
func firstTime(existing, now pgtype.Timestamptz) pgtype.Timestamptz {
	if existing.Valid {
		return existing
	}
	return now
}

// notify tells the other participant of m (not actor) about an event.
func notify(ctx context.Context, tx pgx.Tx, m *Row, actor pgtype.UUID, typ string) error {
	other := m.UserAID
	if m.isUserA(actor) {
		other = m.UserBID
	}
	return notifications.Create(ctx, tx, other, typ, m.ID)
}

// commitThenError signals that the transaction's changes should be committed
// even though the operation reports an error to the caller.
type commitThenError struct{ err error }

func (e commitThenError) Error() string { return e.err.Error() }
func (e commitThenError) Unwrap() error { return e.err }

func errCommitThen(err error) error { return commitThenError{err: err} }

// mutate locks the match, verifies the user participates in it, applies fn,
// persists the result and returns the refreshed view. fn returns the user to
// record as having closed the match (if any).
func (s *Service) mutate(ctx context.Context, id, userID pgtype.UUID, fn func(tx pgx.Tx, m *Row) (pgtype.UUID, error)) (*MatchResponse, error) {
	var deferred error
	err := db.WithTx(ctx, s.repo.Pool(), func(tx pgx.Tx) error {
		m, err := lockForUpdate(ctx, tx, id)
		if err != nil {
			return err
		}
		if !m.hasParticipant(userID) {
			return ErrMatchNotFound
		}

		closedBy, err := fn(tx, m)
		var cte commitThenError
		if errors.As(err, &cte) {
			deferred = cte.err
		} else if err != nil {
			return err
		}
		return save(ctx, tx, m, closedBy)
	})
	if err != nil {
		return nil, err
	}
	if deferred != nil {
		return nil, deferred
	}
	return s.Get(ctx, id, userID)
}

// toResponse renders a match from the viewer's point of view.
func toResponse(d *DetailRow, viewer pgtype.UUID) MatchResponse {
	itemA := summary(d.ItemAID, d.ItemA)
	itemB := summary(d.ItemBID, d.ItemB)

	resp := MatchResponse{
		ID:        d.ID.String(),
		Status:    d.Status,
		Score:     d.Score,
		YouRated:  d.ViewerRated,
		CreatedAt: d.CreatedAt.Time,
		UpdatedAt: d.UpdatedAt.Time,
	}
	if d.isUserA(viewer) {
		resp.YourItem, resp.TheirItem = itemA, itemB
		resp.OtherUser = UserSummary{ID: d.UserBID.String(), FullName: d.UserBName}
		resp.YouAccepted, resp.TheyAccepted = d.AAcceptedAt.Valid, d.BAcceptedAt.Valid
		resp.YouCompleted, resp.TheyCompleted = d.ACompletedAt.Valid, d.BCompletedAt.Valid
	} else {
		resp.YourItem, resp.TheirItem = itemB, itemA
		resp.OtherUser = UserSummary{ID: d.UserAID.String(), FullName: d.UserAName}
		resp.YouAccepted, resp.TheyAccepted = d.BAcceptedAt.Valid, d.AAcceptedAt.Valid
		resp.YouCompleted, resp.TheyCompleted = d.BCompletedAt.Valid, d.ACompletedAt.Valid
	}
	if d.ExchangeMethod.Valid {
		resp.Exchange = &Exchange{
			Method:        d.ExchangeMethod.String,
			ProposedByYou: d.ExchangeProposedBy == viewer,
		}
		if d.ExchangeDetails.Valid {
			resp.Exchange.Details = &d.ExchangeDetails.String
		}
	}
	if d.CompletedAt.Valid {
		t := d.CompletedAt.Time
		resp.CompletedAt = &t
	}
	return resp
}

func summary(id pgtype.UUID, it itemRow) ItemSummary {
	s := ItemSummary{
		ID:        id.String(),
		Title:     it.Title,
		Category:  it.Category,
		Condition: it.Condition,
		ImageURLs: it.ImageURLs,
	}
	if s.ImageURLs == nil {
		s.ImageURLs = []string{}
	}
	if it.EstimatedValue.Valid {
		v := int(it.EstimatedValue.Int32)
		s.EstimatedValue = &v
	}
	return s
}
