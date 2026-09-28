package matches

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// findMatchesSQL discovers new two-way swaps and records them as pending matches.
//
// want_items lists every (want, item) pair where an available item owned by
// someone else satisfies an active want: same category, at least the wanted
// condition, and — if the want has keywords — at least one keyword appears in
// the item's title or description.
//
// A swap exists when user A wants an item of user B's (ib) and user B wants an
// item of user A's (ia). Every swap shows up twice (once from each side), so
// only the orientation with item_a_id < item_b_id is kept, which also matches
// the table's canonical ordering. When several wants produce the same item
// pair, DISTINCT ON keeps the highest scoring one. Pairs that were matched
// before — including declined ones — are skipped by the unique constraint.
//
// The score (0–1) favours swaps of similar estimated value; when either value
// is unknown it defaults to 0.5.
const findMatchesSQL = `
WITH want_items AS (
    SELECT w.id AS want_id, w.user_id AS wanter_id,
           i.id AS item_id, i.owner_id, i.estimated_value
    FROM wants w
    JOIN items i
      ON i.category = w.category
     AND i.owner_id <> w.user_id
     AND i.status = 'available'
     AND item_condition_rank(i.condition) >= item_condition_rank(COALESCE(w.min_condition, 'poor'))
     AND (
          cardinality(w.keywords) = 0
          OR EXISTS (
              SELECT 1 FROM unnest(w.keywords) AS k(word)
              WHERE i.title ILIKE '%' || k.word || '%'
                 OR i.description ILIKE '%' || k.word || '%'
          )
     )
    WHERE w.status = 'active'
)
INSERT INTO matches (user_a_id, item_a_id, want_a_id, user_b_id, item_b_id, want_b_id, score)
SELECT DISTINCT ON (ia.item_id, ib.item_id)
       ia.owner_id, ia.item_id, ib.want_id,
       ib.owner_id, ib.item_id, ia.want_id,
       CASE
           WHEN ia.estimated_value IS NULL OR ib.estimated_value IS NULL THEN 0.5
           WHEN GREATEST(ia.estimated_value, ib.estimated_value) = 0 THEN 1.0
           ELSE 1.0 - abs(ia.estimated_value - ib.estimated_value)::float8
                      / GREATEST(ia.estimated_value, ib.estimated_value)
       END AS score
FROM want_items ia
JOIN want_items ib
  ON ib.wanter_id = ia.owner_id
 AND ib.owner_id  = ia.wanter_id
WHERE ia.item_id < ib.item_id
ORDER BY ia.item_id, ib.item_id, score DESC
ON CONFLICT (item_a_id, item_b_id) DO NOTHING`

// Worker periodically scans listings and wants for new matches.
type Worker struct {
	pool     *pgxpool.Pool
	interval time.Duration
	trigger  chan struct{}
}

// NewWorker creates a matching worker that runs every interval.
func NewWorker(pool *pgxpool.Pool, interval time.Duration) *Worker {
	return &Worker{
		pool:     pool,
		interval: interval,
		trigger:  make(chan struct{}, 1),
	}
}

// RunOnce performs a single matching pass and returns the number of new matches.
func (w *Worker) RunOnce(ctx context.Context) (int64, error) {
	tag, err := w.pool.Exec(ctx, findMatchesSQL)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// Trigger asks the worker to run a pass soon without waiting for the next tick.
// It never blocks; multiple triggers before a run collapse into one.
func (w *Worker) Trigger() {
	select {
	case w.trigger <- struct{}{}:
	default:
	}
}

// Run executes matching passes until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	log.Printf("Matching worker started (interval %s)", w.interval)
	for {
		w.pass(ctx)
		select {
		case <-ctx.Done():
			log.Println("Matching worker stopped")
			return
		case <-ticker.C:
		case <-w.trigger:
		}
	}
}

func (w *Worker) pass(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	n, err := w.RunOnce(ctx)
	if err != nil {
		if ctx.Err() == nil {
			log.Printf("Matching pass failed: %v", err)
		}
		return
	}
	if n > 0 {
		log.Printf("Matching pass created %d new match(es)", n)
	}
}
