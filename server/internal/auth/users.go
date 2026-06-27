package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Sentinel errors let the HTTP layer choose a status code without inspecting
// driver-specific error strings. Both register and login map their "no"
// answers to these.
var (
	// ErrUsernameTaken is returned by Register when the name already exists.
	ErrUsernameTaken = errors.New("username already taken")
	// ErrBadCredentials covers both "no such user" and "wrong password" — the
	// caller deliberately cannot tell which, so the endpoint can't be used to
	// enumerate valid usernames.
	ErrBadCredentials = errors.New("invalid username or password")
)

// Users is the Postgres-backed account store and the only place credentials
// live (proposal §16: "passwords handled only by API"). It hashes on
// Register and verifies on Authenticate; callers pass plaintext and never
// see a hash. The returned UUID is the identity everything else (inventory,
// bases, reputation) references.
type Users struct {
	pool *pgxpool.Pool
}

func NewUsers(pool *pgxpool.Pool) *Users {
	return &Users{pool: pool}
}

// Register creates a new account, hashing the password before it touches the
// database. A unique-violation on username comes back as ErrUsernameTaken so
// the handler can return 409; the SQLSTATE check is isolated to the one
// helper below rather than smeared across the HTTP layer.
func (u *Users) Register(ctx context.Context, username, password string) (UserInfo, error) {
	hash, err := HashPassword(password)
	if err != nil {
		return UserInfo{}, fmt.Errorf("hash password: %w", err)
	}

	var id string
	err = u.pool.QueryRow(ctx, `
		INSERT INTO users (username, password_hash) VALUES ($1, $2)
		RETURNING id::text`, username, hash).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return UserInfo{}, ErrUsernameTaken
		}
		return UserInfo{}, fmt.Errorf("insert user %q: %w", username, err)
	}
	return UserInfo{UserID: id, Username: username}, nil
}

// Authenticate verifies a password against the stored hash. A missing user
// and a wrong password both return ErrBadCredentials, so the caller can't
// distinguish them. (A missing user skips the argon2 work, a minor timing
// signal that's acceptable for this endpoint and will sit behind rate
// limiting before it's public.)
func (u *Users) Authenticate(ctx context.Context, username, password string) (UserInfo, error) {
	var id, hash string
	err := u.pool.QueryRow(ctx, `
		SELECT id::text, password_hash FROM users
		WHERE username = $1`, username).Scan(&id, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return UserInfo{}, ErrBadCredentials
	}
	if err != nil {
		return UserInfo{}, fmt.Errorf("lookup user %q: %w", username, err)
	}

	ok, err := VerifyPassword(password, hash)
	if err != nil {
		// A corrupt/unreadable stored hash is an internal fault, not a wrong
		// password — surface it so it gets noticed, don't mask it as a 401.
		return UserInfo{}, fmt.Errorf("verify user %q: %w", username, err)
	}
	if !ok {
		return UserInfo{}, ErrBadCredentials
	}

	// last_login is best-effort: a failure here shouldn't deny a valid login.
	_, _ = u.pool.Exec(ctx, `UPDATE users SET last_login = now() WHERE id = $1`, id)
	return UserInfo{UserID: id, Username: username}, nil
}

// isUniqueViolation reports whether err is a Postgres unique-constraint
// violation (SQLSTATE 23505), the signal that a username is taken.
func isUniqueViolation(err error) bool {
	var pgErr interface{ SQLState() string }
	return errors.As(err, &pgErr) && pgErr.SQLState() == "23505"
}
