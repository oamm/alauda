package auth

import (
	"strings"
	"testing"
)

func TestNewTokenReturnsSecretAndHash(t *testing.T) {
	secret, hash, err := NewToken()
	if err != nil {
		t.Fatalf("NewToken() error = %v", err)
	}
	if !strings.HasPrefix(secret, tokenPrefix) {
		t.Fatalf("secret prefix = %q, want %q", secret, tokenPrefix)
	}
	if hash == "" || hash == secret {
		t.Fatalf("hash was not generated correctly")
	}
	if HashToken(secret) != hash {
		t.Fatalf("HashToken(secret) did not match returned hash")
	}
}

func TestHashTokenTrimsWhitespace(t *testing.T) {
	if HashToken(" sr_test ") != HashToken("sr_test") {
		t.Fatalf("HashToken should trim surrounding whitespace")
	}
}
