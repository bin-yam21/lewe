package items

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yeabt/lewe/internal/db"
	"github.com/yeabt/lewe/internal/notifications"
	"github.com/yeabt/lewe/internal/request"
)

var ErrItemNotFound = errors.New("item not found")

// Row represents an items row from the database.
type Row struct {
	ID             pgtype.UUID
	OwnerID        pgtype.UUID
	Title          string
	Description    string
	Category       string
	Condition      string
	EstimatedValue pgtype.Int4
	Location       pgtype.Text
	ImageURLs      []string
	Status         string
	CreatedAt      pgtype.Timestamptz
	UpdatedAt      pgtype.Timestamptz
}

const itemColumns = `id, owner_id, title, description, category, condition,
	estimated_value, location, image_urls, status, created_at, updated_at`

func scanItem(row pgx.Row) (*Row, error) {
	it := &Row{}
	err := row.Scan(&it.ID, &it.OwnerID, &it.Title, &it.Description, &it.Category, &it.Condition,
		&it.EstimatedValue, &it.Location, &it.ImageURLs, &it.Status, &it.CreatedAt, &it.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrItemNotFound
		}
		return nil, err
	}
	return it, nil
}

// Repository handles item persistence.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new item repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create inserts a new item owned by ownerID.
func (r *Repository) Create(ctx context.Context, ownerID pgtype.UUID, req ItemRequest) (*Row, error) {
	return scanItem(r.pool.QueryRow(ctx,
		`INSERT INTO items (owner_id, title, description, category, condition, estimated_value, location, image_urls)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING `+itemColumns,
		ownerID, req.Title, req.Description, req.Category, req.Condition, req.EstimatedValue, req.Location, req.ImageURLs,
	))
}

// GetByID retrieves an item by ID.
func (r *Repository) GetByID(ctx context.Context, id pgtype.UUID) (*Row, error) {
	return scanItem(r.pool.QueryRow(ctx, `SELECT `+itemColumns+` FROM items WHERE id = $1`, id))
}

// Filter narrows an item listing query. Empty fields are ignored.
type Filter struct {
	OwnerID  pgtype.UUID
	Category string
	Query    string
	Status   string
}

// List returns items matching the filter, newest first.
func (r *Repository) List(ctx context.Context, f Filter, page request.Page) ([]*Row, error) {
	var (
		conds []string
		args  []any
	)
	add := func(cond string, arg any) {
		args = append(args, arg)
		conds = append(conds, strings.ReplaceAll(cond, "?", "$"+strconv.Itoa(len(args))))
	}

	if f.OwnerID.Valid {
		add("owner_id = ?", f.OwnerID)
	}
	if f.Category != "" {
		add("category = ?", f.Category)
	}
	if f.Status != "" {
		add("status = ?", f.Status)
	}
	if f.Query != "" {
		add("(title ILIKE ? OR description ILIKE ?)", "%"+escapeLike(f.Query)+"%")
	}

	sql := `SELECT ` + itemColumns + ` FROM items`
	if len(conds) > 0 {
		sql += " WHERE " + strings.Join(conds, " AND ")
	}
	args = append(args, page.Limit, page.Offset)
	sql += " ORDER BY created_at DESC, id LIMIT $" + strconv.Itoa(len(args)-1) + " OFFSET $" + strconv.Itoa(len(args))

	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []*Row{}
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, it)
	}
	return result, rows.Err()
}

// Update replaces the editable fields of an item.
func (r *Repository) Update(ctx context.Context, id pgtype.UUID, req ItemRequest) (*Row, error) {
	return scanItem(r.pool.QueryRow(ctx,
		`UPDATE items
		 SET title = $2, description = $3, category = $4, condition = $5,
		     estimated_value = $6, location = $7, image_urls = $8, updated_at = now()
		 WHERE id = $1
		 RETURNING `+itemColumns,
		id, req.Title, req.Description, req.Category, req.Condition, req.EstimatedValue, req.Location, req.ImageURLs,
	))
}

// Withdraw marks an available item as withdrawn and cancels any pending
// matches that involve it. It returns ErrItemNotFound if the item is not
// currently available.
func (r *Repository) Withdraw(ctx context.Context, id pgtype.UUID) error {
	return db.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE items SET status = 'withdrawn', updated_at = now()
			 WHERE id = $1 AND status = 'available'`, id)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrItemNotFound
		}
		return notifications.CancelPendingMatches(ctx, tx, `item_a_id = $1 OR item_b_id = $1`, id)
	})
}

// escapeLike escapes LIKE wildcards so user input is matched literally.
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
