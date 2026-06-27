package auth

import (
	"strings"
	"testing"
)

func TestHashVerifyRoundTrip(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$v=19$") {
		t.Fatalf("unexpected hash format: %s", hash)
	}

	ok, err := VerifyPassword("correct horse battery staple", hash)
	if err != nil || !ok {
		t.Fatalf("right password should verify: ok=%v err=%v", ok, err)
	}

	ok, err = VerifyPassword("wrong password", hash)
	if err != nil {
		t.Fatalf("wrong password is not an error: %v", err)
	}
	if ok {
		t.Fatal("wrong password must not verify")
	}
}

func TestHashIsSaltedPerCall(t *testing.T) {
	a, _ := HashPassword("same")
	b, _ := HashPassword("same")
	if a == b {
		t.Fatal("identical passwords must not produce identical hashes (missing salt)")
	}
}

func TestVerifyRejectsMalformedHash(t *testing.T) {
	for _, bad := range []string{
		"", "not-a-hash", "$argon2id$v=19$bad", "$bcrypt$...",
		"$argon2id$v=18$m=65536,t=2,p=4$c2FsdA$aGFzaA", // wrong version
	} {
		ok, err := VerifyPassword("x", bad)
		if ok {
			t.Errorf("malformed hash %q must not verify", bad)
		}
		if err == nil {
			t.Errorf("malformed hash %q should report an error, not a silent false", bad)
		}
	}
}
