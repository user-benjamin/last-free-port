// Package inventory is the Postgres-backed source of truth for what each
// player owns. It is deliberately service-agnostic: the game server uses it
// today (load on join, grant on gather); if the API later needs to read or
// mutate inventory for REST endpoints, trading, or crafting, it imports the
// same Store rather than reimplementing the rules (proposal §15.1).
package inventory

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store reads and writes player inventories. The quantity arithmetic happens
// in SQL so concurrent grants to the same (user, item) can't lose updates to
// a read-modify-write race.
type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Load returns every item the player owns as item_type -> quantity. A new
// player simply has no rows yet, so the map comes back empty (never nil).
func (s *Store) Load(ctx context.Context, userID string) (map[string]int, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT item_type, quantity FROM inventory_items WHERE user_id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("load inventory for %s: %w", userID, err)
	}
	defer rows.Close()

	items := make(map[string]int)
	for rows.Next() {
		var itemType string
		var qty int
		if err := rows.Scan(&itemType, &qty); err != nil {
			return nil, fmt.Errorf("scan inventory row: %w", err)
		}
		items[itemType] = qty
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate inventory rows: %w", err)
	}
	return items, nil
}

// Grant adds n of itemType to the player and returns the resulting quantity.
// The upsert means a first grant creates the row and a later grant increments
// it, atomically, in a single round-trip.
func (s *Store) Grant(ctx context.Context, userID, itemType string, n int) (int, error) {
	var qty int
	err := s.pool.QueryRow(ctx, `
		INSERT INTO inventory_items (user_id, item_type, quantity)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, item_type)
		DO UPDATE SET quantity = inventory_items.quantity + $3, updated_at = now()
		RETURNING quantity`, userID, itemType, n).Scan(&qty)
	if err != nil {
		return 0, fmt.Errorf("grant %d %s to %s: %w", n, itemType, userID, err)
	}
	return qty, nil
}
