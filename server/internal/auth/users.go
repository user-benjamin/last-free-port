package auth

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Users is the Postgres-backed account store. There are no passwords yet:
// a username IS an account. That's an honest, documented stub for LAN play
// — when real credentials land they slot in here (validate before Upsert)
// without touching the ticket flow at all.
type Users struct {
	pool *pgxpool.Pool
}

func NewUsers(pool *pgxpool.Pool) *Users {
	return &Users{pool: pool}
}

// Upsert returns the stable user id for a username, creating the account
// on first login. One round-trip covers both new and returning players,
// and the returned UUID is the identity everything else (inventory, bases,
// reputation) will reference.
func (u *Users) Upsert(ctx context.Context, username string) (string, error) {
	var id string
	err := u.pool.QueryRow(ctx, `
		INSERT INTO users (username) VALUES ($1)
		ON CONFLICT (username) DO UPDATE SET last_login = now()
		RETURNING id::text`, username).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("upsert user %q: %w", username, err)
	}
	return id, nil
}
