package items

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrItemNotFound = errors.New("item not found")
	ErrNotItemOwner = errors.New("not the item owner")
)

// ItemRow represents an item row from the database.
type ItemRow struct {
	ID             pgtype.UUID
	UserID         pgtype.UUID
	Title          string
	Description    string
	Category       string
	Condition      string
	ExchangeMethod string
	Images         []byte // JSONB stored as raw bytes
	Location       pgtype.Text
	Status         string
	CreatedAt      pgtype.Timestamptz
	UpdatedAt      pgtype.Timestamptz
}

// WantRow represents a want row from the database.
type WantRow struct {
	ID          pgtype.UUID
	ItemID      pgtype.UUID
	Category    string
	Description pgtype.Text
	CreatedAt   pgtype.Timestamptz
}

// Repository handles item persistence.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new item repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create inserts a new item and its associated wants in a single transaction.
func (r *Repository) Create(ctx context.Context, userID pgtype.UUID, title, description, category, condition, exchangeMethod string, images []string, location *string, wants []WantInput) (*ItemRow, []WantRow, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx)

	imagesJSON, err := json.Marshal(images)
	if err != nil {
		return nil, nil, err
	}

	item := &ItemRow{}
	err = tx.QueryRow(ctx,
		`INSERT INTO items (user_id, title, description, category, condition, exchange_method, images, location)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id, user_id, title, description, category, condition, exchange_method, images, location, status, created_at, updated_at`,
		userID, title, description, category, condition, exchangeMethod, imagesJSON, location,
	).Scan(
		&item.ID, &item.UserID, &item.Title, &item.Description, &item.Category,
		&item.Condition, &item.ExchangeMethod, &item.Images, &item.Location,
		&item.Status, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, nil, err
	}

	// Batch insert wants
	wantRows, err := r.insertWants(ctx, tx, item.ID, wants)
	if err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}

	return item, wantRows, nil
}

// GetByID retrieves an item by its UUID.
func (r *Repository) GetByID(ctx context.Context, id pgtype.UUID) (*ItemRow, []WantRow, error) {
	item := &ItemRow{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, title, description, category, condition, exchange_method, images, location, status, created_at, updated_at
		 FROM items WHERE id = $1`,
		id,
	).Scan(
		&item.ID, &item.UserID, &item.Title, &item.Description, &item.Category,
		&item.Condition, &item.ExchangeMethod, &item.Images, &item.Location,
		&item.Status, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrItemNotFound
		}
		return nil, nil, err
	}

	wants, err := r.getWantsByItemID(ctx, item.ID)
	if err != nil {
		return nil, nil, err
	}

	return item, wants, nil
}

// Update modifies an existing item. Only the owner can update.
func (r *Repository) Update(ctx context.Context, id, userID pgtype.UUID, title, description, category, condition, exchangeMethod *string, images *[]string, location *string, wants *[]WantInput) (*ItemRow, []WantRow, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx)

	// Build dynamic UPDATE query
	setClauses := []string{"updated_at = now()"}
	args := []any{id}
	argIdx := 2

	if title != nil {
		setClauses = append(setClauses, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, *title)
		argIdx++
	}
	if description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *description)
		argIdx++
	}
	if category != nil {
		setClauses = append(setClauses, fmt.Sprintf("category = $%d", argIdx))
		args = append(args, *category)
		argIdx++
	}
	if condition != nil {
		setClauses = append(setClauses, fmt.Sprintf("condition = $%d", argIdx))
		args = append(args, *condition)
		argIdx++
	}
	if exchangeMethod != nil {
		setClauses = append(setClauses, fmt.Sprintf("exchange_method = $%d", argIdx))
		args = append(args, *exchangeMethod)
		argIdx++
	}
	if images != nil {
		imagesJSON, err := json.Marshal(*images)
		if err != nil {
			return nil, nil, err
		}
		setClauses = append(setClauses, fmt.Sprintf("images = $%d", argIdx))
		args = append(args, imagesJSON)
		argIdx++
	}
	if location != nil {
		setClauses = append(setClauses, fmt.Sprintf("location = $%d", argIdx))
		args = append(args, *location)
		argIdx++
	}

	// Add user_id filter
	args = append(args, userID)
	userIDArgIdx := argIdx

	query := fmt.Sprintf(
		`UPDATE items SET %s WHERE id = $1 AND user_id = $%d
		 RETURNING id, user_id, title, description, category, condition, exchange_method, images, location, status, created_at, updated_at`,
		strings.Join(setClauses, ", "), userIDArgIdx,
	)

	item := &ItemRow{}
	err = tx.QueryRow(ctx, query, args...).Scan(
		&item.ID, &item.UserID, &item.Title, &item.Description, &item.Category,
		&item.Condition, &item.ExchangeMethod, &item.Images, &item.Location,
		&item.Status, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrNotItemOwner
		}
		return nil, nil, err
	}

	// Replace wants if provided
	var wantRows []WantRow
	if wants != nil {
		// Delete old wants
		if _, err := tx.Exec(ctx, `DELETE FROM wants WHERE item_id = $1`, id); err != nil {
			return nil, nil, err
		}
		wantRows, err = r.insertWants(ctx, tx, item.ID, *wants)
		if err != nil {
			return nil, nil, err
		}
	} else {
		wantRows, err = r.getWantsByItemIDTx(ctx, tx, item.ID)
		if err != nil {
			return nil, nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}

	return item, wantRows, nil
}

// Delete archives an item (soft delete). Only the owner can delete.
func (r *Repository) Delete(ctx context.Context, id, userID pgtype.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE items SET status = 'archived', updated_at = now() WHERE id = $1 AND user_id = $2 AND status != 'archived'`,
		id, userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotItemOwner
	}
	return nil
}

// List retrieves a paginated list of items with optional filters.
func (r *Repository) List(ctx context.Context, f ListItemsFilter) ([]ItemRow, int, error) {
	whereClauses := []string{"1=1"}
	args := []any{}
	argIdx := 1

	if f.Category != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("i.category = $%d", argIdx))
		args = append(args, f.Category)
		argIdx++
	}
	if f.Status != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("i.status = $%d", argIdx))
		args = append(args, f.Status)
		argIdx++
	} else {
		// Default: exclude archived
		whereClauses = append(whereClauses, "i.status != 'archived'")
	}
	if f.UserID != "" {
		var uid pgtype.UUID
		// Fail loudly: silently dropping this clause would widen an owner-scoped
		// listing into a listing of every user's items.
		if err := uid.Scan(f.UserID); err != nil {
			return nil, 0, fmt.Errorf("invalid user filter: %w", err)
		}
		whereClauses = append(whereClauses, fmt.Sprintf("i.user_id = $%d", argIdx))
		args = append(args, uid)
		argIdx++
	}
	if f.Query != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(i.title ILIKE $%d OR i.description ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+f.Query+"%")
		argIdx++
	}

	where := strings.Join(whereClauses, " AND ")

	// Count total
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM items i WHERE %s", where)
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Fetch page
	offset := (f.Page - 1) * f.PerPage
	args = append(args, f.PerPage, offset)
	listQuery := fmt.Sprintf(
		`SELECT id, user_id, title, description, category, condition, exchange_method, images, location, status, created_at, updated_at
		 FROM items i WHERE %s ORDER BY i.created_at DESC LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1,
	)

	rows, err := r.pool.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []ItemRow
	for rows.Next() {
		var item ItemRow
		if err := rows.Scan(
			&item.ID, &item.UserID, &item.Title, &item.Description, &item.Category,
			&item.Condition, &item.ExchangeMethod, &item.Images, &item.Location,
			&item.Status, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}

	return items, total, nil
}

// UpdateStatus changes an item's status.
func (r *Repository) UpdateStatus(ctx context.Context, id pgtype.UUID, status string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE items SET status = $2, updated_at = now() WHERE id = $1`,
		id, status,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrItemNotFound
	}
	return nil
}

// --- internal helpers ---

// insertWants batch-inserts wants within a transaction.
func (r *Repository) insertWants(ctx context.Context, tx pgx.Tx, itemID pgtype.UUID, wants []WantInput) ([]WantRow, error) {
	wantRows := make([]WantRow, 0, len(wants))
	for _, w := range wants {
		wr := WantRow{}
		err := tx.QueryRow(ctx,
			`INSERT INTO wants (item_id, category, description)
			 VALUES ($1, $2, $3)
			 RETURNING id, item_id, category, description, created_at`,
			itemID, w.Category, w.Description,
		).Scan(&wr.ID, &wr.ItemID, &wr.Category, &wr.Description, &wr.CreatedAt)
		if err != nil {
			return nil, err
		}
		wantRows = append(wantRows, wr)
	}
	return wantRows, nil
}

// getWantsByItemID fetches all wants for a given item.
func (r *Repository) getWantsByItemID(ctx context.Context, itemID pgtype.UUID) ([]WantRow, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, item_id, category, description, created_at FROM wants WHERE item_id = $1 ORDER BY created_at`,
		itemID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanWantRows(rows)
}

// getWantsByItemIDTx fetches wants within a transaction.
func (r *Repository) getWantsByItemIDTx(ctx context.Context, tx pgx.Tx, itemID pgtype.UUID) ([]WantRow, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, item_id, category, description, created_at FROM wants WHERE item_id = $1 ORDER BY created_at`,
		itemID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanWantRows(rows)
}

// scanWantRows scans rows into a slice of WantRow.
func scanWantRows(rows pgx.Rows) ([]WantRow, error) {
	var wants []WantRow
	for rows.Next() {
		var w WantRow
		if err := rows.Scan(&w.ID, &w.ItemID, &w.Category, &w.Description, &w.CreatedAt); err != nil {
			return nil, err
		}
		wants = append(wants, w)
	}
	return wants, nil
}

// GetWantCategoriesByItemID returns the want categories for a given item.
func (r *Repository) GetWantCategoriesByItemID(ctx context.Context, itemID pgtype.UUID) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT category FROM wants WHERE item_id = $1`,
		itemID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cats []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		cats = append(cats, c)
	}
	return cats, nil
}
