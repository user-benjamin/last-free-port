// Package inventory is the Postgres-backed source of truth for what each
// player owns. It is deliberately service-agnostic: the game server uses it
// today (load on join, grant on gather); if the API later needs to read or
// mutate inventory for REST endpoints, trading, or crafting, it imports the
// same Store rather than reimplementing the rules (proposal §15.1).
package inventory

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrInsufficient means the player doesn't own enough of a craft's inputs.
// Craft returns it (not a generic error) so the caller can tell "you can't
// afford this" apart from "the database broke" and message accordingly.
var ErrInsufficient = errors.New("insufficient materials")

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

// Craft applies one recipe that the player has explicitly chosen (the hub
// calls this only after receiving a craft_intent — it is never automatic).
// It debits each input — returning ErrInsufficient if the player lacks any —
// and credits the output inside a single database transaction, so the whole
// craft either fully applies or not at all: it can never eat materials
// without producing the item, or produce the item without charging for it.
// It returns the resulting quantity of every affected item so the caller can
// report new authoritative totals to the client.
func (s *Store) Craft(ctx context.Context, userID string, inputs map[string]int, output string, outputQty int) (map[string]int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin craft tx: %w", err)
	}
	defer tx.Rollback(ctx) // no-op once Commit succeeds

	result := make(map[string]int, len(inputs)+1)

	// Debit inputs in a stable (sorted) order so two concurrent crafts that
	// share input items lock rows in the same sequence and can't deadlock.
	for _, item := range sortedKeys(inputs) {
		need := inputs[item]
		var remaining int
		err := tx.QueryRow(ctx, `
			UPDATE inventory_items SET quantity = quantity - $3, updated_at = now()
			WHERE user_id = $1 AND item_type = $2 AND quantity >= $3
			RETURNING quantity`, userID, item, need).Scan(&remaining)
		if errors.Is(err, pgx.ErrNoRows) {
			// No row, or the guard quantity >= need failed: can't afford it.
			return nil, ErrInsufficient
		}
		if err != nil {
			return nil, fmt.Errorf("debit %d %s: %w", need, item, err)
		}
		result[item] = remaining
	}

	var outQty int
	err = tx.QueryRow(ctx, `
		INSERT INTO inventory_items (user_id, item_type, quantity)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, item_type)
		DO UPDATE SET quantity = inventory_items.quantity + $3, updated_at = now()
		RETURNING quantity`, userID, output, outputQty).Scan(&outQty)
	if err != nil {
		return nil, fmt.Errorf("credit %d %s: %w", outputQty, output, err)
	}
	result[output] = outQty

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit craft: %w", err)
	}
	return result, nil
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
