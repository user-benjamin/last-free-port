// Package auth implements the session flow from proposal §16: the API is
// the only service that knows who you are; the game server only ever sees
// short-lived, single-use tickets it redeems against Valkey.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// TicketTTL is how long an issued ticket stays redeemable. Long enough for
// the client to turn around and dial the game server; short enough that a
// leaked ticket is nearly worthless.
const TicketTTL = 30 * time.Second

var ErrInvalidTicket = errors.New("invalid, expired, or already-used ticket")

// UserInfo is what a ticket buys: a stable identity for the session.
type UserInfo struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

// Tickets mints and redeems session tickets in Valkey. The API issues; the
// game server redeems. Valkey is the right home (not Postgres): tickets are
// throwaway state that must expire on their own.
type Tickets struct {
	rdb *redis.Client
}

func NewTickets(rdb *redis.Client) *Tickets {
	return &Tickets{rdb: rdb}
}

func (t *Tickets) Issue(ctx context.Context, user UserInfo) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err) // crypto/rand failing means the host is broken
	}
	ticket := hex.EncodeToString(b)

	payload, err := json.Marshal(user)
	if err != nil {
		return "", err
	}
	if err := t.rdb.Set(ctx, "ticket:"+ticket, payload, TicketTTL).Err(); err != nil {
		return "", fmt.Errorf("store ticket: %w", err)
	}
	return ticket, nil
}

// Redeem consumes a ticket. GETDEL reads and deletes atomically, which
// makes tickets structurally single-use: two connections racing the same
// ticket means exactly one wins, with no lock and no race window.
func (t *Tickets) Redeem(ctx context.Context, ticket string) (UserInfo, error) {
	payload, err := t.rdb.GetDel(ctx, "ticket:"+ticket).Result()
	if errors.Is(err, redis.Nil) {
		return UserInfo{}, ErrInvalidTicket
	}
	if err != nil {
		return UserInfo{}, fmt.Errorf("redeem ticket: %w", err)
	}
	var user UserInfo
	if err := json.Unmarshal([]byte(payload), &user); err != nil {
		return UserInfo{}, fmt.Errorf("corrupt ticket payload: %w", err)
	}
	return user, nil
}
