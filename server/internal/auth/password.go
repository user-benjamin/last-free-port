package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Password hashing is argon2id, the memory-hard winner of the Password
// Hashing Competition and the current OWASP first choice. The cost
// parameters are stored *inside* each hash (PHC string format), so they can
// be raised later without a migration: old hashes keep verifying against
// their own recorded cost, new hashes use the new one.
//
// Only the API service ever calls this (proposal §16: "passwords handled
// only by API"). The game server never sees a password or a hash.

// argon2Params are the cost knobs for newly created hashes. memoryKiB is the
// dominant defense — 64 MiB per hash makes large-scale cracking expensive
// while staying comfortable for a login-rate endpoint.
type argon2Params struct {
	memoryKiB uint32
	time      uint32
	threads   uint8
	keyLen    uint32
	saltLen   uint32
}

var defaultParams = argon2Params{
	memoryKiB: 64 * 1024, // 64 MiB
	time:      2,
	threads:   4,
	keyLen:    32,
	saltLen:   16,
}

// ErrMalformedHash means a stored hash isn't a string we wrote — a corrupt
// row or a hand-edited column. Treated as an internal error, never as "wrong
// password", so it can't silently let anyone in.
var ErrMalformedHash = errors.New("malformed password hash")

// HashPassword returns a self-describing PHC string:
//
//	$argon2id$v=19$m=65536,t=2,p=4$<b64 salt>$<b64 hash>
//
// The salt is fresh per call, so two identical passwords never share a hash.
func HashPassword(password string) (string, error) {
	p := defaultParams
	salt := make([]byte, p.saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("read salt: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, p.time, p.memoryKiB, p.threads, p.keyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.memoryKiB, p.time, p.threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// VerifyPassword reports whether password matches encodedHash. It rederives
// the key using the cost parameters recorded in the hash itself, then
// compares in constant time so a timing side channel can't leak how much of
// the hash was correct.
//
// A false result with a nil error means "wrong password". A non-nil error
// means the stored hash was unreadable (ErrMalformedHash) — the caller must
// treat that as a failure, not a match.
func VerifyPassword(password, encodedHash string) (bool, error) {
	parts := strings.Split(encodedHash, "$")
	// ["", "argon2id", "v=19", "m=...,t=...,p=...", salt, hash]
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, ErrMalformedHash
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false, ErrMalformedHash
	}

	var p argon2Params
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.memoryKiB, &p.time, &p.threads); err != nil {
		return false, ErrMalformedHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, ErrMalformedHash
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, ErrMalformedHash
	}

	got := argon2.IDKey([]byte(password), salt, p.time, p.memoryKiB, p.threads, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}
