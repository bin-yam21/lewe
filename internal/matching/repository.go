package matching

import (
	"context"
	"encoding/hex"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrMatchNotFound     = errors.New("match not found")
	ErrNotMatchParty     = errors.New("not a participant in this match")
	ErrInvalidTransition = errors.New("invalid match status transition")
	ErrAlreadyMatched    = errors.New("match already exists between these items")

	ErrInvalidOffer    = errors.New("cannot offer an item for itself")
	ErrOwnItem         = errors.New("cannot offer a trade on your own item")
	ErrItemUnavailable = errors.New("item is not available to trade")
	ErrOfferExists     = errors.New("a trade between these items already exists")
)

// MatchRow represents a match row from the database.
type MatchRow struct {
	ID             pgtype.UUID
	ItemAID        pgtype.UUID
	ItemBID        pgtype.UUID
	Status         string
	ExchangeMethod pgtype.Text
	ConfirmedA     bool
	ConfirmedB     bool
	/** "discovered" by the mutual-wants query, or "direct" if someone offered. */
	Origin    string
	Message   pgtype.Text
	CreatedAt pgtype.Timestamptz
	UpdatedAt pgtype.Timestamptz
}

// MatchWithItems is a match joined with item summaries.
type MatchWithItems struct {
	Match MatchRow
	// Item A fields
	ItemAUserID   pgtype.UUID
	ItemATitle    string
	ItemACategory string
	ItemAStatus   string
	// Item B fields
	ItemBUserID   pgtype.UUID
	ItemBTitle    string
	ItemBCategory string
	ItemBStatus   string
}

// Repository handles match persistence.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new matching repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// FindMatches finds active items that are potential trade matches for the given item.
// A match exists when:
//  1. Another item's category is in our item's wants
//  2. Our item's category is in the other item's wants
//  3. Both items are active
//  4. The items belong to different users
func (r *Repository) FindMatches(ctx context.Context, itemID, userID pgtype.UUID) ([]MatchWithItems, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT
			m.id, m.item_a_id, m.item_b_id, m.status, m.exchange_method, m.confirmed_a, m.confirmed_b, m.origin, m.message, m.created_at, m.updated_at,
			ia.user_id, ia.title, ia.category, ia.status,
			ib.user_id, ib.title, ib.category, ib.status
		 FROM matches m
		 JOIN items ia ON ia.id = m.item_a_id
		 JOIN items ib ON ib.id = m.item_b_id
		 WHERE (m.item_a_id = $1 OR m.item_b_id = $1)
		   AND m.status != 'cancelled'`,
		itemID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanMatchWithItems(rows)
}

// FindPotentialMatches finds items that could be matched with the given item.
// Returns items where a bidirectional want relationship exists.
func (r *Repository) FindPotentialMatches(ctx context.Context, itemID, userID pgtype.UUID) ([]pgtype.UUID, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT DISTINCT other.id
		 FROM items other
		 -- other item's category is something we want
		 JOIN wants our_wants ON our_wants.item_id = $1 AND our_wants.category = other.category
		 -- our item's category is something they want
		 JOIN items our_item ON our_item.id = $1
		 JOIN wants their_wants ON their_wants.item_id = other.id AND their_wants.category = our_item.category
		 WHERE other.status = 'active'
		   AND other.user_id != $2
		   AND NOT EXISTS (
		       SELECT 1 FROM matches
		       WHERE (item_a_id = $1 AND item_b_id = other.id)
		          OR (item_a_id = other.id AND item_b_id = $1)
		   )`,
		itemID, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []pgtype.UUID
	for rows.Next() {
		var id pgtype.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// CreateMatch inserts a new match between two items.
func (r *Repository) CreateMatch(ctx context.Context, itemAID, itemBID pgtype.UUID) (*MatchRow, error) {
	m := &MatchRow{}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO matches (item_a_id, item_b_id)
		 VALUES ($1, $2)
		 ON CONFLICT (item_a_id, item_b_id) DO UPDATE SET updated_at = now()
		 RETURNING id, item_a_id, item_b_id, status, exchange_method, confirmed_a, confirmed_b, origin, message, created_at, updated_at`,
		itemAID, itemBID,
	).Scan(
		&m.ID, &m.ItemAID, &m.ItemBID, &m.Status, &m.ExchangeMethod,
		&m.ConfirmedA, &m.ConfirmedB, &m.Origin, &m.Message, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// CreateDirectOffer records a trade proposed straight to an owner.
//
// item_a is always the offered item and item_b the item being asked for, which
// keeps "who approached whom" readable from the row alone.
func (r *Repository) CreateDirectOffer(ctx context.Context, offerItemID, targetItemID pgtype.UUID, message *string) (*MatchRow, error) {
	m := &MatchRow{}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO matches (item_a_id, item_b_id, origin, message)
		 VALUES ($1, $2, 'direct', $3)
		 RETURNING id, item_a_id, item_b_id, status, exchange_method, confirmed_a, confirmed_b, origin, message, created_at, updated_at`,
		offerItemID, targetItemID, message,
	).Scan(
		&m.ID, &m.ItemAID, &m.ItemBID, &m.Status, &m.ExchangeMethod,
		&m.ConfirmedA, &m.ConfirmedB, &m.Origin, &m.Message, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// FindBetween returns a live match between two items in either direction.
func (r *Repository) FindBetween(ctx context.Context, itemX, itemY pgtype.UUID) (*MatchRow, error) {
	m := &MatchRow{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, item_a_id, item_b_id, status, exchange_method, confirmed_a, confirmed_b, origin, message, created_at, updated_at
		 FROM matches
		 WHERE ((item_a_id = $1 AND item_b_id = $2) OR (item_a_id = $2 AND item_b_id = $1))
		   AND status != 'cancelled'
		 LIMIT 1`,
		itemX, itemY,
	).Scan(
		&m.ID, &m.ItemAID, &m.ItemBID, &m.Status, &m.ExchangeMethod,
		&m.ConfirmedA, &m.ConfirmedB, &m.Origin, &m.Message, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMatchNotFound
		}
		return nil, err
	}
	return m, nil
}

// GetByID retrieves a match with item details.
func (r *Repository) GetByID(ctx context.Context, id pgtype.UUID) (*MatchWithItems, error) {
	mwi := &MatchWithItems{}
	err := r.pool.QueryRow(ctx,
		`SELECT
			m.id, m.item_a_id, m.item_b_id, m.status, m.exchange_method, m.confirmed_a, m.confirmed_b, m.origin, m.message, m.created_at, m.updated_at,
			ia.user_id, ia.title, ia.category, ia.status,
			ib.user_id, ib.title, ib.category, ib.status
		 FROM matches m
		 JOIN items ia ON ia.id = m.item_a_id
		 JOIN items ib ON ib.id = m.item_b_id
		 WHERE m.id = $1`,
		id,
	).Scan(
		&mwi.Match.ID, &mwi.Match.ItemAID, &mwi.Match.ItemBID, &mwi.Match.Status,
		&mwi.Match.ExchangeMethod, &mwi.Match.ConfirmedA, &mwi.Match.ConfirmedB,
		&mwi.Match.Origin, &mwi.Match.Message,
		&mwi.Match.CreatedAt, &mwi.Match.UpdatedAt,
		&mwi.ItemAUserID, &mwi.ItemATitle, &mwi.ItemACategory, &mwi.ItemAStatus,
		&mwi.ItemBUserID, &mwi.ItemBTitle, &mwi.ItemBCategory, &mwi.ItemBStatus,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMatchNotFound
		}
		return nil, err
	}
	return mwi, nil
}

// ListForUser retrieves all matches where the user owns either item.
func (r *Repository) ListForUser(ctx context.Context, userID pgtype.UUID) ([]MatchWithItems, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT
			m.id, m.item_a_id, m.item_b_id, m.status, m.exchange_method, m.confirmed_a, m.confirmed_b, m.origin, m.message, m.created_at, m.updated_at,
			ia.user_id, ia.title, ia.category, ia.status,
			ib.user_id, ib.title, ib.category, ib.status
		 FROM matches m
		 JOIN items ia ON ia.id = m.item_a_id
		 JOIN items ib ON ib.id = m.item_b_id
		 WHERE ia.user_id = $1 OR ib.user_id = $1
		 ORDER BY m.updated_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanMatchWithItems(rows)
}

// UpdateStatus updates the match status and timestamp.
func (r *Repository) UpdateStatus(ctx context.Context, id pgtype.UUID, status string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE matches SET status = $2, updated_at = now() WHERE id = $1`,
		id, status,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrMatchNotFound
	}
	return nil
}

// SetConfirmation sets confirmed_a or confirmed_b for a match.
func (r *Repository) SetConfirmation(ctx context.Context, id pgtype.UUID, side string, confirmed bool) error {
	var col string
	if side == "a" {
		col = "confirmed_a"
	} else {
		col = "confirmed_b"
	}

	tag, err := r.pool.Exec(ctx,
		`UPDATE matches SET `+col+` = $2, updated_at = now() WHERE id = $1`,
		id, confirmed,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrMatchNotFound
	}
	return nil
}

// SetExchangeMethod records how the two parties will exchange their items.
func (r *Repository) SetExchangeMethod(ctx context.Context, id pgtype.UUID, method string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE matches SET exchange_method = $2, updated_at = now() WHERE id = $1`,
		id, method,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrMatchNotFound
	}
	return nil
}

// GetConfirmations returns confirmed_a and confirmed_b for a match.
func (r *Repository) GetConfirmations(ctx context.Context, id pgtype.UUID) (bool, bool, error) {
	var a, b bool
	err := r.pool.QueryRow(ctx,
		`SELECT confirmed_a, confirmed_b FROM matches WHERE id = $1`,
		id,
	).Scan(&a, &b)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, false, ErrMatchNotFound
		}
		return false, false, err
	}
	return a, b, nil
}

// --- helpers ---

func scanMatchWithItems(rows pgx.Rows) ([]MatchWithItems, error) {
	var result []MatchWithItems
	for rows.Next() {
		var mwi MatchWithItems
		if err := rows.Scan(
			&mwi.Match.ID, &mwi.Match.ItemAID, &mwi.Match.ItemBID, &mwi.Match.Status,
			&mwi.Match.ExchangeMethod, &mwi.Match.ConfirmedA, &mwi.Match.ConfirmedB,
			&mwi.Match.Origin, &mwi.Match.Message,
			&mwi.Match.CreatedAt, &mwi.Match.UpdatedAt,
			&mwi.ItemAUserID, &mwi.ItemATitle, &mwi.ItemACategory, &mwi.ItemAStatus,
			&mwi.ItemBUserID, &mwi.ItemBTitle, &mwi.ItemBCategory, &mwi.ItemBStatus,
		); err != nil {
			return nil, err
		}
		result = append(result, mwi)
	}
	return result, nil
}

// uuidToString converts a pgtype.UUID to its string representation.
func uuidToString(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	b := id.Bytes
	return hex.EncodeToString(b[0:4]) + "-" +
		hex.EncodeToString(b[4:6]) + "-" +
		hex.EncodeToString(b[6:8]) + "-" +
		hex.EncodeToString(b[8:10]) + "-" +
		hex.EncodeToString(b[10:16])
}
