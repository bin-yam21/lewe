package matches

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yeabt/lewe/internal/db"
	"github.com/yeabt/lewe/internal/notifications"
	"github.com/yeabt/lewe/internal/request"
)

var ErrMatchNotFound = errors.New("match not found")

// Row holds the state columns of a matches row.
type Row struct {
	ID                 pgtype.UUID
	UserAID            pgtype.UUID
	ItemAID            pgtype.UUID
	WantAID            pgtype.UUID
	UserBID            pgtype.UUID
	ItemBID            pgtype.UUID
	WantBID            pgtype.UUID
	Score              float64
	Status             string
	AAcceptedAt        pgtype.Timestamptz
	BAcceptedAt        pgtype.Timestamptz
	ExchangeMethod     pgtype.Text
	ExchangeDetails    pgtype.Text
	ExchangeProposedBy pgtype.UUID
	ACompletedAt       pgtype.Timestamptz
	BCompletedAt       pgtype.Timestamptz
	CreatedAt          pgtype.Timestamptz
	UpdatedAt          pgtype.Timestamptz
	CompletedAt        pgtype.Timestamptz
}

// isUserA reports whether userID is the "A" side of the match.
func (m *Row) isUserA(userID pgtype.UUID) bool { return m.UserAID == userID }

// hasParticipant reports whether userID is one of the two users in the match.
func (m *Row) hasParticipant(userID pgtype.UUID) bool {
	return m.UserAID == userID || m.UserBID == userID
}

const matchColumns = `m.id, m.user_a_id, m.item_a_id, m.want_a_id, m.user_b_id, m.item_b_id, m.want_b_id,
	m.score, m.status, m.a_accepted_at, m.b_accepted_at, m.exchange_method, m.exchange_details,
	m.exchange_proposed_by, m.a_completed_at, m.b_completed_at, m.created_at, m.updated_at, m.completed_at`

func (m *Row) scanTargets() []any {
	return []any{&m.ID, &m.UserAID, &m.ItemAID, &m.WantAID, &m.UserBID, &m.ItemBID, &m.WantBID,
		&m.Score, &m.Status, &m.AAcceptedAt, &m.BAcceptedAt, &m.ExchangeMethod, &m.ExchangeDetails,
		&m.ExchangeProposedBy, &m.ACompletedAt, &m.BCompletedAt, &m.CreatedAt, &m.UpdatedAt, &m.CompletedAt}
}

// itemRow is the item data joined into a match view.
type itemRow struct {
	Title          string
	Category       string
	Condition      string
	EstimatedValue pgtype.Int4
	ImageURLs      []string
}

// DetailRow is a match joined with both items, both users' names, and whether
// the viewer has rated the match.
type DetailRow struct {
	Row
	ItemA       itemRow
	ItemB       itemRow
	UserAName   string
	UserBName   string
	ViewerRated bool
}

const detailSelect = `SELECT ` + matchColumns + `,
	ia.title, ia.category, ia.condition, ia.estimated_value, ia.image_urls,
	ib.title, ib.category, ib.condition, ib.estimated_value, ib.image_urls,
	ua.full_name, ub.full_name,
	EXISTS (SELECT 1 FROM ratings r WHERE r.match_id = m.id AND r.rater_id = $1)
FROM matches m
JOIN items ia ON ia.id = m.item_a_id
JOIN items ib ON ib.id = m.item_b_id
JOIN users ua ON ua.id = m.user_a_id
JOIN users ub ON ub.id = m.user_b_id`

func scanDetail(row pgx.Row) (*DetailRow, error) {
	d := &DetailRow{}
	targets := append(d.scanTargets(),
		&d.ItemA.Title, &d.ItemA.Category, &d.ItemA.Condition, &d.ItemA.EstimatedValue, &d.ItemA.ImageURLs,
		&d.ItemB.Title, &d.ItemB.Category, &d.ItemB.Condition, &d.ItemB.EstimatedValue, &d.ItemB.ImageURLs,
		&d.UserAName, &d.UserBName, &d.ViewerRated,
	)
	if err := row.Scan(targets...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMatchNotFound
		}
		return nil, err
	}
	return d, nil
}

// Repository handles match persistence.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new match repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Pool exposes the connection pool for transactional service operations.
func (r *Repository) Pool() *pgxpool.Pool { return r.pool }

// GetDetailForUser loads a match the user participates in.
func (r *Repository) GetDetailForUser(ctx context.Context, id, userID pgtype.UUID) (*DetailRow, error) {
	return scanDetail(r.pool.QueryRow(ctx,
		detailSelect+` WHERE m.id = $2 AND (m.user_a_id = $1 OR m.user_b_id = $1)`, userID, id))
}

// ListForUser lists the user's matches, optionally filtered by status.
func (r *Repository) ListForUser(ctx context.Context, userID pgtype.UUID, status string, page request.Page) ([]*DetailRow, error) {
	rows, err := r.pool.Query(ctx,
		detailSelect+`
		 WHERE (m.user_a_id = $1 OR m.user_b_id = $1) AND ($2 = '' OR m.status = $2)
		 ORDER BY m.created_at DESC, m.id
		 LIMIT $3 OFFSET $4`,
		userID, status, page.Limit, page.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []*DetailRow{}
	for rows.Next() {
		d, err := scanDetail(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

// --- transactional helpers (called by the service inside db.WithTx) ---

// lockForUpdate loads and row-locks a match.
func lockForUpdate(ctx context.Context, q db.DBTX, id pgtype.UUID) (*Row, error) {
	m := &Row{}
	err := q.QueryRow(ctx, `SELECT `+matchColumns+` FROM matches m WHERE m.id = $1 FOR UPDATE`, id).
		Scan(m.scanTargets()...)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMatchNotFound
		}
		return nil, err
	}
	return m, nil
}

// lockAvailableItems row-locks both items (in a stable order to avoid
// deadlocks) and reports whether both are still available.
func lockAvailableItems(ctx context.Context, q db.DBTX, a, b pgtype.UUID) (bool, error) {
	rows, err := q.Query(ctx,
		`SELECT status FROM items WHERE id IN ($1, $2) ORDER BY id FOR UPDATE`, a, b)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	available := 0
	for rows.Next() {
		var status string
		if err := rows.Scan(&status); err != nil {
			return false, err
		}
		if status == "available" {
			available++
		}
	}
	return available == 2, rows.Err()
}

func setItemsStatus(ctx context.Context, q db.DBTX, a, b pgtype.UUID, from, to string) error {
	_, err := q.Exec(ctx,
		`UPDATE items SET status = $4, updated_at = now()
		 WHERE id IN ($1, $2) AND status = $3`, a, b, from, to)
	return err
}

// cancelOtherPendingForItems cancels pending matches (other than keep) that
// involve either item.
func cancelOtherPendingForItems(ctx context.Context, q db.DBTX, keep, a, b pgtype.UUID) error {
	return notifications.CancelPendingMatches(ctx, q,
		`id <> $1 AND (item_a_id IN ($2, $3) OR item_b_id IN ($2, $3))`, keep, a, b)
}

// fulfillWants marks the wants behind a completed match as fulfilled and
// cancels other pending matches that relied on them.
func fulfillWants(ctx context.Context, q db.DBTX, keep, wantA, wantB pgtype.UUID) error {
	if _, err := q.Exec(ctx,
		`UPDATE wants SET status = 'fulfilled', updated_at = now()
		 WHERE id IN ($1, $2) AND status = 'active'`, wantA, wantB); err != nil {
		return err
	}
	return notifications.CancelPendingMatches(ctx, q,
		`id <> $1 AND (want_a_id IN ($2, $3) OR want_b_id IN ($2, $3))`, keep, wantA, wantB)
}

// save writes the mutable state columns of a match.
func save(ctx context.Context, q db.DBTX, m *Row, closedBy pgtype.UUID) error {
	_, err := q.Exec(ctx,
		`UPDATE matches SET
		   status = $2, a_accepted_at = $3, b_accepted_at = $4,
		   exchange_method = $5, exchange_details = $6, exchange_proposed_by = $7,
		   a_completed_at = $8, b_completed_at = $9, completed_at = $10,
		   closed_by = COALESCE($11, closed_by), updated_at = now()
		 WHERE id = $1`,
		m.ID, m.Status, m.AAcceptedAt, m.BAcceptedAt,
		m.ExchangeMethod, m.ExchangeDetails, m.ExchangeProposedBy,
		m.ACompletedAt, m.BCompletedAt, m.CompletedAt, closedBy)
	return err
}
